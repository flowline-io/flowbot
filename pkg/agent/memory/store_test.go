package memory_test

import (
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeScope(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty defaults", input: "", want: "default"},
		{name: "plain name", input: "my-pipeline", want: "my-pipeline"},
		{name: "spaces to underscore", input: "my pipeline", want: "my_pipeline"},
		{name: "special chars collapsed", input: "foo/bar*baz", want: "foo_bar_baz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, memory.SanitizeScope(tt.input))
		})
	}
}

func TestFileStoreReadWriteList(t *testing.T) {
	root := t.TempDir()
	store, err := memory.NewFileStore(root, "MEMORIES.md", 1024)
	require.NoError(t, err)

	tests := []struct {
		name    string
		scope   string
		file    string
		content string
	}{
		{name: "default scope write", scope: "", file: "", content: "hello"},
		{name: "pipeline scope", scope: "sync-bookmarks", file: "NOTES.md", content: "note"},
		{name: "overwrite", scope: "sync-bookmarks", file: "MEMORIES.md", content: "updated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, store.Write(tt.scope, tt.file, tt.content))
			got, err := store.Read(tt.scope, tt.file)
			require.NoError(t, err)
			assert.Equal(t, tt.content, got)
		})
	}

	files, err := store.ListFiles("sync-bookmarks")
	require.NoError(t, err)
	assert.Contains(t, files, "NOTES.md")
	assert.Contains(t, files, "MEMORIES.md")
}

func TestFileStoreRejectsInvalidInput(t *testing.T) {
	root := t.TempDir()
	store, err := memory.NewFileStore(root, "MEMORIES.md", 64)
	require.NoError(t, err)

	tests := []struct {
		name    string
		scope   string
		file    string
		content string
	}{
		{name: "path traversal file", scope: "a", file: "../evil.md", content: "x"},
		{name: "non md file", scope: "a", file: "notes.txt", content: "x"},
		{name: "oversized content", scope: "a", file: "", content: string(make([]byte, 128))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Write(tt.scope, tt.file, tt.content)
			require.Error(t, err)
		})
	}
}

func TestFileStoreScopeIsolation(t *testing.T) {
	root := t.TempDir()
	store, err := memory.NewFileStore(root, "MEMORIES.md", 1024)
	require.NoError(t, err)

	require.NoError(t, store.Write("scope-a", "", "alpha"))
	require.NoError(t, store.Write("scope-b", "", "beta"))

	a, err := store.Read("scope-a", "")
	require.NoError(t, err)
	b, err := store.Read("scope-b", "")
	require.NoError(t, err)
	assert.Equal(t, "alpha", a)
	assert.Equal(t, "beta", b)
	assert.NotEqual(t, filepath.Join(root, "scope-a"), filepath.Join(root, "scope-b"))
}
