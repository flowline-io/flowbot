package dev

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSettingConstants(t *testing.T) {
	assert.Equal(t, "secret", secretSettingKey)
	assert.Equal(t, "number", numberSettingKey)
}

func TestSettingRules_Count(t *testing.T) {
	assert.Len(t, settingRules, 2)
}

func TestSettingRules_Keys(t *testing.T) {
	keys := make(map[string]bool)
	for _, r := range settingRules {
		keys[r.Key] = true
	}

	assert.True(t, keys[secretSettingKey])
	assert.True(t, keys[numberSettingKey])
}

func TestSettingRules_Types(t *testing.T) {
	typeMap := make(map[string]types.FormFieldType)
	for _, r := range settingRules {
		typeMap[r.Key] = r.Type
	}

	assert.Equal(t, types.FormFieldText, typeMap[secretSettingKey])
	assert.Equal(t, types.FormFieldNumber, typeMap[numberSettingKey])
}
