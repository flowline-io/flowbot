package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseExpiresDefault(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "zero", input: "0"},
		{name: "zero d", input: "0d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expires, err := parseExpires(tt.input)
			require.NoError(t, err)
			// Should be ~100 years from now
			farFuture := time.Now().Add(98 * 365 * 24 * time.Hour)
			require.True(t, expires.After(farFuture), "expected far future, got %v", expires)
		})
	}
}

func TestParseExpiresDays(t *testing.T) {
	expires, err := parseExpires("365d")
	require.NoError(t, err)

	expectedMin := time.Now().Add(364 * 24 * time.Hour)
	expectedMax := time.Now().Add(366 * 24 * time.Hour)
	require.True(t, expires.After(expectedMin))
	require.True(t, expires.Before(expectedMax))
}

func TestParseExpiresHours(t *testing.T) {
	expires, err := parseExpires("24h")
	require.NoError(t, err)

	expectedMin := time.Now().Add(23 * time.Hour)
	expectedMax := time.Now().Add(25 * time.Hour)
	require.True(t, expires.After(expectedMin))
	require.True(t, expires.Before(expectedMax))
}

func TestParseExpiresMinutes(t *testing.T) {
	expires, err := parseExpires("30m")
	require.NoError(t, err)

	expectedMin := time.Now().Add(29 * time.Minute)
	expectedMax := time.Now().Add(31 * time.Minute)
	require.True(t, expires.After(expectedMin))
	require.True(t, expires.Before(expectedMax))
}

func TestParseExpiresErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "no unit", input: "365"},
		{name: "unknown unit", input: "10s"},
		{name: "negative", input: "-5d"},
		{name: "zero with h", input: "0h"},
		{name: "zero with m", input: "0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseExpires(tt.input)
			require.Error(t, err)
		})
	}
}

func TestParseExpiresCaseInsensitive(t *testing.T) {
	tests := []string{"1D", "1H", "1M"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := parseExpires(input)
			require.NoError(t, err)
		})
	}
}
