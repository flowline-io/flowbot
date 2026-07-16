package notify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
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
