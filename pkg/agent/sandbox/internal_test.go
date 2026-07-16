package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRunOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		wantErr string
	}{
		{name: "valid options", opts: RunOptions{Workspace: "/ws", Image: "img:1"}},
		{name: "missing workspace", opts: RunOptions{Image: "img:1"}, wantErr: "workspace is required"},
		{name: "missing image", opts: RunOptions{Workspace: "/ws"}, wantErr: "image is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRunOptions(tt.opts)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestBuildCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		want    []string
		wantErr string
	}{
		{name: "argv passthrough", opts: RunOptions{Argv: []string{"python", "main.py"}}, want: []string{"python", "main.py"}},
		{name: "shell wraps command", opts: RunOptions{Command: "echo hi"}, want: []string{"sh", "-c", "echo hi"}},
		{name: "empty command errors", opts: RunOptions{}, wantErr: "empty command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := buildCommand(tt.opts)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripDockerLogHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{name: "short payload unchanged", in: []byte("hi"), want: "hi"},
		{name: "empty payload", in: []byte{}, want: ""},
		{name: "framed docker logs stripped", in: append([]byte{1, 0, 0, 0, 0, 0, 0, 3}, []byte("abc")...), want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripDockerLogHeaders(tt.in))
		})
	}
}

func TestBuildHostConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		wantErr string
	}{
		{name: "binds workspace", opts: RunOptions{Workspace: "/host/ws"}},
		{name: "sets network mode", opts: RunOptions{Workspace: "/host/ws", Network: "bridge"}},
		{name: "invalid memory", opts: RunOptions{Workspace: "/host/ws", Memory: "not-memory"}, wantErr: "memory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hc, err := buildHostConfig(tt.opts)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, hc.Binds)
		})
	}
}
