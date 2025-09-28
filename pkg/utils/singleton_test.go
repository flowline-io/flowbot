package utils

import (
	"testing"
)

// TestCheckSingleton tests the CheckSingleton function
// Note: This is a challenging function to test because it has side effects
// and depends on external network conditions
func TestCheckSingleton(t *testing.T) {
	// Since CheckSingleton may call log.Fatal, we can't easily test the failure case
	// We can only test that it doesn't panic when called
	// This test verifies the function can be called without crashing

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CheckSingleton() panicked: %v", r)
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
	// Test that EmbedServerPort is a valid port number
	if EmbedServerPort == "" {
		t.Error("EmbedServerPort should not be empty")
	}

	// Test that it's the expected value
	expectedPort := "15656"
	if EmbedServerPort != expectedPort {
		t.Errorf("EmbedServerPort = %v, want %v", EmbedServerPort, expectedPort)
	}
}

// TestEmbedServerCreation tests that EmbedServer can be called without immediate panic
func TestEmbedServerCreation(t *testing.T) {
	// We can't easily test EmbedServer() because it starts a blocking HTTP server
	// and logs a fatal error if it can't bind to the port.
	// Instead, we test the components it uses indirectly.

	// Test that the port is valid for network operations
	port := EmbedServerPort
	if len(port) == 0 {
		t.Error("EmbedServerPort should not be empty")
	}

	// Test that port is numeric (basic validation)
	for _, char := range port {
		if char < '0' || char > '9' {
			t.Errorf("EmbedServerPort contains non-numeric character: %c", char)
		}
	}

	// Port should be in valid range (1-65535)
	if port == "0" || len(port) > 5 {
		t.Error("EmbedServerPort should be a valid port number")
	}
}
