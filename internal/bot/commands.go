package bot

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	discordadapter "github.com/na0chan-go/splatoon-team-balancer-bot/internal/adapter/discord"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/util"
)

var roomStore store.Store = store.NewMemoryStore()
var pauseReactionRegistry = newPauseReactionRegistry()
var roomCommandGuards = newRoomCommandGuardMap()

var ErrNoLastMake = errors.New("no previous make result")
var ErrNoPreviousMatch = errors.New("no previous match")
var ErrNotInRoom = errors.New("player not in room")

const rotationDiffSlack = 50

func SetStore(s store.Store) {
	if s == nil {
		return
	}
	roomStore = s
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "ping to bot and receive pong",
	},
	{
		Name:        "help",
		Description: "show usage guide",
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
		Name:        "next",
		Description: "create next match from current participants",
	},
	{
		Name:        "pause",
		Description: "temporarily pause a player for some matches",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "matches",
				Description: "number of matches to pause (default: 3)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "optional pause reason",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "target user (default: yourself)",
				Required:    false,
			},
		},
	},
	{
		Name:        "resume",
		Description: "resume a paused player",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "target user (default: yourself)",
				Required:    false,
			},
		},
	},
	{
		Name:        "paused",
		Description: "show paused players in this room",
	},
	{
		Name:        "whoami",
		Description: "show your current state in this room",
	},
	{
		Name:        "undo",
		Description: "undo last /make or /next result",
	},
	{
		Name:        "reroll",
		Description: "reroll teams using last /make participant snapshot",
	},
	{
		Name:        "reset",
		Description: "reset current room state",
	},
	{
		Name:        "result",
		Description: "record match winner and update ratings",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "winner",
				Description: "winner team",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "alpha", Value: "alpha"},
					{Name: "bravo", Value: "bravo"},
				},
			},
		},
	},
	{
		Name:        "export",
		Description: "export matches and player stats",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "export file type",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "csv", Value: "csv"},
					{Name: "json", Value: "json"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "scope",
				Description: "export scope",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "room", Value: "room"},
					{Name: "all", Value: "all"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "recent match limit (default 100)",
				Required:    false,
			},
		},
	},
	{
		Name:        "settings",
		Description: "show or update room settings",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "show",
				Description: "show room settings",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "set",
				Description: "set room setting (admin only)",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "key",
						Description: "setting key",
						Required:    true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: domain.RoomSettingMakeNextCooldownSeconds, Value: domain.RoomSettingMakeNextCooldownSeconds},
							{Name: domain.RoomSettingSpectatorRotationWeight, Value: domain.RoomSettingSpectatorRotationWeight},
							{Name: domain.RoomSettingSameTeamAvoidanceWeight, Value: domain.RoomSettingSameTeamAvoidanceWeight},
							{Name: domain.RoomSettingPauseDefaultMatches, Value: domain.RoomSettingPauseDefaultMatches},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "value",
						Description: "setting value",
						Required:    true,
					},
				},
			},
		},
	},
}

func RegisterCommands(s *discordgo.Session, appID, guildID string) error {
	target := "global"
	if strings.TrimSpace(guildID) != "" {
		target = "guild"
	}
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			return fmt.Errorf("failed to register %s command %q: %w", target, cmd.Name, err)
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
	case "help":
		handleHelp(s, ic)
	case "join":
		handleJoin(s, ic)
	case "leave":
		handleLeave(s, ic)
	case "list":
		handleList(s, ic)
	case "make":
		handleMake(s, ic)
	case "next":
		handleNext(s, ic)
	case "pause":
		handlePause(s, ic)
	case "resume":
		handleResume(s, ic)
	case "paused":
		handlePaused(s, ic)
	case "whoami":
		handleWhoAmI(s, ic)
	case "undo":
		handleUndo(s, ic)
	case "reroll":
		handleReroll(s, ic)
	case "reset":
		handleReset(s, ic)
	case "result":
		handleResult(s, ic)
	case "export":
		handleExport(s, ic)
	case "settings":
		handleSettings(s, ic)
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
		showOnboarding, err := roomStore.TryMarkOnboardingShown(ic.GuildID, ic.ChannelID)
		if err != nil {
			respond(s, ic, "参加登録に失敗しました。", true)
			return
		}
		if showOnboarding {
			respondEmbed(s, ic, discordadapter.OnboardingEmbed(), false)
			return
		}
		respond(s, ic, fmt.Sprintf("参加登録しました: %s (%d)", player.Name, player.XPower), false)
		return
	}
	respond(s, ic, fmt.Sprintf("参加情報を更新しました: %s (%d)", player.Name, player.XPower), false)
}

