package store

import (
	"errors"
	"sort"
	"sync"
	"time"

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
	Players             []domain.Player
	LastResult          domain.MatchResult
	LastSeed            int64
	LastPlayersSnapshot []domain.Player
	SpectatorHistory    map[string]SpectatorHistory
	PreviousState       *RoomStateSnapshot
}

type SpectatorHistory struct {
	SpectatorCount  int   `json:"spectator_count"`
	LastSpectatedAt int64 `json:"last_spectated_at"`
}

type RoomStateSnapshot struct {
	Players             []domain.Player             `json:"players"`
	LastResult          domain.MatchResult          `json:"last_result"`
	LastSeed            int64                       `json:"last_seed"`
	LastPlayersSnapshot []domain.Player             `json:"last_players_snapshot"`
	SpectatorHistory    map[string]SpectatorHistory `json:"spectator_history"`
}

type MemoryStore struct {
	mu          sync.RWMutex
	rooms       map[string]RoomState
	playerStats map[string]PlayerStat
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms:       make(map[string]RoomState),
		playerStats: make(map[string]PlayerStat),
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
			player.PauseRemaining = p.PauseRemaining
			player.PauseReason = p.PauseReason
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

func (s *MemoryStore) SaveLastMatch(guildID, channelID string, seed int64, players []domain.Player, result domain.MatchResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state := s.rooms[key]
	state.LastSeed = seed
	state.LastPlayersSnapshot = copyPlayers(players)
	state.LastResult = copyResult(result)
	if state.SpectatorHistory == nil {
		state.SpectatorHistory = make(map[string]SpectatorHistory)
	}
	now := time.Now().Unix()
	for _, spectator := range result.Spectators {
		h := state.SpectatorHistory[spectator.ID]
		h.SpectatorCount++
		h.LastSpectatedAt = now
		state.SpectatorHistory[spectator.ID] = h
	}
	s.rooms[key] = state
}

func (s *MemoryStore) SnapshotRoomState(guildID, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state := s.rooms[key]
	snap := snapshotFromState(state)
	state.PreviousState = &snap
	s.rooms[key] = state
}

func (s *MemoryStore) UndoRoomState(guildID, channelID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok || state.PreviousState == nil {
		return false, nil
	}

	snapshot := *state.PreviousState
	restored := stateFromSnapshot(snapshot)
	restored.PreviousState = nil
	s.rooms[key] = restored
	return true, nil
}

func (s *MemoryStore) ResetRoom(guildID, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.rooms, roomKey(guildID, channelID))
}

func (s *MemoryStore) GetState(guildID, channelID string) (RoomState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok {
		return RoomState{}, false
	}

	return RoomState{
		Players:             copyPlayers(state.Players),
		LastResult:          copyResult(state.LastResult),
		LastSeed:            state.LastSeed,
		LastPlayersSnapshot: copyPlayers(state.LastPlayersSnapshot),
		SpectatorHistory:    copySpectatorHistory(state.SpectatorHistory),
		PreviousState:       copySnapshot(state.PreviousState),
	}, true
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

func (s *MemoryStore) Paused(guildID, channelID string) []domain.Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok {
		return nil
	}

	var paused []domain.Player
	for _, p := range state.Players {
		if p.PauseRemaining > 0 {
			paused = append(paused, p)
		}
	}
	sort.Slice(paused, func(i, j int) bool {
		if paused[i].PauseRemaining == paused[j].PauseRemaining {
			return paused[i].Name < paused[j].Name
		}
		return paused[i].PauseRemaining > paused[j].PauseRemaining
	})
	return paused
}

func (s *MemoryStore) SetPause(guildID, channelID, userID string, matches int, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok {
		return ErrNotJoined
	}
	for i, p := range state.Players {
		if p.ID != userID {
			continue
		}
		p.PauseRemaining = matches
		p.PauseReason = reason
		state.Players[i] = p
		s.rooms[key] = state
		return nil
	}
	return ErrNotJoined
}

