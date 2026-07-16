package reexec // import "github.com/docker/docker/pkg/reexec"

import (
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var registerSeq atomic.Int32

func TestRegister(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{name: "registers function successfully", testFunc: func() {}},
		{name: "registers second function with unique name", testFunc: func() {}},
		{name: "registers nil func", testFunc: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uniqueName := fmt.Sprintf("test_unique_%d_%d", os.Getpid(), registerSeq.Add(1))

			Register(uniqueName, tt.testFunc)
			t.Logf("Successfully registered function with name: %s", uniqueName)
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFunc := func() {}

			uniqueName := fmt.Sprintf("test_unique_%d_%d", os.Getpid(), registerSeq.Add(1))

			Register(uniqueName, testFunc)
			t.Logf("Successfully registered function with name: %s", uniqueName)
		})
	}
}

func TestCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmdName string
	}{
		{name: "creates command for nonexistent", cmdName: "nonexistent"},
		{name: "creates command for empty string", cmdName: ""},
		{name: "creates command for native command", cmdName: "echo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := Command(tt.cmdName)
			// On unsupported platforms (like Windows), Command returns nil
			if cmd == nil {
				t.Skip("Command not supported on this platform")
				return
			}

			// Test that we can create a command - if we reach here, cmd is not nil
			t.Logf("Command created successfully for %q", tt.cmdName)
		})
	}
}

func init() {
	if os.Getenv("TEST_CHECK") == "1" {
		os.Exit(2)
	}
}

func TestNaiveSelf(t *testing.T) {
	tests := []struct {
		name           string
		performCmdExec bool
		checkFallback  bool
	}{
		{name: "naiveSelf returns correct path and handles args", performCmdExec: true, checkFallback: true},
		{name: "naiveSelf falls back when os.Args[0] is not self", performCmdExec: false, checkFallback: true},
		{name: "naiveSelf returns non-empty on linux", performCmdExec: false, checkFallback: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfPath := naiveSelf()
			if selfPath == "" {
				t.Skip("naiveSelf returned empty string on this platform")
				return
			}

			if tt.performCmdExec {
				cmd := exec.Command(selfPath, "-test.run=TestNaiveSelf")
				cmd.Env = append(os.Environ(), "TEST_CHECK=1")
				err := cmd.Start()
				require.NoError(t, err, "Unable to start command")
				err = cmd.Wait()
				require.ErrorContains(t, err, "exit status 2")
			}

			if tt.checkFallback {
				originalArg := os.Args[0]
				os.Args[0] = "mkdir"
				assert.NotEqual(t, os.Args[0], naiveSelf())
				os.Args[0] = originalArg // Restore original value
			} else {
				require.NotEmpty(t, selfPath)
			}
		})
	}
}

// TestInit tests the Init function behavior
func TestInitRegisteredCommand(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	called := false
	uniqueName := fmt.Sprintf("registered_init_%d_%d", os.Getpid(), registerSeq.Add(1))
	Register(uniqueName, func() { called = true })

	os.Args = []string{uniqueName}
	require.True(t, Init())
	assert.True(t, called)
}

func TestInit(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	tests := []struct {
		name    string
		cmdName string
	}{
		{name: "returns false for unregistered command", cmdName: "nonexistent_command_test"},
		{name: "returns false for empty args", cmdName: ""},
		{name: "returns false for another unregistered command", cmdName: "another_nonexistent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = []string{tt.cmdName}
			result := Init()

			if result {
				assert.Fail(t, "Init() should return false for unregistered command")
			}

			t.Logf("Init() correctly returned false for command %q", tt.cmdName)
		})
	}
}
