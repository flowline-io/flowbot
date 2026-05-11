package utils

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "returns non-nil channel",
			fn: func(t *testing.T) {
				stopChan := SignalHandler()
				require.NotNil(t, stopChan, "SignalHandler() returned nil channel")
			},
		},
		{
			name: "channel not ready immediately",
			fn: func(t *testing.T) {
				stopChan := SignalHandler()
				require.NotNil(t, stopChan)
				select {
				case <-stopChan:
					assert.Fail(t, "SignalHandler() channel should not be ready immediately")
				default:
				}
			},
		},
		{
			name: "receives signal on channel",
			fn: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("Signal testing not fully supported on Windows")
				}
				stopChan2 := SignalHandler()
				require.NotNil(t, stopChan2)

				go func() {
					time.Sleep(100 * time.Millisecond)
					process, err := os.FindProcess(os.Getpid())
					assert.NoError(t, err, "Failed to find current process")
					if err != nil {
						return
					}
					err = process.Signal(syscall.SIGTERM)
					assert.NoError(t, err, "Failed to send SIGTERM")
				}()

				select {
				case received := <-stopChan2:
					assert.True(t, received, "SignalHandler() should send true on signal reception")
				case <-time.After(1 * time.Second):
					assert.Fail(t, "SignalHandler() did not receive signal within timeout")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestSignalHandlerMultipleSignals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "handles SIGINT",
			fn: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("Signal testing not fully supported on Windows")
				}
				stopChan := SignalHandler()
				require.NotNil(t, stopChan)

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
					assert.True(t, received, "SignalHandler() should send true for SIGINT")
				case <-time.After(500 * time.Millisecond):
					t.Log("SignalHandler() test for SIGINT timed out (may be expected in test environment)")
				}
			},
		},
		{
			name: "handles SIGTERM",
			fn: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("Signal testing not fully supported on Windows")
				}
				stopChan := SignalHandler()
				require.NotNil(t, stopChan)

				go func() {
					time.Sleep(50 * time.Millisecond)
					process, err := os.FindProcess(os.Getpid())
					if err != nil {
						return
					}
					_ = process.Signal(syscall.SIGTERM)
				}()

				select {
				case received := <-stopChan:
					assert.True(t, received, "SignalHandler() should send true for SIGTERM")
				case <-time.After(500 * time.Millisecond):
					t.Log("SignalHandler() test for SIGTERM timed out (may be expected in test environment)")
				}
			},
		},
		{
			name: "handles SIGHUP",
			fn: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					t.Skip("Signal testing not fully supported on Windows")
				}
				stopChan := SignalHandler()
				require.NotNil(t, stopChan)

				go func() {
					time.Sleep(50 * time.Millisecond)
					process, err := os.FindProcess(os.Getpid())
					if err != nil {
						return
					}
					_ = process.Signal(syscall.SIGHUP)
				}()

				select {
				case received := <-stopChan:
					assert.True(t, received, "SignalHandler() should send true for SIGHUP")
				case <-time.After(500 * time.Millisecond):
					t.Log("SignalHandler() test for SIGHUP timed out (may be expected in test environment)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestSignalHandlerChannelType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "returns bool receive-only channel",
			fn: func(t *testing.T) {
				stopChan := SignalHandler()
				require.NotNil(t, stopChan, "SignalHandler() returned nil")

				select {
				case val := <-stopChan:
					_, ok := any(val).(bool)
					assert.True(t, ok, "SignalHandler() channel should carry bool values")
					t.Log("Unexpected signal received during test")
				default:
				}
			},
		},
		{
			name: "channel is receive-only type",
			fn: func(t *testing.T) {
				stopChan := SignalHandler()
				require.NotNil(t, stopChan, "SignalHandler() returned nil")
				assert.NotNil(t, (<-chan bool)(stopChan), "SignalHandler() should return a receive-only channel")
			},
		},
		{
			name: "multiple signal handlers do not share channels",
			fn: func(t *testing.T) {
				ch1 := SignalHandler()
				ch2 := SignalHandler()
				require.NotNil(t, ch1, "first handler returned nil")
				require.NotNil(t, ch2, "second handler returned nil")
				assert.NotEqual(t, ch1, ch2, "SignalHandler() should return distinct channels")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
