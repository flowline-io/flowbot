package dcg

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryCheckerFailClosedMissingBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		emptyCfg   bool
		command    string
		wantSubstr string
	}{
		{
			name:       "missing path",
			path:       "definitely-missing-dcg-binary",
			command:    "echo ok",
			wantSubstr: "dcg",
		},
		{
			name:       "empty command",
			path:       "definitely-missing-dcg-binary",
			command:    "  ",
			wantSubstr: "command is required",
		},
		{
			name:       "empty config path",
			path:       "definitely-missing-dcg-binary",
			emptyCfg:   true,
			command:    "echo ok",
			wantSubstr: "config path is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := t.TempDir()
			if tt.emptyCfg {
				cfg = ""
			}
			checker := NewBinaryChecker(BinaryCheckerOptions{
				Path:       tt.path,
				ConfigPath: cfg,
			})
			_, err := checker.Check(context.Background(), tt.command)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantSubstr)
		})
	}
}

func TestBinaryCheckerWithRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		run       commandRunner
		wantAllow bool
		wantErr   bool
		wantReas  string
	}{
		{
			name: "allow",
			run: func(context.Context, string, []string, []string) (string, int, error) {
				return `{"decision":"allow","command":"echo ok"}`, 0, nil
			},
			wantAllow: true,
		},
		{
			name: "deny",
			run: func(context.Context, string, []string, []string) (string, int, error) {
				return `{"decision":"deny","reason":"blocked","command":"rm -rf /"}`, 1, nil
			},
			wantAllow: false,
			wantReas:  "blocked",
		},
		{
			name: "runner error fail closed",
			run: func(context.Context, string, []string, []string) (string, int, error) {
				return "", 0, errors.New("spawn failed")
			},
			wantErr: true,
		},
		{
			name: "bad json fail closed",
			run: func(context.Context, string, []string, []string) (string, int, error) {
				return "not-json", 0, nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checker := NewBinaryChecker(BinaryCheckerOptions{
				Path:       "dcg",
				ConfigPath: "cfg.toml",
				Runner:     tt.run,
			})
			d, err := checker.Check(context.Background(), "echo ok")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllow, d.Allow)
			if tt.wantReas != "" {
				assert.Equal(t, tt.wantReas, d.Reason)
			}
		})
	}
}

func TestStripBypassEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "removes bypass",
			in:   []string{"PATH=/bin", "DCG_BYPASS=1", "HOME=/tmp"},
			want: []string{"PATH=/bin", "HOME=/tmp"},
		},
		{
			name: "case insensitive name",
			in:   []string{"dcg_bypass=1", "FOO=bar"},
			want: []string{"FOO=bar"},
		},
		{
			name: "empty",
			in:   nil,
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, StripBypassEnv(tt.in))
		})
	}
}

func TestMaterializeConfig(t *testing.T) {
	// Process-wide cache: do not parallelize with other MaterializeConfig callers.
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "writes embed once",
			run: func(t *testing.T) {
				path, err := MaterializeConfig()
				require.NoError(t, err)
				require.FileExists(t, path)
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Contains(t, string(data), "windows.filesystem")
			},
		},
		{
			name: "second call reuses path",
			run: func(t *testing.T) {
				first, err := MaterializeConfig()
				require.NoError(t, err)
				second, err := MaterializeConfig()
				require.NoError(t, err)
				assert.Equal(t, first, second)
			},
		},
		{
			name: "rewrites when temp removed",
			run: func(t *testing.T) {
				first, err := MaterializeConfig()
				require.NoError(t, err)
				require.NoError(t, os.Remove(first))
				second, err := MaterializeConfig()
				require.NoError(t, err)
				require.FileExists(t, second)
				assert.NotEqual(t, first, second)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestBinaryCheckerStripsBypassInRunnerEnv(t *testing.T) {
	tests := []struct {
		name   string
		setEnv bool
	}{
		{name: "strips DCG_BYPASS when set", setEnv: true},
		{name: "ok when bypass unset", setEnv: false},
		{name: "strips regardless of other env", setEnv: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawEnv []string
			checker := NewBinaryChecker(BinaryCheckerOptions{
				Path:       "dcg",
				ConfigPath: filepath.Join(t.TempDir(), "cfg.toml"),
				Runner: func(_ context.Context, _ string, _ []string, env []string) (string, int, error) {
					sawEnv = append([]string(nil), env...)
					return `{"decision":"allow"}`, 0, nil
				},
			})
			if tt.setEnv {
				t.Setenv("DCG_BYPASS", "1")
			}
			_, err := checker.Check(context.Background(), "echo ok")
			require.NoError(t, err)
			for _, e := range sawEnv {
				key, _, _ := strings.Cut(e, "=")
				assert.False(t, strings.EqualFold(key, "DCG_BYPASS"), "env entry %q", e)
			}
		})
	}
}
