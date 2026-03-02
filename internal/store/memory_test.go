package store

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

func TestMemoryStoreJoinUpdatesDuplicate(t *testing.T) {
	s := NewMemoryStore()
	player := domain.Player{ID: "u1", Name: "p1", XPower: 2500}

	created, err := s.Join("g1", "c1", player)
	if err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	if !created {
		t.Fatal("expected first join to create participant")
	}

	updatedPlayer := domain.Player{ID: "u1", Name: "p1", XPower: 2700}
	created, err = s.Join("g1", "c1", updatedPlayer)
	if err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if created {
		t.Fatal("expected second join to update participant")
	}

	list := s.List("g1", "c1")
	if got, want := len(list), 1; got != want {
		t.Fatalf("unexpected list size: got %d want %d", got, want)
	}
	if list[0].XPower != 2700 {
		t.Fatalf("expected updated xpower 2700, got %d", list[0].XPower)
	}
}

func TestMemoryStoreJoinRejectsOver10Players(t *testing.T) {
	s := NewMemoryStore()
	for i := 1; i <= 10; i++ {
		_, err := s.Join("g1", "c1", domain.Player{
			ID:     fmt.Sprintf("u%d", i),
			Name:   fmt.Sprintf("p%d", i),
			XPower: 2000 + i,
		})
		if err != nil {
			t.Fatalf("join %d failed: %v", i, err)
		}
	}

	_, err := s.Join("g1", "c1", domain.Player{
		ID:     "u11",
		Name:   "p11",
		XPower: 2111,
	})
	if !errors.Is(err, ErrRoomFull) {
		t.Fatalf("expected ErrRoomFull, got %v", err)
	}
}

func TestMemoryStoreListReturnsXPowerDesc(t *testing.T) {
	s := NewMemoryStore()
	input := []domain.Player{
		{ID: "u1", Name: "p1", XPower: 2200},
		{ID: "u2", Name: "p2", XPower: 2500},
		{ID: "u3", Name: "p3", XPower: 2400},
	}
	for _, p := range input {
		if _, err := s.Join("g1", "c1", p); err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}

	list := s.List("g1", "c1")
	if got, want := len(list), 3; got != want {
		t.Fatalf("unexpected list size: got %d want %d", got, want)
	}
	if list[0].ID != "u2" || list[1].ID != "u3" || list[2].ID != "u1" {
		t.Fatalf("unexpected order: %+v", list)
	}
}

func TestMemoryStoreJoinRejectsOutOfRangeXPower(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.Join("g1", "c1", domain.Player{ID: "u1", Name: "p1", XPower: -1})
	if !errors.Is(err, ErrInvalidXPower) {
		t.Fatalf("expected ErrInvalidXPower for -1, got %v", err)
	}

	_, err = s.Join("g1", "c1", domain.Player{ID: "u2", Name: "p2", XPower: 5001})
	if !errors.Is(err, ErrInvalidXPower) {
		t.Fatalf("expected ErrInvalidXPower for 5001, got %v", err)
	}
}

func TestMemoryStoreSaveAndGetLastMatchState(t *testing.T) {
	s := NewMemoryStore()
	players := []domain.Player{
		{ID: "u1", Name: "p1", XPower: 2500},
		{ID: "u2", Name: "p2", XPower: 2400},
		{ID: "u3", Name: "p3", XPower: 2300},
		{ID: "u4", Name: "p4", XPower: 2200},
		{ID: "u5", Name: "p5", XPower: 2100},
		{ID: "u6", Name: "p6", XPower: 2000},
		{ID: "u7", Name: "p7", XPower: 1900},
		{ID: "u8", Name: "p8", XPower: 1800},
		{ID: "u9", Name: "p9", XPower: 1700},
	}
	result := domain.MatchResult{
		TeamA: players[:4], TeamB: players[4:8], Spectators: []domain.Player{players[8]},
		SumA: 9400, SumB: 7800, Diff: 1600,
	}

	s.SaveLastMatch("g1", "c1", 42, players, result)

	state, ok := s.GetState("g1", "c1")
	if !ok {
		t.Fatal("expected state to exist")
	}
	if state.LastSeed != 42 {
		t.Fatalf("expected LastSeed=42, got %d", state.LastSeed)
	}
	if !reflect.DeepEqual(state.LastPlayersSnapshot, players) {
		t.Fatalf("unexpected LastPlayersSnapshot: %+v", state.LastPlayersSnapshot)
	}
	if !reflect.DeepEqual(state.LastResult, result) {
		t.Fatalf("unexpected LastResult: %+v", state.LastResult)
	}
	if got := state.SpectatorHistory["u9"].SpectatorCount; got != 1 {
		t.Fatalf("expected spectator count of u9 to be 1, got %d", got)
	}
	if got := state.SpectatorHistory["u9"].LastSpectatedAt; got == 0 {
		t.Fatal("expected LastSpectatedAt to be set")
	}
}

func TestMemoryStoreResetRoomClearsState(t *testing.T) {
	s := NewMemoryStore()
	players := []domain.Player{
		{ID: "u1", Name: "p1", XPower: 2500},
		{ID: "u2", Name: "p2", XPower: 2400},
		{ID: "u3", Name: "p3", XPower: 2300},
		{ID: "u4", Name: "p4", XPower: 2200},
		{ID: "u5", Name: "p5", XPower: 2100},
		{ID: "u6", Name: "p6", XPower: 2000},
		{ID: "u7", Name: "p7", XPower: 1900},
		{ID: "u8", Name: "p8", XPower: 1800},
	}
	for _, p := range players {
		if _, err := s.Join("g1", "c1", p); err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}
	s.SaveLastMatch("g1", "c1", 1, players, domain.MatchResult{SumA: 1, SumB: 2, Diff: 1})

	s.ResetRoom("g1", "c1")

	if got := s.List("g1", "c1"); len(got) != 0 {
		t.Fatalf("expected empty players after reset, got %+v", got)
	}
	if _, ok := s.GetState("g1", "c1"); ok {
		t.Fatal("expected room state to be removed after reset")
	}
}

func TestMemoryStoreRecordMatchResultUpdatesStatsAndClamp(t *testing.T) {
	s := NewMemoryStore()
	result := domain.MatchResult{
		TeamA: []domain.Player{
			{ID: "u1", Name: "p1", XPower: 2400},
			{ID: "u2", Name: "p2", XPower: 2300},
			{ID: "u3", Name: "p3", XPower: 2200},
			{ID: "u4", Name: "p4", XPower: 2100},
		},
		TeamB: []domain.Player{
			{ID: "u5", Name: "p5", XPower: 2400},
			{ID: "u6", Name: "p6", XPower: 2300},
			{ID: "u7", Name: "p7", XPower: 2200},
			{ID: "u8", Name: "p8", XPower: 2100},
		},
	}

	for i := 0; i < 30; i++ {
		if err := s.RecordMatchResult("g1", "c1", "alpha", result); err != nil {
			t.Fatalf("RecordMatchResult failed: %v", err)
		}
	}

	stats := s.GetPlayerStats([]string{"u1", "u5"})
	if got := stats["u1"].Rating; got != 200 {
		t.Fatalf("expected winner rating clamped to 200, got %d", got)
	}
	if got := stats["u5"].Rating; got != -200 {
		t.Fatalf("expected loser rating clamped to -200, got %d", got)
	}
	if stats["u1"].Wins == 0 || stats["u5"].Losses == 0 {
		t.Fatalf("expected wins/losses to be updated: %+v", stats)
	}
}
