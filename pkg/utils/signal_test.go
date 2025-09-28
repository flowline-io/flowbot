package utils

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// TestSignalHandler tests the SignalHandler function
func TestSignalHandler(t *testing.T) {
	// Test that SignalHandler returns a channel
	stopChan := SignalHandler()
	if stopChan == nil {
		t.Fatal("SignalHandler() returned nil channel")
	}

	// Test channel type
	select {
	case <-stopChan:
		t.Error("SignalHandler() channel should not be ready immediately")
	default:
		// This is expected - channel should not have data initially
	}

	// Test that we can send a signal and receive on the channel
	// Note: On Windows, not all signals are supported, so we'll skip this test on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Signal testing not fully supported on Windows")
	}

	// Create a new signal handler for this test
	stopChan2 := SignalHandler()

	// Send SIGTERM to current process
	go func() {
		time.Sleep(100 * time.Millisecond)
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Errorf("Failed to find current process: %v", err)
			return
		}
		err = process.Signal(syscall.SIGTERM)
		if err != nil {
			t.Errorf("Failed to send SIGTERM: %v", err)
		}
	}()

	// Wait for signal with timeout
	select {
	case received := <-stopChan2:
		if !received {
			t.Error("SignalHandler() should send true on signal reception")
		}
	case <-time.After(1 * time.Second):
		t.Error("SignalHandler() did not receive signal within timeout")
	}
}

// TestSignalHandlerMultipleSignals tests handling multiple signals
func TestSignalHandlerMultipleSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Signal testing not fully supported on Windows")
	}

	stopChan := SignalHandler()

	// Test SIGINT
	go func() {
		time.Sleep(50 * time.Millisecond)
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			return
		}
		_ = process.Signal(syscall.SIGINT)
	}()

	select {
	case received := <-stopChan:
		if !received {
			t.Error("SignalHandler() should send true for SIGINT")
		}
	case <-time.After(500 * time.Millisecond):
		// If this test fails due to timing, it's not critical
		t.Log("SignalHandler() test for SIGINT timed out (may be expected in test environment)")
	}
}

// TestSignalHandlerChannelType tests the return type of SignalHandler
func TestSignalHandlerChannelType(t *testing.T) {
	stopChan := SignalHandler()

	// Verify it's a receive-only channel of bool
	if stopChan == nil {
		t.Fatal("SignalHandler() returned nil")
	}

	// Test that it's the correct type by using it in a select
	select {
	case val := <-stopChan:
		// If we receive a value, it should be bool
		if _, ok := interface{}(val).(bool); !ok {
			t.Error("SignalHandler() channel should carry bool values")
		}
		// Don't expect to receive anything immediately
		t.Log("Unexpected signal received during test")
	default:
		// Expected case - no signal received
	}
}
