package chatagent_test

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateKnowledgePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "valid nested markdown", path: "/docs/develop/api-specs.md"},
		{name: "valid root markdown", path: "/readme.md"},
		{name: "empty path", path: "", wantErr: "path is required"},
		{name: "missing leading slash", path: "docs/api.md", wantErr: "path must start with /"},
		{name: "missing md suffix", path: "/docs/api", wantErr: "path must end with .md"},
		{name: "parent segment", path: "/docs/../secret.md", wantErr: "path must not contain parent segments"},
		{name: "empty segment", path: "/docs//api.md", wantErr: "path must not contain empty segments"},
		{name: "invalid characters", path: "/docs/api specs.md", wantErr: "path contains invalid characters"},
		{name: "only slash", path: "/", wantErr: "path must end with .md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := chatagent.ValidateKnowledgePath(tt.path)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
