package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUserConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     permission.Config
		wantErr bool
	}{
		{
			name: "valid pattern map",
			cfg: permission.Config{
				"bash": {
					Patterns: []permission.PatternRule{
						{Pattern: "git *", Action: permission.ActionAllow},
						{Pattern: "ls *", Action: permission.ActionAsk},
					},
				},
			},
		},
		{
			name:    "reject bash default allow",
			cfg:     permission.Config{"bash": {Default: permission.ActionAllow}},
			wantErr: true,
		},
		{
			name: "reject wildcard pattern",
			cfg: permission.Config{
				"bash": {
					Patterns: []permission.PatternRule{
						{Pattern: "*", Action: permission.ActionAsk},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "reject wildcard key",
			cfg:     permission.Config{permission.KeyWildcard: {Default: permission.ActionAsk}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := permission.ValidateUserConfig(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateUserConfigAllowsNonSensitiveKeys(t *testing.T) {
	cfg := permission.Config{"skill": {Default: permission.ActionAllow}}
	assert.NoError(t, permission.ValidateUserConfig(cfg))
}
