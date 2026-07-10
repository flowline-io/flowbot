package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() Type {
	return Type{
		Redis: Redis{
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "secret",
		},
		Store: StoreType{
			UseAdapter: "postgres",
			Adapters: map[string]any{
				"postgres": map[string]any{
					"dsn": "postgres://user:pass@localhost/flowbot?sslmode=disable",
				},
			},
		},
	}
}

func TestValidate_Required(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
	}{
		{
			name:    "missing redis host",
			mutate:  func(c *Type) { c.Redis.Host = "" },
			wantErr: "redis.Host",
		},
		{
			name:    "missing redis password",
			mutate:  func(c *Type) { c.Redis.Password = "" },
			wantErr: "redis.Password",
		},
		{
			name:    "redis port zero",
			mutate:  func(c *Type) { c.Redis.Port = 0 },
			wantErr: "redis.Port",
		},
		{
			name:    "redis port too high",
			mutate:  func(c *Type) { c.Redis.Port = 99999 },
			wantErr: "redis.Port",
		},
		{
			name:    "missing store adapter name",
			mutate:  func(c *Type) { c.Store.UseAdapter = "" },
			wantErr: "store.use_adapter",
		},
		{
			name: "use_adapter not found in adapters map",
			mutate: func(c *Type) {
				c.Store.UseAdapter = "mysql"
			},
			wantErr: "not found",
		},
		{
			name: "missing DSN",
			mutate: func(c *Type) {
				c.Store.Adapters = map[string]any{
					"postgres": map[string]any{},
				}
			},
			wantErr: "dsn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			require.Error(t, err)
			if tt.wantErr != "" {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			assert.Contains(t, err.Error(), "Fix:")
		})
	}
}

func TestValidate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
		noErr   bool
	}{
		{
			name:   "valid log level debug",
			mutate: func(c *Type) { c.Log.Level = "debug" },
			noErr:  true,
		},
		{
			name:   "valid log level info",
			mutate: func(c *Type) { c.Log.Level = "info" },
			noErr:  true,
		},
		{
			name:   "valid log level warn",
			mutate: func(c *Type) { c.Log.Level = "warn" },
			noErr:  true,
		},
		{
			name:    "invalid log level",
			mutate:  func(c *Type) { c.Log.Level = "verbose" },
			wantErr: "log.Level",
		},
		{
			name: "tracing enabled, missing endpoint",
			mutate: func(c *Type) {
				c.Tracing.Enabled = true
				c.Tracing.Endpoint = ""
			},
			wantErr: "tracing.Endpoint",
		},
		{
			name: "tracing enabled, invalid URL",
			mutate: func(c *Type) {
				c.Tracing.Enabled = true
				c.Tracing.Endpoint = "not-a-url"
			},
			wantErr: "tracing.Endpoint",
		},
		{
			name: "tracing disabled, missing endpoint OK",
			mutate: func(c *Type) {
				c.Tracing.Enabled = false
				c.Tracing.Endpoint = ""
			},
			noErr: true,
		},
		{
			name: "sample rate out of range high",
			mutate: func(c *Type) {
				c.Tracing.SampleRate = 2.0
			},
			wantErr: "tracing.SampleRate",
		},
		{
			name:    "invalid listen address",
			mutate:  func(c *Type) { c.Listen = "::99999" },
			wantErr: "listen",
		},
		{
			name: "invalid probe timeout duration",
			mutate: func(c *Type) {
				c.Homelab.Discovery.ProbeTimeout = "xyz"
			},
			wantErr: "probe_timeout",
		},
		{
			name: "invalid expiry duration",
			mutate: func(c *Type) {
				c.Ability.EventPool.ExpiryDuration = "bad"
			},
			wantErr: "expiry_duration",
		},
		{
			name:   "valid flowbot url",
			mutate: func(c *Type) { c.Flowbot.URL = "http://example.com" },
			noErr:  true,
		},
		{
			name:    "invalid flowbot url",
			mutate:  func(c *Type) { c.Flowbot.URL = "not-a-url" },
			wantErr: "flowbot.URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.noErr {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				if tt.wantErr != "" {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_Conditional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
		noErr   bool
	}{
		{
			name: "slack enabled, empty creds OK",
			mutate: func(c *Type) {
				c.Platform.Slack.Enabled = true
				c.Platform.Slack.AppID = ""
				c.Platform.Slack.ClientID = ""
				c.Platform.Slack.ClientSecret = ""
				c.Platform.Slack.SigningSecret = ""
			},
			noErr: true,
		},
		{
			name: "discord enabled, missing bot_token",
			mutate: func(c *Type) {
				c.Platform.Discord.Enabled = true
				c.Platform.Discord.BotToken = ""
			},
			wantErr: "platform.discord.BotToken",
		},
		{
			name: "discord disabled, missing bot_token OK",
			mutate: func(c *Type) {
				c.Platform.Discord.Enabled = false
				c.Platform.Discord.BotToken = ""
			},
			noErr: true,
		},
		{
			name: "tailchat enabled, invalid api_url",
			mutate: func(c *Type) {
				c.Platform.Tailchat.Enabled = true
				c.Platform.Tailchat.ApiURL = "not-a-url"
			},
			wantErr: "platform.tailchat.ApiURL",
		},
		{
			name: "chat agent references unknown model",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "openai", ModelNames: []string{"gpt4"}, BaseUrl: "https://api.openai.com"},
				}
				c.ChatAgent = ChatAgentConfig{ChatModel: "nonexistent"}
			},
			wantErr: "chat_agent.chat_model",
		},
		{
			name: "ability tool requires readonly",
			mutate: func(c *Type) {
				c.ChatAgent = ChatAgentConfig{
					AbilityTools: []AbilityToolConfig{{
						Name: "memo_list", Capability: "memo", Operation: "list", Readonly: false,
					}},
				}
			},
			wantErr: "readonly must be true",
		},
		{
			name: "ability tool requires capability",
			mutate: func(c *Type) {
				c.ChatAgent = ChatAgentConfig{
					AbilityTools: []AbilityToolConfig{{
						Name: "memo_list", Operation: "list", Readonly: true,
					}},
				}
			},
			wantErr: "capability is required",
		},
		{
			name: "ability tool readonly ok",
			mutate: func(c *Type) {
				c.ChatAgent = ChatAgentConfig{
					AbilityTools: []AbilityToolConfig{{
						Name: "memo_list", Capability: "memo", Operation: "list", Readonly: true,
					}},
				}
			},
			noErr: true,
		},
		{
			name: "model missing provider",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "", BaseUrl: "https://api.openai.com"},
				}
			},
			wantErr: "models[0].provider",
		},
		{
			name: "model invalid provider alias",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "openai-compatible", ModelNames: []string{"deepseek-v4-flash"}},
				}
			},
			wantErr: "models[0].provider: unsupported value",
		},
		{
			name: "model supported openai compatible provider",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "openai_compatible", ModelNames: []string{"deepseek-v4-flash"}},
				}
			},
			noErr: true,
		},
		{
			name: "model invalid base_url",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "openai", BaseUrl: "not-a-url"},
				}
			},
			wantErr: "models[0].base_url",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.noErr {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				if tt.wantErr != "" {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestValidate_Accumulated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mutate       func(*Type)
		wantContains []string
		minNewlines  int
	}{
		{
			name:         "redis host empty only",
			mutate:       func(c *Type) { c.Redis.Host = "" },
			wantContains: []string{"redis.Host"},
		},
		{
			name: "dsn empty only",
			mutate: func(c *Type) {
				c.Store.Adapters = map[string]any{
					"postgres": map[string]any{},
				}
			},
			wantContains: []string{"dsn"},
		},
		{
			name: "both empty",
			mutate: func(c *Type) {
				c.Redis.Host = ""
				c.Store.UseAdapter = ""
				c.Store.Adapters = nil
			},
			wantContains: []string{"redis.Host", "store.use_adapter"},
			minNewlines:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			require.Error(t, err)
			errStr := err.Error()
			for _, want := range tt.wantContains {
				assert.Contains(t, errStr, want)
			}
			if tt.minNewlines > 0 {
				lines := 0
				for _, ch := range errStr {
					if ch == '\n' {
						lines++
					}
				}
				assert.GreaterOrEqual(t, lines, tt.minNewlines)
			}
		})
	}
}

