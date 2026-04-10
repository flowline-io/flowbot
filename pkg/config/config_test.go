package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeStruct_Fields(t *testing.T) {
	cfg := Type{
		Listen:  ":8080",
		ApiPath: "/api/",
		DevMode: true,
	}

	assert.Equal(t, ":8080", cfg.Listen)
	assert.Equal(t, "/api/", cfg.ApiPath)
	assert.True(t, cfg.DevMode)
}

func TestTypeStruct_DefaultApiPath(t *testing.T) {
	cfg := Type{}
	assert.Empty(t, cfg.ApiPath)
}

func TestTypeStruct_BotsAndVendors(t *testing.T) {
	cfg := Type{
		Bots:    map[string]any{"test": "value"},
		Vendors: map[string]any{"vendor1": "val"},
	}
	assert.NotNil(t, cfg.Bots)
	assert.NotNil(t, cfg.Vendors)
}

func TestStoreType_Adapters(t *testing.T) {
	store := StoreType{
		MaxResults: 100,
		UseAdapter: "mysql",
		Adapters: map[string]any{
			"mysql": map[string]string{
				"host": "localhost",
				"port": "3306",
			},
		},
	}

	assert.Equal(t, 100, store.MaxResults)
	assert.Equal(t, "mysql", store.UseAdapter)
	assert.NotNil(t, store.Adapters)
}

func TestLog(t *testing.T) {
	log := Log{Level: "debug"}
	assert.Equal(t, "debug", log.Level)

	log = Log{Level: "info"}
	assert.Equal(t, "info", log.Level)
}

func TestRedis(t *testing.T) {
	redis := Redis{
		Host:     "localhost",
		Port:     6379,
		DB:       0,
		Password: "secret",
	}

	assert.Equal(t, "localhost", redis.Host)
	assert.Equal(t, 6379, redis.Port)
	assert.Equal(t, 0, redis.DB)
	assert.Equal(t, "secret", redis.Password)
}

func TestPlatformConfigs(t *testing.T) {
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

	discord := Discord{
		Enabled:      true,
		AppID:        "D123",
		ClientID:     "DC123",
		ClientSecret: "secret",
		BotToken:     "Bot token",
	}
	assert.True(t, discord.Enabled)
	assert.Equal(t, "D123", discord.AppID)

	telegram := Telegram{Enabled: true}
	assert.True(t, telegram.Enabled)

	tailchat := Tailchat{
		Enabled:   true,
		ApiURL:    "https://api.tailchat.com",
		AppID:     "T123",
		AppSecret: "secret",
	}
	assert.True(t, tailchat.Enabled)
	assert.Equal(t, "https://api.tailchat.com", tailchat.ApiURL)
}

func TestExecutor(t *testing.T) {
	executor := Executor{Type: "docker"}
	assert.Equal(t, "docker", executor.Type)

	executor.Limits.Cpus = "2.0"
	executor.Limits.Memory = "4g"
	assert.Equal(t, "2.0", executor.Limits.Cpus)
	assert.Equal(t, "4g", executor.Limits.Memory)

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
}

func TestMetrics(t *testing.T) {
	metrics := Metrics{
		Enabled:  true,
		Endpoint: "/metrics",
	}
	assert.True(t, metrics.Enabled)
	assert.Equal(t, "/metrics", metrics.Endpoint)
}

func TestSearch(t *testing.T) {
	search := Search{
		Enabled:   true,
		Endpoint:  "https://search.example.com",
		MasterKey: "key123",
		DataIndex: "flowbot",
		UrlBaseMap: map[string]string{
			"github": "https://github.com",
		},
	}
	assert.True(t, search.Enabled)
	assert.Equal(t, "https://search.example.com", search.Endpoint)
	assert.Equal(t, "key123", search.MasterKey)
	assert.Equal(t, "flowbot", search.DataIndex)
	assert.Len(t, search.UrlBaseMap, 1)
}

func TestFlowbot(t *testing.T) {
	flowbot := Flowbot{
		URL:         "https://flowbot.example.com",
		ChannelPath: "/channels",
		Language:    "en",
		MCPToken:    "token123",
	}
	assert.Equal(t, "https://flowbot.example.com", flowbot.URL)
	assert.Equal(t, "/channels", flowbot.ChannelPath)
	assert.Equal(t, "en", flowbot.Language)
	assert.Equal(t, "token123", flowbot.MCPToken)
}

