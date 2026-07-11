package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUserConfigFromForm(t *testing.T) {
	defaults := permission.DefaultConfig()

	tests := []struct {
		name      string
		form      permission.FormValues
		wantKeys  []string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "inherit all yields empty user config",
			form: permission.FormValues{
				Simple: map[string]string{
					"websearch": "inherit",
					"skill":     "inherit",
				},
			},
			wantKeys: nil,
		},
		{
			name: "simple override stores only changed key",
			form: permission.FormValues{
				Simple: map[string]string{
					"websearch": "allow",
					"skill":     "inherit",
				},
			},
			wantKeys: []string{"websearch"},
		},
		{
			name: "matching default value is not stored",
			form: permission.FormValues{
				Simple: map[string]string{
					"skill": "allow",
				},
			},
			wantKeys: nil,
		},
		{
			name: "pattern override stores pattern rules",
			form: permission.FormValues{
				Patterns: map[string][]permission.FormPatternRow{
					"bash": {
						{Pattern: "git *", Action: "allow"},
					},
				},
			},
			wantKeys: []string{"bash"},
		},
		{
			name: "reject wildcard pattern",
			form: permission.FormValues{
				Patterns: map[string][]permission.FormPatternRow{
					"bash": {
						{Pattern: "*", Action: "ask"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "reject sensitive default allow",
			form: permission.FormValues{
				Simple: map[string]string{
					"bash": "allow",
				},
			},
			wantErr:   true,
			errSubstr: "default allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, errs, err := permission.BuildUserConfigFromForm(defaults, tt.form)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				if tt.name == "reject wildcard pattern" {
					assert.Contains(t, errs["perm.bash.patterns.0.pattern"], "too broad")
				}
				return
			}
			require.NoError(t, err)
			assert.Empty(t, errs)
			if tt.wantKeys == nil {
				assert.Empty(t, cfg)
				return
			}
			for _, key := range tt.wantKeys {
				_, ok := cfg[key]
				assert.True(t, ok, "expected key %q in config", key)
			}
		})
	}
}

func TestParseFormPostArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]string
		wantSimple   map[string]string
		wantPatterns map[string][]permission.FormPatternRow
	}{
		{
			name: "parses simple permission fields",
			args: map[string]string{
				"perm[websearch]": "allow",
				"perm[skill]":     "inherit",
			},
			wantSimple: map[string]string{
				"websearch": "allow",
				"skill":     "inherit",
			},
		},
		{
			name: "parses pattern permission fields",
			args: map[string]string{
				"perm[bash][patterns][0][pattern]": "git *",
				"perm[bash][patterns][0][action]":  "allow",
				"perm[bash][patterns][1][pattern]": "npm *",
				"perm[bash][patterns][1][action]":  "ask",
			},
			wantPatterns: map[string][]permission.FormPatternRow{
				"bash": {
					{Pattern: "git *", Action: "allow"},
					{Pattern: "npm *", Action: "ask"},
				},
			},
		},
		{
			name: "ignores unknown keys",
			args: map[string]string{
				"submit_mode": "form",
				"rules":       "{}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := permission.ParseFormPostArgs(tt.args)
			if tt.wantSimple != nil {
				assert.Equal(t, tt.wantSimple, got.Simple)
			}
			if tt.wantPatterns != nil {
				assert.Equal(t, tt.wantPatterns, got.Patterns)
			}
		})
	}
}
