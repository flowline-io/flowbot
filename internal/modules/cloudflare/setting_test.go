package cloudflare

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSettingConstants(t *testing.T) {
	assert.Equal(t, "token", tokenSettingKey)
	assert.Equal(t, "zone_id", zoneIdSettingKey)
	assert.Equal(t, "account_id", accountIdSettingKey)
}

func TestSettingRules_Count(t *testing.T) {
	assert.Len(t, settingRules, 3)
}

func TestSettingRules_Keys(t *testing.T) {
	keys := make(map[string]bool)
	for _, r := range settingRules {
		keys[r.Key] = true
	}

	assert.True(t, keys[tokenSettingKey])
	assert.True(t, keys[zoneIdSettingKey])
	assert.True(t, keys[accountIdSettingKey])
}

func TestSettingRules_Types(t *testing.T) {
	for _, r := range settingRules {
		assert.Equal(t, types.FormFieldText, r.Type)
	}
}

func TestSettingRules_Titles(t *testing.T) {
	titles := make(map[string]string)
	for _, r := range settingRules {
		titles[r.Key] = r.Title
	}

	assert.Equal(t, "Token", titles[tokenSettingKey])
	assert.Equal(t, "Zone Id", titles[zoneIdSettingKey])
	assert.Equal(t, "Account Id", titles[accountIdSettingKey])
}
