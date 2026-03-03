package store

import (
	roomdomain "github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain/room"
	botstore "github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

type RoomRepository struct {
	store botstore.Store
}

func NewRoomRepository(s botstore.Store) *RoomRepository {
	return &RoomRepository{store: s}
}

func (r *RoomRepository) Load(guildID, channelID string) (roomdomain.State, error) {
	state, ok := r.store.GetState(guildID, channelID)
	if !ok {
		return roomdomain.State{}, nil
	}
	return state, nil
}

func (r *RoomRepository) Save(guildID, channelID string, state roomdomain.State) error {
	return r.store.ReplaceState(guildID, channelID, state)
}
