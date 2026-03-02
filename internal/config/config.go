package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DiscordToken   string
	DiscordAppID   string
	DiscordGuildID string
	SQLitePath     string
}

func Load() (Config, error) {
	cfg := Config{
		DiscordToken:   strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		DiscordAppID:   strings.TrimSpace(os.Getenv("DISCORD_APP_ID")),
		DiscordGuildID: strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID")),
		SQLitePath:     strings.TrimSpace(os.Getenv("SQLITE_PATH")),
	}
	if cfg.SQLitePath == "" {
		cfg.SQLitePath = "./data.db"
	}

	var missing []string
	if cfg.DiscordToken == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if cfg.DiscordAppID == "" {
		missing = append(missing, "DISCORD_APP_ID")
	}
	if cfg.DiscordGuildID == "" {
		missing = append(missing, "DISCORD_GUILD_ID")
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}
