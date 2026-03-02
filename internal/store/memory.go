package store

import (
	"errors"
	"sort"
	"sync"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

var (
	ErrRoomFull      = errors.New("room is full")
	ErrAlreadyJoined = errors.New("player already joined")
	ErrNotJoined     = errors.New("player not joined")
)

const maxPlayers = 10

type RoomState struct {
	Players []domain.Player
}

type MemoryStore struct {
	mu    sync.RWMutex
	rooms map[string]RoomState
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms: make(map[string]RoomState),
	}
}

func (s *MemoryStore) Join(guildID, channelID string, player domain.Player) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state := s.rooms[key]

	for _, p := range state.Players {
		if p.ID == player.ID {
			return ErrAlreadyJoined
		}
	}
	if len(state.Players) >= maxPlayers {
		return ErrRoomFull
	}

	state.Players = append(state.Players, player)
	s.rooms[key] = state
	return nil
}

func (s *MemoryStore) Leave(guildID, channelID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok {
		return ErrNotJoined
	}

	index := -1
	for i, p := range state.Players {
		if p.ID == userID {
			index = i
			break
		}
	}
	if index == -1 {
		return ErrNotJoined
	}

	state.Players = append(state.Players[:index], state.Players[index+1:]...)
	s.rooms[key] = state
	return nil
}

func (s *MemoryStore) List(guildID, channelID string) []domain.Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok {
		return nil
	}

	players := make([]domain.Player, len(state.Players))
	copy(players, state.Players)
	sort.Slice(players, func(i, j int) bool {
		if players[i].XPower == players[j].XPower {
			return players[i].ID < players[j].ID
		}
		return players[i].XPower > players[j].XPower
	})
	return players
}

func roomKey(guildID, channelID string) string {
	return guildID + ":" + channelID
}
