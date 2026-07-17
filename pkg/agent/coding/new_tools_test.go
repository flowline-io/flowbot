package coding_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDirTool_Execute(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(root, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("b"), 0o644))

	tool := coding.ListDirTool{Workspace: coding.Workspace{Root: root}}
	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		contains  string
	}{
		{name: "lists root", args: map[string]any{"path": "."}, contains: "a.txt"},
		{name: "recursive lists nested", args: map[string]any{"path": ".", "recursive": true}, contains: "sub/b.txt"},
		{name: "traversal blocked", args: map[string]any{"path": ".."}, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				assert.Contains(t, textFromResult(t, result), tt.contains)
			}
		})
	}
}

func TestGlobFilesTool_Execute(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "pkg", "x"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pkg", "x", "a.go"), []byte("package x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.md"), []byte("#"), 0o644))

	tool := coding.GlobFilesTool{Workspace: coding.Workspace{Root: root}}
	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		contains  string
	}{
		{name: "matches go files", args: map[string]any{"pattern": "**/*.go"}, contains: "pkg/x/a.go"},
		{name: "no matches", args: map[string]any{"pattern": "**/*.rs"}, contains: "No files matched"},
		{name: "empty pattern", args: map[string]any{"pattern": "  "}, wantError: true},
		{name: "path escape", args: map[string]any{"pattern": "*.go", "path": ".."}, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				assert.Contains(t, textFromResult(t, result), tt.contains)
			}
		})
	}
}

func TestGrepFilesTool_Execute(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\nfunc Hello() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("hello world\n"), 0o644))

	tool := coding.GrepFilesTool{Workspace: coding.Workspace{Root: root}}
	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		contains  string
	}{
		{name: "finds content", args: map[string]any{"pattern": "Hello"}, contains: "a.go:2:"},
		{name: "glob filter", args: map[string]any{"pattern": "hello", "glob": "**/*.txt"}, contains: "b.txt:1:"},
		{name: "invalid regexp", args: map[string]any{"pattern": "("}, wantError: true},
		{name: "empty pattern", args: map[string]any{"pattern": ""}, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				assert.Contains(t, textFromResult(t, result), tt.contains)
			}
		})
	}
}

func TestApplyPatchTool_Execute(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "exist.txt"), []byte("hello\nworld\n"), 0o644))

	tool := coding.ApplyPatchTool{Workspace: coding.Workspace{Root: root}}
	addPatch := "*** Begin Patch\n*** Add File: new.txt\n+alpha\n+beta\n*** End Patch\n"
	updatePatch := "*** Begin Patch\n*** Update File: exist.txt\n@@\n hello\n-world\n+universe\n*** End Patch\n"
	deletePatch := "*** Begin Patch\n*** Delete File: exist.txt\n*** End Patch\n"
	badHunk := "*** Begin Patch\n*** Update File: exist.txt\n@@\n missing\n-context\n*** End Patch\n"

	tests := []struct {
		name      string
		patch     string
		wantError bool
		check     func(t *testing.T)
	}{
		{
			name:  "adds file",
			patch: addPatch,
			check: func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(root, "new.txt"))
				require.NoError(t, err)
				assert.Equal(t, "alpha\nbeta", string(data))
			},
		},
		{
			name:  "updates file",
			patch: updatePatch,
			check: func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(root, "exist.txt"))
				require.NoError(t, err)
				assert.Contains(t, string(data), "universe")
			},
		},
		{
			name:  "deletes file",
			patch: deletePatch,
			check: func(t *testing.T) {
				_, err := os.Stat(filepath.Join(root, "exist.txt"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{name: "hunk mismatch fails", patch: badHunk, wantError: true},
		{name: "empty patch", patch: "  ", wantError: true},
		{name: "oversized patch", patch: strings.Repeat("x", coding.MaxPatchBytes+1), wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "deletes file" || tt.name == "updates file" || tt.name == "hunk mismatch fails" {
				require.NoError(t, os.WriteFile(filepath.Join(root, "exist.txt"), []byte("hello\nworld\n"), 0o644))
			}
			result, err := tool.Execute(context.Background(), "id", map[string]any{"patch": tt.patch}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if tt.check != nil {
				tt.check(t)
			}
			if tt.name == "hunk mismatch fails" {
				data, err := os.ReadFile(filepath.Join(root, "exist.txt"))
				require.NoError(t, err)
				assert.Equal(t, "hello\nworld\n", string(data))
			}
		})
	}
}

func TestApplyPatchTool_RollbackOnApplyFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep.txt"), []byte("original\n"), 0o644))

	failEnv := &failNthWriteEnv{failAt: 2}
	tool := coding.ApplyPatchTool{Workspace: coding.Workspace{Root: root}, Env: failEnv}
	patch := "*** Begin Patch\n*** Update File: keep.txt\n@@\n-original\n+changed\n*** Add File: other.txt\n+new\n*** End Patch\n"
	result, err := tool.Execute(context.Background(), "id", map[string]any{"patch": patch}, nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	data, err := os.ReadFile(filepath.Join(root, "keep.txt"))
	require.NoError(t, err)
	assert.Equal(t, "original\n", string(data))
	_, err = os.Stat(filepath.Join(root, "other.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestMatchPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "double star go", pattern: "**/*.go", path: "pkg/x/a.go", want: true},
		{name: "no match", pattern: "**/*.go", path: "a.txt", want: false},
		{name: "single segment", pattern: "*.md", path: "readme.md", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := coding.MatchPath(tt.pattern, tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
