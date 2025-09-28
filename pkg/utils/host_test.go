package utils

import (
	"testing"
)

// TestHostInfo tests the HostInfo function which retrieves host ID and hostname
func TestHostInfo(t *testing.T) {
	hostID, hostname, err := HostInfo()

	// Test that no error occurs
	if err != nil {
		t.Errorf("HostInfo() returned error: %v", err)
	}

	// Test that hostID is not empty
	if hostID == "" {
		t.Error("HostInfo() returned empty hostID")
	}

	// Test that hostname is not empty
	if hostname == "" {
		t.Error("HostInfo() returned empty hostname")
	}

	// Test that both values are strings
	if len(hostID) == 0 {
		t.Error("HostInfo() hostID should not be empty string")
	}

	if len(hostname) == 0 {
		t.Error("HostInfo() hostname should not be empty string")
	}
}
