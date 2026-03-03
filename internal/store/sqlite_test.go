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
		{ID: "u9", Name: "p9", XPower: 1700},
	}
	for _, p := range players {
		if _, err := s.Join("g1", "c1", p); err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}
	result := domain.MatchResult{
		TeamA:      players[:4],
		TeamB:      players[4:8],
		Spectators: []domain.Player{players[8]},
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
	if got, want := len(list), 9; got != want {
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
	if got := state.SpectatorHistory["u9"].SpectatorCount; got != 1 {
		t.Fatalf("expected spectator count of u9 to be 1, got %d", got)
	}
	if got := state.SpectatorHistory["u9"].LastSpectatedAt; got == 0 {
		t.Fatal("expected LastSpectatedAt to be set")
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

func TestSQLiteStoreRecordMatchResultUpdatesStatsAndHistory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer s.Close()

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

	if err := s.RecordMatchResult("g1", "c1", "alpha", result); err != nil {
		t.Fatalf("RecordMatchResult failed: %v", err)
	}

	stats := s.GetPlayerStats([]string{"u1", "u5"})
	if got := stats["u1"].Rating; got != 10 {
		t.Fatalf("expected winner rating 10, got %d", got)
	}
	if got := stats["u5"].Rating; got != -10 {
		t.Fatalf("expected loser rating -10, got %d", got)
	}

	var matchCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM matches`).Scan(&matchCount); err != nil {
		t.Fatalf("failed to count matches: %v", err)
	}
	if matchCount != 1 {
		t.Fatalf("expected 1 match record, got %d", matchCount)
	}
}

func TestSQLiteStoreUndoRoomStatePersists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}

	for i := 1; i <= 9; i++ {
		_, err := s.Join("g1", "c1", domain.Player{
			ID:     fmt.Sprintf("u%d", i),
			Name:   fmt.Sprintf("p%d", i),
			XPower: 2000 + i,
		})
		if err != nil {
			t.Fatalf("join failed: %v", err)
		}
	}
	base := s.List("g1", "c1")
	s.SaveLastMatch("g1", "c1", 100, base, domain.MatchResult{
		TeamA: base[:4], TeamB: base[4:8], Spectators: []domain.Player{base[8]},
	})
	s.SnapshotRoomState("g1", "c1")

	if err := s.Leave("g1", "c1", "u9"); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after := s.List("g1", "c1")
	s.SaveLastMatch("g1", "c1", 200, after, domain.MatchResult{
		TeamA: after[:4], TeamB: after[4:8],
	})

	ok, err := s.UndoRoomState("g1", "c1")
	if err != nil {
		t.Fatalf("UndoRoomState failed: %v", err)
	}
	if !ok {
		t.Fatal("expected undo to restore previous state")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer s2.Close()

	state, exists := s2.GetState("g1", "c1")
	if !exists {
		t.Fatal("expected state to exist")
	}
	if state.LastSeed != 100 {
		t.Fatalf("expected LastSeed restored to 100, got %d", state.LastSeed)
	}
	if got := len(state.Players); got != 9 {
		t.Fatalf("expected players restored to 9, got %d", got)
	}
}

func TestSQLiteStoreTryMarkOnboardingShownPersists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}

	first, err := s.TryMarkOnboardingShown("g1", "c1")
	if err != nil {
		t.Fatalf("first TryMarkOnboardingShown failed: %v", err)
	}
	if !first {
		t.Fatal("expected first mark to return true")
	}
	second, err := s.TryMarkOnboardingShown("g1", "c1")
	if err != nil {
		t.Fatalf("second TryMarkOnboardingShown failed: %v", err)
	}
	if second {
		t.Fatal("expected second mark to return false")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer s2.Close()

	state, ok := s2.GetState("g1", "c1")
	if !ok {
		t.Fatal("expected persisted state")
	}
	if !state.OnboardingShown {
		t.Fatal("expected onboarding flag to persist as true")
	}
}

func TestSQLiteStoreRoomSettingsPersist(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}

	if err := s.SetRoomSetting("g1", "c1", "k1", "v1"); err != nil {
		t.Fatalf("SetRoomSetting failed: %v", err)
	}
	if err := s.SetRoomSetting("g1", "c1", "k1", "v2"); err != nil {
		t.Fatalf("SetRoomSetting update failed: %v", err)
	}
	if err := s.SetRoomSetting("g1", "c1", "k2", "v3"); err != nil {
		t.Fatalf("SetRoomSetting second key failed: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer s2.Close()

	got, err := s2.GetRoomSettings("g1", "c1")
	if err != nil {
		t.Fatalf("GetRoomSettings failed: %v", err)
	}
	if got["k1"] != "v2" || got["k2"] != "v3" {
		t.Fatalf("unexpected settings: %+v", got)
	}
}
