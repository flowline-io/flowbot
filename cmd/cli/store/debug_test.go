package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveLoadDebug(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := SaveDebug(true, "")
	require.NoError(t, err)

	enabled, err := LoadDebug("")
	require.NoError(t, err)
	require.True(t, enabled)
}

func TestSaveLoadDebugFalse(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := SaveDebug(false, "")
	require.NoError(t, err)

	enabled, err := LoadDebug("")
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestLoadDebugNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	enabled, err := LoadDebug("")
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestSaveLoadDebugWithProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := SaveDebug(true, "dev")
	require.NoError(t, err)

	enabled, err := LoadDebug("dev")
	require.NoError(t, err)
	require.True(t, enabled)
}

func TestDebugProfileIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, SaveDebug(true, ""))
	require.NoError(t, SaveDebug(false, "dev"))

	defaultDebug, err := LoadDebug("")
	require.NoError(t, err)
	require.True(t, defaultDebug)

	devDebug, err := LoadDebug("dev")
	require.NoError(t, err)
	require.False(t, devDebug)
}

func TestGetDebugPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := GetDebugPath("")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "debug"), path)
}

func TestGetDebugPathWithProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := GetDebugPath("staging")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "debug.staging"), path)
}

func TestSaveDebugFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, SaveDebug(true, ""))

	path, err := GetDebugPath("")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "true", string(data))
}

func TestSaveDebugFileContentFalse(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, SaveDebug(false, ""))

	path, err := GetDebugPath("")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "false", string(data))
}
