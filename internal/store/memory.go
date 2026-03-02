package store

import (
	"errors"
	"sort"
	"sync"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
)

var (
	ErrRoomFull      = errors.New("room is full")
	ErrNotJoined     = errors.New("player not joined")
	ErrInvalidXPower = errors.New("xpower must be between 0 and 5000")
)

const (
	maxPlayers = 10
	minXPower  = 0
	maxXPower  = 5000
)

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

// Join adds a player to room or updates existing player's profile.
// The returned bool is true when added as a new participant and false when updated.
func (s *MemoryStore) Join(guildID, channelID string, player domain.Player) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if player.XPower < minXPower || player.XPower > maxXPower {
		return false, ErrInvalidXPower
	}

	key := roomKey(guildID, channelID)
	state := s.rooms[key]

	for i, p := range state.Players {
		if p.ID == player.ID {
			state.Players[i] = player
			s.rooms[key] = state
			return false, nil
		}
	}
	if len(state.Players) >= maxPlayers {
		return false, ErrRoomFull
	}

	state.Players = append(state.Players, player)
	s.rooms[key] = state
	return true, nil
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
