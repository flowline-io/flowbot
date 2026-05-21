package utils

import (
	"net/http"
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

func TestHTTPTransport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "returns non-nil transport",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				require.NotNil(t, tr, "HTTPTransport() returned nil")
			},
		},
		{
			name: "has MaxIdleConnsPerHost set to 10",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				assert.Equal(t, 10, tr.MaxIdleConnsPerHost, "HTTPTransport() MaxIdleConnsPerHost should be 10")
			},
		},
		{
			name: "has MaxIdleConns set to 100",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				assert.Equal(t, 100, tr.MaxIdleConns, "HTTPTransport() MaxIdleConns should be 100")
			},
		},
		{
			name: "returns same instance on repeated calls",
			fn: func(t *testing.T) {
				t.Parallel()
				tr1 := HTTPTransport()
				tr2 := HTTPTransport()
				assert.Same(t, tr1, tr2, "HTTPTransport() should return the same instance")
			},
		},
		{
			name: "transport is a clone of http.DefaultTransport",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				defaultTr, ok := http.DefaultTransport.(*http.Transport)
				require.True(t, ok, "http.DefaultTransport should be *http.Transport")
				assert.Equal(t, defaultTr.TLSClientConfig, tr.TLSClientConfig, "TLS config should match DefaultTransport")
				assert.NotNil(t, tr.Proxy, "Proxy func should be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
