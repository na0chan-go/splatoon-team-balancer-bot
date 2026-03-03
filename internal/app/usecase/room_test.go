package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

type fixedRNG struct {
	values []int64
	index  int
}

func (r *fixedRNG) Int63() int64 {
	if len(r.values) == 0 {
		return 1
	}
	if r.index >= len(r.values) {
		return r.values[len(r.values)-1]
	}
	v := r.values[r.index]
	r.index++
	return v
}

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time { return c.now }

type pseudoStore struct {
	*store.MemoryStore
	exportMatches []store.MatchRecord
	exportStats   []store.PlayerStat
	exportErr     error
	lastScope     string
	lastLimit     int
	lastGuildID   string
	lastChannelID string
}

func (s *pseudoStore) GetExportData(guildID, channelID, scope string, limit int) ([]store.MatchRecord, []store.PlayerStat, error) {
	s.lastGuildID = guildID
	s.lastChannelID = channelID
	s.lastScope = scope
	s.lastLimit = limit
	if s.exportErr != nil {
		return nil, nil, s.exportErr
	}
	return s.exportMatches, s.exportStats, nil
}

func TestRoomServiceMainFlowWithPseudoStore(t *testing.T) {
	ps := &pseudoStore{
		MemoryStore: store.NewMemoryStore(),
		exportMatches: []store.MatchRecord{
			{ID: 1, GuildID: "g1", ChannelID: "c1", WinnerTeam: "alpha"},
		},
		exportStats: []store.PlayerStat{
			{UserID: "u1", RatingDelta: 10, Wins: 1, Losses: 0},
		},
	}
	svc := NewRoomService(ps)
	svc.SetRNG(&fixedRNG{values: []int64{11, 22, 33}})
	svc.SetClock(fakeClock{now: time.Unix(1700000000, 0)})

	guildID := "g1"
	channelID := "c1"
	players := makePlayers(9)

	created, onboarding, err := svc.Join(guildID, channelID, players[0])
	if err != nil {
		t.Fatalf("Join first failed: %v", err)
	}
	if !created || !onboarding {
		t.Fatalf("expected created=true onboarding=true, got created=%v onboarding=%v", created, onboarding)
	}
	created, onboarding, err = svc.Join(guildID, channelID, domain.Player{
		ID: players[0].ID, Name: players[0].Name, XPower: players[0].XPower + 50,
	})
	if err != nil {
		t.Fatalf("Join update failed: %v", err)
	}
	if created || onboarding {
		t.Fatalf("expected update with no onboarding, got created=%v onboarding=%v", created, onboarding)
	}
	for i := 1; i < len(players); i++ {
		if _, _, err := svc.Join(guildID, channelID, players[i]); err != nil {
			t.Fatalf("Join player %d failed: %v", i+1, err)
		}
	}

	if err := svc.UpdateRoomSetting(guildID, channelID, domain.RoomSettingPauseDefaultMatches, "4"); err != nil {
		t.Fatalf("UpdateRoomSetting failed: %v", err)
	}
	settings, err := svc.RoomSettings(guildID, channelID)
	if err != nil {
		t.Fatalf("RoomSettings failed: %v", err)
	}
	if settings.PauseDefaultMatches != 4 {
		t.Fatalf("expected pause default 4, got %d", settings.PauseDefaultMatches)
	}

	makeResult, err := svc.Make(guildID, channelID, settings)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}
	if len(makeResult.TeamA) != 4 || len(makeResult.TeamB) != 4 {
		t.Fatalf("unexpected make team sizes: A=%d B=%d", len(makeResult.TeamA), len(makeResult.TeamB))
	}
	stateAfterMake, ok := ps.GetState(guildID, channelID)
	if !ok {
		t.Fatal("expected state after make")
	}
	if stateAfterMake.LastSeed != 11 {
		t.Fatalf("expected LastSeed=11, got %d", stateAfterMake.LastSeed)
	}

	if err := svc.SetPause(guildID, channelID, "u9", 2, "rest"); err != nil {
		t.Fatalf("SetPause failed: %v", err)
	}
	nextResult, err := svc.Next(guildID, channelID, settings)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if len(nextResult.TeamA) != 4 || len(nextResult.TeamB) != 4 {
		t.Fatalf("unexpected next team sizes: A=%d B=%d", len(nextResult.TeamA), len(nextResult.TeamB))
	}
	stateAfterNext, _ := ps.GetState(guildID, channelID)
	if stateAfterNext.LastSeed != 22 {
		t.Fatalf("expected LastSeed=22 after next, got %d", stateAfterNext.LastSeed)
	}
	u9 := findPlayer(stateAfterNext.Players, "u9")
	if u9.PauseRemaining != 1 {
		t.Fatalf("expected u9 pause remaining 1 after next decrement, got %d", u9.PauseRemaining)
	}

	undone, err := svc.Undo(guildID, channelID)
	if err != nil {
		t.Fatalf("Undo failed: %v", err)
	}
	if !undone {
		t.Fatal("expected Undo to restore previous state")
	}
	stateAfterUndo, _ := ps.GetState(guildID, channelID)
	if stateAfterUndo.LastSeed != 11 {
		t.Fatalf("expected LastSeed restored to 11, got %d", stateAfterUndo.LastSeed)
	}
	u9 = findPlayer(stateAfterUndo.Players, "u9")
	if u9.PauseRemaining != 2 {
		t.Fatalf("expected u9 pause remaining restored to 2, got %d", u9.PauseRemaining)
	}

	if err := svc.Resume(guildID, channelID, "u9"); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	stateAfterResume, _ := ps.GetState(guildID, channelID)
	u9 = findPlayer(stateAfterResume.Players, "u9")
	if u9.PauseRemaining != 0 {
		t.Fatalf("expected u9 resumed, got pause remaining %d", u9.PauseRemaining)
	}

	if err := svc.RecordResult(guildID, channelID, "alpha"); err != nil {
		t.Fatalf("RecordResult failed: %v", err)
	}
	stats := ps.GetPlayerStats([]string{stateAfterUndo.LastResult.TeamA[0].ID, stateAfterUndo.LastResult.TeamB[0].ID})
	alpha := stats[stateAfterUndo.LastResult.TeamA[0].ID]
	bravo := stats[stateAfterUndo.LastResult.TeamB[0].ID]
	if alpha.Wins != 1 || alpha.RatingDelta != 10 {
		t.Fatalf("expected alpha player to get win/+10, got %+v", alpha)
	}
	if bravo.Losses != 1 || bravo.RatingDelta != -10 {
		t.Fatalf("expected bravo player to get loss/-10, got %+v", bravo)
	}

	_, _, err = svc.Export(guildID, channelID, "room", 50)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if ps.lastGuildID != guildID || ps.lastChannelID != channelID || ps.lastScope != "room" || ps.lastLimit != 50 {
		t.Fatalf("unexpected export query args: guild=%s channel=%s scope=%s limit=%d", ps.lastGuildID, ps.lastChannelID, ps.lastScope, ps.lastLimit)
	}
}

func makePlayers(n int) []domain.Player {
	out := make([]domain.Player, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, domain.Player{
			ID:     fmt.Sprintf("u%d", i),
			Name:   fmt.Sprintf("p%d", i),
			XPower: 2500 - i*30,
		})
	}
	return out
}

func findPlayer(players []domain.Player, userID string) domain.Player {
	for _, p := range players {
		if p.ID == userID {
			return p
		}
	}
	return domain.Player{}
}
