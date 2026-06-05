package wasm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/plugin"
)

func TestNewWasmRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest *plugin.Manifest
		wantErr  bool
	}{
		{
			name: "valid wasm manifest creates runner",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeWasm,
				Wasm: &plugin.WasmConfig{
					Module: "./testdata/empty.wasm",
				},
			},
		},
		{
			name: "default timeout when no execution limit",
			manifest: &plugin.Manifest{
				Name:    "test2",
				Runtime: plugin.RuntimeWasm,
				Wasm: &plugin.WasmConfig{
					Module: "./testdata/empty.wasm",
				},
			},
		},
		{
			name: "nil wasm config returns error",
			manifest: &plugin.Manifest{
				Name:    "test3",
				Runtime: plugin.RuntimeWasm,
				Wasm:    nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewWasmRunner(tt.manifest)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, runner)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, runner)
			assert.NotNil(t, runner.runtime)
		})
	}
}

func TestWasmRunnerHealthUnstarted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupFn   func() *WasmRunner
		wantErr   bool
		errSubstr string
	}{
		{
			name: "unstarted runner rejects Call",
			setupFn: func() *WasmRunner {
				r, err := NewWasmRunner(&plugin.Manifest{
					Name:    "test",
					Runtime: plugin.RuntimeWasm,
					Wasm: &plugin.WasmConfig{
						Module: "./testdata/empty.wasm",
					},
				})
				assert.NoError(t, err)
				return r
			},
			wantErr:   true,
			errSubstr: "not started",
		},
		{
			name: "zero-value runner rejects Call",
			setupFn: func() *WasmRunner {
				return &WasmRunner{}
			},
			wantErr:   true,
			errSubstr: "not started",
		},
		{
			name: "nil instance rejects Call",
			setupFn: func() *WasmRunner {
				r := &WasmRunner{}
				r.started.Store(true)
				return r
			},
			wantErr:   true,
			errSubstr: "no available instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := tt.setupFn()
			_, err := runner.Call(context.Background(), "command", json.RawMessage(`{}`))
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestWasmRunnerCustomTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		timeout     string
		wantTimeout time.Duration
	}{
		{
			name:        "custom 10s timeout",
			timeout:     "10s",
			wantTimeout: 10 * time.Second,
		},
		{
			name:        "custom 1m timeout",
			timeout:     "1m",
			wantTimeout: 1 * time.Minute,
		},
		{
			name:        "invalid timeout string uses default 30s",
			timeout:     "invalid",
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewWasmRunner(&plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeWasm,
				Wasm: &plugin.WasmConfig{
					Module: "./testdata/empty.wasm",
					Permissions: &plugin.WasmPermissions{
						Execution: &plugin.ExecutionLimit{Timeout: tt.timeout},
					},
				},
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTimeout, runner.timeout)
		})
	}
}

func TestWasmRunnerDefaultTimeout(t *testing.T) {
	t.Parallel()

	runner, err := NewWasmRunner(&plugin.Manifest{
		Name:    "default-timeout",
		Runtime: plugin.RuntimeWasm,
		Wasm: &plugin.WasmConfig{
			Module: "./testdata/empty.wasm",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, runner.timeout)
}

func TestWasmRunnerNilPermissions(t *testing.T) {
	t.Parallel()

	runner, err := NewWasmRunner(&plugin.Manifest{
		Name:    "nil-perms",
		Runtime: plugin.RuntimeWasm,
		Wasm: &plugin.WasmConfig{
			Module:      "./testdata/empty.wasm",
			Permissions: nil,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, runner.timeout)
	assert.Equal(t, uint32(64*1024*1024), runner.memMax)
}