func TestAlarm(t *testing.T) {
	alarm := Alarm{
		Enabled:      true,
		Filter:       "error|fatal",
		SlackWebhook: "https://hooks.slack.com/test",
	}
	assert.True(t, alarm.Enabled)
	assert.Equal(t, "error|fatal", alarm.Filter)
	assert.Equal(t, "https://hooks.slack.com/test", alarm.SlackWebhook)
}

func TestModel(t *testing.T) {
	model := Model{
		Provider:   "openai",
		BaseUrl:    "https://api.openai.com",
		ApiKey:     "sk-test",
		ModelNames: []string{"gpt-4", "gpt-3.5-turbo"},
	}
	assert.Equal(t, "openai", model.Provider)
	assert.Equal(t, "https://api.openai.com", model.BaseUrl)
	assert.Equal(t, "sk-test", model.ApiKey)
	assert.Len(t, model.ModelNames, 2)
}

func TestAgent(t *testing.T) {
	agent := Agent{
		Name:    "assistant",
		Enabled: true,
		Model:   "gpt-4",
	}
	assert.Equal(t, "assistant", agent.Name)
	assert.True(t, agent.Enabled)
	assert.Equal(t, "gpt-4", agent.Model)
}

func TestMediaConfig(t *testing.T) {
	media := mediaConfig{
		UseHandler:        "s3",
		MaxFileUploadSize: 104857600,
		GcPeriod:          3600,
		GcBlockSize:       100,
		Handlers: map[string]any{
			"s3": map[string]string{
				"bucket": "mybucket",
				"region": "us-east-1",
			},
		},
	}
	assert.Equal(t, "s3", media.UseHandler)
	assert.Equal(t, int64(104857600), media.MaxFileUploadSize)
	assert.Equal(t, 3600, media.GcPeriod)
	assert.Equal(t, 100, media.GcBlockSize)
	assert.NotNil(t, media.Handlers)
}

func TestTypeJSONMarshaling(t *testing.T) {
	cfg := Type{
		Listen:  ":9090",
		ApiPath: "/v1/",
		DevMode: false,
		Store: StoreType{
			MaxResults: 50,
			UseAdapter: "mysql",
		},
	}

	data, err := json.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, string(data), ":9090")
	assert.Contains(t, string(data), "/v1/")

	var unmarshaled Type
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, ":9090", unmarshaled.Listen)
	assert.Equal(t, "/v1/", unmarshaled.ApiPath)
	assert.Equal(t, 50, unmarshaled.Store.MaxResults)
}

func TestTelegramConfig(t *testing.T) {
	telegram := Telegram{Enabled: false}
	assert.False(t, telegram.Enabled)
}

func TestDiscordConfig_AllFields(t *testing.T) {
	discord := Discord{
		Enabled:      true,
		AppID:        "app1",
		PublicKey:    "pk1",
		ClientID:     "cid1",
		ClientSecret: "cs1",
		BotToken:     "bt1",
	}
	assert.Equal(t, "app1", discord.AppID)
	assert.Equal(t, "pk1", discord.PublicKey)
	assert.Equal(t, "cid1", discord.ClientID)
	assert.Equal(t, "cs1", discord.ClientSecret)
	assert.Equal(t, "bt1", discord.BotToken)
}

func TestSlackConfig_AllFields(t *testing.T) {
	slack := Slack{
		Enabled:           true,
		AppID:             "app1",
		ClientID:          "cid1",
		ClientSecret:      "cs1",
		SigningSecret:     "ss1",
		VerificationToken: "vt1",
		AppToken:          "at1",
		BotToken:          "bt1",
	}
	assert.Equal(t, "app1", slack.AppID)
	assert.Equal(t, "cid1", slack.ClientID)
	assert.Equal(t, "cs1", slack.ClientSecret)
	assert.Equal(t, "ss1", slack.SigningSecret)
	assert.Equal(t, "vt1", slack.VerificationToken)
	assert.Equal(t, "at1", slack.AppToken)
	assert.Equal(t, "bt1", slack.BotToken)
}

func TestModelsSlice(t *testing.T) {
	cfg := Type{
		Models: []Model{
			{Provider: "openai", BaseUrl: "https://api.openai.com", ApiKey: "sk1", ModelNames: []string{"gpt-4"}},
			{Provider: "anthropic", BaseUrl: "https://api.anthropic.com", ApiKey: "sk2", ModelNames: []string{"claude-3"}},
		},
	}
	assert.Len(t, cfg.Models, 2)
	assert.Equal(t, "openai", cfg.Models[0].Provider)
	assert.Equal(t, "anthropic", cfg.Models[1].Provider)
}

func TestAgentsSlice(t *testing.T) {
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
}