func TestValidate_HappyPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Type)
	}{
		{
			name:   "default valid config",
			mutate: func(_ *Type) {},
		},
		{
			name: "with optional fields set",
			mutate: func(c *Type) {
				c.Flowbot.URL = "http://example.com"
				c.Log.Level = "debug"
			},
		},
		{
			name: "with all platforms disabled",
			mutate: func(c *Type) {
				c.Platform.Slack.Enabled = false
				c.Platform.Discord.Enabled = false
				c.Platform.Tailchat.Enabled = false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestReachabilityCheck_RedisUnreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reachability test in short mode")
	}
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
	}{
		{
			name: "unreachable host",
			mutate: func(c *Type) {
				c.Redis.Host = "255.255.255.255"
				c.Redis.Port = 9999
				c.Redis.Password = "nope"
			},
			wantErr: "redis",
		},
		{
			name: "wrong port",
			mutate: func(c *Type) {
				c.Redis.Host = "127.0.0.1"
				c.Redis.Port = 9999
				c.Redis.Password = "secret"
			},
			wantErr: "redis",
		},
		{
			name: "unreachable subnet",
			mutate: func(c *Type) {
				c.Redis.Host = "10.255.255.255"
				c.Redis.Port = 6379
				c.Redis.Password = "secret"
			},
			wantErr: "redis",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.ReachabilityCheck(t.Context())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestReachabilityCheck_PostgresUnreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reachability test in short mode")
	}
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
	}{
		{
			name: "unreachable host",
			mutate: func(c *Type) {
				c.Store.Adapters = map[string]any{
					"postgres": map[string]any{
						"dsn": "postgres://nonexistent:bad@255.255.255.255:9999/fake?sslmode=disable",
					},
				}
			},
			wantErr: "postgres",
		},
		{
			name: "invalid DSN",
			mutate: func(c *Type) {
				c.Store.Adapters = map[string]any{
					"postgres": map[string]any{
						"dsn": "not-a-valid-dsn",
					},
				}
			},
			wantErr: "postgres",
		},
		{
			name: "connection refused",
			mutate: func(c *Type) {
				c.Store.Adapters = map[string]any{
					"postgres": map[string]any{
						"dsn": "postgres://user:pass@127.0.0.1:5433/flowbot?sslmode=disable",
					},
				}
			},
			wantErr: "postgres",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.ReachabilityCheck(t.Context())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
