package github

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSettingConstants(t *testing.T) {
	assert.Equal(t, "repo", repoSettingKey)
}

func TestSettingRules_Count(t *testing.T) {
	assert.Len(t, settingRules, 1)
}

func TestSettingRules_Key(t *testing.T) {
	assert.Equal(t, repoSettingKey, settingRules[0].Key)
}

func TestSettingRules_Type(t *testing.T) {
	assert.Equal(t, types.FormFieldText, settingRules[0].Type)
}

func TestSettingRules_Title(t *testing.T) {
	assert.Equal(t, "Repo", settingRules[0].Title)
}
