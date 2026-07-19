package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandConfigFileEnv(t *testing.T) {
	tests := []struct {
		name    string
		content string
		env     map[string]string
		wantURL string
	}{
		{
			name:    "expands braced secret in redis url",
			content: "listen: \":6060\"\nredis:\n  url: redis://:${REDIS_PASSWORD}@127.0.0.1:6379/0\n",
			env:     map[string]string{"REDIS_PASSWORD": "s3cret"},
			wantURL: "redis://:s3cret@127.0.0.1:6379/0",
		},
		{
			name:    "leaves empty when env unset",
			content: "listen: \":6060\"\nredis:\n  url: redis://:${MISSING_REDIS_PASS}@127.0.0.1:6379/0\n",
			env:     map[string]string{},
			wantURL: "redis://:@127.0.0.1:6379/0",
		},
		{
			name:    "empty path is no-op",
			content: "",
			env:     nil,
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			viper.Reset()
			viper.SetConfigType("yaml")

			if tt.content == "" {
				require.NoError(t, expandConfigFileEnv(""))
				return
			}

			dir := t.TempDir()
			path := filepath.Join(dir, "flowbot.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))

			require.NoError(t, viper.ReadConfig(strings.NewReader(tt.content)))
			require.NoError(t, expandConfigFileEnv(path))

			var cfg Type
			require.NoError(t, viper.Unmarshal(&cfg))
			assert.Equal(t, tt.wantURL, cfg.Redis.URL)
		})
	}
}
