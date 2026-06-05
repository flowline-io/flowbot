package grpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/plugin"
)

func TestNewGrpcRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest *plugin.Manifest
	}{
		{
			name: "valid grpc manifest creates runner",
			manifest: &plugin.Manifest{
				Name:    "test",
				Runtime: plugin.RuntimeGRPC,
				GRPC:    &plugin.GRPCConfig{Binary: "/nonexistent/plugin", Args: []string{}},
			},
		},
		{
			name: "runner stores manifest",
			manifest: &plugin.Manifest{
				Name:    "test2",
				Version: "1.0.0",
				Runtime: plugin.RuntimeGRPC,
				GRPC:    &plugin.GRPCConfig{Binary: "/bin/echo", Args: []string{}},
			},
		},
		{
			name: "runner with args",
			manifest: &plugin.Manifest{
				Name:    "test3",
				Runtime: plugin.RuntimeGRPC,
				GRPC:    &plugin.GRPCConfig{Binary: "/bin/ls", Args: []string{"-la"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewGrpcRunner(tt.manifest)
			assert.NoError(t, err)
			assert.NotNil(t, runner)
			assert.NotNil(t, runner.client)
			assert.Equal(t, tt.manifest, runner.manifest)
		})
	}
}

func TestGrpcRunnerHealthUnstarted(t *testing.T) {
	t.Parallel()

	runner := &GrpcRunner{started: false}
	_, err := runner.Call(context.Background(), "command", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestGrpcRunnerUnknownFunction(t *testing.T) {
	t.Parallel()

	runner := &GrpcRunner{started: true}
	_, err := runner.Call(context.Background(), "nonexistent", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown function")
}
