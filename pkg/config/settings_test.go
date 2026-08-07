package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSensitivePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		tag  bool
		want bool
	}{
		{name: "explicit tag", path: "redis.url", tag: true, want: true},
		{name: "dsn segment", path: "postgres.dsn", want: true},
		{name: "api_key", path: "models[0].api_key", want: true},
		{name: "sign_secret", path: "media.sign_secret", want: true},
		{name: "access_token", path: "chat_agent.sandbox.access_token", want: true},
		{name: "password", path: "search.password", want: true},
		{name: "encryption_key", path: "modules[0].auth.encryption_key", want: true},
		{name: "consumer_key", path: "vendors.twitter.consumer_key", want: true},
		{name: "bare key", path: "vendors.foo.key", want: true},
		{name: "ssh_key suffix", path: "homelab.ssh_key", want: true},
		{name: "token_header is not a secret value", path: "gateway.auth.token_header", want: false},
		{name: "encryption_key_dir is a path", path: "modules[0].auth.encryption_key_dir", want: false},
		{name: "plain listen", path: "listen", want: false},
		{name: "level", path: "log.level", want: false},
		{name: "dynamic vendor secret", path: "vendors.openai.api_key", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isSensitivePath(tt.path, tt.tag))
		})
	}
}

func TestSettingsCatalog_redactsAndDescribes(t *testing.T) {
	t.Parallel()
	prevDocs := FieldDocs
	FieldDocs = map[string]string{
		"listen":       "HTTP listen address",
		"postgres.dsn": "PostgreSQL connection string",
		"redis.url":    "Redis connection URI",
		"log.level":    "Log level",
	}
	t.Cleanup(func() { FieldDocs = prevDocs })

	cfg := Type{
		Listen: ":8080",
		Postgres: PostgresConfig{
			DSN: "postgres://user:secret@localhost/db",
		},
		Redis: Redis{
			URL: "redis://:hunter2@127.0.0.1:6379/0",
		},
		Log: Log{Level: "info"},
	}

	groups := SettingsCatalog(cfg)
	byPath := indexSettingsByPath(groups)

	listen, ok := byPath["listen"]
	require.True(t, ok)
	assert.Equal(t, ":8080", listen.Value)
	assert.Equal(t, "HTTP listen address", listen.Description)
	assert.False(t, listen.Sensitive)

	dsn, ok := byPath["postgres.dsn"]
	require.True(t, ok)
	assert.Equal(t, maskedSecret, dsn.Value)
	assert.True(t, dsn.Sensitive)
	assert.Equal(t, "PostgreSQL connection string", dsn.Description)

	redisURL, ok := byPath["redis.url"]
	require.True(t, ok)
	assert.Equal(t, maskedSecret, redisURL.Value)
	assert.True(t, redisURL.Sensitive)

	level, ok := byPath["log.level"]
	require.True(t, ok)
	assert.Equal(t, "info", level.Value)
}

func TestSettingsCatalog_zeroValuesAndNilPointer(t *testing.T) {
	t.Parallel()
	groups := SettingsCatalog(Type{})
	byPath := indexSettingsByPath(groups)

	listen, ok := byPath["listen"]
	require.True(t, ok)
	assert.Equal(t, emptyDisplay, listen.Value)

	dsn, ok := byPath["postgres.dsn"]
	require.True(t, ok)
	assert.Equal(t, notSetDisplay, dsn.Value)
	assert.True(t, dsn.Sensitive)

	// media is a nil pointer: child fields still appear as not set
	useHandler, ok := byPath["media.use_handler"]
	require.True(t, ok)
	assert.Equal(t, notSetDisplay, useHandler.Value)

	signSecret, ok := byPath["media.sign_secret"]
	require.True(t, ok)
	assert.Equal(t, notSetDisplay, signSecret.Value)
	assert.True(t, signSecret.Sensitive)
}

