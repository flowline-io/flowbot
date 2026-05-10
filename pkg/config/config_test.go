package config

import (
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeStruct(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Type
		wantList string
		wantApi  string
		wantDev  bool
	}{
		{
			name:     "populated fields",
			cfg:      Type{Listen: ":8080", ApiPath: "/api/", DevMode: true},
			wantList: ":8080",
			wantApi:  "/api/",
			wantDev:  true,
		},
		{
			name:     "default api path",
			cfg:      Type{},
			wantList: "",
			wantApi:  "",
			wantDev:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantList, tt.cfg.Listen)
			assert.Equal(t, tt.wantApi, tt.cfg.ApiPath)
			assert.Equal(t, tt.wantDev, tt.cfg.DevMode)
		})
	}

	t.Run("bots and vendors are not nil", func(t *testing.T) {
		cfg := Type{
			Bots:    map[string]any{"test": "value"},
			Vendors: map[string]any{"vendor1": "val"},
		}
		assert.NotNil(t, cfg.Bots)
		assert.NotNil(t, cfg.Vendors)
	})
}

func TestStoreType(t *testing.T) {
	tests := []struct {
		name      string
		store     StoreType
		wantMax   int
		wantAdapt string
		wantMap   bool
	}{
		{
			name: "mysql adapter",
			store: StoreType{
				MaxResults: 100,
				UseAdapter: "mysql",
				Adapters:   map[string]any{"mysql": map[string]string{"host": "localhost", "port": "3306"}},
			},
			wantMax:   100,
			wantAdapt: "mysql",
			wantMap:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMax, tt.store.MaxResults)
			assert.Equal(t, tt.wantAdapt, tt.store.UseAdapter)
			if tt.wantMap {
				assert.NotNil(t, tt.store.Adapters)
			}
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{name: "debug", level: "debug"},
		{name: "info", level: "info"},
		{name: "warn", level: "warn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := Log{Level: tt.level}
			assert.Equal(t, tt.level, log.Level)
		})
	}
}

func TestRedis(t *testing.T) {
	tests := []struct {
		name     string
		redis    Redis
		wantHost string
		wantPort int
		wantDB   int
		wantPass string
	}{
		{
			name:     "full config",
			redis:    Redis{Host: "localhost", Port: 6379, DB: 0, Password: "secret"},
			wantHost: "localhost",
			wantPort: 6379,
			wantDB:   0,
			wantPass: "secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantHost, tt.redis.Host)
			assert.Equal(t, tt.wantPort, tt.redis.Port)
			assert.Equal(t, tt.wantDB, tt.redis.DB)
			assert.Equal(t, tt.wantPass, tt.redis.Password)
		})
	}
}

func TestPlatformConfigs(t *testing.T) {
	t.Run("Slack", func(t *testing.T) {
		slack := Slack{
			Enabled:       true,
			AppID:         "A123",
			ClientID:      "C123",
			ClientSecret:  "secret",
			SigningSecret: "signing",
			BotToken:      "xoxb-test",
		}
		assert.True(t, slack.Enabled)
		assert.Equal(t, "A123", slack.AppID)
		assert.Equal(t, "xoxb-test", slack.BotToken)
	})

	t.Run("Discord", func(t *testing.T) {
		discord := Discord{Enabled: true, AppID: "D123", ClientID: "DC123", ClientSecret: "secret", BotToken: "Bot token"}
		assert.True(t, discord.Enabled)
		assert.Equal(t, "D123", discord.AppID)
	})

	t.Run("Telegram", func(t *testing.T) {
		tests := []struct {
			name   string
			tg     Telegram
			wantOn bool
		}{
			{name: "enabled", tg: Telegram{Enabled: true}, wantOn: true},
			{name: "disabled", tg: Telegram{Enabled: false}, wantOn: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.wantOn, tt.tg.Enabled)
			})
		}
	})

	t.Run("Tailchat", func(t *testing.T) {
		tailchat := Tailchat{Enabled: true, ApiURL: "https://api.tailchat.com", AppID: "T123", AppSecret: "secret"}
		assert.True(t, tailchat.Enabled)
		assert.Equal(t, "https://api.tailchat.com", tailchat.ApiURL)
	})

	t.Run("Slack all fields", func(t *testing.T) {
		slack := Slack{Enabled: true, AppID: "app1", ClientID: "cid1", ClientSecret: "cs1", SigningSecret: "ss1", VerificationToken: "vt1", AppToken: "at1", BotToken: "bt1"}
		assert.Equal(t, "app1", slack.AppID)
		assert.Equal(t, "cid1", slack.ClientID)
		assert.Equal(t, "cs1", slack.ClientSecret)
		assert.Equal(t, "ss1", slack.SigningSecret)
		assert.Equal(t, "vt1", slack.VerificationToken)
		assert.Equal(t, "at1", slack.AppToken)
		assert.Equal(t, "bt1", slack.BotToken)
	})

	t.Run("Discord all fields", func(t *testing.T) {
		discord := Discord{Enabled: true, AppID: "app1", PublicKey: "pk1", ClientID: "cid1", ClientSecret: "cs1", BotToken: "bt1"}
		assert.Equal(t, "app1", discord.AppID)
		assert.Equal(t, "pk1", discord.PublicKey)
		assert.Equal(t, "cid1", discord.ClientID)
		assert.Equal(t, "cs1", discord.ClientSecret)
		assert.Equal(t, "bt1", discord.BotToken)
	})
}

