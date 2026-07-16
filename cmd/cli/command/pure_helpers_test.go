package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToEnvKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "hyphens become underscores", key: "redis-password", want: "REDIS_PASSWORD"},
		{name: "already uppercase", key: "HOST", want: "HOST"},
		{name: "mixed case with hyphens", key: "http-cors-allow-origins", want: "HTTP_CORS_ALLOW_ORIGINS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, toEnvKey(tt.key))
		})
	}
}

func TestFormatBool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   bool
		want string
	}{
		{name: "true returns on", in: true, want: "on"},
		{name: "false returns off", in: false, want: "off"},
		{name: "zero value is off", in: false, want: "off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatBool(tt.in))
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ts   int
		want string
	}{
		{name: "zero returns N/A", ts: 0, want: "N/A"},
		{name: "unix epoch", ts: 0, want: "N/A"},
		{name: "known timestamp", ts: 1704067200, want: time.Unix(1704067200, 0).Format(time.RFC3339)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatTimestamp(tt.ts))
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{name: "shorter than max unchanged", input: "hello", maxLen: 10, want: "hello"},
		{name: "exact length unchanged", input: "hello", maxLen: 5, want: "hello"},
		{name: "longer than max truncated", input: "hello world", maxLen: 5, want: "hello..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, truncate(tt.input, tt.maxLen))
		})
	}
}
