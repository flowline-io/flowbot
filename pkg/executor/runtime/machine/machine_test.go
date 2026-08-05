package machine

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

func mustHostKeyB64(t *testing.T) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal())
}

func TestNewRuntime_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "host key required",
			cfg:     Config{Host: "127.0.0.1", Port: 22, Username: "u", Password: "p"},
			wantErr: "host key is required",
		},
		{
			name:    "invalid host key base64",
			cfg:     Config{Host: "127.0.0.1", Port: 22, Username: "u", Password: "p", HostKey: "%%%"},
			wantErr: "invalid host key base64",
		},
		{
			name: "valid base64 but not a public key",
			cfg: Config{
				Host: "127.0.0.1", Port: 22, Username: "u", Password: "p",
				HostKey: base64.StdEncoding.EncodeToString([]byte("not-a-key")),
			},
			wantErr: "failed to parse host key",
		},
		{
			name: "parsed host key but dial fails",
			cfg: Config{
				Host: "127.0.0.1", Port: 1, Username: "u", Password: "p",
				HostKey: "", // filled below
			},
			wantErr: "connect",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			if tt.name == "parsed host key but dial fails" {
				cfg.HostKey = mustHostKeyB64(t)
			}
			rt, err := NewRuntime(WithConfig(cfg))
			require.Error(t, err)
			assert.Nil(t, rt)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestStopCloseAndEmptyTaskID(t *testing.T) {
	rt := &Runtime{tasks: new(syncx.Map[string, *ssh.Session])}

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "stop missing task is no-op",
			fn:   func() error { return rt.Stop(context.Background(), &types.Task{ID: "missing"}) },
		},
		{
			name: "close nil client is no-op",
			fn:   func() error { return rt.Close() },
		},
		{
			name: "doRun requires task id",
			fn:   func() error { return rt.doRun(context.Background(), &types.Task{ID: "", Run: "true"}) },
		},
		{
			name: "Run requires task id on main task",
			fn:   func() error { return rt.Run(context.Background(), &types.Task{ID: "", Run: "true"}) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if tt.name == "doRun requires task id" || tt.name == "Run requires task id on main task" {
				require.EqualError(t, err, "task id is required")
				return
			}
			require.NoError(t, err)
		})
	}
}