func handleHelp(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	respondEmbed(s, ic, discordadapter.HelpEmbed(), false)
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
	respond(s, ic, discordadapter.ParticipantList(players), false)
}

func handleMake(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()
	settings, err := roomSettings(ic.GuildID, ic.ChannelID)
	if err != nil {
		respond(s, ic, "settings の読み込みに失敗しました。", true)
		return
	}
	if !checkAndConsumeMakeNextCooldown(s, ic, lock, settings.MakeNextCooldownSeconds) {
		return
	}
	if !deferAck(s, ic) {
		return
	}

	players := roomStore.List(ic.GuildID, ic.ChannelID)
	roomStore.SnapshotRoomState(ic.GuildID, ic.ChannelID)
	result, err := runMatchAndStore(ic.GuildID, ic.ChannelID, players, settings, time.Now().UnixNano())
	if errors.Is(err, domain.ErrNotEnoughPlayers) {
		editDeferredContent(s, ic, "参加者が8人未満のためチーム分けできません。")
		return
	}
	if err != nil {
		editDeferredContent(s, ic, "チーム分けに失敗しました。")
		return
	}

	editDeferredEmbed(s, ic, util.MatchResultEmbed(result))
}

func handleReroll(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()

	settings, err := roomSettings(ic.GuildID, ic.ChannelID)
	if err != nil {
		respond(s, ic, "settings の読み込みに失敗しました。", true)
		return
	}
	result, err := rerollFromLastSnapshot(ic.GuildID, ic.ChannelID, settings, time.Now().UnixNano())
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

func handleNext(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()
	settings, err := roomSettings(ic.GuildID, ic.ChannelID)
	if err != nil {
		respond(s, ic, "settings の読み込みに失敗しました。", true)
		return
	}
	if !checkAndConsumeMakeNextCooldown(s, ic, lock, settings.MakeNextCooldownSeconds) {
		return
	}
	if !deferAck(s, ic) {
		return
	}

	result, err := nextMatchFromCurrentParticipants(ic.GuildID, ic.ChannelID, settings, time.Now().UnixNano())
	if errors.Is(err, ErrNoPreviousMatch) {
		editDeferredContent(s, ic, "先に /make を実行してください。")
		return
	}
	if errors.Is(err, domain.ErrNotEnoughPlayers) {
		editDeferredContent(s, ic, "参加者が8人未満のため次試合を作成できません。")
		return
	}
	if err != nil {
		editDeferredContent(s, ic, "次試合の作成に失敗しました。")
		return
	}

	editDeferredEmbed(s, ic, util.MatchResultEmbed(result))
}

func handlePause(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()

	settings, err := roomSettings(ic.GuildID, ic.ChannelID)
	if err != nil {
		respond(s, ic, "settings の読み込みに失敗しました。", true)
		return
	}

	targetID := ""
	targetName := ""
	if u, ok := userOption(s, ic.ApplicationCommandData().Options, "user"); ok && u != nil {
		targetID = u.ID
		targetName = u.Username
	} else {
		u := interactionUser(ic)
		if u == nil {
			respond(s, ic, "ユーザー情報を取得できませんでした。", true)
			return
		}
		targetID = u.ID
		targetName = displayName(ic)
	}

	matches := settings.PauseDefaultMatches
	if v, ok := intOption(ic.ApplicationCommandData().Options, "matches"); ok {
		matches = v
	}
	if matches < 1 {
		respond(s, ic, "matches は1以上を指定してください。", true)
		return
	}
	reason, _ := stringOption(ic.ApplicationCommandData().Options, "reason")

	if err := roomStore.SetPause(ic.GuildID, ic.ChannelID, targetID, matches, reason); err != nil {
		if errors.Is(err, store.ErrNotJoined) {
			respond(s, ic, "対象ユーザーはこの部屋に参加していません。", true)
			return
		}
		respond(s, ic, "pause の設定に失敗しました。", true)
		return
	}

	msg := fmt.Sprintf("%s を %d 試合 pause しました。", targetName, matches)
	if strings.TrimSpace(reason) != "" {
		msg += fmt.Sprintf(" 理由: %s", reason)
	}
	respond(s, ic, msg, false)

	notice := fmt.Sprintf("<@%s> が復帰するにはこのメッセージに 👍 リアクションしてください。", targetID)
	sent, err := s.ChannelMessageSend(ic.ChannelID, notice)
	if err == nil && sent != nil {
		pauseReactionRegistry.put(sent.ID, pauseReactionTarget{
			GuildID:   ic.GuildID,
			ChannelID: ic.ChannelID,
			UserID:    targetID,
		})
	}
}

func handleResume(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()

	targetID := ""
	targetName := ""
	if u, ok := userOption(s, ic.ApplicationCommandData().Options, "user"); ok && u != nil {
		targetID = u.ID
		targetName = u.Username
	} else {
		u := interactionUser(ic)
		if u == nil {
			respond(s, ic, "ユーザー情報を取得できませんでした。", true)
			return
		}
		targetID = u.ID
		targetName = displayName(ic)
	}

	if err := roomStore.Resume(ic.GuildID, ic.ChannelID, targetID); err != nil {
		if errors.Is(err, store.ErrNotJoined) {
			respond(s, ic, "対象ユーザーはこの部屋に参加していません。", true)
			return
		}
		respond(s, ic, "resume に失敗しました。", true)
		return
	}
	respond(s, ic, fmt.Sprintf("%s を復帰させました。", targetName), false)
}

func handlePaused(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}

	paused := roomStore.Paused(ic.GuildID, ic.ChannelID)
	respond(s, ic, discordadapter.PausedList(paused), false)
}

