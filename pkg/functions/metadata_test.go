package functions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/functions"
)

func TestParseMetadataYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		yaml    string
		wantErr string
		want    string
	}{
		{
			name: "token auth",
			yaml: "name: parse-bill\nhttp:\n  auth:\n    token: secret\nenv:\n  mode: strict\n",
			want: "parse-bill",
		},
		{
			name: "hmac only",
			yaml: "name: hmac-fn\nhttp:\n  auth:\n    hmac_secret: hs\n",
			want: "hmac-fn",
		},
		{
			name:    "missing auth",
			yaml:    "name: no-auth\nhttp:\n  auth: {}\n",
			wantErr: "http.auth.token",
		},
		{
			name:    "invalid name",
			yaml:    "name: Bad Name\nhttp:\n  auth:\n    token: t\n",
			wantErr: "invalid function name",
		},
		{
			name:    "invalid yaml",
			yaml:    ": not yaml",
			wantErr: "invalid metadata YAML",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := functions.ParseMetadataYAML(tt.yaml)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.Name)
		})
	}
}
