package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestLoadFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		errContains string
		check       func(t *testing.T, wf *types.WorkflowMetadata)
	}{
		{
			name: "valid workflow file",
			setup: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(t.TempDir(), "wf.yaml")
				content := []byte(`name: file-wf
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo hello
`)
				require.NoError(t, os.WriteFile(path, content, 0o600))
				return path
			},
			check: func(t *testing.T, wf *types.WorkflowMetadata) {
				assert.Equal(t, "file-wf", wf.Name)
				assert.Equal(t, []string{"step1"}, wf.Pipeline)
			},
		},
		{
			name: "missing file",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing.yaml")
			},
			wantErr:     true,
			errContains: "read workflow file",
		},
		{
			name: "invalid yaml content",
			setup: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(t.TempDir(), "bad.yaml")
				require.NoError(t, os.WriteFile(path, []byte("{{invalid"), 0o600))
				return path
			},
			wantErr:     true,
			errContains: "parse workflow yaml",
		},
		{
			name: "dag validation failure",
			setup: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(t.TempDir(), "cycle.yaml")
				content := []byte(`name: cycle-wf
pipeline:
  - a
tasks:
  - id: a
    action: echo
    conn: [b]
  - id: b
    action: echo
    conn: [a]
`)
				require.NoError(t, os.WriteFile(path, content, 0o600))
				return path
			},
			wantErr:     true,
			errContains: "workflow dag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := tt.setup(t)
			wf, err := LoadFile(path)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, wf)
			if tt.check != nil {
				tt.check(t, wf)
			}
		})
	}
}
