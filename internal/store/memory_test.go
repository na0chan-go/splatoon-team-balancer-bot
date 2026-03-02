package store

import (
	"errors"
	"fmt"
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
