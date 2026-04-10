package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeStruct(t *testing.T) {
	// Test that Type struct can be instantiated
	cfg := Type{
		Listen:  ":8080",
		ApiPath: "/api/",
	}

	assert.Equal(t, ":8080", cfg.Listen)
	assert.Equal(t, "/api/", cfg.ApiPath)
}

func TestStoreType(t *testing.T) {
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
	// Test Slack config
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

	// Test Discord config
	discord := Discord{
		Enabled:      true,
		AppID:        "D123",
		ClientID:     "DC123",
		ClientSecret: "secret",
		BotToken:     "Bot token",
	}
	assert.True(t, discord.Enabled)
	assert.Equal(t, "D123", discord.AppID)

	// Test Tailchat config
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
	executor := Executor{
		Type: "docker",
	}
	assert.Equal(t, "docker", executor.Type)

	// Test limits
	executor.Limits.Cpus = "2.0"
	executor.Limits.Memory = "4g"
	assert.Equal(t, "2.0", executor.Limits.Cpus)
	assert.Equal(t, "4g", executor.Limits.Memory)

	// Test Docker config
	executor.Docker.Config = "/etc/docker/daemon.json"
	assert.Equal(t, "/etc/docker/daemon.json", executor.Docker.Config)
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
		MaxFileUploadSize: 104857600, // 100MB
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
