package plugin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		wantErr string
		wantRT  RuntimeKind
	}{
		{
			name: "valid grpc manifest",
			yaml: `name: test-plugin
version: "1.0.0"
runtime: grpc
grpc:
  binary: ./server`,
			wantRT: RuntimeGRPC,
		},
		{
			name: "valid wasm manifest",
			yaml: `name: test-plugin
version: "1.0.0"
runtime: wasm
wasm:
  module: ./plugin.wasm`,
			wantRT: RuntimeWasm,
		},
		{
			name:    "missing name",
			yaml:    `runtime: grpc`,
			wantErr: "missing name",
		},
		{
			name: "invalid runtime",
			yaml: `name: test
runtime: invalid`,
			wantErr: "invalid runtime",
		},
		{
			name: "grpc without grpc config",
			yaml: `name: test
runtime: grpc`,
			wantErr: "grpc config required",
		},
		{
			name: "wasm without wasm config",
			yaml: `name: test
runtime: wasm`,
			wantErr: "wasm config required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ParseManifest([]byte(tt.yaml))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRT, m.Runtime)
		})
	}
}

func TestManifestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schema  json.RawMessage
		config  json.RawMessage
		wantErr string
	}{
		{
			name:    "no schema passes",
			schema:  nil,
			config:  json.RawMessage(`{"any": true}`),
			wantErr: "",
		},
		{
			name: "valid config passes",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {"api_key": {"type": "string"}},
				"required": ["api_key"]
			}`),
			config:  json.RawMessage(`{"api_key": "secret"}`),
			wantErr: "",
		},
		{
			name: "missing required field fails",
			schema: json.RawMessage(`{
				"type": "object",
				"properties": {"api_key": {"type": "string"}},
				"required": ["api_key"]
			}`),
			config:  json.RawMessage(`{}`),
			wantErr: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{ConfigSchema: tt.schema}
			err := m.ValidateConfig(tt.config)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
