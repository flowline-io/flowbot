package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePipelineName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "ascii letters and digits",
			input:   "my-pipeline_01",
			wantErr: false,
		},
		{
			name:    "chinese name",
			input:   "数据同步",
			wantErr: false,
		},
		{
			name:    "mixed chinese and ascii",
			input:   "同步-bookmarks",
			wantErr: false,
		},
		{
			name:    "empty name rejected",
			input:   "",
			wantErr: true,
		},
		{
			name:    "leading hyphen rejected",
			input:   "-bad-name",
			wantErr: true,
		},
		{
			name:    "space rejected",
			input:   "bad name",
			wantErr: true,
		},
		{
			name:    "slash rejected",
			input:   "bad/name",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePipelineName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, PipelineNamePattern.MatchString(tt.input))
		})
	}
}
