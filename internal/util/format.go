package util

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

func MatchResultEmbed(result domain.MatchResult) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Team Balancer",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Alpha",
				Value: formatPlayers(result.TeamA),
			},
			{
				Name:  "Bravo",
				Value: formatPlayers(result.TeamB),
			},
			{
				Name:  "Spectators",
				Value: formatPlayers(result.Spectators),
			},
			{
				Name: "Summary",
				Value: fmt.Sprintf(
					"SumA: %d\nSumB: %d\nDiff: %d",
					result.SumA,
					result.SumB,
					result.Diff,
				),
			},
		},
	}
}

func WhoAmIEmbed(name string, xpower int, pauseRemaining int, participationCount int, spectatorCount int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Who Am I",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Player", Value: name, Inline: true},
			{Name: "XPower", Value: fmt.Sprintf("%d", xpower), Inline: true},
			{Name: "Pause Remaining", Value: fmt.Sprintf("%d", pauseRemaining), Inline: true},
			{Name: "Past Participation", Value: fmt.Sprintf("%d", participationCount), Inline: true},
			{Name: "Spectator Count", Value: fmt.Sprintf("%d", spectatorCount), Inline: true},
		},
	}
}

func formatPlayers(players []domain.Player) string {
	if len(players) == 0 {
		return "- none"
	}

	var b strings.Builder
	for _, p := range players {
		fmt.Fprintf(&b, "- %s (%d)\n", p.Name, p.XPower)
	}
	return strings.TrimSpace(b.String())
}
