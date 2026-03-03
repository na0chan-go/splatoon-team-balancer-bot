package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	storeadapter "github.com/na0chan-go/splatoon-team-balancer-bot/internal/adapter/store"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/bot"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	persistentStore, err := storeadapter.NewSQLiteStore(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("failed to initialize sqlite store: %v", err)
	}
	defer persistentStore.Close()
	bot.SetStore(persistentStore)
	bot.SetSQLitePath(cfg.SQLitePath)

	token := cfg.DiscordToken
	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	session, err := discordgo.New(token)
	if err != nil {
		log.Fatalf("failed to create discord session: %v", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessageReactions
	session.AddHandler(bot.HandleInteraction)
	session.AddHandler(bot.HandleReactionAdd)

	if err := session.Open(); err != nil {
		log.Fatalf("failed to open discord session: %v", err)
	}
	defer session.Close()

	registerGuildID := strings.TrimSpace(cfg.DiscordGuildID)
	registerTarget := "global"
	if registerGuildID != "" {
		registerTarget = "guild"
	}
	if err := bot.RegisterCommands(session, cfg.DiscordAppID, registerGuildID); err != nil {
		log.Fatalf("failed to register commands: %v", err)
	}
	log.Printf("registered slash commands target=%s guild_id=%q", registerTarget, registerGuildID)

	log.Println("bot started. press Ctrl+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
