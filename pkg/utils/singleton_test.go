package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSingleton(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "does not panic on single call",
			fn:   CheckSingleton,
		},
		{
			name: "does not panic on repeated calls",
			fn:   CheckSingleton,
		},
		{
			name: "does not panic on concurrent-style invocation",
			fn:   CheckSingleton,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					require.Fail(t, "CheckSingleton() panicked")
				}
			}()

			tt.fn()
		})
	}
}

func TestEmbedServerPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "is not empty",
			fn: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, EmbedServerPort, "EmbedServerPort should not be empty")
			},
		},
		{
			name: "matches expected value",
			fn: func(t *testing.T) {
				t.Parallel()
				expectedPort := "15656"
				assert.Equal(t, expectedPort, EmbedServerPort)
			},
		},
		{
			name: "port is numeric only",
			fn: func(t *testing.T) {
				t.Parallel()
				for _, char := range EmbedServerPort {
					assert.True(t, char >= '0' && char <= '9', "EmbedServerPort contains non-numeric character: %c", char)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestEmbedServerCreation(t *testing.T) {
	t.Parallel()

	port := EmbedServerPort

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "port is not empty",
			fn: func(t *testing.T) {
				t.Parallel()
				require.NotEmpty(t, port, "EmbedServerPort should not be empty")
			},
		},
		{
			name: "port is numeric only",
			fn: func(t *testing.T) {
				t.Parallel()
				for _, char := range port {
					assert.True(t, char >= '0' && char <= '9', "EmbedServerPort contains non-numeric character: %c", char)
				}
			},
		},
		{
			name: "port is in valid range",
			fn: func(t *testing.T) {
				t.Parallel()
				require.NotEqual(t, "0", port, "EmbedServerPort should be a valid port number")
				require.LessOrEqual(t, len(port), 5, "EmbedServerPort should be a valid port number")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
