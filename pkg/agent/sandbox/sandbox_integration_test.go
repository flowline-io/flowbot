//go:build sandbox

package sandbox_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerRunnerIntegration(t *testing.T) {
	if os.Getenv("DOCKER_HOST") == "" {
		// Still attempt local docker socket; skip only when docker is clearly unavailable.
	}
	image := os.Getenv("FLOWBOT_SANDBOX_IMAGE")
	if image == "" {
		image = "alpine:3.20"
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644))

	e := sandbox.New(sandbox.Config{
		Image:     image,
		Workspace: dir,
		Network:   "none",
		Memory:    "64m",
	}, env.Default(), sandbox.DockerRunner{})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	got := e.Exec(ctx, env.ExecOptions{
		Command: "cat hello.txt",
		Dir:     dir,
		Timeout: ctx,
	})
	require.True(t, got.IsOk(), "exec failed: %v", got.ErrorValue())
	assert.Contains(t, got.Value().Stdout, "hi")
	assert.Equal(t, 0, got.Value().ExitCode)
}
