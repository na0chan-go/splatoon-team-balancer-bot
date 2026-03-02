package config

import "testing"

func TestLoadFailsWhenRequiredMissing(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "")
	t.Setenv("DISCORD_APP_ID", "")
	t.Setenv("DISCORD_GUILD_ID", "")
	t.Setenv("SQLITE_PATH", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when required env vars are missing")
	}
}

func TestLoadUsesDefaultSQLitePath(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "token")
	t.Setenv("DISCORD_APP_ID", "app")
	t.Setenv("DISCORD_GUILD_ID", "guild")
	t.Setenv("SQLITE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.SQLitePath != "./data.db" {
		t.Fatalf("expected default sqlite path ./data.db, got %s", cfg.SQLitePath)
	}
}
