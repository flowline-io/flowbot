package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveLoadDebug(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load debug true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveDebug(true, "")
			require.NoError(t, err)

			enabled, err := LoadDebug("")
			require.NoError(t, err)
			require.True(t, enabled)
		})
	}
}

func TestSaveLoadDebugFalse(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load debug false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveDebug(false, "")
			require.NoError(t, err)

			enabled, err := LoadDebug("")
			require.NoError(t, err)
			require.False(t, enabled)
		})
	}
}

func TestLoadDebugNotExist(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "load debug when file does not exist returns false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			enabled, err := LoadDebug("")
			require.NoError(t, err)
			require.False(t, enabled)
		})
	}
}

func TestSaveLoadDebugWithProfile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load debug with profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveDebug(true, "dev")
			require.NoError(t, err)

			enabled, err := LoadDebug("dev")
			require.NoError(t, err)
			require.True(t, enabled)
		})
	}
}

func TestDebugProfileIsolation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "debug profiles are isolated from each other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestGetDebugPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "get debug path for default profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			path, err := GetDebugPath("")
			require.NoError(t, err)
			require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "debug"), path)
		})
	}
}

func TestGetDebugPathWithProfile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "get debug path with profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			path, err := GetDebugPath("staging")
			require.NoError(t, err)
			require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "debug.staging"), path)
		})
	}
}

func TestSaveDebugFileContent(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save debug writes true to file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			require.NoError(t, SaveDebug(true, ""))

			path, err := GetDebugPath("")
			require.NoError(t, err)

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "true", string(data))
		})
	}
}

func TestSaveDebugFileContentFalse(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save debug writes false to file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			require.NoError(t, SaveDebug(false, ""))

			path, err := GetDebugPath("")
			require.NoError(t, err)

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "false", string(data))
		})
	}
}
