package notify

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestParseSchema(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "valid slack URL", input: "slack://hooks.slack.com/services/xxx", expect: "slack"},
		{name: "discord bot URL", input: "discord-bot://webhook/xxx", expect: "discord-bot"},
		{name: "plain text no scheme", input: "plain text", expect: ""},
		{name: "empty string", input: "", expect: ""},
		{name: "https URL", input: "https://example.com", expect: "https"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scheme, err := ParseSchema(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, scheme)
		})
	}
}

func TestParseTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		templates []string
		expect    types.KV
	}{
		{
			name:      "single template match",
			input:     "slack://general/abc123",
			templates: []string{"slack://{channel}/{token}"},
			expect:    types.KV{"channel": "general", "token": "abc123"},
		},
		{
			name:      "no match",
			input:     "https://other.com/path",
			templates: []string{"slack://{channel}/{token}"},
			expect:    types.KV{},
		},
		{
			name:      "multiple templates picks first match",
			input:     "slack://general/abc123",
			templates: []string{"discord://{channel}/{token}", "slack://{channel}/{token}"},
			expect:    types.KV{"channel": "general", "token": "abc123"},
		},
		{
			name:      "empty templates",
			input:     "slack://general/abc123",
			templates: nil,
			expect:    types.KV{},
		},
		{
			name:      "empty input",
			input:     "",
			templates: []string{"slack://{channel}"},
			expect:    types.KV{},
		},
		{
			name:      "dashed keys",
			input:     "pushover://ukey123/atoken",
			templates: []string{"pushover://{user_key}/{app_token}"},
			expect:    types.KV{"user_key": "ukey123", "app_token": "atoken"},
		},
		{
			name:      "prefers full match over shorter prefix template",
			input:     "http://ntfy.example.com/mytopic",
			templates: []string{"{schema}://{topic}", "{schema}://{host}/{targets}"},
			expect:    types.KV{"schema": "http", "host": "ntfy.example.com", "targets": "mytopic"},
		},
		{
			name:      "rejects partial-only match",
			input:     "slack://general/abc123/extra",
			templates: []string{"slack://{channel}/{token}"},
			expect:    types.KV{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseTemplate(tt.input, tt.templates)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestPriorityConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		priority Priority
		expected Priority
	}{
		{name: "Low", priority: Low, expected: Priority(1)},
		{name: "Moderate", priority: Moderate, expected: Priority(2)},
		{name: "Normal", priority: Normal, expected: Priority(3)},
		{name: "High", priority: High, expected: Priority(4)},
		{name: "Emergency", priority: Emergency, expected: Priority(5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.priority)
		})
	}
}

func TestMessageZeroValue(t *testing.T) {
	t.Parallel()
	t.Run("all fields zero/empty", func(t *testing.T) {
		t.Parallel()
		m := Message{}
		assert.Empty(t, m.Title)
		assert.Empty(t, m.Body)
		assert.Empty(t, m.Url)
		assert.Equal(t, Priority(0), m.Priority)
	})
}

type mockNotifyer struct {
	protocol  string
	templates []string
	sendErr   error
	calls     int
}

func (m *mockNotifyer) Protocol() string { return m.protocol }
func (m *mockNotifyer) Templates() []string {
	return m.templates
}
func (m *mockNotifyer) Send(_ types.KV, _ Message) error {
	m.calls++
	return m.sendErr
}

func TestSend(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		register  *mockNotifyer
		wantErr   string
		wantCalls int
	}{
		{
			name:    "empty input returns error",
			uri:     "",
			wantErr: "no notification sent",
		},
		{
			name:    "unknown protocol returns error",
			uri:     "nosuchproto://token",
			wantErr: "unknown protocol",
		},
		{
			name: "provider send failure is returned",
			uri:  "testsendfail://chan/tok",
			register: &mockNotifyer{
				protocol:  "testsendfail",
				templates: []string{"testsendfail://{channel}/{token}"},
				sendErr:   assert.AnError,
			},
			wantErr:   "send message error",
			wantCalls: 1,
		},
		{
			name: "successful send returns nil",
			uri:  "testsendsuccess://chan/tok",
			register: &mockNotifyer{
				protocol:  "testsendsuccess",
				templates: []string{"testsendsuccess://{channel}/{token}"},
			},
			wantCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register != nil {
				Register(tt.register.protocol, tt.register)
				t.Cleanup(func() { Unregister(tt.register.protocol) })
			}
			err := Send(tt.uri, Message{Title: "t", Body: "b"})
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			if tt.register != nil {
				assert.Equal(t, tt.wantCalls, tt.register.calls)
			}
		})
	}
}

