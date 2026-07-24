package exec_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/exec"
)

func TestResolveWorkDir(t *testing.T) {
	root := t.TempDir()
	cfg := exec.Config{Workspace: root}

	dir, err := cfg.ResolveWorkDir("")
	require.NoError(t, err)
	assert.Equal(t, root, dir)

	dir, err = cfg.ResolveWorkDir("sub")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "sub"), dir)

	_, err = cfg.ResolveWorkDir("..")
	require.Error(t, err)

	_, err = cfg.ResolveWorkDir(root)
	require.Error(t, err)
}

func TestRunCode_Python(t *testing.T) {
	root := t.TempDir()
	cfg := exec.Config{Workspace: root}
	res, err := exec.RunCode(context.Background(), cfg, "python", "print(1+1)", "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Contains(t, res.Output, "2")
}
