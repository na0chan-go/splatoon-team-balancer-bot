package util

import (
	"github.com/bwmarrin/discordgo"
	discordadapter "github.com/na0chan-go/splatoon-team-balancer-bot/internal/adapter/discord"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

func MatchResultEmbed(result domain.MatchResult) *discordgo.MessageEmbed {
	return discordadapter.MatchResultEmbed(result)
}

func WhoAmIEmbed(name string, xpower int, pauseRemaining int, participationCount int, spectatorCount int, ratingDelta int, wins int, losses int) *discordgo.MessageEmbed {
	return discordadapter.WhoAmIEmbed(name, xpower, pauseRemaining, participationCount, spectatorCount, ratingDelta, wins, losses)
}

func HelpEmbed() *discordgo.MessageEmbed {
	return discordadapter.HelpEmbed()
}

func OnboardingEmbed() *discordgo.MessageEmbed {
	return discordadapter.OnboardingEmbed()
}

func SettingsEmbed(settings map[string]string) *discordgo.MessageEmbed {
	return discordadapter.SettingsEmbed(settings)
}
