package dev

import (
	"encoding/json"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	assert.Equal(t, "dev", Name)
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

	assert.Contains(t, defines, "dev setting")
	assert.Contains(t, defines, "id")
	assert.Contains(t, defines, "form test")
	assert.Contains(t, defines, "queue test")
	assert.Contains(t, defines, "instruct test")
	assert.Contains(t, defines, "page test")
	assert.Contains(t, defines, "docker test")
	assert.Contains(t, defines, "torrent test")
	assert.Contains(t, defines, "slash test")
	assert.Contains(t, defines, "llm test")
	assert.Contains(t, defines, "notify test")
	assert.Contains(t, defines, "fs test")
	assert.Contains(t, defines, "event test")
	assert.Contains(t, defines, "test")
}

func TestCommandRules_HaveHandlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestFormRules_Defined(t *testing.T) {
	assert.NotEmpty(t, formRules)

	found := false
	for _, r := range formRules {
		if r.Id == devFormID {
			found = true
			assert.NotEmpty(t, r.Title)
			assert.NotEmpty(t, r.Field)
			assert.NotNil(t, r.Handler)
		}
	}
	assert.True(t, found, "dev_form rule should be defined")
}

func TestInstructRules_Defined(t *testing.T) {
	assert.NotEmpty(t, instructRules)

	ids := make(map[string]bool)
	for _, r := range instructRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[ExampleInstructID])
}

func TestEventRules_Defined(t *testing.T) {
	assert.NotEmpty(t, eventRules)

	ids := make(map[string]bool)
	for _, r := range eventRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[types.ExampleBotEventID])
}

func TestWebhookRules_Defined(t *testing.T) {
	assert.NotEmpty(t, webhookRules)

	ids := make(map[string]bool)
	for _, r := range webhookRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[ExampleWebhookID])
	assert.True(t, ids[ChatWebhookID])
}

func TestWebhookRules_SecretFlags(t *testing.T) {
	for _, r := range webhookRules {
		assert.True(t, r.Secret, "webhook %q should have Secret=true", r.Id)
	}
}

func TestSettingRules_Defined(t *testing.T) {
	assert.NotEmpty(t, settingRules)

	keys := make(map[string]bool)
	for _, r := range settingRules {
		keys[r.Key] = true
	}

	assert.True(t, keys[secretSettingKey])
	assert.True(t, keys[numberSettingKey])
}

func TestPageRules_Defined(t *testing.T) {
	assert.NotEmpty(t, pageRules)

	ids := make(map[string]bool)
	for _, r := range pageRules {
		ids[r.Id] = true
	}

	assert.True(t, ids["dev"])
}

func TestWebserviceRules_Defined(t *testing.T) {
	assert.NotEmpty(t, webserviceRules)
}

func TestToolRules_Defined(t *testing.T) {
	assert.NotEmpty(t, toolRules)
}

func TestCollectRules_Defined(t *testing.T) {
	assert.NotEmpty(t, collectRules)

	ids := make(map[string]bool)
	for _, r := range collectRules {
		ids[r.Id] = true
	}

	assert.True(t, ids["import_collect"])
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	handler = bot{initialized: true}
	rules := handler.Rules()
	assert.NotEmpty(t, rules)
	assert.Len(t, rules, 9)
}
