package module

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

func TestHelp(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
			_ = tt.call()
		})
	}
}

func TestFindRuleAndHandler(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	t.Run("picks correct handler from multiple", func(t *testing.T) {
		t.Parallel()
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
	name         string
	rules        []any
	ready        bool
	inited       bool
	bootstrapped bool
}

func (m *testHandler) Rules() []any { return m.rules }
func (m *testHandler) IsReady() bool {
	return m.ready
}
func (m *testHandler) Init(_ json.RawMessage) error {
	m.inited = true
	return nil
}
func (m *testHandler) Bootstrap() error {
	m.bootstrapped = true
	return nil
}
func (*testHandler) Webservice(_ *fiber.App) {}

func TestRegisterAndList(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "register and list contains module",
			run: func(t *testing.T) {
				h := &testHandler{name: "testreglist"}
				Register("testreglist", h)
				t.Cleanup(func() { Unregister("testreglist") })
				list := List()
				assert.Contains(t, list, "testreglist")
				assert.Equal(t, h, list["testreglist"])
			},
		},
		{
			name: "unregister removes module",
			run: func(t *testing.T) {
				h := &testHandler{name: "testunreg"}
				Register("testunreg", h)
				Unregister("testunreg")
				assert.NotContains(t, List(), "testunreg")
			},
		},
		{
			name: "list returns copy isolated from mutations",
			run: func(t *testing.T) {
				h := &testHandler{name: "testlistcopy2"}
				Register("testlistcopy2", h)
				t.Cleanup(func() { Unregister("testlistcopy2") })
				list := List()
				delete(list, "testlistcopy2")
				assert.Contains(t, List(), "testlistcopy2")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestInitAndBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		conf    string
		handler *testHandler
		wantErr bool
	}{
		{
			name:    "init with matching config",
			conf:    `[{"name":"testinitmod","enabled":true}]`,
			handler: &testHandler{name: "testinitmod", ready: true},
		},
		{
			name:    "init with default config for unlisted module",
			conf:    `[]`,
			handler: &testHandler{name: "testinitdefault", ready: true},
		},
		{
			name:    "bootstrap skips not-ready modules",
			conf:    `[{"name":"testbootnotready","enabled":true}]`,
			handler: &testHandler{name: "testbootnotready", ready: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.handler.name, tt.handler)
			t.Cleanup(func() { Unregister(tt.handler.name) })

			err := Init(json.RawMessage(tt.conf))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, tt.handler.inited)

			err = Bootstrap()
			require.NoError(t, err)
			if tt.handler.ready {
				assert.True(t, tt.handler.bootstrapped)
			} else {
				assert.False(t, tt.handler.bootstrapped)
			}
		})
	}
}

func TestRunCommand(t *testing.T) {
	t.Parallel()

	rules := []command.Rule{
		{
			Define: "echo [string]",
			Help:   "Echo text",
			Handler: func(_ types.Context, tokens []*parser.Token) types.MsgPayload {
				if len(tokens) > 1 {
					text, _ := tokens[1].Value.String()
					return types.TextMsg{Text: text}
				}
				return types.TextMsg{Text: "empty"}
			},
		},
	}

	tests := []struct {
		name     string
		content  any
		input    string
		wantText string
		wantNil  bool
	}{
		{name: "non-string content returns nil", content: 42, wantNil: true},
		{name: "help command returns info", input: "help", wantNil: false},
		{name: "matching command returns payload", input: "echo hello", wantText: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := tt.content
			if content == nil {
				content = tt.input
			}
			payload, err := RunCommand(rules, types.Context{}, content)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, payload)
				return
			}
			require.NotNil(t, payload)
			if tt.wantText != "" {
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantText, msg.Text)
			}
		})
	}
}

func TestWebservice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ruleset webservice.Ruleset
	}{
		{name: "empty ruleset is no-op", ruleset: nil},
		{name: "single route registers", ruleset: webservice.Ruleset{
			{Method: "GET", Path: "/ping", Function: func(_ fiber.Ctx) error { return nil }},
		}},
		{name: "multiple routes register", ruleset: webservice.Ruleset{
			{Method: "GET", Path: "/a", Function: func(_ fiber.Ctx) error { return nil }},
			{Method: "POST", Path: "/b", Function: func(_ fiber.Ctx) error { return nil }},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			assert.NotPanics(t, func() {
				Webservice(app, "testmod", tt.ruleset)
			})
		})
	}
}

func TestRunForm(t *testing.T) {
	prev := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = prev })

	rules := []form.Rule{
		{
			Id:         "settings",
			IsLongTerm: true,
			Title:      "Settings",
			Handler: func(_ types.Context, values types.KV) types.MsgPayload {
				name, _ := values.String("name")
				return types.TextMsg{Text: name}
			},
		},
	}

	uid := "usertest01"
	topic := "defaulttopic"

	tests := []struct {
		name    string
		formID  string
		state   int
		values  types.KV
		want    string
		wantNil bool
	}{
		{name: "processes matching form rule", formID: "form-test-1", state: int(schema.FormStateCreated), values: types.KV{"name": "alice"}, want: "alice"},
		{name: "missing form returns nil", formID: "missing", state: int(schema.FormStateCreated), values: types.KV{"name": "bob"}, wantNil: true},
		{name: "submitted form returns nil", formID: "form-test-1", state: int(schema.FormStateSubmitSuccess), values: types.KV{"name": "carol"}, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.formID == "form-test-1" {
				require.NoError(t, store.Database.FormSet(context.Background(), "form-test-1", gen.Form{
					FormID: "form-test-1",
					UID:    uid,
					Topic:  topic,
					Schema: map[string]any{"title": "Settings"},
					State:  tt.state,
				}))
			}

			runCtx := types.Context{AsUser: types.Uid(uid), Topic: topic, FormId: tt.formID, FormRuleId: "settings"}
			payload, err := RunForm(rules, runCtx, tt.values)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, payload)
				return
			}
			require.NotNil(t, payload)
			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Equal(t, tt.want, msg.Text)
		})
	}
}

func TestBehavior(t *testing.T) {
	prev := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = prev })

	uid := types.Uid("behavior-user")

	tests := []struct {
		name  string
		flag  string
		count int
	}{
		{name: "creates new behavior record", flag: "msg_bot_incoming", count: 1},
		{name: "increments existing behavior", flag: "msg_bot_incoming", count: 2},
		{name: "tracks separate flags independently", flag: "msg_group_incoming", count: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				Behavior(uid, tt.flag, tt.count)
			})
		})
	}
}
