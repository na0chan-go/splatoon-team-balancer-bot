package domain

import "testing"

func TestValidateRoomSetting(t *testing.T) {
	tests := []struct {
		key   string
		value string
		ok    bool
	}{
		{RoomSettingMakeNextCooldownSeconds, "3", true},
		{RoomSettingMakeNextCooldownSeconds, "61", false},
		{RoomSettingSpectatorRotationWeight, "0", true},
		{RoomSettingSpectatorRotationWeight, "200", false},
		{RoomSettingSameTeamAvoidanceWeight, "50", true},
		{RoomSettingSameTeamAvoidanceWeight, "-1", false},
		{RoomSettingPauseDefaultMatches, "3", true},
		{RoomSettingPauseDefaultMatches, "0", false},
		{"unknown", "1", false},
	}
	for _, tt := range tests {
		err := ValidateRoomSetting(tt.key, tt.value)
		if (err == nil) != tt.ok {
			t.Fatalf("ValidateRoomSetting(%q,%q) ok=%v err=%v", tt.key, tt.value, tt.ok, err)
		}
	}
}

func TestRoomSettingsFromMap(t *testing.T) {
	got := RoomSettingsFromMap(map[string]string{
		RoomSettingMakeNextCooldownSeconds: "5",
		RoomSettingSpectatorRotationWeight: "2",
		RoomSettingSameTeamAvoidanceWeight: "80",
		RoomSettingPauseDefaultMatches:     "4",
	})
	if got.MakeNextCooldownSeconds != 5 || got.SpectatorRotationWeight != 2 || got.SameTeamAvoidanceWeight != 80 || got.PauseDefaultMatches != 4 {
		t.Fatalf("unexpected settings: %+v", got)
	}
}
