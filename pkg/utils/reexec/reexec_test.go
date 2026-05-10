package reexec // import "github.com/docker/docker/pkg/reexec"

import (
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"

	"gotest.tools/v3/assert"
)

var registerSeq atomic.Int32

func TestRegister(t *testing.T) {
	testFunc := func() {}

	uniqueName := fmt.Sprintf("test_unique_%d_%d", os.Getpid(), registerSeq.Add(1))

	Register(uniqueName, testFunc)
	t.Logf("Successfully registered function with name: %s", uniqueName)
}

func TestCommand(t *testing.T) {
	t.Parallel()

	cmd := Command("nonexistent")
	// On unsupported platforms (like Windows), Command returns nil
	if cmd == nil {
		t.Skip("Command not supported on this platform")
		return
	}

	// Test that we can create a command - if we reach here, cmd is not nil
	t.Logf("Command created successfully for nonexistent command")
}

func TestNaiveSelf(t *testing.T) {
	if os.Getenv("TEST_CHECK") == "1" {
		os.Exit(2)
	}

	selfPath := naiveSelf()
	if selfPath == "" {
		t.Skip("naiveSelf returned empty string on this platform")
		return
	}

	cmd := exec.Command(selfPath, "-test.run=TestNaiveSelf")
	cmd.Env = append(os.Environ(), "TEST_CHECK=1")
	err := cmd.Start()
	assert.NilError(t, err, "Unable to start command")
	err = cmd.Wait()
	assert.Error(t, err, "exit status 2")

	originalArg := os.Args[0]
	os.Args[0] = "mkdir"
	assert.Check(t, naiveSelf() != os.Args[0])
	os.Args[0] = originalArg // Restore original value
}

// TestInit tests the Init function behavior
func TestInit(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Test with a non-existent command name
	os.Args = []string{"nonexistent_command_test"}
	result := Init()

	// Should return false because no initializer is registered for this name
	if result {
		t.Error("Init() should return false for unregistered command")
	}

	t.Logf("Init() correctly returned false for unregistered command")
}
