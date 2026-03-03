package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
			player.PauseRemaining = p.PauseRemaining
			player.PauseReason = p.PauseReason
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

func (s *SQLiteStore) Paused(guildID, channelID string) []domain.Player {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, _, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil {
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

func (s *SQLiteStore) SetPause(guildID, channelID, userID string, matches int, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, _, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil {
		return err
	}
	for i, p := range state.Players {
		if p.ID != userID {
			continue
		}
		p.PauseRemaining = matches
		p.PauseReason = reason
		state.Players[i] = p
		return s.upsertStateLocked(guildID, channelID, state)
	}
	return ErrNotJoined
}

func (s *SQLiteStore) Resume(guildID, channelID, userID string) error {
	return s.SetPause(guildID, channelID, userID, 0, "")
}

func (s *SQLiteStore) DecrementPauses(guildID, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok, err := s.getRoomStateLocked(guildID, channelID)
	if err != nil || !ok {
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
	_ = s.upsertStateLocked(guildID, channelID, state)
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
		SpectatorHistory:    copySpectatorHistory(state.SpectatorHistory),
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
	const roomStateSchema = `
CREATE TABLE IF NOT EXISTS room_states (
  guild_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  players_json TEXT NOT NULL,
  last_result_json TEXT NOT NULL,
  last_seed INTEGER NOT NULL,
  last_players_snapshot_json TEXT NOT NULL,
  spectator_history_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (guild_id, channel_id)
);`

	if _, err := s.db.Exec(roomStateSchema); err != nil {
		return fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}
	_, err := s.db.Exec(`ALTER TABLE room_states ADD COLUMN spectator_history_json TEXT NOT NULL DEFAULT '{}'`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("failed to migrate sqlite schema: %w", err)
	}

	const playerStatsSchema = `
CREATE TABLE IF NOT EXISTS player_stats (
  user_id TEXT PRIMARY KEY,
  rating INTEGER NOT NULL DEFAULT 0,
  wins INTEGER NOT NULL DEFAULT 0,
  losses INTEGER NOT NULL DEFAULT 0
);`
	if _, err := s.db.Exec(playerStatsSchema); err != nil {
		return fmt.Errorf("failed to initialize player_stats schema: %w", err)
	}

	const matchesSchema = `
CREATE TABLE IF NOT EXISTS matches (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  guild_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  winner_team TEXT NOT NULL,
  team_a_json TEXT NOT NULL,
  team_b_json TEXT NOT NULL,
  spectators_json TEXT NOT NULL,
  sum_a INTEGER NOT NULL,
  sum_b INTEGER NOT NULL,
  diff INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);`
	if _, err := s.db.Exec(matchesSchema); err != nil {
		return fmt.Errorf("failed to initialize matches schema: %w", err)
	}

	return nil
}

func (s *SQLiteStore) getRoomStateLocked(guildID, channelID string) (RoomState, bool, error) {
	var playersJSON string
	var lastResultJSON string
	var lastSeed int64
	var lastPlayersSnapshotJSON string
	var spectatorHistoryJSON string

	err := s.db.QueryRow(
		`SELECT players_json, last_result_json, last_seed, last_players_snapshot_json, spectator_history_json
		 FROM room_states WHERE guild_id = ? AND channel_id = ?`,
		guildID, channelID,
	).Scan(&playersJSON, &lastResultJSON, &lastSeed, &lastPlayersSnapshotJSON, &spectatorHistoryJSON)
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
	if err := json.Unmarshal([]byte(spectatorHistoryJSON), &state.SpectatorHistory); err != nil {
		return RoomState{}, false, fmt.Errorf("failed to unmarshal spectator history: %w", err)
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
	spectatorHistoryJSON, err := json.Marshal(state.SpectatorHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal spectator history: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO room_states
		  (guild_id, channel_id, players_json, last_result_json, last_seed, last_players_snapshot_json, spectator_history_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(guild_id, channel_id) DO UPDATE SET
		   players_json = excluded.players_json,
		   last_result_json = excluded.last_result_json,
		   last_seed = excluded.last_seed,
		   last_players_snapshot_json = excluded.last_players_snapshot_json,
		   spectator_history_json = excluded.spectator_history_json`,
		guildID,
		channelID,
		string(playersJSON),
		string(lastResultJSON),
		state.LastSeed,
		string(lastPlayersSnapshotJSON),
		string(spectatorHistoryJSON),
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

func (s *SQLiteStore) GetPlayerStats(userIDs []string) map[string]PlayerStat {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := make(map[string]PlayerStat, len(userIDs))
	for _, userID := range userIDs {
		stats[userID] = PlayerStat{UserID: userID}
	}
	if len(userIDs) == 0 {
		return stats
	}

	query, args := inClause("user_id", userIDs)
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT user_id, rating, wins, losses FROM player_stats WHERE %s`, query),
		args...,
	)
	if err != nil {
		return stats
	}
	defer rows.Close()

	for rows.Next() {
		var st PlayerStat
		if err := rows.Scan(&st.UserID, &st.Rating, &st.Wins, &st.Losses); err != nil {
			continue
		}
		st.Rating = clampRating(st.Rating)
		stats[st.UserID] = st
	}
	return stats
}

func (s *SQLiteStore) RecordMatchResult(guildID, channelID, winnerTeam string, result domain.MatchResult) error {
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

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, p := range winners {
		if err := updatePlayerStat(tx, p.ID, true); err != nil {
			return err
		}
	}
	for _, p := range losers {
		if err := updatePlayerStat(tx, p.ID, false); err != nil {
			return err
		}
	}

	teamAJSON, err := json.Marshal(result.TeamA)
	if err != nil {
		return err
	}
	teamBJSON, err := json.Marshal(result.TeamB)
	if err != nil {
		return err
	}
	spectatorsJSON, err := json.Marshal(result.Spectators)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO matches
		   (guild_id, channel_id, winner_team, team_a_json, team_b_json, spectators_json, sum_a, sum_b, diff, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		guildID,
		channelID,
		winnerTeam,
		string(teamAJSON),
		string(teamBJSON),
		string(spectatorsJSON),
		result.SumA,
		result.SumB,
		result.Diff,
		time.Now().Unix(),
	); err != nil {
		return err
	}

	return tx.Commit()
}

func inClause(column string, values []string) (string, []any) {
	parts := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	for _, v := range values {
		parts = append(parts, column+" = ?")
		args = append(args, v)
	}
	return strings.Join(parts, " OR "), args
}

func updatePlayerStat(tx *sql.Tx, userID string, won bool) error {
	var rating, wins, losses int
	err := tx.QueryRow(
		`SELECT rating, wins, losses FROM player_stats WHERE user_id = ?`,
		userID,
	).Scan(&rating, &wins, &losses)
	if errors.Is(err, sql.ErrNoRows) {
		rating = 0
		wins = 0
		losses = 0
	} else if err != nil {
		return err
	}

	if won {
		wins++
		rating = clampRating(rating + 10)
	} else {
		losses++
		rating = clampRating(rating - 10)
	}

	_, err = tx.Exec(
		`INSERT INTO player_stats (user_id, rating, wins, losses)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   rating = excluded.rating,
		   wins = excluded.wins,
		   losses = excluded.losses`,
		userID,
		rating,
		wins,
		losses,
	)
	return err
}
