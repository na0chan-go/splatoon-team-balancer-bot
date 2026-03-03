package domain

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	RoomSettingMakeNextCooldownSeconds = "make_next_cooldown_seconds"
	RoomSettingSpectatorRotationWeight = "spectator_rotation_weight"
	RoomSettingSameTeamAvoidanceWeight = "same_team_avoidance_weight"
	RoomSettingPauseDefaultMatches     = "pause_default_matches"
)

type RoomSettings struct {
	MakeNextCooldownSeconds int
	SpectatorRotationWeight int
	SameTeamAvoidanceWeight int
	PauseDefaultMatches     int
}

func DefaultRoomSettings() RoomSettings {
	return RoomSettings{
		MakeNextCooldownSeconds: 3,
		SpectatorRotationWeight: 1,
		SameTeamAvoidanceWeight: 50,
		PauseDefaultMatches:     3,
	}
}

func RoomSettingsKeys() []string {
	return []string{
		RoomSettingMakeNextCooldownSeconds,
		RoomSettingSpectatorRotationWeight,
		RoomSettingSameTeamAvoidanceWeight,
		RoomSettingPauseDefaultMatches,
	}
}

func ValidateRoomSetting(key, value string) error {
	key = strings.TrimSpace(key)
	v, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("value must be integer")
	}

	switch key {
	case RoomSettingMakeNextCooldownSeconds:
		if v < 0 || v > 60 {
			return fmt.Errorf("cooldown must be between 0 and 60")
		}
	case RoomSettingSpectatorRotationWeight:
		if v < 0 || v > 100 {
			return fmt.Errorf("spectator_rotation_weight must be between 0 and 100")
		}
	case RoomSettingSameTeamAvoidanceWeight:
		if v < 0 || v > 1000 {
			return fmt.Errorf("same_team_avoidance_weight must be between 0 and 1000")
		}
	case RoomSettingPauseDefaultMatches:
		if v < 1 || v > 20 {
			return fmt.Errorf("pause_default_matches must be between 1 and 20")
		}
	default:
		return fmt.Errorf("unknown setting key")
	}

	return nil
}

func RoomSettingsFromMap(values map[string]string) RoomSettings {
	cfg := DefaultRoomSettings()
	if len(values) == 0 {
		return cfg
	}

	apply := func(key string, set func(int)) {
		raw, ok := values[key]
		if !ok {
			return
		}
		v, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return
		}
		if ValidateRoomSetting(key, raw) != nil {
			return
		}
		set(v)
	}

	apply(RoomSettingMakeNextCooldownSeconds, func(v int) { cfg.MakeNextCooldownSeconds = v })
	apply(RoomSettingSpectatorRotationWeight, func(v int) { cfg.SpectatorRotationWeight = v })
	apply(RoomSettingSameTeamAvoidanceWeight, func(v int) { cfg.SameTeamAvoidanceWeight = v })
	apply(RoomSettingPauseDefaultMatches, func(v int) { cfg.PauseDefaultMatches = v })
	return cfg
}
