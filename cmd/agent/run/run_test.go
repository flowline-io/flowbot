package run_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/cmd/agent/run"
)

func TestExecute_RequiresConfig(t *testing.T) {
	_, err := run.Execute(context.Background(), run.Options{
		Workspace: t.TempDir(),
		Prompt:    "hi",
	})
	require.Error(t, err)
}

func TestExecute_UnreachableProxy(t *testing.T) {
	cfg := &config.Config{FlowbotURL: "http://127.0.0.1:9", AccessToken: "tok"}
	_, err := run.Execute(context.Background(), run.Options{
		Config:    cfg,
		Workspace: filepath.Clean(t.TempDir()),
		Prompt:    "hi",
		Force:     false,
		Timeout:   time.Second,
	})
	require.Error(t, err)
}
