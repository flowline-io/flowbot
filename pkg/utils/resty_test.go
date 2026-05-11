package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRestyClient(t *testing.T) {
	t.Parallel()

	client := DefaultRestyClient()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "returns non-nil client",
			fn: func(t *testing.T) {
				t.Parallel()
				require.NotNil(t, client, "DefaultRestyClient() returned nil")
			},
		},
		{
			name: "can create requests",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				assert.NotNil(t, req, "DefaultRestyClient() client.R() returned nil")
			},
		},
		{
			name: "request has headers initialized",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				assert.NotNil(t, req.Header, "DefaultRestyClient() request headers are nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestDefaultRestyClientTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		timeout     time.Duration
		expectError bool
	}{
		{
			name:        "nanosecond timeout triggers error immediately",
			timeout:     1 * time.Nanosecond,
			expectError: true,
		},
		{
			name:        "default timeout does not prevent request creation",
			timeout:     0,
			expectError: false,
		},
		{
			name:        "moderate timeout still allows request creation",
			timeout:     30 * time.Second,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := DefaultRestyClient()
			if tt.timeout > 0 {
				client.SetTimeout(tt.timeout)
			}

			if tt.expectError {
				start := time.Now()
				_, err := client.R().Get("http://example.com/")
				elapsed := time.Since(start)

				require.Error(t, err, "Expected timeout error but got nil")
				assert.LessOrEqual(t, elapsed, 5*time.Second, "Request took too long: %v, expected immediate timeout", elapsed)
			} else {
				assert.NotNil(t, client, "DefaultRestyClient should return valid client")
				req := client.R()
				assert.NotNil(t, req, "should be able to create requests with default timeout")
			}
		})
	}
}

func TestDefaultRestyClientHeaders(t *testing.T) {
	t.Parallel()

	client := DefaultRestyClient()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "can create request object",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				assert.NotNil(t, req, "DefaultRestyClient() should be able to create requests")
			},
		},
		{
			name: "can set and get custom header on request",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				req.SetHeader("Test-Header", "test-value")
				assert.Equal(t, "test-value", req.Header.Get("Test-Header"), "Should be able to set headers on requests")
			},
		},
		{
			name: "can override default headers",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				req.SetHeader("Content-Type", "application/json")
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"), "Should be able to override Content-Type header")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