func handleWhoAmI(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	user := interactionUser(ic)
	if user == nil {
		respond(s, ic, "ユーザー情報を取得できませんでした。", true)
		return
	}

	info, err := whoAmIState(ic.GuildID, ic.ChannelID, user.ID)
	if errors.Is(err, ErrNotInRoom) {
		respond(s, ic, "この部屋に参加していません。", true)
		return
	}
	if err != nil {
		respond(s, ic, "状態の取得に失敗しました。", true)
		return
	}

	embed := discordadapter.WhoAmIEmbed(
		info.Name,
		info.XPower,
		info.PauseRemaining,
		info.ParticipationCount,
		info.SpectatorCount,
		info.RatingDelta,
		info.Wins,
		info.Losses,
	)
	respondEmbed(s, ic, embed, false)
}

func handleReset(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()
	if !hasResetPermission(ic) {
		respond(s, ic, "権限がありません", true)
		return
	}

	roomStore.ResetRoom(ic.GuildID, ic.ChannelID)
	respond(s, ic, "部屋の状態をリセットしました。", false)
}

func handleUndo(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()

	ok, err := undoLastRoomState(ic.GuildID, ic.ChannelID)
	if err != nil {
		respond(s, ic, "undo に失敗しました。", true)
		return
	}
	if !ok {
		respond(s, ic, "戻せる直前状態がありません。", true)
		return
	}
	respond(s, ic, "直前の状態に戻しました。", false)
}

