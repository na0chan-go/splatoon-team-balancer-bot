package store

import (
	"path/filepath"
	"testing"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

func TestSQLiteStorePersistsRoomState(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}

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
	s.SaveLastMatch("g1", "c1", 123, players, domain.MatchResult{SumA: 1, SumB: 2, Diff: 1})
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("reopen NewSQLiteStore failed: %v", err)
	}
	defer s2.Close()

	list := s2.List("g1", "c1")
	if got, want := len(list), 8; got != want {
		t.Fatalf("unexpected persisted list size: got %d want %d", got, want)
	}
	state, ok := s2.GetState("g1", "c1")
	if !ok {
		t.Fatal("expected persisted state")
	}
	if state.LastSeed != 123 {
		t.Fatalf("expected LastSeed=123, got %d", state.LastSeed)
	}
}
