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
	ID     string
	Name   string
	XPower int
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
			candidates = poolCandidates
			continue
		}
		if poolBestDiff == bestDiff {
			candidates = append(candidates, poolCandidates...)
		}
	}

	r := rand.New(rand.NewSource(seed))
	chosen := candidates[r.Intn(len(candidates))]

	return buildResult(players, chosen), nil
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
