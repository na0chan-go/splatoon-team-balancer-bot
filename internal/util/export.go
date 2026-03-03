package util

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

type ExportFile struct {
	Name string
	Data []byte
}

func BuildExportFiles(format string, matches []store.MatchRecord, stats []store.PlayerStat) ([]ExportFile, error) {
	switch format {
	case "csv":
		matchCSV, err := buildMatchesCSV(matches)
		if err != nil {
			return nil, err
		}
		statsCSV, err := buildStatsCSV(stats)
		if err != nil {
			return nil, err
		}
		return []ExportFile{
			{Name: "matches.csv", Data: matchCSV},
			{Name: "player_stats.csv", Data: statsCSV},
		}, nil
	case "json":
		body, err := json.MarshalIndent(map[string]any{
			"matches":      matches,
			"player_stats": stats,
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		return []ExportFile{
			{Name: "export.json", Data: body},
		}, nil
	default:
		return nil, fmt.Errorf("unknown export format: %s", format)
	}
}

func buildMatchesCSV(matches []store.MatchRecord) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{
		"match_id", "guild_id", "channel_id", "created_at", "winner_team",
		"sum_a", "sum_b", "diff", "team_a", "team_b", "spectators",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, m := range matches {
		row := []string{
			strconv.FormatInt(m.ID, 10),
			m.GuildID,
			m.ChannelID,
			strconv.FormatInt(m.CreatedAt, 10),
			m.WinnerTeam,
			strconv.Itoa(m.SumA),
			strconv.Itoa(m.SumB),
			strconv.Itoa(m.Diff),
			joinPlayers(m.TeamA),
			joinPlayers(m.TeamB),
			joinPlayers(m.Spectators),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildStatsCSV(stats []store.PlayerStat) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{"user_id", "rating", "wins", "losses"}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, s := range stats {
		row := []string{
			s.UserID,
			strconv.Itoa(s.Rating),
			strconv.Itoa(s.Wins),
			strconv.Itoa(s.Losses),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func joinPlayers(players []domain.Player) string {
	if len(players) == 0 {
		return ""
	}
	out := make([]string, 0, len(players))
	for _, p := range players {
		out = append(out, p.ID+":"+p.Name+"("+strconv.Itoa(p.XPower)+")")
	}
	return strings.Join(out, ";")
}
