package life_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/life"
)

func TestRewardOnCooldown(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	hourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name           string
		lastRedeemed   *time.Time
		cooldownHours  int
		wantOnCooldown bool
	}{
		{name: "no cooldown hours", lastRedeemed: &hourAgo, cooldownHours: 0, wantOnCooldown: false},
		{name: "nil last redeemed", lastRedeemed: nil, cooldownHours: 24, wantOnCooldown: false},
		{name: "still cooling", lastRedeemed: &hourAgo, cooldownHours: 3, wantOnCooldown: true},
		{name: "cooldown elapsed", lastRedeemed: &twoHoursAgo, cooldownHours: 1, wantOnCooldown: false},
		{name: "negative hours treated as none", lastRedeemed: &hourAgo, cooldownHours: -1, wantOnCooldown: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantOnCooldown, life.RewardOnCooldown(tt.lastRedeemed, tt.cooldownHours, now))
		})
	}
}

func TestRewardCooldownEndsAt(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	last := now.Add(-1 * time.Hour)

	tests := []struct {
		name          string
		lastRedeemed  *time.Time
		cooldownHours int
		wantNil       bool
		wantEnds      time.Time
	}{
		{name: "not on cooldown", lastRedeemed: &last, cooldownHours: 0, wantNil: true},
		{name: "on cooldown", lastRedeemed: &last, cooldownHours: 3, wantNil: false, wantEnds: last.Add(3 * time.Hour)},
		{name: "nil last", lastRedeemed: nil, cooldownHours: 5, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := life.RewardCooldownEndsAt(tt.lastRedeemed, tt.cooldownHours, now)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
			assert.True(t, tt.wantEnds.Equal(*got))
		})
	}
}
