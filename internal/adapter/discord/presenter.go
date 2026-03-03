package discord

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

func MatchResultEmbed(result domain.MatchResult) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Team Balancer",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Alpha", Value: formatPlayers(result.TeamA)},
			{Name: "Bravo", Value: formatPlayers(result.TeamB)},
			{Name: "Spectators", Value: formatPlayers(result.Spectators)},
			{
				Name:  "Summary",
				Value: fmt.Sprintf("SumA: %d\nSumB: %d\nDiff: %d", result.SumA, result.SumB, result.Diff),
			},
		},
	}
}

func WhoAmIEmbed(name string, xpower int, pauseRemaining int, participationCount int, spectatorCount int, ratingDelta int, wins int, losses int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Who Am I",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Player", Value: name, Inline: true},
			{Name: "XPower", Value: fmt.Sprintf("%d", xpower), Inline: true},
			{Name: "Rating Delta", Value: fmt.Sprintf("%d", ratingDelta), Inline: true},
			{Name: "Pause Remaining", Value: fmt.Sprintf("%d", pauseRemaining), Inline: true},
			{Name: "Past Participation", Value: fmt.Sprintf("%d", participationCount), Inline: true},
			{Name: "Spectator Count", Value: fmt.Sprintf("%d", spectatorCount), Inline: true},
			{Name: "Wins/Losses", Value: fmt.Sprintf("%d/%d", wins, losses), Inline: true},
		},
	}
}

func HelpEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Team Balancer Help",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "最短フロー",
				Value: strings.Join([]string{
					"- `/join xpower` で参加",
					"- `/make` で初回のチーム分け",
					"- `/next` で次試合を作成",
				}, "\n"),
			},
			{
				Name: "よく使う補助コマンド",
				Value: strings.Join([]string{
					"- `/pause matches:3 reason:トイレ`",
					"- `/resume` で復帰",
					"- `/undo` で直前の /make /next を取り消し",
				}, "\n"),
			},
		},
	}
}

func OnboardingEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Team Balancer Onboarding",
		Description: strings.Join([]string{
			"この部屋での基本操作",
			"`/join -> /make -> /next`",
			"困ったら `/help` を実行してください。",
		}, "\n"),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "運用例",
				Value: "- 数試合抜ける: `/pause matches:3 reason:トイレ`\n- 戻る: `/resume`",
			},
		},
	}
}

func SettingsEmbed(settings map[string]string) *discordgo.MessageEmbed {
	var b strings.Builder
	for _, key := range []string{
		domain.RoomSettingMakeNextCooldownSeconds,
		domain.RoomSettingSpectatorRotationWeight,
		domain.RoomSettingSameTeamAvoidanceWeight,
		domain.RoomSettingPauseDefaultMatches,
	} {
		fmt.Fprintf(&b, "- `%s`: `%s`\n", key, settings[key])
	}
	return &discordgo.MessageEmbed{
		Title:       "Room Settings",
		Description: strings.TrimSpace(b.String()),
	}
}

func DiagnoseEmbed(guildID, channelID, roomKey string, activePlayers, pausedPlayers int, locked bool, makeNextCooldownRemainingSeconds int, sqlitePath string, lastResultAt int64) *discordgo.MessageEmbed {
	lockedText := "no"
	if locked {
		lockedText = "yes"
	}
	cooldownText := "make: 0s\nnext: 0s"
	if makeNextCooldownRemainingSeconds < 0 {
		cooldownText = "make: processing\nnext: processing"
	} else if makeNextCooldownRemainingSeconds > 0 {
		cooldownText = fmt.Sprintf("make: %ds\nnext: %ds", makeNextCooldownRemainingSeconds, makeNextCooldownRemainingSeconds)
	}

	lastResultText := "none"
	if lastResultAt > 0 {
		lastResultText = time.Unix(lastResultAt, 0).UTC().Format(time.RFC3339)
	}

	return &discordgo.MessageEmbed{
		Title: "Team Balancer Diagnose",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Room", Value: fmt.Sprintf("key: `%s`\nguild_id: `%s`\nchannel_id: `%s`", roomKey, guildID, channelID)},
			{Name: "Players", Value: fmt.Sprintf("active: %d\npaused: %d", activePlayers, pausedPlayers), Inline: true},
			{Name: "Locked", Value: lockedText, Inline: true},
			{Name: "Cooldown", Value: cooldownText, Inline: true},
			{Name: "SQLite", Value: fmt.Sprintf("path: `%s`", sqlitePath)},
			{Name: "Last Result", Value: fmt.Sprintf("timestamp: %s", lastResultText)},
		},
	}
}

func ParticipantList(players []domain.Player) string {
	if len(players) == 0 {
		return "参加者はいません。"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "現在の参加者 (%d/10)\n", len(players))
	for _, p := range players {
		fmt.Fprintf(&b, "- %s (<@%s>) : %d\n", p.Name, p.ID, p.XPower)
	}
	return strings.TrimSpace(b.String())
}

func PausedList(players []domain.Player) string {
	if len(players) == 0 {
		return "pause中のプレイヤーはいません。"
	}
	var b strings.Builder
	b.WriteString("pause中のプレイヤー\n")
	for _, p := range players {
		if p.PauseReason != "" {
			fmt.Fprintf(&b, "- %s: 残り%d試合（%s）\n", p.Name, p.PauseRemaining, p.PauseReason)
			continue
		}
		fmt.Fprintf(&b, "- %s: 残り%d試合\n", p.Name, p.PauseRemaining)
	}
	return strings.TrimSpace(b.String())
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
