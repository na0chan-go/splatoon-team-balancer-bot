package bot

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/util"
)

var roomStore = store.NewMemoryStore()

var ErrNoLastMake = errors.New("no previous make result")

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "ping to bot and receive pong",
	},
	{
		Name:        "join",
		Description: "join current room with your xpower",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "xpower",
				Description: "your xpower",
				Required:    true,
			},
		},
	},
	{
		Name:        "leave",
		Description: "leave current room",
	},
	{
		Name:        "list",
		Description: "show participants of current room",
	},
	{
		Name:        "make",
		Description: "create balanced 4v4 teams from participants",
	},
	{
		Name:        "reroll",
		Description: "reroll teams using last /make participant snapshot",
	},
}

func RegisterGuildCommands(s *discordgo.Session, appID, guildID string) error {
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			return fmt.Errorf("failed to register command %q: %w", cmd.Name, err)
		}
	}
	return nil
}

func HandleInteraction(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch ic.ApplicationCommandData().Name {
	case "ping":
		respond(s, ic, "pong", false)
	case "join":
		handleJoin(s, ic)
	case "leave":
		handleLeave(s, ic)
	case "list":
		handleList(s, ic)
	case "make":
		handleMake(s, ic)
	case "reroll":
		handleReroll(s, ic)
	}
}

func handleJoin(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	user := interactionUser(ic)
	if user == nil {
		respond(s, ic, "ユーザー情報を取得できませんでした。", true)
		return
	}

	xpower, ok := intOption(ic.ApplicationCommandData().Options, "xpower")
	if !ok {
		respond(s, ic, "xpower の取得に失敗しました。", true)
		return
	}

	player := domain.Player{
		ID:     user.ID,
		Name:   displayName(ic),
		XPower: xpower,
	}

	created, err := roomStore.Join(ic.GuildID, ic.ChannelID, player)
	if errors.Is(err, store.ErrRoomFull) {
		respond(s, ic, "参加者が10人に達しているため参加できません。", true)
		return
	}
	if errors.Is(err, store.ErrInvalidXPower) {
		respond(s, ic, "XPower は 0〜5000 の範囲で入力してください。", true)
		return
	}
	if err != nil {
		respond(s, ic, "参加登録に失敗しました。", true)
		return
	}

	if created {
		respond(s, ic, fmt.Sprintf("参加登録しました: %s (%d)", player.Name, player.XPower), false)
		return
	}
	respond(s, ic, fmt.Sprintf("参加情報を更新しました: %s (%d)", player.Name, player.XPower), false)
}

func handleLeave(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	user := interactionUser(ic)
	if user == nil {
		respond(s, ic, "ユーザー情報を取得できませんでした。", true)
		return
	}

	err := roomStore.Leave(ic.GuildID, ic.ChannelID, user.ID)
	if errors.Is(err, store.ErrNotJoined) {
		respond(s, ic, "この部屋には参加していません。", true)
		return
	}
	if err != nil {
		respond(s, ic, "退出処理に失敗しました。", true)
		return
	}

	respond(s, ic, "退出しました。", false)
}

func handleList(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	players := roomStore.List(ic.GuildID, ic.ChannelID)
	if len(players) == 0 {
		respond(s, ic, "参加者はいません。", false)
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "現在の参加者 (%d/10)\n", len(players))
	for _, p := range players {
		fmt.Fprintf(&b, "- %s (<@%s>) : %d\n", p.Name, p.ID, p.XPower)
	}

	respond(s, ic, strings.TrimSpace(b.String()), false)
}

func handleMake(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}

	players := roomStore.List(ic.GuildID, ic.ChannelID)
	result, err := runMatchAndStore(ic.GuildID, ic.ChannelID, players, time.Now().UnixNano())
	if errors.Is(err, domain.ErrNotEnoughPlayers) {
		respond(s, ic, "参加者が8人未満のためチーム分けできません。", true)
		return
	}
	if err != nil {
		respond(s, ic, "チーム分けに失敗しました。", true)
		return
	}

	respondEmbed(s, ic, util.MatchResultEmbed(result), false)
}

func handleReroll(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}

	result, err := rerollFromLastSnapshot(ic.GuildID, ic.ChannelID, time.Now().UnixNano())
	if errors.Is(err, ErrNoLastMake) {
		respond(s, ic, "先に /make を実行してください。", true)
		return
	}
	if err != nil {
		respond(s, ic, "再抽選に失敗しました。", true)
		return
	}

	respondEmbed(s, ic, util.MatchResultEmbed(result), false)
}

func respond(s *discordgo.Session, ic *discordgo.InteractionCreate, content string, ephemeral bool) {
	data := &discordgo.InteractionResponseData{
		Content: content,
	}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	_ = s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func respondEmbed(s *discordgo.Session, ic *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, ephemeral bool) {
	data := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	_ = s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func intOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) (int, bool) {
	for _, opt := range opts {
		if opt.Name != name {
			continue
		}
		return int(opt.IntValue()), true
	}
	return 0, false
}

func interactionUser(ic *discordgo.InteractionCreate) *discordgo.User {
	if ic.Member != nil && ic.Member.User != nil {
		return ic.Member.User
	}
	if ic.User != nil {
		return ic.User
	}
	return nil
}

func displayName(ic *discordgo.InteractionCreate) string {
	if ic.Member != nil {
		if ic.Member.Nick != "" {
			return ic.Member.Nick
		}
		if ic.Member.User != nil {
			if ic.Member.User.GlobalName != "" {
				return ic.Member.User.GlobalName
			}
			return ic.Member.User.Username
		}
	}
	if ic.User != nil {
		if ic.User.GlobalName != "" {
			return ic.User.GlobalName
		}
		return ic.User.Username
	}
	return "unknown"
}

func runMatchAndStore(guildID, channelID string, players []domain.Player, seed int64) (domain.MatchResult, error) {
	result, err := domain.BuildMatch(players, seed)
	if err != nil {
		return domain.MatchResult{}, err
	}
	roomStore.SaveLastMatch(guildID, channelID, seed, players, result)
	return result, nil
}

func rerollFromLastSnapshot(guildID, channelID string, seed int64) (domain.MatchResult, error) {
	state, ok := roomStore.GetState(guildID, channelID)
	if !ok || len(state.LastPlayersSnapshot) == 0 {
		return domain.MatchResult{}, ErrNoLastMake
	}
	return runMatchAndStore(guildID, channelID, state.LastPlayersSnapshot, seed)
}
