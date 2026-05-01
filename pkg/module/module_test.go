package module

import (
	"encoding/json"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelp_CommandRules(t *testing.T) {
	rules := []any{
		[]command.Rule{
			{Define: "test cmd", Help: "Run a test"},
			{Define: "info", Help: "Show info"},
		},
	}
	result, err := Help(rules)
	require.NoError(t, err)
	assert.Contains(t, result, "command")
	assert.Len(t, result["command"], 2)
	assert.Contains(t, result["command"], "test cmd : Run a test")
	assert.Contains(t, result["command"], "info : Show info")
}

func TestHelp_CronRules(t *testing.T) {
	rules := []any{
		[]cron.Rule{
			{Name: "cleanup", Help: "Clean up old data"},
		},
	}
	result, err := Help(rules)
	require.NoError(t, err)
	assert.Contains(t, result, "cron")
	assert.Len(t, result["cron"], 1)
	assert.Contains(t, result["cron"], "cleanup : Clean up old data")
}

func TestHelp_MixedRules(t *testing.T) {
	rules := []any{
		[]command.Rule{{Define: "cmd", Help: "test"}},
		[]cron.Rule{{Name: "task", Help: "schedule"}},
	}
	result, err := Help(rules)
	require.NoError(t, err)
	assert.Contains(t, result, "command")
	assert.Contains(t, result, "cron")
}

func TestHelp_EmptyRules(t *testing.T) {
	result, err := Help(nil)
	require.NoError(t, err)
	assert.Empty(t, result)

	result, err = Help([]any{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestHelp_UnknownType(t *testing.T) {
	result, err := Help([]any{"not a rule"})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestHelp_EmptyCommandRules(t *testing.T) {
	rules := []any{
		[]command.Rule{},
	}
	result, err := Help(rules)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestBase_DefaultMethodsReturnZero(t *testing.T) {
	var b Base

	payload, err := b.Command(types.Context{}, nil)
	assert.NoError(t, err)
	assert.Nil(t, payload)

	payload, err = b.Form(types.Context{}, nil)
	assert.NoError(t, err)
	assert.Nil(t, payload)

	html, err := b.Page(types.Context{}, "", nil)
	assert.NoError(t, err)
	assert.Empty(t, html)

	rs, err := b.Cron()
	assert.NoError(t, err)
	assert.Nil(t, rs)

	err = b.Event(types.Context{}, nil)
	assert.NoError(t, err)

	payload, err = b.Webhook(types.Context{}, nil)
	assert.NoError(t, err)
	assert.Nil(t, payload)

	payload, err = b.Input(types.Context{}, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, payload)

	err = b.Bootstrap()
	assert.NoError(t, err)

	help, err := b.Help()
	assert.NoError(t, err)
	assert.NotNil(t, help)

	assert.Nil(t, b.Rules())
}

func TestFindRuleAndHandler_Found(t *testing.T) {
	h := &testHandler{rules: []any{
		[]form.Rule{{Id: "form1", Title: "Test Form"}},
	}}

	rule, handler := FindRuleAndHandler[form.Rule]("form1", map[string]Handler{
		"test": h,
	})
	assert.Equal(t, "form1", rule.Id)
	assert.NotNil(t, handler)
}

func TestFindRuleAndHandler_NotFound(t *testing.T) {
	h := &testHandler{rules: []any{
		[]form.Rule{{Id: "form1"}},
	}}

	rule, handler := FindRuleAndHandler[form.Rule]("nonexistent", map[string]Handler{
		"test": h,
	})
	assert.Equal(t, "", rule.Id)
	assert.Nil(t, handler)
}

func TestFindRuleAndHandler_EmptyHandlers(t *testing.T) {
	rule, handler := FindRuleAndHandler[form.Rule]("any", map[string]Handler{})
	assert.Equal(t, "", rule.Id)
	assert.Nil(t, handler)
}

func TestFindRuleAndHandler_MultipleHandlers(t *testing.T) {
	a := &testHandler{rules: []any{[]form.Rule{{Id: "a1"}}}}
	b := &testHandler{rules: []any{[]form.Rule{{Id: "b1"}}}}

	rule, handler := FindRuleAndHandler[form.Rule]("b1", map[string]Handler{
		"a": a,
		"b": b,
	})
	assert.Equal(t, "b1", rule.Id)
	assert.Equal(t, b, handler)
}

type testHandler struct {
	Base
	rules []any
}

func (m *testHandler) Rules() []any { return m.rules }
func (m *testHandler) IsReady() bool { return true }
func (m *testHandler) Init(_ json.RawMessage) error { return nil }
func (m *testHandler) Webservice(_ *fiber.App) {}
