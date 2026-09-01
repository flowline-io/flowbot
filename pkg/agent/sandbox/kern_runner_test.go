package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectRunner(t *testing.T) {
	t.Parallel()
	assert.IsType(t, DockerRunner{}, selectRunner(Config{}))
	assert.IsType(t, KernRunner{}, selectRunner(Config{Runtime: "kern"}))
	assert.IsType(t, KernRunner{}, selectRunner(Config{Runtime: "KERN"}))
}

func TestValidateKernNetwork(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateKernNetwork(""))
	require.NoError(t, validateKernNetwork("host"))
	require.NoError(t, validateKernNetwork("none"))
	assert.Error(t, validateKernNetwork("bridge"))
}

func TestSandboxRuntimeLabel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "docker", sandboxRuntimeLabel(""))
	assert.Equal(t, "kern", sandboxRuntimeLabel("kern"))
}
