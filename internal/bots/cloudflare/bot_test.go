package cloudflare

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	assert.Equal(t, "cloudflare", Name)
}

func TestBotInit_Enabled(t *testing.T) {
	handler = bot{} // reset
	config := configType{Enabled: true}
	data, _ := json.Marshal(config)
	err := handler.Init(data)
	require.NoError(t, err)
	assert.True(t, handler.IsReady())
}

func TestBotInit_Disabled(t *testing.T) {
	handler = bot{} // reset
	config := configType{Enabled: false}
	data, _ := json.Marshal(config)
	err := handler.Init(data)
	require.NoError(t, err)
	assert.False(t, handler.IsReady())
}

func TestBotInit_InvalidJSON(t *testing.T) {
	handler = bot{} // reset
	err := handler.Init(json.RawMessage(`{invalid`))
	assert.Error(t, err)
}

func TestBotInit_AlreadyInitialized(t *testing.T) {
	handler = bot{initialized: true}
	err := handler.Init(json.RawMessage(`{"enabled":true}`))
	assert.Error(t, err)
}

func TestCommandRules_Defined(t *testing.T) {
	assert.NotEmpty(t, commandRules)

	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "cloudflare setting")
	assert.Contains(t, defines, "cloudflare test")
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestSettingRules_Defined(t *testing.T) {
	assert.NotEmpty(t, settingRules)

	keys := make(map[string]bool)
	for _, r := range settingRules {
		keys[r.Key] = true
	}

	assert.True(t, keys[tokenSettingKey])
	assert.True(t, keys[zoneIdSettingKey])
	assert.True(t, keys[accountIdSettingKey])
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	handler = bot{initialized: true}
	rules := handler.Rules()
	assert.NotEmpty(t, rules)
	assert.Len(t, rules, 2) // commandRules, formRules
}