func (s *MemoryStore) Resume(guildID, channelID, userID string) error {
	return s.SetPause(guildID, channelID, userID, 0, "")
}

func (s *MemoryStore) DecrementPauses(guildID, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := roomKey(guildID, channelID)
	state, ok := s.rooms[key]
	if !ok {
		return
	}
	for i, p := range state.Players {
		if p.PauseRemaining <= 0 {
			continue
		}
		p.PauseRemaining--
		if p.PauseRemaining < 0 {
			p.PauseRemaining = 0
		}
		if p.PauseRemaining == 0 {
			p.PauseReason = ""
		}
		state.Players[i] = p
	}
	s.rooms[key] = state
}

func (s *MemoryStore) GetPlayerStats(userIDs []string) map[string]PlayerStat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]PlayerStat, len(userIDs))
	for _, userID := range userIDs {
		st, ok := s.playerStats[userID]
		if !ok {
			stats[userID] = PlayerStat{UserID: userID}
			continue
		}
		stats[userID] = st
	}
	return stats
}

func (s *MemoryStore) RecordMatchResult(guildID, channelID, winnerTeam string, result domain.MatchResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var winners []domain.Player
	var losers []domain.Player
	switch winnerTeam {
	case "alpha":
		winners = result.TeamA
		losers = result.TeamB
	case "bravo":
		winners = result.TeamB
		losers = result.TeamA
	default:
		return errors.New("winner team must be alpha or bravo")
	}

	for _, p := range winners {
		st := s.playerStats[p.ID]
		st.UserID = p.ID
		st.Wins++
		st.Rating = clampRating(st.Rating + 10)
		s.playerStats[p.ID] = st
	}
	for _, p := range losers {
		st := s.playerStats[p.ID]
		st.UserID = p.ID
		st.Losses++
		st.Rating = clampRating(st.Rating - 10)
		s.playerStats[p.ID] = st
	}
	return nil
}

func copyPlayers(players []domain.Player) []domain.Player {
	if len(players) == 0 {
		return nil
	}
	cp := make([]domain.Player, len(players))
	copy(cp, players)
	return cp
}

func copyResult(result domain.MatchResult) domain.MatchResult {
	return domain.MatchResult{
		TeamA:      copyPlayers(result.TeamA),
		TeamB:      copyPlayers(result.TeamB),
		Spectators: copyPlayers(result.Spectators),
		SumA:       result.SumA,
		SumB:       result.SumB,
		Diff:       result.Diff,
	}
}

func copySpectatorHistory(in map[string]SpectatorHistory) map[string]SpectatorHistory {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]SpectatorHistory, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func roomKey(guildID, channelID string) string {
	return guildID + ":" + channelID
}

func snapshotFromState(state RoomState) RoomStateSnapshot {
	return RoomStateSnapshot{
		Players:             copyPlayers(state.Players),
		LastResult:          copyResult(state.LastResult),
		LastSeed:            state.LastSeed,
		LastPlayersSnapshot: copyPlayers(state.LastPlayersSnapshot),
		SpectatorHistory:    copySpectatorHistory(state.SpectatorHistory),
	}
}

func stateFromSnapshot(snapshot RoomStateSnapshot) RoomState {
	return RoomState{
		Players:             copyPlayers(snapshot.Players),
		LastResult:          copyResult(snapshot.LastResult),
		LastSeed:            snapshot.LastSeed,
		LastPlayersSnapshot: copyPlayers(snapshot.LastPlayersSnapshot),
		SpectatorHistory:    copySpectatorHistory(snapshot.SpectatorHistory),
	}
}

func copySnapshot(in *RoomStateSnapshot) *RoomStateSnapshot {
	if in == nil {
		return nil
	}
	cp := snapshotFromState(stateFromSnapshot(*in))
	return &cp
}

func clampRating(r int) int {
	switch {
	case r < -200:
		return -200
	case r > 200:
		return 200
	default:
		return r
	}
}
