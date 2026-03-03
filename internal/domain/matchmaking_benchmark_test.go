package domain

import "testing"

func BenchmarkBuildMatchWorstCase10Players(b *testing.B) {
	players := []Player{
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

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := BuildMatch(players, int64(i)); err != nil {
			b.Fatalf("BuildMatch failed: %v", err)
		}
	}
}
