package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/bot"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/config"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	persistentStore, err := store.NewSQLiteStore(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("failed to initialize sqlite store: %v", err)
	}
	defer persistentStore.Close()
	bot.SetStore(persistentStore)

	token := cfg.DiscordToken
	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	session, err := discordgo.New(token)
	if err != nil {
		log.Fatalf("failed to create discord session: %v", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds
	session.AddHandler(bot.HandleInteraction)

	if err := session.Open(); err != nil {
		log.Fatalf("failed to open discord session: %v", err)
	}
	defer session.Close()

	if err := bot.RegisterGuildCommands(session, cfg.DiscordAppID, cfg.DiscordGuildID); err != nil {
		log.Fatalf("failed to register commands: %v", err)
	}

	log.Println("bot started. press Ctrl+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