func TestSettingsCatalog_structSliceAndScalarSlice(t *testing.T) {
	t.Parallel()
	cfg := Type{
		Models: []Model{
			{Provider: "openai", ApiKey: "sk-test", ModelNames: []string{"gpt-4"}},
		},
		Profiling: Profiling{
			ProfileTypes: []string{"cpu", "alloc_objects"},
		},
	}
	groups := SettingsCatalog(cfg)
	byPath := indexSettingsByPath(groups)

	provider, ok := byPath["models[0].provider"]
	require.True(t, ok)
	assert.Equal(t, "openai", provider.Value)

	apiKey, ok := byPath["models[0].api_key"]
	require.True(t, ok)
	assert.Equal(t, maskedSecret, apiKey.Value)

	names, ok := byPath["models[0].model_names"]
	require.True(t, ok)
	assert.Equal(t, `["gpt-4"]`, names.Value)

	types, ok := byPath["profiling.profile_types"]
	require.True(t, ok)
	assert.Equal(t, `["cpu","alloc_objects"]`, types.Value)

	emptyCfg := Type{Models: nil}
	emptyGroups := SettingsCatalog(emptyCfg)
	emptyByPath := indexSettingsByPath(emptyGroups)
	models, ok := emptyByPath["models"]
	require.True(t, ok)
	assert.Equal(t, emptyDisplay, models.Value)
	_, hasIndex := emptyByPath["models[0].provider"]
	assert.False(t, hasIndex)
}

func TestSettingsCatalog_dynamicMapAndAny(t *testing.T) {
	t.Parallel()
	cfg := Type{
		Modules: map[string]any{
			"demo": map[string]any{"api_key": "secret", "enabled": true},
		},
		Vendors: nil,
	}
	groups := SettingsCatalog(cfg)
	byPath := indexSettingsByPath(groups)

	key, ok := byPath["modules.demo.api_key"]
	require.True(t, ok)
	assert.Equal(t, maskedSecret, key.Value)

	enabled, ok := byPath["modules.demo.enabled"]
	require.True(t, ok)
	assert.Equal(t, "true", enabled.Value)

	vendors, ok := byPath["vendors"]
	require.True(t, ok)
	assert.Equal(t, notSetDisplay, vendors.Value)
}

func TestSettingsCatalog_modulesSliceRedactsNestedPassword(t *testing.T) {
	t.Parallel()
	cfg := Type{
		Modules: []any{
			map[string]any{
				"name": "web",
				"auth": map[string]any{
					"username": "admin",
					"password": "flowbot-dev-pass",
				},
			},
		},
	}
	groups := SettingsCatalog(cfg)
	byPath := indexSettingsByPath(groups)

	_, collapsed := byPath["modules"]
	assert.False(t, collapsed, "modules slice must expand, not dump as one JSON leaf")

	pass, ok := byPath["modules[0].auth.password"]
	require.True(t, ok, "paths=%v", pathsOf(byPath))
	assert.Equal(t, maskedSecret, pass.Value)
	assert.True(t, pass.Sensitive)
	assert.NotContains(t, pass.Value, "flowbot-dev-pass")

	user, ok := byPath["modules[0].auth.username"]
	require.True(t, ok)
	assert.Equal(t, "admin", user.Value)

	for _, e := range byPath {
		assert.NotContains(t, e.Value, "flowbot-dev-pass")
	}
}

func TestSettingsCatalog_durationAndGroups(t *testing.T) {
	t.Parallel()
	cfg := Type{
		Redis: Redis{DialTimeout: 5 * time.Second},
	}
	groups := SettingsCatalog(cfg)
	require.NotEmpty(t, groups)

	var foundListen, foundRedis bool
	for _, g := range groups {
		if g.Name == "root" {
			foundListen = true
		}
		if g.Name == "redis" {
			foundRedis = true
			byPath := map[string]SettingEntry{}
			for _, e := range g.Entries {
				byPath[e.Path] = e
			}
			assert.Equal(t, "5s", byPath["redis.dial_timeout"].Value)
		}
	}
	assert.True(t, foundListen)
	assert.True(t, foundRedis)
}

func TestNormalizeDocPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"models[0].api_key", "models.api_key"},
		{"modules.demo.enabled", "modules.demo.enabled"},
		{"listen", "listen"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, normalizeDocPath(tt.in))
		})
	}
}

func indexSettingsByPath(groups []SettingGroup) map[string]SettingEntry {
	out := make(map[string]SettingEntry)
	for _, g := range groups {
		for _, e := range g.Entries {
			out[e.Path] = e
		}
	}
	return out
}

func TestSettingsCatalog_includesStore(t *testing.T) {
	t.Parallel()
	cfg := Type{}
	cfg.Normalize()
	groups := SettingsCatalog(cfg)
	byPath := indexSettingsByPath(groups)
	_, ok := byPath["store.use_adapter"]
	require.True(t, ok, "store fields must appear; paths=%v", pathsOf(byPath))
}

func pathsOf(byPath map[string]SettingEntry) string {
	parts := make([]string, 0, len(byPath))
	for p := range byPath {
		parts = append(parts, p)
	}
	return strings.Join(parts, ",")
}