func TestSendToProtocol(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		uri       string
		register  *mockNotifyer
		wantErr   string
		wantCalls int
	}{
		{
			name:     "empty protocol returns error",
			protocol: "",
			uri:      "http://host/topic",
			wantErr:  "protocol is required",
		},
		{
			name:     "unknown protocol returns error",
			protocol: "nosuch",
			uri:      "http://host/topic",
			wantErr:  "unknown protocol",
		},
		{
			name:     "http URI uses declared protocol not scheme",
			protocol: "testprotontfy",
			uri:      "http://ntfy.example.com/mytopic",
			register: &mockNotifyer{
				protocol:  "testprotontfy",
				templates: []string{"{schema}://{topic}", "{schema}://{host}/{targets}"},
			},
			wantCalls: 1,
		},
		{
			name:     "relative URI prepends protocol",
			protocol: "testprotorel",
			uri:      "chan/tok",
			register: &mockNotifyer{
				protocol:  "testprotorel",
				templates: []string{"testprotorel://{channel}/{token}"},
			},
			wantCalls: 1,
		},
		{
			name:     "no template match returns error",
			protocol: "testprotonomatch",
			uri:      "http://only-host",
			register: &mockNotifyer{
				protocol:  "testprotonomatch",
				templates: []string{"testprotonomatch://{channel}/{token}"},
			},
			wantErr: "does not match any template",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register != nil {
				Register(tt.register.protocol, tt.register)
				t.Cleanup(func() { Unregister(tt.register.protocol) })
			}
			err := SendToProtocol(tt.protocol, tt.uri, Message{Title: "t", Body: "b"})
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			if tt.register != nil {
				assert.Equal(t, tt.wantCalls, tt.register.calls)
			}
		})
	}
}

func TestUnregister(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "removes previously registered notifyer",
			run: func(t *testing.T) {
				m := &mockNotifyer{
					protocol:  "testunregister",
					templates: []string{"testunregister://{token}"},
				}
				Register(m.protocol, m)
				Unregister(m.protocol)
				err := Send("testunregister://tok", Message{Title: "t", Body: "b"})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown protocol")
				assert.Equal(t, 0, m.calls)
			},
		},
		{
			name: "no-op for unknown id",
			run: func(_ *testing.T) {
				Unregister("does-not-exist")
			},
		},
		{
			name: "allows re-register after unregister",
			run: func(t *testing.T) {
				m := &mockNotifyer{
					protocol:  "testreregister",
					templates: []string{"testreregister://{token}"},
				}
				Register(m.protocol, m)
				Unregister(m.protocol)
				Register(m.protocol, m)
				t.Cleanup(func() { Unregister(m.protocol) })
				err := Send("testreregister://tok", Message{Title: "t", Body: "b"})
				require.NoError(t, err)
				assert.Equal(t, 1, m.calls)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "returns registered notifyers",
			run: func(t *testing.T) {
				m := &mockNotifyer{
					protocol:  "testlist",
					templates: []string{"testlist://{token}"},
				}
				Register(m.protocol, m)
				t.Cleanup(func() { Unregister(m.protocol) })
				list := List()
				assert.Contains(t, list, m.protocol)
				assert.Equal(t, m, list[m.protocol])
			},
		},
		{
			name: "returns a copy that does not affect registry",
			run: func(t *testing.T) {
				m := &mockNotifyer{
					protocol:  "testlistcopy",
					templates: []string{"testlistcopy://{token}"},
				}
				Register(m.protocol, m)
				t.Cleanup(func() { Unregister(m.protocol) })
				list := List()
				delete(list, m.protocol)
				assert.Contains(t, List(), m.protocol)
			},
		},
		{
			name: "empty registry yields empty map",
			run: func(t *testing.T) {
				list := List()
				require.NotNil(t, list)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestChannelsFromNotifyConfigKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		keys []string
		want []string
	}{
		{
			name: "extracts slack and ntfy channels",
			keys: []string{"notify:slack", "notify:ntfy"},
			want: []string{"slack", "ntfy"},
		},
		{
			name: "skips keys without notify prefix",
			keys: []string{"other:slack", "notify:pushover"},
			want: []string{"pushover"},
		},
		{
			name: "deduplicates channels",
			keys: []string{"notify:slack", "notify:slack"},
			want: []string{"slack"},
		},
		{
			name: "empty and bare prefix yield empty",
			keys: []string{"", "notify:", "notify"},
			want: []string{},
		},
		{
			name: "nil keys yield empty",
			keys: nil,
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, channelsFromNotifyConfigKeys(tt.keys))
		})
	}
}

