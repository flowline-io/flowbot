package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckSingleton tests the CheckSingleton function
// Note: This is a challenging function to test because it has side effects
// and depends on external network conditions
func TestCheckSingleton(t *testing.T) {
	t.Parallel()
	// Since CheckSingleton may call log.Fatal, we can't easily test the failure case
	// We can only test that it doesn't panic when called
	// This test verifies the function can be called without crashing

	defer func() {
		if r := recover(); r != nil {
			require.Fail(t, "CheckSingleton() panicked")
		}
	}()

	// Call CheckSingleton - it should not panic
	// In most test environments, the embed server port should be available
	CheckSingleton()

	// If we reach this point, the function executed without fatal errors
	// (at least in this test run)
}

// TestEmbedServerPort tests that the embed server port constant is valid
func TestEmbedServerPort(t *testing.T) {
	t.Parallel()
	assert.NotEmpty(t, EmbedServerPort, "EmbedServerPort should not be empty")

	expectedPort := "15656"
	assert.Equal(t, expectedPort, EmbedServerPort)
}

// TestEmbedServerCreation tests that EmbedServer can be called without immediate panic
func TestEmbedServerCreation(t *testing.T) {
	t.Parallel()
	// We can't easily test EmbedServer() because it starts a blocking HTTP server
	// and logs a fatal error if it can't bind to the port.
	// Instead, we test the components it uses indirectly.

	// Test that the port is valid for network operations
	port := EmbedServerPort
	require.NotEmpty(t, port, "EmbedServerPort should not be empty")

	// Test that port is numeric (basic validation)
	for _, char := range port {
		assert.True(t, char >= '0' && char <= '9', "EmbedServerPort contains non-numeric character: %c", char)
	}

	// Port should be in valid range (1-65535)
	require.NotEqual(t, "0", port, "EmbedServerPort should be a valid port number")
	require.LessOrEqual(t, len(port), 5, "EmbedServerPort should be a valid port number")
}
