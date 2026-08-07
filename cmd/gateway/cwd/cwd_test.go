package cwd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/cmd/gateway/cwd"
)

func TestResolveDefaultAndAllowlist(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got, err := cwd.Resolve("", root, []string{root})
	require.NoError(t, err)
	absRoot, err := filepath.Abs(root)
	require.NoError(t, err)
	require.Equal(t, absRoot, got)

	sub := filepath.Join(root, "proj")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	got, err = cwd.Resolve(sub, root, []string{root})
	require.NoError(t, err)
	absSub, err := filepath.Abs(sub)
	require.NoError(t, err)
	require.Equal(t, absSub, got)
}

func TestResolveRejectsOutside(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := t.TempDir()
	_, err := cwd.Resolve(outside, root, []string{root})
	require.Error(t, err)
}