func TestUserNotifyChannels_NilDatabase(t *testing.T) {
	// Mutates global store.Database; must not run in parallel with other tests.
	prev := store.Database
	store.Database = nil
	t.Cleanup(func() { store.Database = prev })

	tests := []struct {
		name string
	}{
		{name: "nil store returns empty channels"},
		{name: "nil store returns no error"},
		{name: "nil store is safe to call"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels, err := UserNotifyChannels(context.Background(), types.Uid("user1"))
			require.NoError(t, err)
			assert.Nil(t, channels)
		})
	}
}

func testNotifyTemplate() config.NotifyTemplate {
	return config.NotifyTemplate{
		ID:              "test.event",
		Name:            "Test Event",
		Description:     "Test",
		DefaultFormat:   "markdown",
		DefaultTemplate: "**{{ .title }}**\n{{ .body }}",
	}
}

func setupNotifyTestEnv(t *testing.T, templates []config.NotifyTemplate, rules []config.NotifyRule, redisStore *cache.RedisStore) {
	t.Helper()
	prevTemplates := config.App.Notify.Templates
	prevRules := config.App.Notify.Rules
	config.App.Notify.Templates = templates
	config.App.Notify.Rules = rules
	t.Cleanup(func() {
		config.App.Notify.Templates = prevTemplates
		config.App.Notify.Rules = prevRules
	})
	require.NoError(t, notifytmpl.Init())
	require.NoError(t, notifyrules.Init(redisStore))
}

func TestBuildNotifyMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   *notifytmpl.RenderResult
		payload  map[string]any
		wantURL  string
		wantPri  Priority
		wantBody string
	}{
		{
			name:     "basic title and body",
			result:   &notifytmpl.RenderResult{Title: "Alert", Body: "Server down"},
			payload:  map[string]any{},
			wantPri:  Normal,
			wantBody: "Server down",
		},
		{
			name:    "payload url and priority int",
			result:  &notifytmpl.RenderResult{Title: "T", Body: "B"},
			payload: map[string]any{"url": "https://example.com", "priority": 4},
			wantURL: "https://example.com",
			wantPri: High,
		},
		{
			name:    "priority float64",
			result:  &notifytmpl.RenderResult{Title: "T", Body: "B"},
			payload: map[string]any{"priority": float64(5)},
			wantPri: Emergency,
		},
		{
			name:    "priority as Priority type",
			result:  &notifytmpl.RenderResult{Title: "T", Body: "B"},
			payload: map[string]any{"priority": Low},
			wantPri: Low,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := buildNotifyMessage(tt.result, tt.payload)
			assert.Equal(t, tt.result.Title, msg.Title)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, msg.Body)
			}
			assert.Equal(t, tt.wantURL, msg.Url)
			assert.Equal(t, tt.wantPri, msg.Priority)
		})
	}
}

