package bot

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

func TestRerollFromLastSnapshotErrorsWithoutMake(t *testing.T) {
	roomStore = store.NewMemoryStore()

	_, err := rerollFromLastSnapshot("g1", "c1", 1)
	if !errors.Is(err, ErrNoLastMake) {
		t.Fatalf("expected ErrNoLastMake, got %v", err)
	}
}

func TestRunMatchAndStoreSavesStateForReroll(t *testing.T) {
	roomStore = store.NewMemoryStore()
	players := []domain.Player{
		{ID: "u1", Name: "p1", XPower: 2500},
		{ID: "u2", Name: "p2", XPower: 2450},
		{ID: "u3", Name: "p3", XPower: 2400},
		{ID: "u4", Name: "p4", XPower: 2350},
		{ID: "u5", Name: "p5", XPower: 2300},
		{ID: "u6", Name: "p6", XPower: 2250},
		{ID: "u7", Name: "p7", XPower: 2200},
		{ID: "u8", Name: "p8", XPower: 2150},
		{ID: "u9", Name: "p9", XPower: 2100},
	}

	if _, err := runMatchAndStore("g1", "c1", players, 100); err != nil {
		t.Fatalf("runMatchAndStore failed: %v", err)
	}
	state, ok := roomStore.GetState("g1", "c1")
	if !ok {
		t.Fatal("expected room state to exist")
	}
	if state.LastSeed != 100 {
		t.Fatalf("expected LastSeed=100, got %d", state.LastSeed)
	}
	if got, want := len(state.LastPlayersSnapshot), 9; got != want {
		t.Fatalf("expected LastPlayersSnapshot len=%d, got %d", want, got)
	}

	if _, err := rerollFromLastSnapshot("g1", "c1", 200); err != nil {
		t.Fatalf("rerollFromLastSnapshot failed: %v", err)
	}
	state, _ = roomStore.GetState("g1", "c1")
	if state.LastSeed != 200 {
		t.Fatalf("expected LastSeed=200 after reroll, got %d", state.LastSeed)
	}
}

func TestHasResetPermission(t *testing.T) {
	adminInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Permissions: discordgo.PermissionAdministrator},
		},
	}
	if !hasResetPermission(adminInteraction) {
		t.Fatal("expected administrator to have reset permission")
	}

	manageInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Permissions: discordgo.PermissionManageGuild},
		},
	}
	if !hasResetPermission(manageInteraction) {
		t.Fatal("expected manage server permission to have reset permission")
	}

	normalInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Permissions: discordgo.PermissionViewChannel},
		},
	}
	if hasResetPermission(normalInteraction) {
		t.Fatal("expected normal member to not have reset permission")
	}
}

func TestNextMatchFromCurrentParticipantsRequiresPreviousMake(t *testing.T) {
	roomStore = store.NewMemoryStore()

	_, err := nextMatchFromCurrentParticipants("g1", "c1", 1)
	if !errors.Is(err, ErrNoPreviousMatch) {
		t.Fatalf("expected ErrNoPreviousMatch, got %v", err)
	}
}

func TestNextMatchFromCurrentParticipantsRequiresAtLeast8Players(t *testing.T) {
	roomStore = store.NewMemoryStore()

	players := make([]domain.Player, 0, 8)
	for i := 1; i <= 8; i++ {
		p := domain.Player{
			ID:     fmt.Sprintf("u%d", i),
			Name:   fmt.Sprintf("p%d", i),
			XPower: 2000 + i*10,
		}
		players = append(players, p)
		if _, err := roomStore.Join("g1", "c1", p); err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}

	if _, err := runMatchAndStore("g1", "c1", players, 42); err != nil {
		t.Fatalf("runMatchAndStore failed: %v", err)
	}
	if err := roomStore.SetPause("g1", "c1", "u7", 1, "break"); err != nil {
		t.Fatalf("SetPause failed: %v", err)
	}

	if err := roomStore.Leave("g1", "c1", "u8"); err != nil {
		t.Fatalf("leave failed: %v", err)
	}

	_, err := nextMatchFromCurrentParticipants("g1", "c1", 43)
	if !errors.Is(err, domain.ErrNotEnoughPlayers) {
		t.Fatalf("expected ErrNotEnoughPlayers, got %v", err)
	}

	state, ok := roomStore.GetState("g1", "c1")
	if !ok {
		t.Fatal("expected state to exist")
	}
	for _, p := range state.Players {
		if p.ID == "u7" && p.PauseRemaining != 0 {
			t.Fatalf("expected u7 to auto-resume after decrement, got %d", p.PauseRemaining)
		}
	}
}

