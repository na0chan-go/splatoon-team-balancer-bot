package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		path = "./data.db"
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create sqlite directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Join(guildID, channelID string, player domain.Player) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if player.XPower < minXPower || player.XPower > maxXPower {
		return false, ErrInvalidXPower
	}

	state, _, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil {
		return false, err
	}

	for i, p := range state.Players {
		if p.ID == player.ID {
			state.Players[i] = player
			if err := s.upsertStateLocked(guildID, channelID, state); err != nil {
				return false, err
			}
			return false, nil
		}
	}

	if len(state.Players) >= maxPlayers {
		return false, ErrRoomFull
	}

	state.Players = append(state.Players, player)
	if err := s.upsertStateLocked(guildID, channelID, state); err != nil {
		return false, err
	}

	return true, nil
}

func (s *SQLiteStore) Leave(guildID, channelID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, _, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil {
		return err
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
	return s.upsertStateLocked(guildID, channelID, state)
}

func (s *SQLiteStore) List(guildID, channelID string) []domain.Player {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, _, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil {
		return nil
	}

	players := copyPlayers(state.Players)
	sortPlayers(players)
	return players
}

func (s *SQLiteStore) SaveLastMatch(guildID, channelID string, seed int64, players []domain.Player, result domain.MatchResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, _, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil {
		return
	}

	state.LastSeed = seed
	state.LastPlayersSnapshot = copyPlayers(players)
	state.LastResult = copyResult(result)
	_ = s.upsertStateLocked(guildID, channelID, state)
}

func (s *SQLiteStore) GetState(guildID, channelID string) (RoomState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil || !ok {
		return RoomState{}, false
	}

	return RoomState{
		Players:             copyPlayers(state.Players),
		LastResult:          copyResult(state.LastResult),
		LastSeed:            state.LastSeed,
		LastPlayersSnapshot: copyPlayers(state.LastPlayersSnapshot),
	}, true
}

func (s *SQLiteStore) ResetRoom(guildID, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, _ = s.db.Exec(
		`DELETE FROM room_states WHERE guild_id = ? AND channel_id = ?`,
		guildID, channelID,
	)
}

func (s *SQLiteStore) init() error {
	const schema = `
CREATE TABLE IF NOT EXISTS room_states (
  guild_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  players_json TEXT NOT NULL,
  last_result_json TEXT NOT NULL,
  last_seed INTEGER NOT NULL,
  last_players_snapshot_json TEXT NOT NULL,
  PRIMARY KEY (guild_id, channel_id)
);`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) getRoomStateLocked(guildID, channelID string) (RoomState, bool, error) {
	var playersJSON string
	var lastResultJSON string
	var lastSeed int64
	var lastPlayersSnapshotJSON string

	err := s.db.QueryRow(
		`SELECT players_json, last_result_json, last_seed, last_players_snapshot_json
		 FROM room_states WHERE guild_id = ? AND channel_id = ?`,
		guildID, channelID,
	).Scan(&playersJSON, &lastResultJSON, &lastSeed, &lastPlayersSnapshotJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return RoomState{}, false, nil
	}
	if err != nil {
		return RoomState{}, false, err
	}

	var state RoomState
	if err := json.Unmarshal([]byte(playersJSON), &state.Players); err != nil {
		return RoomState{}, false, fmt.Errorf("failed to unmarshal players: %w", err)
	}
	if err := json.Unmarshal([]byte(lastResultJSON), &state.LastResult); err != nil {
		return RoomState{}, false, fmt.Errorf("failed to unmarshal last result: %w", err)
	}
	if err := json.Unmarshal([]byte(lastPlayersSnapshotJSON), &state.LastPlayersSnapshot); err != nil {
		return RoomState{}, false, fmt.Errorf("failed to unmarshal last players snapshot: %w", err)
	}
	state.LastSeed = lastSeed

	return state, true, nil
}

func (s *SQLiteStore) upsertStateLocked(guildID, channelID string, state RoomState) error {
	playersJSON, err := json.Marshal(state.Players)
	if err != nil {
		return fmt.Errorf("failed to marshal players: %w", err)
	}
	lastResultJSON, err := json.Marshal(state.LastResult)
	if err != nil {
		return fmt.Errorf("failed to marshal last result: %w", err)
	}
	lastPlayersSnapshotJSON, err := json.Marshal(state.LastPlayersSnapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal last players snapshot: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO room_states
		  (guild_id, channel_id, players_json, last_result_json, last_seed, last_players_snapshot_json)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(guild_id, channel_id) DO UPDATE SET
		   players_json = excluded.players_json,
		   last_result_json = excluded.last_result_json,
		   last_seed = excluded.last_seed,
		   last_players_snapshot_json = excluded.last_players_snapshot_json`,
		guildID,
		channelID,
		string(playersJSON),
		string(lastResultJSON),
		state.LastSeed,
		string(lastPlayersSnapshotJSON),
	)
	if err != nil {
		return err
	}
	return nil
}

func sortPlayers(players []domain.Player) {
	sort.Slice(players, func(i, j int) bool {
		if players[i].XPower == players[j].XPower {
			return players[i].ID < players[j].ID
		}
		return players[i].XPower > players[j].XPower
	})
}
