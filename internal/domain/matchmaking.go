package domain

import (
	"errors"
	"math"
	"math/rand"
)

var (
	ErrNotEnoughPlayers = errors.New("at least 8 players are required")
	ErrTooManyPlayers   = errors.New("at most 10 players are allowed")
)

type Player struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	XPower         int    `json:"xpower"`
	PauseRemaining int    `json:"pause_remaining"`
	PauseReason    string `json:"pause_reason,omitempty"`
}

type MatchResult struct {
	TeamA      []Player
	TeamB      []Player
	Spectators []Player
	SumA       int
	SumB       int
	Diff       int
}

type matchCandidate struct {
	teamAIdx []int
	teamBIdx []int
	sumA     int
	sumB     int
	diff     int
}

// BuildMatch creates a 4v4 match from 8-10 players by exhaustive search.
func BuildMatch(players []Player, seed int64) (MatchResult, error) {
	return buildMatch(players, seed, 0, nil)
}

// BuildMatchWithSpectatorPenalty creates a match with spectator-rotation preference.
// Team balance (Diff) is still primary, and rotation is applied within diffSlack range.
func BuildMatchWithSpectatorPenalty(players []Player, seed int64, diffSlack int, penaltyFn func([]Player) int) (MatchResult, error) {
	return buildMatch(players, seed, diffSlack, penaltyFn)
}

func buildMatch(players []Player, seed int64, diffSlack int, penaltyFn func([]Player) int) (MatchResult, error) {
	if len(players) < 8 {
		return MatchResult{}, ErrNotEnoughPlayers
	}
	if len(players) > 10 {
		return MatchResult{}, ErrTooManyPlayers
	}

	teamPoolCombos := combinations(indexRange(len(players)), 8)
	bestDiff := math.MaxInt
	var candidates []matchCandidate

	for _, pool := range teamPoolCombos {
		poolCandidates, poolBestDiff := bestSplitsForPool(pool, players)
		if poolBestDiff < bestDiff {
			bestDiff = poolBestDiff
		}
		candidates = append(candidates, poolCandidates...)
	}

	candidates = selectCandidatesByBalanceAndPenalty(players, candidates, bestDiff, diffSlack, penaltyFn)
	r := rand.New(rand.NewSource(seed))
	chosen := candidates[r.Intn(len(candidates))]

	return buildResult(players, chosen), nil
}

func selectCandidatesByBalanceAndPenalty(
	players []Player,
	candidates []matchCandidate,
	bestDiff int,
	diffSlack int,
	penaltyFn func([]Player) int,
) []matchCandidate {
	if penaltyFn == nil || diffSlack <= 0 {
		return filterByDiff(candidates, bestDiff)
	}

	limit := bestDiff + diffSlack
	eligible := make([]matchCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.diff <= limit {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) == 0 {
		return filterByDiff(candidates, bestDiff)
	}

	bestPenalty := math.MaxInt
	penaltyByKey := make(map[string]int, len(eligible))
	for _, c := range eligible {
		spectators := spectatorsForCandidate(players, c)
		penalty := penaltyFn(spectators)
		key := candidateKey(c)
		penaltyByKey[key] = penalty
		if penalty < bestPenalty {
			bestPenalty = penalty
		}
	}

	penalized := make([]matchCandidate, 0, len(eligible))
	minDiff := math.MaxInt
	for _, c := range eligible {
		if penaltyByKey[candidateKey(c)] != bestPenalty {
			continue
		}
		if c.diff < minDiff {
			minDiff = c.diff
			penalized = penalized[:0]
			penalized = append(penalized, c)
			continue
		}
		if c.diff == minDiff {
			penalized = append(penalized, c)
		}
	}
	if len(penalized) > 0 {
		return penalized
	}
	return filterByDiff(candidates, bestDiff)
}

func filterByDiff(candidates []matchCandidate, diff int) []matchCandidate {
	filtered := make([]matchCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.diff == diff {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func spectatorsForCandidate(players []Player, c matchCandidate) []Player {
	inMatch := make(map[int]bool, 8)
	for _, idx := range c.teamAIdx {
		inMatch[idx] = true
	}
	for _, idx := range c.teamBIdx {
		inMatch[idx] = true
	}

	spectators := make([]Player, 0, len(players)-8)
	for idx, p := range players {
		if !inMatch[idx] {
			spectators = append(spectators, p)
		}
	}
	return spectators
}

func candidateKey(c matchCandidate) string {
	// Candidate uniqueness is determined by team indices.
	return intsKey(c.teamAIdx) + "|" + intsKey(c.teamBIdx)
}

func intsKey(indices []int) string {
	key := ""
	for i, idx := range indices {
		if i > 0 {
			key += ","
		}
		key += itoa(idx)
	}
	return key
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(buf[i:])
}

func bestSplitsForPool(pool []int, players []Player) ([]matchCandidate, int) {
	// Fix the first player in TeamA to avoid mirrored duplicates.
	first := pool[0]
	rest := pool[1:]
	choose3 := combinations(rest, 3)
	bestDiff := math.MaxInt
	var candidates []matchCandidate

	for _, extraA := range choose3 {
		teamA := make([]int, 0, 4)
		teamA = append(teamA, first)
		teamA = append(teamA, extraA...)

		inA := make(map[int]bool, 4)
		for _, idx := range teamA {
			inA[idx] = true
		}

		teamB := make([]int, 0, 4)
		for _, idx := range pool {
			if !inA[idx] {
				teamB = append(teamB, idx)
			}
		}

		sumA := sumXPower(players, teamA)
		sumB := sumXPower(players, teamB)
		diff := abs(sumA - sumB)

		candidate := matchCandidate{
			teamAIdx: teamA,
			teamBIdx: teamB,
			sumA:     sumA,
			sumB:     sumB,
			diff:     diff,
		}

		if diff < bestDiff {
			bestDiff = diff
			candidates = []matchCandidate{candidate}
			continue
		}
		if diff == bestDiff {
			candidates = append(candidates, candidate)
		}
	}

	return candidates, bestDiff
}

func buildResult(players []Player, candidate matchCandidate) MatchResult {
	teamA := make([]Player, 0, len(candidate.teamAIdx))
	teamB := make([]Player, 0, len(candidate.teamBIdx))

	inMatch := make(map[int]bool, 8)
	for _, idx := range candidate.teamAIdx {
		teamA = append(teamA, players[idx])
		inMatch[idx] = true
	}
	for _, idx := range candidate.teamBIdx {
		teamB = append(teamB, players[idx])
		inMatch[idx] = true
	}

	spectators := make([]Player, 0, len(players)-8)
	for idx, p := range players {
		if !inMatch[idx] {
			spectators = append(spectators, p)
		}
	}

	return MatchResult{
		TeamA:      teamA,
		TeamB:      teamB,
		Spectators: spectators,
		SumA:       candidate.sumA,
		SumB:       candidate.sumB,
		Diff:       candidate.diff,
	}
}

func combinations(items []int, choose int) [][]int {
	var result [][]int
	var current []int

	var dfs func(start int)
	dfs = func(start int) {
		if len(current) == choose {
			comb := make([]int, choose)
			copy(comb, current)
			result = append(result, comb)
			return
		}

		need := choose - len(current)
		for i := start; i <= len(items)-need; i++ {
			current = append(current, items[i])
			dfs(i + 1)
			current = current[:len(current)-1]
		}
	}

	dfs(0)
	return result
}

func indexRange(n int) []int {
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	return indices
}

func sumXPower(players []Player, indices []int) int {
	sum := 0
	for _, idx := range indices {
		sum += players[idx].XPower
	}
	return sum
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
