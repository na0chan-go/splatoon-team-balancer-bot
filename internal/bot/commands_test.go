package bot

import (
	"errors"
	"testing"

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
