package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetNameInYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		yaml    string
		newName string
		wantErr bool
	}{
		{
			name:    "empty input rejected",
			yaml:    "",
			newName: "renamed",
			wantErr: true,
		},
		{
			name:    "invalid yaml rejected",
			yaml:    "name: [",
			newName: "renamed",
			wantErr: true,
		},
		{
			name:    "invalid new name rejected",
			yaml:    "name: old\ntriggers: []\nsteps: []",
			newName: "-bad",
			wantErr: true,
		},
		{
			name:    "updates top-level name",
			yaml:    "name: old\nenabled: true\ntriggers: []\nsteps: []",
			newName: "renamed",
		},
		{
			name:    "supports unicode name",
			yaml:    "name: old\ntriggers: []\nsteps: []",
			newName: "数据同步",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := SetNameInYAML(tt.yaml, tt.newName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			def, parseErr := ParseEditorYAML(got)
			require.NoError(t, parseErr)
			assert.Equal(t, tt.newName, def.Name)
		})
	}
}