func TestNextMatchSkipsPausedPlayersAndDecrementsOnSuccess(t *testing.T) {
	roomStore = store.NewMemoryStore()

	players := make([]domain.Player, 0, 9)
	for i := 1; i <= 9; i++ {
		p := domain.Player{
			ID:     fmt.Sprintf("u%d", i),
			Name:   fmt.Sprintf("p%d", i),
			XPower: 2100 + i*10,
		}
		players = append(players, p)
		if _, err := roomStore.Join("g1", "c1", p); err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}
	if err := roomStore.SetPause("g1", "c1", "u9", 2, "rest"); err != nil {
		t.Fatalf("SetPause failed: %v", err)
	}

	if _, err := runMatchAndStore("g1", "c1", players, 100); err != nil {
		t.Fatalf("runMatchAndStore failed: %v", err)
	}

	got, err := nextMatchFromCurrentParticipants("g1", "c1", 101)
	if err != nil {
		t.Fatalf("nextMatchFromCurrentParticipants failed: %v", err)
	}
	if len(got.Spectators) != 0 {
		t.Fatalf("expected no spectators with 8 active players, got %d", len(got.Spectators))
	}
	for _, p := range append(append([]domain.Player{}, got.TeamA...), got.TeamB...) {
		if p.ID == "u9" {
			t.Fatal("expected paused player u9 to be excluded from next match")
		}
	}

	state, _ := roomStore.GetState("g1", "c1")
	for _, p := range state.Players {
		if p.ID == "u9" && p.PauseRemaining != 1 {
			t.Fatalf("expected u9 pause remaining to decrement to 1, got %d", p.PauseRemaining)
		}
	}
}

func TestUndoLastRoomStateAfterNextRestoresPreviousState(t *testing.T) {
	roomStore = store.NewMemoryStore()

	players := make([]domain.Player, 0, 9)
	for i := 1; i <= 9; i++ {
		p := domain.Player{
			ID:     fmt.Sprintf("u%d", i),
			Name:   fmt.Sprintf("p%d", i),
			XPower: 2200 + i*10,
		}
		players = append(players, p)
		if _, err := roomStore.Join("g1", "c1", p); err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}

	if _, err := runMatchAndStore("g1", "c1", players, 100); err != nil {
		t.Fatalf("runMatchAndStore failed: %v", err)
	}
	makeState, _ := roomStore.GetState("g1", "c1")

	if _, err := nextMatchFromCurrentParticipants("g1", "c1", 101); err != nil {
		t.Fatalf("nextMatchFromCurrentParticipants failed: %v", err)
	}
	nextState, _ := roomStore.GetState("g1", "c1")
	if nextState.LastSeed == makeState.LastSeed {
		t.Fatal("expected next state to differ from make state before undo")
	}

	ok, err := undoLastRoomState("g1", "c1")
	if err != nil {
		t.Fatalf("undoLastRoomState failed: %v", err)
	}
	if !ok {
		t.Fatal("expected undoLastRoomState to restore previous state")
	}

	restored, _ := roomStore.GetState("g1", "c1")
	if restored.LastSeed != makeState.LastSeed {
		t.Fatalf("expected LastSeed restored to %d, got %d", makeState.LastSeed, restored.LastSeed)
	}
}
