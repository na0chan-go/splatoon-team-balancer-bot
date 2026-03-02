package store

import (
	"fmt"
	"path/filepath"
	"reflect"
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
	result := domain.MatchResult{
		TeamA:      players[:4],
		TeamB:      players[4:8],
		Spectators: nil,
		SumA:       9400,
		SumB:       7800,
		Diff:       1600,
	}
	s.SaveLastMatch("g1", "c1", 123, players, result)
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
	if !reflect.DeepEqual(state.LastPlayersSnapshot, players) {
		t.Fatalf("unexpected LastPlayersSnapshot: %+v", state.LastPlayersSnapshot)
	}
	if !reflect.DeepEqual(state.LastResult, result) {
		t.Fatalf("unexpected LastResult: %+v", state.LastResult)
	}
}

func TestSQLiteStoreJoinUpdatesDuplicate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer s.Close()

	created, err := s.Join("g1", "c1", domain.Player{ID: "u1", Name: "p1", XPower: 2500})
	if err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	if !created {
		t.Fatal("expected first join to create participant")
	}

	created, err = s.Join("g1", "c1", domain.Player{ID: "u1", Name: "p1", XPower: 2700})
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

func TestSQLiteStoreJoinRejectsOver10Players(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer s.Close()

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

	_, err = s.Join("g1", "c1", domain.Player{
		ID:     "u-over",
		Name:   "p-over",
		XPower: 2111,
	})
	if err != ErrRoomFull {
		t.Fatalf("expected ErrRoomFull, got %v", err)
	}
}
