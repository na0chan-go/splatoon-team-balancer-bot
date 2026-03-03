package usecase

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/domain"
	"github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"
)

var ErrNoLastMake = errors.New("no previous make result")
var ErrNoPreviousMatch = errors.New("no previous match")
var ErrNotInRoom = errors.New("player not in room")

const rotationDiffSlack = 50

type RoomService struct {
	store store.Store
	now   func() time.Time
}

type WhoAmIInfo struct {
	Name               string
	XPower             int
	PauseRemaining     int
	ParticipationCount int
	SpectatorCount     int
	RatingDelta        int
	Wins               int
	Losses             int
}

func NewRoomService(s store.Store) *RoomService {
	return &RoomService{
		store: s,
		now:   time.Now,
	}
}

func (u *RoomService) SetStore(s store.Store) {
	if s != nil {
		u.store = s
	}
}

func (u *RoomService) Join(guildID, channelID string, player domain.Player) (bool, bool, error) {
	created, err := u.store.Join(guildID, channelID, player)
	if err != nil {
		return false, false, err
	}
	showOnboarding := false
	if created {
		showOnboarding, err = u.store.TryMarkOnboardingShown(guildID, channelID)
		if err != nil {
			return false, false, err
		}
	}
	return created, showOnboarding, nil
}

func (u *RoomService) Leave(guildID, channelID, userID string) error {
	return u.store.Leave(guildID, channelID, userID)
}

func (u *RoomService) List(guildID, channelID string) []domain.Player {
	return u.store.List(guildID, channelID)
}

func (u *RoomService) Paused(guildID, channelID string) []domain.Player {
	return u.store.Paused(guildID, channelID)
}

func (u *RoomService) SetPause(guildID, channelID, userID string, matches int, reason string) error {
	return u.store.SetPause(guildID, channelID, userID, matches, reason)
}

func (u *RoomService) Resume(guildID, channelID, userID string) error {
	return u.store.Resume(guildID, channelID, userID)
}

func (u *RoomService) Reset(guildID, channelID string) {
	u.store.ResetRoom(guildID, channelID)
}

func (u *RoomService) Undo(guildID, channelID string) (bool, error) {
	return u.store.UndoRoomState(guildID, channelID)
}

func (u *RoomService) DecrementPauses(guildID, channelID string) {
	u.store.DecrementPauses(guildID, channelID)
}

func (u *RoomService) RoomSettings(guildID, channelID string) (domain.RoomSettings, error) {
	raw, err := u.store.GetRoomSettings(guildID, channelID)
	if err != nil {
		return domain.RoomSettings{}, err
	}
	return domain.RoomSettingsFromMap(raw), nil
}

func (u *RoomService) RoomSettingsMap(s domain.RoomSettings) map[string]string {
	return map[string]string{
		domain.RoomSettingMakeNextCooldownSeconds: strconv.Itoa(s.MakeNextCooldownSeconds),
		domain.RoomSettingSpectatorRotationWeight: strconv.Itoa(s.SpectatorRotationWeight),
		domain.RoomSettingSameTeamAvoidanceWeight: strconv.Itoa(s.SameTeamAvoidanceWeight),
		domain.RoomSettingPauseDefaultMatches:     strconv.Itoa(s.PauseDefaultMatches),
	}
}

func (u *RoomService) UpdateRoomSetting(guildID, channelID, key, value string) error {
	if err := domain.ValidateRoomSetting(key, value); err != nil {
		return err
	}
	return u.store.SetRoomSetting(guildID, channelID, key, value)
}

func (u *RoomService) Make(guildID, channelID string, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	players := u.store.List(guildID, channelID)
	u.store.SnapshotRoomState(guildID, channelID)
	return u.RunMatchWithPlayers(guildID, channelID, players, settings, seed)
}

func (u *RoomService) Reroll(guildID, channelID string, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	state, ok := u.store.GetState(guildID, channelID)
	if !ok || len(state.LastPlayersSnapshot) == 0 {
		return domain.MatchResult{}, ErrNoLastMake
	}
	return u.RunMatchWithPlayers(guildID, channelID, state.LastPlayersSnapshot, settings, seed)
}

func (u *RoomService) Next(guildID, channelID string, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	state, ok := u.store.GetState(guildID, channelID)
	if !ok || len(state.LastResult.TeamA) == 0 || len(state.LastResult.TeamB) == 0 {
		return domain.MatchResult{}, ErrNoPreviousMatch
	}
	u.store.SnapshotRoomState(guildID, channelID)
	defer u.store.DecrementPauses(guildID, channelID)

	players := u.store.List(guildID, channelID)
	active := make([]domain.Player, 0, len(players))
	for _, p := range players {
		if p.PauseRemaining > 0 {
			continue
		}
		active = append(active, p)
	}
	return u.RunMatchWithPlayers(guildID, channelID, active, settings, seed)
}

