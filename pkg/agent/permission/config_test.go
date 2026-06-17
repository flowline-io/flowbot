package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantKey string
		wantAct permission.Action
		wantErr bool
	}{
		{name: "simple action", raw: `{"bash":"allow"}`, wantKey: "bash", wantAct: permission.ActionAllow},
		{name: "pattern map", raw: `{"bash":{"*":"ask","git *":"allow"}}`, wantKey: "bash"},
		{name: "invalid action", raw: `{"bash":"nope"}`, wantErr: true},
		{name: "empty", raw: "", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := permission.ParseConfig([]byte(tt.raw))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantKey == "" {
				return
			}
			rs, ok := cfg[tt.wantKey]
			require.True(t, ok)
			if tt.wantAct != "" {
				assert.Equal(t, tt.wantAct, rs.Default)
			} else {
				assert.NotEmpty(t, rs.Patterns)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		base permission.Config
		over permission.Config
		key  string
		want permission.Action
	}{
		{name: "overlay replaces key", base: permission.DefaultConfig(), over: permission.Config{"bash": {Default: permission.ActionDeny}}, key: "bash", want: permission.ActionDeny},
		{name: "keeps other keys", base: permission.DefaultConfig(), over: permission.Config{"bash": {Default: permission.ActionDeny}}, key: "read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := permission.Merge(tt.base, tt.over)
			rs := merged[tt.key]
			if tt.want != "" {
				if len(rs.Patterns) > 0 {
					assert.Equal(t, tt.want, rs.Patterns[len(rs.Patterns)-1].Action)
				} else {
					assert.Equal(t, tt.want, rs.Default)
				}
			} else {
				assert.NotEmpty(t, merged["read"].Patterns)
			}
		})
	}
}
