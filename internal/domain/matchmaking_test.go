package domain

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBuildMatch_8Players_DiffZero(t *testing.T) {
	players := makePlayers([]int{2500, 2450, 2400, 2350, 2300, 2250, 2200, 2150})

	result, err := BuildMatch(players, 42)
	if err != nil {
		t.Fatalf("BuildMatch returned error: %v", err)
	}

	if result.Diff != 0 {
		t.Fatalf("expected Diff=0, got %d", result.Diff)
	}
	if len(result.TeamA) != 4 || len(result.TeamB) != 4 {
		t.Fatalf("expected TeamA/TeamB to be 4 players each, got %d/%d", len(result.TeamA), len(result.TeamB))
	}
	if len(result.Spectators) != 0 {
		t.Fatalf("expected no spectators, got %d", len(result.Spectators))
	}
}

func TestBuildMatch_10Players_Has2Spectators(t *testing.T) {
	players := makePlayers([]int{2600, 2550, 2500, 2450, 2400, 2350, 2300, 2250, 1800, 1700})

	result, err := BuildMatch(players, 42)
	if err != nil {
		t.Fatalf("BuildMatch returned error: %v", err)
	}

	if len(result.TeamA) != 4 || len(result.TeamB) != 4 {
		t.Fatalf("expected TeamA/TeamB to be 4 players each, got %d/%d", len(result.TeamA), len(result.TeamB))
	}
	if len(result.Spectators) != 2 {
		t.Fatalf("expected 2 spectators, got %d", len(result.Spectators))
	}
}

func TestBuildMatch_DeterministicWithSameSeed(t *testing.T) {
	players := makePlayers([]int{2600, 2550, 2500, 2450, 2400, 2350, 2300, 2250, 1800, 1700})
	seed := int64(20260302)

	got1, err := BuildMatch(players, seed)
	if err != nil {
		t.Fatalf("first BuildMatch returned error: %v", err)
	}

	got2, err := BuildMatch(players, seed)
	if err != nil {
		t.Fatalf("second BuildMatch returned error: %v", err)
	}

	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("expected deterministic result with same seed; got1=%+v got2=%+v", got1, got2)
	}
}

func TestBuildMatch_ErrorsWhenPlayersLessThan8(t *testing.T) {
	players := makePlayers([]int{2500, 2450, 2400, 2350, 2300, 2250, 2200})

	_, err := BuildMatch(players, 1)
	if err == nil {
		t.Fatal("expected error for less than 8 players, got nil")
	}
	if err != ErrNotEnoughPlayers {
		t.Fatalf("expected ErrNotEnoughPlayers, got %v", err)
	}
}

func TestBuildMatch_ErrorsWhenPlayersMoreThan10(t *testing.T) {
	players := makePlayers([]int{2600, 2550, 2500, 2450, 2400, 2350, 2300, 2250, 2200, 2150, 2100})

	_, err := BuildMatch(players, 1)
	if err == nil {
		t.Fatal("expected error for more than 10 players, got nil")
	}
	if err != ErrTooManyPlayers {
		t.Fatalf("expected ErrTooManyPlayers, got %v", err)
	}
}

func makePlayers(powers []int) []Player {
	players := make([]Player, 0, len(powers))
	for i, p := range powers {
		players = append(players, Player{
			ID:     fmt.Sprintf("p%d", i+1),
			Name:   fmt.Sprintf("player-%d", i+1),
			XPower: p,
		})
	}
	return players
}
