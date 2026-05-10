package module

import (
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
)

func TestHelp(t *testing.T) {
	tests := []struct {
		name         string
		rules        []any
		wantKV       map[string]int
		wantContains map[string][]string
	}{
		{
			name:   "command rules produce help",
			rules:  []any{[]command.Rule{{Define: "test cmd", Help: "Run a test"}, {Define: "info", Help: "Show info"}}},
			wantKV: map[string]int{"command": 2},
			wantContains: map[string][]string{
				"command": {"test cmd : Run a test", "info : Show info"},
			},
		},
		{
			name:   "cron rules produce help",
			rules:  []any{[]cron.Rule{{Name: "cleanup", Help: "Clean up old data"}}},
			wantKV: map[string]int{"cron": 1},
			wantContains: map[string][]string{
				"cron": {"cleanup : Clean up old data"},
			},
		},
		{
			name:   "mixed command and cron rules",
			rules:  []any{[]command.Rule{{Define: "cmd", Help: "test"}}, []cron.Rule{{Name: "task", Help: "schedule"}}},
			wantKV: map[string]int{"command": 1, "cron": 1},
		},
		{
			name:   "nil rules returns empty",
			rules:  nil,
			wantKV: map[string]int{},
		},
		{
			name:   "empty rules returns empty",
			rules:  []any{},
			wantKV: map[string]int{},
		},
		{
			name:   "unknown rule type returns empty",
			rules:  []any{"not a rule"},
			wantKV: map[string]int{},
		},
		{
			name:   "empty command rules returns empty",
			rules:  []any{[]command.Rule{}},
			wantKV: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Help(tt.rules)
			require.NoError(t, err)
			for k, wantLen := range tt.wantKV {
				assert.Contains(t, result, k)
				assert.Len(t, result[k], wantLen)
			}
			for k, wantSubstrs := range tt.wantContains {
				for _, wantSubstr := range wantSubstrs {
					assert.Contains(t, result[k], wantSubstr)
				}
			}
			if len(tt.wantKV) == 0 {
				assert.Empty(t, result)
			}
		})
	}
}

func TestBase_DefaultMethodsReturnZero(t *testing.T) {
	var b Base

	tests := []struct {
		name    string
		call    func() error
		wantErr bool
	}{
		{
			name: "Command returns nil payload",
			call: func() error {
				payload, err := b.Command(types.Context{}, nil)
				require.NoError(t, err)
				assert.Nil(t, payload)
				return nil
			},
		},
		{
			name: "Form returns nil payload",
			call: func() error {
				payload, err := b.Form(types.Context{}, nil)
				require.NoError(t, err)
				assert.Nil(t, payload)
				return nil
			},
		},
		{
			name: "Page returns empty HTML",
			call: func() error {
				html, err := b.Page(types.Context{}, "", nil)
				require.NoError(t, err)
				assert.Empty(t, html)
				return nil
			},
		},
		{
			name: "Cron returns nil ruleset",
			call: func() error {
				rs, err := b.Cron()
				require.NoError(t, err)
				assert.Nil(t, rs)
				return nil
			},
		},
		{
			name: "Event returns no error",
			call: func() error {
				err := b.Event(types.Context{}, nil)
				assert.NoError(t, err)
				return nil
			},
		},
		{
			name: "Webhook returns nil payload",
			call: func() error {
				payload, err := b.Webhook(types.Context{}, nil)
				require.NoError(t, err)
				assert.Nil(t, payload)
				return nil
			},
		},
		{
			name: "Input returns nil payload",
			call: func() error {
				payload, err := b.Input(types.Context{}, nil, nil)
				require.NoError(t, err)
				assert.Nil(t, payload)
				return nil
			},
		},
		{
			name: "Bootstrap returns no error",
			call: func() error {
				err := b.Bootstrap()
				assert.NoError(t, err)
				return nil
			},
		},
		{
			name: "Help returns non-nil map",
			call: func() error {
				help, err := b.Help()
				require.NoError(t, err)
				assert.NotNil(t, help)
				return nil
			},
		},
		{
			name: "Rules returns nil",
			call: func() error {
				assert.Nil(t, b.Rules())
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.call()
		})
	}
}

func TestFindRuleAndHandler(t *testing.T) {
	tests := []struct {
		name        string
		handlers    map[string]Handler
		flag        string
		wantRuleID  string
		wantHandler Handler
	}{
		{
			name: "finds rule in single handler",
			handlers: map[string]Handler{
				"test": &testHandler{rules: []any{[]form.Rule{{Id: "form1", Title: "Test Form"}}}},
			},
			flag:       "form1",
			wantRuleID: "form1",
		},
		{
			name: "not found returns zero values",
			handlers: map[string]Handler{
				"test": &testHandler{rules: []any{[]form.Rule{{Id: "form1"}}}},
			},
			flag:       "nonexistent",
			wantRuleID: "",
		},
		{
			name:       "empty handlers returns zero values",
			handlers:   map[string]Handler{},
			flag:       "any",
			wantRuleID: "",
		},
		{
			name: "finds rule across multiple handlers",
			handlers: map[string]Handler{
				"a": &testHandler{rules: []any{[]form.Rule{{Id: "a1"}}}},
				"b": &testHandler{rules: []any{[]form.Rule{{Id: "b1"}}}},
			},
			flag:       "b1",
			wantRuleID: "b1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, handler := FindRuleAndHandler[form.Rule](tt.flag, tt.handlers)
			assert.Equal(t, tt.wantRuleID, rule.Id)
			if tt.wantRuleID != "" {
				assert.NotNil(t, handler)
			} else {
				assert.Nil(t, handler)
			}
		})
	}
}

func TestFindRuleAndHandler_MultipleHandlersSelectsCorrect(t *testing.T) {
	t.Run("picks correct handler from multiple", func(t *testing.T) {
		a := &testHandler{rules: []any{[]form.Rule{{Id: "a1"}}}}
		b := &testHandler{rules: []any{[]form.Rule{{Id: "b1"}}}}

		rule, handler := FindRuleAndHandler[form.Rule]("b1", map[string]Handler{
			"a": a,
			"b": b,
		})
		assert.Equal(t, "b1", rule.Id)
		assert.Equal(t, b, handler)
	})
}

type testHandler struct {
	Base
	rules []any
}

func (m *testHandler) Rules() []any                 { return m.rules }
func (m *testHandler) IsReady() bool                { return true }
func (m *testHandler) Init(_ json.RawMessage) error { return nil }
func (m *testHandler) Webservice(_ *fiber.App)      {}
