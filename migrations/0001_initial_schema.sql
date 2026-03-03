CREATE TABLE IF NOT EXISTS room_states (
  guild_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  players_json TEXT NOT NULL,
  last_result_json TEXT NOT NULL,
  last_seed INTEGER NOT NULL,
  last_result_at INTEGER NOT NULL DEFAULT 0,
  last_players_snapshot_json TEXT NOT NULL,
  spectator_history_json TEXT NOT NULL DEFAULT '{}',
  participation_counts_json TEXT NOT NULL DEFAULT '{}',
  onboarding_shown INTEGER NOT NULL DEFAULT 0,
  previous_state_json TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (guild_id, channel_id)
);

CREATE TABLE IF NOT EXISTS player_stats (
  user_id TEXT PRIMARY KEY,
  rating INTEGER NOT NULL DEFAULT 0,
  rating_delta INTEGER NOT NULL DEFAULT 0,
  wins INTEGER NOT NULL DEFAULT 0,
  losses INTEGER NOT NULL DEFAULT 0,
  last_played_at INTEGER NOT NULL DEFAULT 0
);

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
);

CREATE TABLE IF NOT EXISTS room_settings (
  guild_id TEXT NOT NULL,
  channel_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (guild_id, channel_id, key)
);