func TestExecutor(t *testing.T) {
	t.Run("type and limits", func(t *testing.T) {
		executor := Executor{Type: "docker"}
		assert.Equal(t, "docker", executor.Type)

		executor.Limits.Cpus = "2.0"
		executor.Limits.Memory = "4g"
		assert.Equal(t, "2.0", executor.Limits.Cpus)
		assert.Equal(t, "4g", executor.Limits.Memory)
	})

	t.Run("sub-configs", func(t *testing.T) {
		executor := Executor{}
		executor.Docker.Config = "/etc/docker/daemon.json"
		assert.Equal(t, "/etc/docker/daemon.json", executor.Docker.Config)

		executor.Shell.CMD = []string{"/bin/bash", "-c"}
		executor.Shell.UID = "1000"
		executor.Shell.GID = "1000"
		assert.Equal(t, []string{"/bin/bash", "-c"}, executor.Shell.CMD)
		assert.Equal(t, "1000", executor.Shell.UID)
		assert.Equal(t, "1000", executor.Shell.GID)

		executor.Machine.Host = "192.168.1.1"
		executor.Machine.Port = 22
		executor.Machine.Username = "user"
		executor.Machine.Password = "pass"
		executor.Machine.HostKey = "abc123"
		assert.Equal(t, "192.168.1.1", executor.Machine.Host)
		assert.Equal(t, 22, executor.Machine.Port)
		assert.Equal(t, "user", executor.Machine.Username)
		assert.Equal(t, "pass", executor.Machine.Password)
		assert.Equal(t, "abc123", executor.Machine.HostKey)

		executor.Mounts.Bind.Allowed = true
		assert.True(t, executor.Mounts.Bind.Allowed)
	})
}

func TestMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metrics Metrics
		wantOn  bool
		wantEp  string
	}{
		{
			name:    "enabled with endpoint",
			metrics: Metrics{Enabled: true, Endpoint: "/metrics"},
			wantOn:  true,
			wantEp:  "/metrics",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantOn, tt.metrics.Enabled)
			assert.Equal(t, tt.wantEp, tt.metrics.Endpoint)
		})
	}
}

func TestSearch(t *testing.T) {
	search := Search{
		Enabled:    true,
		Endpoint:   "https://search.example.com",
		MasterKey:  "key123",
		DataIndex:  "flowbot",
		UrlBaseMap: map[string]string{"github": "https://github.com"},
	}

	t.Run("search fields", func(t *testing.T) {
		assert.True(t, search.Enabled)
		assert.Equal(t, "https://search.example.com", search.Endpoint)
		assert.Equal(t, "key123", search.MasterKey)
		assert.Equal(t, "flowbot", search.DataIndex)
		assert.Len(t, search.UrlBaseMap, 1)
	})
}