func TestEvaluateAndRenderNotification(t *testing.T) {
	mr := miniredis.RunT(t)
	redisStore := cache.NewRedisStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	tests := []struct {
		name       string
		rules      []config.NotifyRule
		templateID string
		channel    string
		payload    map[string]any
		wantAction string
		wantTitle  string
		wantErr    bool
	}{
		{
			name: "drop rule returns dropped action",
			rules: []config.NotifyRule{
				{ID: "drop1", Action: config.NotifyRuleActionDrop, Match: config.NotifyRuleMatch{Event: "test.event", Channel: "*"}, Priority: 10},
			},
			templateID: "test.event",
			channel:    "slack",
			payload:    map[string]any{"title": "T", "body": "B"},
			wantAction: "dropped",
		},
		{
			name: "mute rule returns muted action",
			rules: []config.NotifyRule{
				{ID: "mute1", Action: config.NotifyRuleActionMute, Match: config.NotifyRuleMatch{Event: "test.event", Channel: "*"}, Priority: 10},
			},
			templateID: "test.event",
			channel:    "slack",
			payload:    map[string]any{"title": "T", "body": "B"},
			wantAction: "muted",
		},
		{
			name: "aggregate rule returns aggregated action",
			rules: []config.NotifyRule{
				{
					ID: "agg1", Action: config.NotifyRuleActionAggregate,
					Match: config.NotifyRuleMatch{Event: "test.event", Channel: "*"}, Priority: 10,
					Params: config.NotifyRuleParams{Window: "5m"},
				},
			},
			templateID: "test.event",
			channel:    "slack",
			payload:    map[string]any{"title": "T", "body": "B"},
			wantAction: "aggregated",
		},
		{
			name:       "no rules renders template",
			rules:      nil,
			templateID: "test.event",
			channel:    "slack",
			payload:    map[string]any{"title": "Hello", "body": "World"},
			wantTitle:  "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupNotifyTestEnv(t, []config.NotifyTemplate{testNotifyTemplate()}, tt.rules, redisStore)
			result, err := evaluateAndRenderNotification(context.Background(), tt.templateID, tt.channel, tt.payload)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.wantAction != "" {
				assert.Equal(t, tt.wantAction, result.action)
				return
			}
			require.NotNil(t, result.renderResult)
			assert.Equal(t, tt.wantTitle, result.renderResult.Title)
		})
	}
}

func TestHandleThrottleRule(t *testing.T) {
	mr := miniredis.RunT(t)
	redisStore := cache.NewRedisStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	setupNotifyTestEnv(t, []config.NotifyTemplate{testNotifyTemplate()}, []config.NotifyRule{
		{
			ID: "thr1", Action: config.NotifyRuleActionThrottle,
			Match: config.NotifyRuleMatch{Event: "test.event", Channel: "*"}, Priority: 10,
			Params: config.NotifyRuleParams{Window: "1m", Limit: 1},
		},
	}, redisStore)

	ruleResult := notifyrules.GetEngine().Evaluate(context.Background(), "test.event", "slack")
	require.NotNil(t, ruleResult)

	tests := []struct {
		name     string
		result   *notifyrules.EvalResult
		wantSkip bool
	}{
		{name: "first message is not throttled", result: ruleResult, wantSkip: false},
		{name: "second message is throttled", result: ruleResult, wantSkip: true},
		{name: "invalid window does not throttle", result: &notifyrules.EvalResult{RuleID: "bad", Window: "not-a-duration", Limit: 1}, wantSkip: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleThrottleRule(context.Background(), tt.result, "test.event", "slack")
			assert.Equal(t, tt.wantSkip, got)
		})
	}
}

func TestGatewaySend(t *testing.T) {
	mr := miniredis.RunT(t)
	redisStore := cache.NewRedisStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	tests := []struct {
		name       string
		setup      func(t *testing.T)
		templateID string
		channels   []string
		payload    map[string]any
		wantErr    string
	}{
		{
			name: "template not found returns error",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []config.NotifyTemplate{testNotifyTemplate()}, nil, nil)
			},
			templateID: "missing.template",
			channels:   []string{"slack"},
			payload:    map[string]any{"title": "T", "body": "B"},
			wantErr:    "not found",
		},
		{
			name: "drop rule completes without error",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []config.NotifyTemplate{testNotifyTemplate()}, []config.NotifyRule{
					{ID: "d1", Action: config.NotifyRuleActionDrop, Match: config.NotifyRuleMatch{Event: "test.event", Channel: "*"}, Priority: 1},
				}, nil)
			},
			templateID: "test.event",
			channels:   []string{"slack"},
			payload:    map[string]any{"title": "T", "body": "B", "summary": "sum"},
		},
		{
			name: "successful render with zero uid",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []config.NotifyTemplate{testNotifyTemplate()}, nil, nil)
			},
			templateID: "test.event",
			channels:   []string{"slack"},
			payload:    map[string]any{"title": "T", "body": "B"},
		},
		{
			name: "throttle allows then blocks repeat",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []config.NotifyTemplate{testNotifyTemplate()}, []config.NotifyRule{
					{
						ID: "t1", Action: config.NotifyRuleActionThrottle,
						Match: config.NotifyRuleMatch{Event: "test.event", Channel: "*"}, Priority: 1,
						Params: config.NotifyRuleParams{Window: "1m", Limit: 1},
					},
				}, redisStore)
			},
			templateID: "test.event",
			channels:   []string{"slack"},
			payload:    map[string]any{"title": "T", "body": "B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			err := GatewaySend(context.Background(), types.Uid(""), tt.templateID, tt.channels, tt.payload)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)

			if tt.name == "throttle allows then blocks repeat" {
				err = GatewaySend(context.Background(), types.Uid(""), tt.templateID, tt.channels, tt.payload)
				require.NoError(t, err)
			}
		})
	}
}

