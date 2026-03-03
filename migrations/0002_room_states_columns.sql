ALTER TABLE room_states ADD COLUMN spectator_history_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE room_states ADD COLUMN previous_state_json TEXT NOT NULL DEFAULT '';
ALTER TABLE room_states ADD COLUMN participation_counts_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE room_states ADD COLUMN onboarding_shown INTEGER NOT NULL DEFAULT 0;

