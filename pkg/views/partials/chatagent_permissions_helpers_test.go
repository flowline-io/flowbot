package partials_test

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/views/partials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPermissionFormFields(t *testing.T) {
	defaults := permission.DefaultConfig()

	tests := []struct {
		name           string
		user           permission.Config
		wantOverridden []string
		wantSelected   map[string]string
	}{
		{
			name:           "no overrides marks inherit for simple keys",
			user:           permission.Config{},
			wantOverridden: nil,
			wantSelected: map[string]string{
				"websearch": permission.FormActionInherit,
				"skill":     permission.FormActionInherit,
			},
		},
		{
			name: "user override selects stored action",
			user: permission.Config{
				"websearch": {Default: permission.ActionAllow},
			},
			wantOverridden: []string{"websearch"},
			wantSelected: map[string]string{
				"websearch": "allow",
			},
		},
		{
			name: "pattern override exposes pattern rows",
			user: permission.Config{
				"bash": {
					Patterns: []permission.PatternRule{
						{Pattern: "git *", Action: permission.ActionAllow},
					},
				},
			},
			wantOverridden: []string{"bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := chatagent.PermissionsView{
				Defaults:  defaults,
				User:      tt.user,
				Effective: permission.EffectiveConfig(tt.user),
			}
			fields := partials.BuildPermissionFormFields(view)
			require.NotEmpty(t, fields)
			byKey := make(map[string]partials.PermissionFormField)
			for _, field := range fields {
				byKey[field.Meta.Key] = field
			}
			for key, want := range tt.wantSelected {
				assert.Equal(t, want, byKey[key].SelectedAction, "key %s", key)
			}
			for _, key := range tt.wantOverridden {
				assert.True(t, byKey[key].IsOverridden, "expected %s overridden", key)
			}
			if tt.name == "pattern override exposes pattern rows" {
				assert.Len(t, byKey["bash"].PatternRows, 1)
				assert.Equal(t, "git *", byKey["bash"].PatternRows[0].Pattern)
			}
		})
	}
}

func TestFormatRuleSetSummary(t *testing.T) {
	tests := []struct {
		name string
		rs   permission.RuleSet
		want string
	}{
		{name: "simple action", rs: permission.RuleSet{Default: permission.ActionAsk}, want: "ask"},
		{name: "pattern list", rs: permission.RuleSet{
			Patterns: []permission.PatternRule{
				{Pattern: "*.env", Action: permission.ActionDeny},
				{Pattern: "*", Action: permission.ActionAllow},
			},
		}, want: "*.env → deny, * → allow"},
		{name: "empty defaults to ask", rs: permission.RuleSet{}, want: "ask"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, partials.FormatRuleSetSummary(tt.rs))
		})
	}
}
