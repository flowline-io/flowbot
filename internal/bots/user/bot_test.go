package user

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	assert.Equal(t, "user", Name)
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

func TestWebserviceRules_Defined(t *testing.T) {
	assert.NotEmpty(t, webserviceRules)
	assert.GreaterOrEqual(t, len(webserviceRules), 4)
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	handler = bot{initialized: true}
	rules := handler.Rules()
	assert.NotEmpty(t, rules)
	assert.Len(t, rules, 2) // commandRules, webserviceRules
}