func handleResult(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	lock, ok := lockRoomMutation(s, ic)
	if !ok {
		return
	}
	defer lock.mu.Unlock()

	winner, ok := stringOption(ic.ApplicationCommandData().Options, "winner")
	if !ok || (winner != "alpha" && winner != "bravo") {
		respond(s, ic, "winner は alpha か bravo を指定してください。", true)
		return
	}

	state, ok := roomStore.GetState(ic.GuildID, ic.ChannelID)
	if !ok || len(state.LastResult.TeamA) == 0 || len(state.LastResult.TeamB) == 0 {
		respond(s, ic, "先に /make を実行してください。", true)
		return
	}

	if err := roomStore.RecordMatchResult(ic.GuildID, ic.ChannelID, winner, state.LastResult); err != nil {
		respond(s, ic, "結果の記録に失敗しました。", true)
		return
	}

	respond(s, ic, fmt.Sprintf("試合結果を記録しました。勝利: %s", winner), false)
}

func handleExport(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	fileType, ok := stringOption(ic.ApplicationCommandData().Options, "type")
	if !ok || (fileType != "csv" && fileType != "json") {
		respond(s, ic, "type は csv か json を指定してください。", true)
		return
	}
	scope, ok := stringOption(ic.ApplicationCommandData().Options, "scope")
	if !ok || (scope != "room" && scope != "all") {
		respond(s, ic, "scope は room か all を指定してください。", true)
		return
	}

	limit := 100
	if v, ok := intOption(ic.ApplicationCommandData().Options, "limit"); ok {
		limit = v
	}
	if limit < 1 || limit > 5000 {
		respond(s, ic, "limit は 1〜5000 の範囲で指定してください。", true)
		return
	}

	if !deferAck(s, ic) {
		return
	}

	matches, stats, err := roomStore.GetExportData(ic.GuildID, ic.ChannelID, scope, limit)
	if err != nil {
		editDeferredContent(s, ic, "export の取得に失敗しました。")
		return
	}
	files, err := util.BuildExportFiles(fileType, matches, stats)
	if err != nil {
		editDeferredContent(s, ic, "export ファイルの生成に失敗しました。")
		return
	}
	sendDeferredFiles(s, ic, fmt.Sprintf("export completed: type=%s scope=%s matches=%d stats=%d", fileType, scope, len(matches), len(stats)), files)
}

func handleSettings(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.GuildID == "" {
		respond(s, ic, "このコマンドはサーバー内で実行してください。", true)
		return
	}
	opts := ic.ApplicationCommandData().Options
	if len(opts) == 0 {
		respond(s, ic, "使い方: /settings show または /settings set key value", true)
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "show":
		cfg, err := roomSettings(ic.GuildID, ic.ChannelID)
		if err != nil {
			respond(s, ic, "settings の読み込みに失敗しました。", true)
			return
		}
		respondEmbed(s, ic, discordadapter.SettingsEmbed(roomSettingsToMap(cfg)), false)
	case "set":
		if !hasResetPermission(ic) {
			respond(s, ic, "権限がありません", true)
			return
		}
		lock, ok := lockRoomMutation(s, ic)
		if !ok {
			return
		}
		defer lock.mu.Unlock()

		key, ok := stringOption(sub.Options, "key")
		if !ok {
			respond(s, ic, "key の取得に失敗しました。", true)
			return
		}
		value, ok := stringOption(sub.Options, "value")
		if !ok {
			respond(s, ic, "value の取得に失敗しました。", true)
			return
		}
		if err := domain.ValidateRoomSetting(key, value); err != nil {
			respond(s, ic, "不正な設定です: "+err.Error(), true)
			return
		}
		if err := roomStore.SetRoomSetting(ic.GuildID, ic.ChannelID, key, value); err != nil {
			respond(s, ic, "settings の保存に失敗しました。", true)
			return
		}
		respond(s, ic, fmt.Sprintf("設定を更新しました: %s=%s", key, value), false)
	default:
		respond(s, ic, "使い方: /settings show または /settings set key value", true)
	}
}

func lockRoomMutation(s *discordgo.Session, ic *discordgo.InteractionCreate) (*roomCommandGuardState, bool) {
	state, ok := roomCommandGuards.tryLock(roomKey(ic.GuildID, ic.ChannelID))
	if ok {
		return state, true
	}
	respond(s, ic, "現在処理中です。少し待って再実行してください。", true)
	return nil, false
}

