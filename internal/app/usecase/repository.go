package usecase

import roomdomain "github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain/room"

type RoomRepository interface {
	Load(guildID, channelID string) (roomdomain.State, error)
	Save(guildID, channelID string, state roomdomain.State) error
}
