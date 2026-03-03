package store

import "github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"

type PlayerStat struct {
	UserID       string `json:"user_id"`
	RatingDelta  int    `json:"rating_delta"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	LastPlayedAt int64  `json:"last_played_at"`
}

type MatchRecord struct {
	ID         int64
	GuildID    string
	ChannelID  string
	WinnerTeam string
	TeamA      []domain.Player
	TeamB      []domain.Player
	Spectators []domain.Player
	SumA       int
	SumB       int
	Diff       int
	CreatedAt  int64
}

type Store interface {
	Join(guildID, channelID string, player domain.Player) (bool, error)
	TryMarkOnboardingShown(guildID, channelID string) (bool, error)
	GetRoomSettings(guildID, channelID string) (map[string]string, error)
	SetRoomSetting(guildID, channelID, key, value string) error
	Leave(guildID, channelID, userID string) error
	List(guildID, channelID string) []domain.Player
	Paused(guildID, channelID string) []domain.Player
	SnapshotRoomState(guildID, channelID string)
	UndoRoomState(guildID, channelID string) (bool, error)
	SaveLastMatch(guildID, channelID string, seed int64, players []domain.Player, result domain.MatchResult)
	GetState(guildID, channelID string) (RoomState, bool)
	ReplaceState(guildID, channelID string, state RoomState) error
	ResetRoom(guildID, channelID string)
	SetPause(guildID, channelID, userID string, matches int, reason string) error
	Resume(guildID, channelID, userID string) error
	DecrementPauses(guildID, channelID string)
	GetPlayerStats(userIDs []string) map[string]PlayerStat
	RecordMatchResult(guildID, channelID, winnerTeam string, result domain.MatchResult) error
	GetExportData(guildID, channelID, scope string, limit int) ([]MatchRecord, []PlayerStat, error)
}