func checkAndConsumeMakeNextCooldown(s *discordgo.Session, ic *discordgo.InteractionCreate, state *roomCommandGuardState, cooldownSeconds int) bool {
	cooldown := time.Duration(cooldownSeconds) * time.Second
	remaining, ok := consumeCooldown(&state.makeNextCooldown, time.Now(), cooldown)
	if ok {
		return true
	}
	respond(s, ic, fmt.Sprintf("クールダウン中です。あと%d秒待って再実行してください。", remainingSeconds(remaining)), true)
	return false
}

func deferAck(s *discordgo.Session, ic *discordgo.InteractionCreate) bool {
	err := s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	return err == nil
}

func editDeferredContent(s *discordgo.Session, ic *discordgo.InteractionCreate, content string) {
	_, _ = s.InteractionResponseEdit(ic.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}

func editDeferredEmbed(s *discordgo.Session, ic *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	_, _ = s.InteractionResponseEdit(ic.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func sendDeferredFiles(s *discordgo.Session, ic *discordgo.InteractionCreate, content string, files []util.ExportFile) {
	_, _ = s.InteractionResponseEdit(ic.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	dFiles := make([]*discordgo.File, 0, len(files))
	for _, f := range files {
		dFiles = append(dFiles, &discordgo.File{
			Name:   f.Name,
			Reader: bytes.NewReader(f.Data),
		})
	}
	_, _ = s.FollowupMessageCreate(ic.Interaction, false, &discordgo.WebhookParams{
		Content: content,
		Files:   dFiles,
	})
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

func stringOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) (string, bool) {
	for _, opt := range opts {
		if opt.Name != name {
			continue
		}
		return opt.StringValue(), true
	}
	return "", false
}

func userOption(s *discordgo.Session, opts []*discordgo.ApplicationCommandInteractionDataOption, name string) (*discordgo.User, bool) {
	for _, opt := range opts {
		if opt.Name != name {
			continue
		}
		return opt.UserValue(s), true
	}
	return nil, false
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

func runMatchAndStore(guildID, channelID string, players []domain.Player, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	state, _ := roomStore.GetState(guildID, channelID)
	penaltyFn := combinedPenaltyFunc(state, settings, time.Now().Unix())
	effectivePlayers := applyRatings(players, roomStore.GetPlayerStats(playerIDs(players)))

	result, err := domain.BuildMatchWithResultPenalty(effectivePlayers, seed, rotationDiffSlack, penaltyFn)
	if err != nil {
		return domain.MatchResult{}, err
	}
	roomStore.SaveLastMatch(guildID, channelID, seed, players, result)
	return result, nil
}

func rerollFromLastSnapshot(guildID, channelID string, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	state, ok := roomStore.GetState(guildID, channelID)
	if !ok || len(state.LastPlayersSnapshot) == 0 {
		return domain.MatchResult{}, ErrNoLastMake
	}
	return runMatchAndStore(guildID, channelID, state.LastPlayersSnapshot, settings, seed)
}

func nextMatchFromCurrentParticipants(guildID, channelID string, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	state, ok := roomStore.GetState(guildID, channelID)
	if !ok || len(state.LastResult.TeamA) == 0 || len(state.LastResult.TeamB) == 0 {
		return domain.MatchResult{}, ErrNoPreviousMatch
	}
	roomStore.SnapshotRoomState(guildID, channelID)
	defer roomStore.DecrementPauses(guildID, channelID)

	players := roomStore.List(guildID, channelID)
	var active []domain.Player
	for _, p := range players {
		if p.PauseRemaining > 0 {
			continue
		}
		active = append(active, p)
	}
	return runMatchAndStore(guildID, channelID, active, settings, seed)
}

func undoLastRoomState(guildID, channelID string) (bool, error) {
	return roomStore.UndoRoomState(guildID, channelID)
}

type whoAmIInfo struct {
	Name               string
	XPower             int
	PauseRemaining     int
	ParticipationCount int
	SpectatorCount     int
	RatingDelta        int
	Wins               int
	Losses             int
}

type pauseReactionTarget struct {
	GuildID   string
	ChannelID string
	UserID    string
}

type pauseReactionMap struct {
	mu      sync.RWMutex
	entries map[string]pauseReactionTarget
}

type roomCommandGuardState struct {
	mu               sync.Mutex
	makeNextCooldown time.Time
}

type roomCommandGuardMap struct {
	mu    sync.Mutex
	rooms map[string]*roomCommandGuardState
}

func newPauseReactionRegistry() *pauseReactionMap {
	return &pauseReactionMap{entries: make(map[string]pauseReactionTarget)}
}

func newRoomCommandGuardMap() *roomCommandGuardMap {
	return &roomCommandGuardMap{
		rooms: make(map[string]*roomCommandGuardState),
	}
}

func (m *pauseReactionMap) put(messageID string, target pauseReactionTarget) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[messageID] = target
}

func (m *pauseReactionMap) get(messageID string) (pauseReactionTarget, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	target, ok := m.entries[messageID]
	return target, ok
}

func (m *pauseReactionMap) delete(messageID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, messageID)
}

func (m *roomCommandGuardMap) get(roomKey string) *roomCommandGuardState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.rooms[roomKey]
	if ok {
		return state
	}
	state = &roomCommandGuardState{}
	m.rooms[roomKey] = state
	return state
}

func (m *roomCommandGuardMap) tryLock(roomKey string) (*roomCommandGuardState, bool) {
	state := m.get(roomKey)
	if !state.mu.TryLock() {
		return nil, false
	}
	return state, true
}

func consumeCooldown(until *time.Time, now time.Time, duration time.Duration) (time.Duration, bool) {
	if now.Before(*until) {
		return until.Sub(now), false
	}
	*until = now.Add(duration)
	return 0, true
}

func remainingSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(math.Ceil(d.Seconds()))
}

func roomKey(guildID, channelID string) string {
	return guildID + ":" + channelID
}

func whoAmIState(guildID, channelID, userID string) (whoAmIInfo, error) {
	state, ok := roomStore.GetState(guildID, channelID)
	if !ok {
		return whoAmIInfo{}, ErrNotInRoom
	}
	var player *domain.Player
	for i := range state.Players {
		if state.Players[i].ID == userID {
			player = &state.Players[i]
			break
		}
	}
	if player == nil {
		return whoAmIInfo{}, ErrNotInRoom
	}
	stats := roomStore.GetPlayerStats([]string{userID})[userID]

	return whoAmIInfo{
		Name:               player.Name,
		XPower:             player.XPower,
		PauseRemaining:     player.PauseRemaining,
		ParticipationCount: state.ParticipationCounts[userID],
		SpectatorCount:     state.SpectatorHistory[userID].SpectatorCount,
		RatingDelta:        stats.RatingDelta,
		Wins:               stats.Wins,
		Losses:             stats.Losses,
	}, nil
}

func HandleReactionAdd(s *discordgo.Session, ev *discordgo.MessageReactionAdd) {
	if ev == nil || ev.MessageReaction == nil {
		return
	}
	if ev.UserID == "" || ev.Emoji.Name != "👍" {
		return
	}

	resumed, targetChannelID := processPauseResumeReaction(
		ev.MessageID,
		ev.UserID,
		ev.GuildID,
		ev.ChannelID,
	)
	if !resumed {
		return
	}

	_, _ = s.ChannelMessageSend(targetChannelID, fmt.Sprintf("<@%s> を復帰させました。", ev.UserID))
}

func processPauseResumeReaction(messageID, reactorUserID, guildID, channelID string) (bool, string) {
	target, ok := pauseReactionRegistry.get(messageID)
	if !ok {
		return false, ""
	}
	if target.UserID != reactorUserID {
		return false, ""
	}
	if target.GuildID != guildID || target.ChannelID != channelID {
		return false, ""
	}

	if err := roomStore.Resume(guildID, channelID, reactorUserID); err != nil {
		return false, ""
	}
	pauseReactionRegistry.delete(messageID)
	return true, target.ChannelID
}

func hasResetPermission(ic *discordgo.InteractionCreate) bool {
	if ic.Member == nil {
		return false
	}
	perms := ic.Member.Permissions
	return perms&discordgo.PermissionAdministrator != 0 || perms&discordgo.PermissionManageGuild != 0
}

func roomSettings(guildID, channelID string) (domain.RoomSettings, error) {
	raw, err := roomStore.GetRoomSettings(guildID, channelID)
	if err != nil {
		return domain.RoomSettings{}, err
	}
	return domain.RoomSettingsFromMap(raw), nil
}

func roomSettingsToMap(s domain.RoomSettings) map[string]string {
	return map[string]string{
		domain.RoomSettingMakeNextCooldownSeconds: strconv.Itoa(s.MakeNextCooldownSeconds),
		domain.RoomSettingSpectatorRotationWeight: strconv.Itoa(s.SpectatorRotationWeight),
		domain.RoomSettingSameTeamAvoidanceWeight: strconv.Itoa(s.SameTeamAvoidanceWeight),
		domain.RoomSettingPauseDefaultMatches:     strconv.Itoa(s.PauseDefaultMatches),
	}
}

func combinedPenaltyFunc(state store.RoomState, settings domain.RoomSettings, nowUnix int64) func(domain.MatchResult) int {
	spectatorPenalty := spectatorPenaltyFunc(state.SpectatorHistory, nowUnix)
	sameTeamPenalty := sameTeamPenaltyFunc(state.LastResult)
	return func(result domain.MatchResult) int {
		total := 0
		if settings.SpectatorRotationWeight > 0 {
			total += spectatorPenalty(result.Spectators) * settings.SpectatorRotationWeight
		}
		if settings.SameTeamAvoidanceWeight > 0 {
			total += sameTeamPenalty(result) * settings.SameTeamAvoidanceWeight
		}
		return total
	}
}

func spectatorPenaltyFunc(history map[string]store.SpectatorHistory, nowUnix int64) func([]domain.Player) int {
	return func(spectators []domain.Player) int {
		if len(history) == 0 || len(spectators) == 0 {
			return 0
		}

		penalty := 0
		for _, p := range spectators {
			h := history[p.ID]
			penalty += h.SpectatorCount * 100

			if h.LastSpectatedAt <= 0 {
				continue
			}

			age := nowUnix - h.LastSpectatedAt
			switch {
			case age < 3600:
				penalty += 300
			case age < 6*3600:
				penalty += 150
			case age < 24*3600:
				penalty += 60
			}
		}
		return penalty
	}
}

func sameTeamPenaltyFunc(last domain.MatchResult) func(domain.MatchResult) int {
	if len(last.TeamA) == 0 || len(last.TeamB) == 0 {
		return func(domain.MatchResult) int { return 0 }
	}
	lastPairs := teammatePairs(last.TeamA)
	for k := range teammatePairs(last.TeamB) {
		lastPairs[k] = struct{}{}
	}
	return func(result domain.MatchResult) int {
		penalty := 0
		for k := range teammatePairs(result.TeamA) {
			if _, ok := lastPairs[k]; ok {
				penalty++
			}
		}
		for k := range teammatePairs(result.TeamB) {
			if _, ok := lastPairs[k]; ok {
				penalty++
			}
		}
		return penalty
	}
}

func teammatePairs(team []domain.Player) map[string]struct{} {
	pairs := make(map[string]struct{})
	for i := 0; i < len(team); i++ {
		for j := i + 1; j < len(team); j++ {
			a := team[i].ID
			b := team[j].ID
			if a > b {
				a, b = b, a
			}
			pairs[a+":"+b] = struct{}{}
		}
	}
	return pairs
}

func playerIDs(players []domain.Player) []string {
	ids := make([]string, 0, len(players))
	for _, p := range players {
		ids = append(ids, p.ID)
	}
	return ids
}

func applyRatings(players []domain.Player, stats map[string]store.PlayerStat) []domain.Player {
	effective := make([]domain.Player, len(players))
	for i, p := range players {
		st := stats[p.ID]
		p.XPower += domain.ClampRatingDelta(st.RatingDelta)
		effective[i] = p
	}
	return effective
}