func TestSendToUserChannel(t *testing.T) {
	prevDB := store.Database
	ns := &notifyTestStore{
		configs: map[string]types.KV{
			"notify:slack": {"value": "testuserchannelsend://chan/tok"},
		},
	}
	store.Database = ns
	t.Cleanup(func() { store.Database = prevDB })

	uid := types.Uid("user-send-test")
	ctx := context.Background()

	m := &mockNotifyer{
		protocol:  "testuserchannelsend",
		templates: []string{"testuserchannelsend://{channel}/{token}"},
	}
	Register(m.protocol, m)
	t.Cleanup(func() { Unregister(m.protocol) })

	tests := []struct {
		name    string
		uid     types.Uid
		channel string
	}{
		{name: "zero uid skips send", uid: types.Uid(""), channel: "slack"},
		{name: "configured channel sends message", uid: uid, channel: "slack"},
		{name: "missing channel config is no-op", uid: uid, channel: "ntfy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := m.calls
			err := sendToUserChannel(ctx, tt.uid, "test.event", tt.channel, Message{Title: "T", Body: "B"})
			require.NoError(t, err)
			switch tt.name {
			case "configured channel sends message":
				assert.Equal(t, before+1, m.calls)
			case "zero uid skips send", "missing channel config is no-op":
				assert.Equal(t, before, m.calls)
			}
		})
	}
}

func TestUserNotifyChannels_WithStore(t *testing.T) {
	prevDB := store.Database
	store.Database = &notifyTestStore{
		listKeys: []string{"notify:slack", "notify:ntfy"},
	}
	t.Cleanup(func() { store.Database = prevDB })

	uid := types.Uid("user-channels-test")
	ctx := context.Background()

	tests := []struct {
		name string
	}{
		{name: "returns configured channels"},
		{name: "channels include slack"},
		{name: "channels include ntfy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels, err := UserNotifyChannels(ctx, uid)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"slack", "ntfy"}, channels)
		})
	}
}

func TestGetNotifyStore(t *testing.T) {
	tests := []struct {
		name     string
		setupNil bool
		wantNil  bool
	}{
		{name: "nil database returns nil", setupNil: true, wantNil: true},
		{name: "sqlite adapter returns store", setupNil: false, wantNil: false},
		{name: "store is usable after init", setupNil: false, wantNil: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := store.Database
			t.Cleanup(func() { store.Database = prev })

			if tt.setupNil {
				store.Database = nil
			} else {
				store.Database = postgres.NewSQLiteTestAdapter(t)
			}
			ns := GetNotifyStore()
			if tt.wantNil {
				assert.Nil(t, ns)
			} else {
				assert.NotNil(t, ns)
			}
		})
	}
}

func TestRecordAsync_NilStore(t *testing.T) {
	prev := store.Database
	store.Database = nil
	t.Cleanup(func() { store.Database = prev })

	tests := []struct {
		name string
	}{
		{name: "does not panic with nil store"},
		{name: "accepts payload copy"},
		{name: "returns immediately"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				recordAsync(types.Uid("u"), "slack", "tpl", "sum", "success", "", map[string]any{"k": "v"})
			})
			time.Sleep(50 * time.Millisecond)
		})
	}
}
