package exec_test

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
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
	res, err := exec.RunCode(context.Background(), cfg, "python", "print(1+1)", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Contains(t, res.Output, "2")
}

func TestRunEntrypoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entrypoint string
		source     string
		stdin      []byte
		wantErr    string
		wantOut    string
		checkEnv   bool
	}{
		{
			name:       "python with stdin JSON",
			entrypoint: "main.py",
			source:     "import sys,json\ndata=json.load(sys.stdin)\nprint(json.dumps({\"echo\":data}))\n",
			stdin:      []byte(`{"hello":"world"}`),
			wantOut:    `"hello": "world"`,
		},
		{
			name:       "reject bad entrypoint",
			entrypoint: "script.py",
			source:     "print(1)\n",
			wantErr:    "entrypoint must be",
		},
		{
			name:       "go command env",
			entrypoint: "main.go",
			source:     "package main\nfunc main() {}\n",
			checkEnv:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.checkEnv {
				fake := &captureEnv{ExecutionEnv: env.Default()}
				cfg := exec.Config{Workspace: root, Env: fake}
				_, err := exec.RunEntrypoint(context.Background(), cfg, tt.entrypoint, tt.source, tt.stdin, nil)
				// Prefer capturing argv/env even when go is missing on PATH.
				require.NotNil(t, fake.last)
				assert.Equal(t, []string{"go", "run", "main.go"}, fake.last.Argv)
				assert.Contains(t, fake.last.Env, "GOPROXY=off")
				assert.Contains(t, fake.last.Env, "GOSUMDB=off")
				assert.Contains(t, fake.last.Env, "GOTELEMETRY=off")
				assert.Contains(t, fake.last.Env, "GOTOOLCHAIN=local")
				assert.Contains(t, fake.last.Env, "CGO_ENABLED=0")
				assert.True(t, hasEnvPrefix(fake.last.Env, "GOCACHE="+filepath.Join(root, ".gocache")))
				assert.True(t, hasEnvPrefix(fake.last.Env, "GOPATH="+filepath.Join(root, ".gopath")))
				mod := filepath.Join(root, "go.mod")
				data := fake.ReadFile(context.Background(), mod)
				require.True(t, data.IsOk(), "go.mod should be written")
				assert.Contains(t, string(data.Value()), "module flowbotfn")
				assert.Contains(t, string(data.Value()), "go 1.26")
				if err != nil && !strings.Contains(err.Error(), "go") {
					// Unexpected non-go failure after env assertions.
					require.NoError(t, err)
				}
				return
			}
			cfg := exec.Config{Workspace: root}
			res, err := exec.RunEntrypoint(context.Background(), cfg, tt.entrypoint, tt.source, tt.stdin, nil)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 0, res.ExitCode)
			assert.Contains(t, res.Stdout, tt.wantOut)
		})
	}
}

type captureEnv struct {
	env.ExecutionEnv
	last *env.ExecOptions
}

func (c *captureEnv) Exec(_ context.Context, opts env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	copied := opts
	copied.Argv = append([]string(nil), opts.Argv...)
	copied.Env = append([]string(nil), opts.Env...)
	c.last = &copied
	return result.Ok[env.Capture, result.ExecutionError](env.Capture{ExitCode: 0})
}

func hasEnvPrefix(envVars []string, want string) bool {
	return slices.Contains(envVars, want)
}