func (u *RoomService) WhoAmI(guildID, channelID, userID string) (WhoAmIInfo, error) {
	state, ok := u.store.GetState(guildID, channelID)
	if !ok {
		return WhoAmIInfo{}, ErrNotInRoom
	}
	var player *domain.Player
	for i := range state.Players {
		if state.Players[i].ID == userID {
			player = &state.Players[i]
			break
		}
	}
	if player == nil {
		return WhoAmIInfo{}, ErrNotInRoom
	}
	stats := u.store.GetPlayerStats([]string{userID})[userID]
	return WhoAmIInfo{
		Name:               player.Name,
		XPower:             player.XPower,
		PauseRemaining:     player.PauseRemaining,
		ParticipationCount: state.ParticipationCounts[userID],
		SpectatorCount:     state.SpectatorHistory[userID].SpectatorCount,
		RatingDelta:        stats.RatingDelta,
		Wins:               stats.Wins,
		Losses:             stats.Losses,
	}, nil
}

func (u *RoomService) RecordResult(guildID, channelID, winner string) error {
	state, ok := u.store.GetState(guildID, channelID)
	if !ok || len(state.LastResult.TeamA) == 0 || len(state.LastResult.TeamB) == 0 {
		return ErrNoPreviousMatch
	}
	return u.store.RecordMatchResult(guildID, channelID, winner, state.LastResult)
}

func (u *RoomService) Export(guildID, channelID, scope string, limit int) ([]store.MatchRecord, []store.PlayerStat, error) {
	return u.store.GetExportData(guildID, channelID, scope, limit)
}

func (u *RoomService) RunMatchWithPlayers(guildID, channelID string, players []domain.Player, settings domain.RoomSettings, seed int64) (domain.MatchResult, error) {
	state, _ := u.store.GetState(guildID, channelID)
	penaltyFn := combinedPenaltyFunc(state, settings, u.now().Unix())
	effectivePlayers := applyRatings(players, u.store.GetPlayerStats(playerIDs(players)))

	result, err := domain.BuildMatchWithResultPenalty(effectivePlayers, seed, rotationDiffSlack, penaltyFn)
	if err != nil {
		return domain.MatchResult{}, err
	}
	u.store.SaveLastMatch(guildID, channelID, seed, players, result)
	return result, nil
}

func SameTeamPenalty(last domain.MatchResult, result domain.MatchResult) int {
	return sameTeamPenaltyFunc(last)(result)
}

func combinedPenaltyFunc(state store.RoomState, settings domain.RoomSettings, nowUnix int64) func(domain.MatchResult) int {
	spectatorPenalty := spectatorPenaltyFunc(state.SpectatorHistory, nowUnix)
	sameTeamPenalty := sameTeamPenaltyFunc(state.LastResult)
	return func(result domain.MatchResult) int {
		total := 0
		if settings.SpectatorRotationWeight > 0 {
			total += spectatorPenalty(result.Spectators) * settings.SpectatorRotationWeight
		}
		if settings.SameTeamAvoidanceWeight > 0 {
			total += sameTeamPenalty(result) * settings.SameTeamAvoidanceWeight
		}
		return total
	}
}

func spectatorPenaltyFunc(history map[string]store.SpectatorHistory, nowUnix int64) func([]domain.Player) int {
	return func(spectators []domain.Player) int {
		if len(history) == 0 || len(spectators) == 0 {
			return 0
		}
		penalty := 0
		for _, p := range spectators {
			h := history[p.ID]
			penalty += h.SpectatorCount * 100
			if h.LastSpectatedAt <= 0 {
				continue
			}
			age := nowUnix - h.LastSpectatedAt
			switch {
			case age < 3600:
				penalty += 300
			case age < 6*3600:
				penalty += 150
			case age < 24*3600:
				penalty += 60
			}
		}
		return penalty
	}
}

func sameTeamPenaltyFunc(last domain.MatchResult) func(domain.MatchResult) int {
	if len(last.TeamA) == 0 || len(last.TeamB) == 0 {
		return func(domain.MatchResult) int { return 0 }
	}
	lastPairs := teammatePairs(last.TeamA)
	for k := range teammatePairs(last.TeamB) {
		lastPairs[k] = struct{}{}
	}
	return func(result domain.MatchResult) int {
		penalty := 0
		for k := range teammatePairs(result.TeamA) {
			if _, ok := lastPairs[k]; ok {
				penalty++
			}
		}
		for k := range teammatePairs(result.TeamB) {
			if _, ok := lastPairs[k]; ok {
				penalty++
			}
		}
		return penalty
	}
}

func teammatePairs(team []domain.Player) map[string]struct{} {
	pairs := make(map[string]struct{})
	for i := 0; i < len(team); i++ {
		for j := i + 1; j < len(team); j++ {
			a := team[i].ID
			b := team[j].ID
			if a > b {
				a, b = b, a
			}
			pairs[a+":"+b] = struct{}{}
		}
	}
	return pairs
}

func playerIDs(players []domain.Player) []string {
	ids := make([]string, 0, len(players))
	for _, p := range players {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids
}

func applyRatings(players []domain.Player, stats map[string]store.PlayerStat) []domain.Player {
	effective := make([]domain.Player, len(players))
	for i, p := range players {
		st := stats[p.ID]
		p.XPower += domain.ClampRatingDelta(st.RatingDelta)
		effective[i] = p
	}
	return effective
}
