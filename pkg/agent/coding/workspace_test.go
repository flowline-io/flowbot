package coding_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace_ResolvePath(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "relative file", path: "src/main.go", wantErr: false},
		{name: "nested relative", path: "./pkg/util.go", wantErr: false},
		{name: "empty path", path: "", wantErr: true},
		{name: "path traversal", path: "../outside.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ws := coding.Workspace{Root: root}
			gotResult := ws.ResolvePath(tt.path)
			if tt.wantErr {
				assert.False(t, gotResult.IsOk())
				return
			}
			assert.True(t, gotResult.IsOk())
			assert.Contains(t, gotResult.Value(), root)
		})
	}
}

func TestWorkspace_TruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		input    string
		contains string
	}{
		{name: "under limit", limit: 100, input: "hello", contains: "hello"},
		{name: "over limit", limit: 5, input: "hello world", contains: "truncated"},
		{name: "default limit", limit: 0, input: "short", contains: "short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ws := coding.Workspace{MaxOutput: tt.limit}
			got := ws.TruncateOutput(tt.input)
			assert.Contains(t, got, tt.contains)
		})
	}
}