func TestFlowbot(t *testing.T) {
	fb := Flowbot{URL: "https://flowbot.example.com", ChannelPath: "/channels", Language: "en"}

	t.Run("flowbot fields", func(t *testing.T) {
		assert.Equal(t, "https://flowbot.example.com", fb.URL)
		assert.Equal(t, "/channels", fb.ChannelPath)
		assert.Equal(t, "en", fb.Language)
	})
}

func TestAlarm(t *testing.T) {
	alarm := Alarm{Enabled: true, Filter: "error|fatal", SlackWebhook: "https://hooks.slack.com/test"}

	t.Run("alarm fields", func(t *testing.T) {
		assert.True(t, alarm.Enabled)
		assert.Equal(t, "error|fatal", alarm.Filter)
		assert.Equal(t, "https://hooks.slack.com/test", alarm.SlackWebhook)
	})
}

func TestModel(t *testing.T) {
	model := Model{Provider: "openai", BaseUrl: "https://api.openai.com", ApiKey: "sk-test", ModelNames: []string{"gpt-4", "gpt-3.5-turbo"}}

	t.Run("model fields", func(t *testing.T) {
		assert.Equal(t, "openai", model.Provider)
		assert.Equal(t, "https://api.openai.com", model.BaseUrl)
		assert.Equal(t, "sk-test", model.ApiKey)
		assert.Len(t, model.ModelNames, 2)
	})
}

func TestAgent(t *testing.T) {
	agent := Agent{Name: "assistant", Enabled: true, Model: "gpt-4"}

	t.Run("agent fields", func(t *testing.T) {
		assert.Equal(t, "assistant", agent.Name)
		assert.True(t, agent.Enabled)
		assert.Equal(t, "gpt-4", agent.Model)
	})
}

func TestMediaConfig(t *testing.T) {
	media := mediaConfig{
		UseHandler:        "s3",
		MaxFileUploadSize: 104857600,
		GcPeriod:          3600,
		GcBlockSize:       100,
		Handlers:          map[string]any{"s3": map[string]string{"bucket": "mybucket", "region": "us-east-1"}},
	}

	t.Run("media config fields", func(t *testing.T) {
		assert.Equal(t, "s3", media.UseHandler)
		assert.Equal(t, int64(104857600), media.MaxFileUploadSize)
		assert.Equal(t, 3600, media.GcPeriod)
		assert.Equal(t, 100, media.GcBlockSize)
		assert.NotNil(t, media.Handlers)
	})
}

func TestTypeJSONMarshaling(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		cfg := Type{
			Listen: ":9090", ApiPath: "/v1/", DevMode: false,
			Store: StoreType{MaxResults: 50, UseAdapter: "mysql"},
		}

		data, err := sonic.Marshal(cfg)
		require.NoError(t, err)
		assert.Contains(t, string(data), ":9090")
		assert.Contains(t, string(data), "/v1/")

		var unmarshaled Type
		err = sonic.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, ":9090", unmarshaled.Listen)
		assert.Equal(t, "/v1/", unmarshaled.ApiPath)
		assert.Equal(t, 50, unmarshaled.Store.MaxResults)
	})
}

func TestModelAndAgentSlices(t *testing.T) {
	t.Run("models slice", func(t *testing.T) {
		cfg := Type{
			Models: []Model{
				{Provider: "openai", BaseUrl: "https://api.openai.com", ApiKey: "sk1", ModelNames: []string{"gpt-4"}},
				{Provider: "anthropic", BaseUrl: "https://api.anthropic.com", ApiKey: "sk2", ModelNames: []string{"claude-3"}},
			},
		}
		assert.Len(t, cfg.Models, 2)
		assert.Equal(t, "openai", cfg.Models[0].Provider)
		assert.Equal(t, "anthropic", cfg.Models[1].Provider)
	})

	t.Run("agents slice", func(t *testing.T) {
		cfg := Type{
			Agents: []Agent{
				{Name: "chat", Enabled: true, Model: "gpt-4"},
				{Name: "react", Enabled: false, Model: "gpt-3.5-turbo"},
			},
		}
		assert.Len(t, cfg.Agents, 2)
		assert.Equal(t, "chat", cfg.Agents[0].Name)
		assert.True(t, cfg.Agents[0].Enabled)
		assert.False(t, cfg.Agents[1].Enabled)
	})
}
