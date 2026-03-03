package store

import (
	"path/filepath"
	"testing"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

func BenchmarkSQLiteStoreLoadSave(b *testing.B) {
	dbPath := filepath.Join(b.TempDir(), "bench.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		b.Fatalf("NewSQLiteStore failed: %v", err)
	}
	b.Cleanup(func() {
		_ = s.Close()
	})

	players := []domain.Player{
		{ID: "u1", Name: "p1", XPower: 2600},
		{ID: "u2", Name: "p2", XPower: 2550},
		{ID: "u3", Name: "p3", XPower: 2500},
		{ID: "u4", Name: "p4", XPower: 2450},
		{ID: "u5", Name: "p5", XPower: 2400},
		{ID: "u6", Name: "p6", XPower: 2350},
		{ID: "u7", Name: "p7", XPower: 2300},
		{ID: "u8", Name: "p8", XPower: 2250},
		{ID: "u9", Name: "p9", XPower: 1800},
		{ID: "u10", Name: "p10", XPower: 1700},
	}
	for _, p := range players {
		if _, err := s.Join("g1", "c1", p); err != nil {
			b.Fatalf("join failed: %v", err)
		}
	}
	result := domain.MatchResult{
		TeamA:      players[:4],
		TeamB:      players[4:8],
		Spectators: players[8:],
		SumA:       10100,
		SumB:       9300,
		Diff:       800,
	}

	b.Run("save", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s.SaveLastMatch("g1", "c1", int64(i), players, result)
		}
	})

	s.SaveLastMatch("g1", "c1", 1, players, result)

	b.Run("load", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			state, ok := s.GetState("g1", "c1")
			if !ok || len(state.Players) == 0 {
				b.Fatal("GetState returned empty state")
			}
		}
	})
}

