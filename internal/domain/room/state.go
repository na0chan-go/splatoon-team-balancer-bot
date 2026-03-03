package room

import "github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"

type State struct {
	Players             []domain.Player
	LastResult          domain.MatchResult
	LastSeed            int64
	LastPlayersSnapshot []domain.Player
	SpectatorHistory    map[string]SpectatorHistory
	ParticipationCounts map[string]int
	OnboardingShown     bool
	PreviousState       *Snapshot
}

type SpectatorHistory struct {
	SpectatorCount  int   `json:"spectator_count"`
	LastSpectatedAt int64 `json:"last_spectated_at"`
}

type Snapshot struct {
	Players             []domain.Player             `json:"players"`
	LastResult          domain.MatchResult          `json:"last_result"`
	LastSeed            int64                       `json:"last_seed"`
	LastPlayersSnapshot []domain.Player             `json:"last_players_snapshot"`
	SpectatorHistory    map[string]SpectatorHistory `json:"spectator_history"`
	ParticipationCounts map[string]int              `json:"participation_counts"`
	OnboardingShown     bool                        `json:"onboarding_shown"`
}
