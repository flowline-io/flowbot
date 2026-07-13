package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalStringListParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		key     string
		want    []string
		wantErr bool
	}{
		{
			name:   "missing key returns nil",
			params: map[string]any{},
			key:    "tools",
		},
		{
			name:   "nil value returns nil",
			params: map[string]any{"tools": nil},
			key:    "tools",
		},
		{
			name:   "empty string slice returns nil",
			params: map[string]any{"tools": []string{}},
			key:    "tools",
		},
		{
			name:   "string slice is copied",
			params: map[string]any{"tools": []string{"read_file", "web_search"}},
			key:    "tools",
			want:   []string{"read_file", "web_search"},
		},
		{
			name:   "any slice strings are parsed",
			params: map[string]any{"skills": []any{"skill-a", "skill-b"}},
			key:    "skills",
			want:   []string{"skill-a", "skill-b"},
		},
		{
			name:    "non-array value returns error",
			params:  map[string]any{"tools": "read_file"},
			key:     "tools",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := optionalStringListParam(tt.params, tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
