package store

import "github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"

type Store interface {
	Join(guildID, channelID string, player domain.Player) (bool, error)
	Leave(guildID, channelID, userID string) error
	List(guildID, channelID string) []domain.Player
	SaveLastMatch(guildID, channelID string, seed int64, players []domain.Player, result domain.MatchResult)
	GetState(guildID, channelID string) (RoomState, bool)
	ResetRoom(guildID, channelID string)
}
