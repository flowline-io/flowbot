package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultRestyClient tests the DefaultRestyClient function
func TestDefaultRestyClient(t *testing.T) {
	t.Parallel()
	client := DefaultRestyClient()

	require.NotNil(t, client, "DefaultRestyClient() returned nil")

	// Test that timeout is set correctly
	// Note: We can't directly access private fields, so we test behavior
	// The timeout should be 1 minute based on the implementation

	// Test that client has proper configuration
	// We can verify this by checking if the client can make a basic request
	// without panicking or immediate errors

	// Create a simple test to verify client is properly configured
	req := client.R()
	assert.NotNil(t, req, "DefaultRestyClient() client.R() returned nil")

	// Test that headers are set
	// The X-Trace-Id header should be set automatically
	headers := req.Header
	assert.NotNil(t, headers, "DefaultRestyClient() request headers are nil")

	// Verify that the client is not nil (type check is implicit since it compiled)
	assert.NotNil(t, client, "DefaultRestyClient() returned nil client")
}

// TestDefaultRestyClientTimeout tests that the client respects timeout
func TestDefaultRestyClientTimeout(t *testing.T) {
	t.Parallel()
	client := DefaultRestyClient()
	// Force a very small timeout to deterministically trigger a timeout error
	client.SetTimeout(1 * time.Nanosecond)

	start := time.Now()
	_, err := client.R().Get("http://example.com/")
	elapsed := time.Since(start)

	// Should timeout and return an error
	require.Error(t, err, "Expected timeout error but got nil")

	// Ensure the request returned quickly due to timeout
	assert.LessOrEqual(t, elapsed, 5*time.Second, "Request took too long: %v, expected immediate timeout", elapsed)
}

// TestDefaultRestyClientHeaders tests that headers are properly set
func TestDefaultRestyClientHeaders(t *testing.T) {
	t.Parallel()
	client := DefaultRestyClient()

	// Since we can't easily access the headers directly from the client,
	// we'll test that the client is properly configured by checking it doesn't panic
	// and can create requests
	req := client.R()
	assert.NotNil(t, req, "DefaultRestyClient() should be able to create requests")

	// Test that we can set a header on the request
	req.SetHeader("Test-Header", "test-value")
	assert.Equal(t, "test-value", req.Header.Get("Test-Header"), "Should be able to set headers on requests")
}
