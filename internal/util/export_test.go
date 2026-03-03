package util

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

func TestBuildExportFilesCSVHeadersAndColumns(t *testing.T) {
	matches := []store.MatchRecord{
		{
			ID:         1,
			GuildID:    "g1",
			ChannelID:  "c1",
			WinnerTeam: "alpha",
			TeamA:      []domain.Player{{ID: "u1", Name: "p1", XPower: 2400}},
			TeamB:      []domain.Player{{ID: "u2", Name: "p2", XPower: 2300}},
			Spectators: []domain.Player{{ID: "u3", Name: "p3", XPower: 2200}},
			SumA:       2400,
			SumB:       2300,
			Diff:       100,
			CreatedAt:  1000,
		},
	}
	stats := []store.PlayerStat{{UserID: "u1", RatingDelta: 10, Wins: 1, Losses: 0, LastPlayedAt: 1000}}

	files, err := BuildExportFiles("csv", matches, stats)
	if err != nil {
		t.Fatalf("BuildExportFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	matchCSV := findFile(files, "matches.csv")
	matchRows, err := csv.NewReader(strings.NewReader(string(matchCSV))).ReadAll()
	if err != nil {
		t.Fatalf("read matches csv failed: %v", err)
	}
	if len(matchRows) < 2 {
		t.Fatalf("expected header + 1 row, got %d", len(matchRows))
	}
	if got := len(matchRows[0]); got != 11 {
		t.Fatalf("expected matches header columns 11, got %d", got)
	}
	if matchRows[0][0] != "match_id" {
		t.Fatalf("unexpected matches header: %+v", matchRows[0])
	}
	if got := len(matchRows[1]); got != 11 {
		t.Fatalf("expected matches row columns 11, got %d", got)
	}

	statsCSV := findFile(files, "player_stats.csv")
	statsRows, err := csv.NewReader(strings.NewReader(string(statsCSV))).ReadAll()
	if err != nil {
		t.Fatalf("read stats csv failed: %v", err)
	}
	if len(statsRows) < 2 {
		t.Fatalf("expected header + 1 row, got %d", len(statsRows))
	}
	if got := len(statsRows[0]); got != 5 {
		t.Fatalf("expected stats header columns 5, got %d", got)
	}
	if statsRows[0][0] != "user_id" {
		t.Fatalf("unexpected stats header: %+v", statsRows[0])
	}
	if got := len(statsRows[1]); got != 5 {
		t.Fatalf("expected stats row columns 5, got %d", got)
	}
}

func findFile(files []ExportFile, name string) []byte {
	for _, f := range files {
		if f.Name == name {
			return f.Data
		}
	}
	return nil
}
