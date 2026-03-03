package store

import "github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"

type PlayerStat struct {
	UserID string
	Rating int
	Wins   int
	Losses int
}

type Store interface {
	Join(guildID, channelID string, player domain.Player) (bool, error)
	TryMarkOnboardingShown(guildID, channelID string) (bool, error)
	Leave(guildID, channelID, userID string) error
	List(guildID, channelID string) []domain.Player
	Paused(guildID, channelID string) []domain.Player
	SnapshotRoomState(guildID, channelID string)
	UndoRoomState(guildID, channelID string) (bool, error)
	SaveLastMatch(guildID, channelID string, seed int64, players []domain.Player, result domain.MatchResult)
	GetState(guildID, channelID string) (RoomState, bool)
	ResetRoom(guildID, channelID string)
	SetPause(guildID, channelID, userID string, matches int, reason string) error
	Resume(guildID, channelID, userID string) error
	DecrementPauses(guildID, channelID string)
	GetPlayerStats(userIDs []string) map[string]PlayerStat
	RecordMatchResult(guildID, channelID, winnerTeam string, result domain.MatchResult) error
}
