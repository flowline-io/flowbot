package eval_test

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
)

// policyCase is a YAML-driven permission / DCG outcome expectation.
type policyCase struct {
	Name           string         `yaml:"name"`
	Kind           string         `yaml:"kind"` // permission | dcg
	Tool           string         `yaml:"tool"`
	Path           string         `yaml:"path"`
	Command        string         `yaml:"command"`
	Args           map[string]any `yaml:"args"`
	ExternalPath   bool           `yaml:"external_path"`
	WantAction     string         `yaml:"want_action"`
	WantDCGAllow   *bool          `yaml:"want_dcg_allow"`
	PermissionJSON string         `yaml:"permission_json"`
}

func TestPolicyOutcomesFromTestdata(t *testing.T) {
	t.Parallel()
	root := filepath.Join("testdata", "safety")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(root, ent.Name())
		t.Run(ent.Name(), func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var tc policyCase
			require.NoError(t, yaml.Unmarshal(raw, &tc))

			switch tc.Kind {
			case "permission", "":
				cfg := permission.DefaultConfig()
				if tc.PermissionJSON != "" {
					parsed, err := permission.ParseConfig([]byte(tc.PermissionJSON))
					require.NoError(t, err)
					cfg = permission.Merge(cfg, parsed)
				}
				ev := permission.NewEvaluator(cfg)
				args := map[string]any{}
				maps.Copy(args, tc.Args)
				if tc.Path != "" {
					args["path"] = tc.Path
				}
				if tc.Command != "" {
					args["command"] = tc.Command
				}
				got := ev.Evaluate(permission.Request{
					Tool:         tc.Tool,
					Args:         args,
					ExternalPath: tc.ExternalPath,
				}, permission.NewSessionState())
				assert.Equal(t, permission.Action(tc.WantAction), got.Action)
			case "dcg":
				checker := dcg.NewBinaryChecker(dcg.BinaryCheckerOptions{
					ConfigPath: "unused",
					Runner: func(_ context.Context, _ string, _ []string, _ []string) (string, int, error) {
						if tc.WantDCGAllow != nil && *tc.WantDCGAllow {
							return `{"command":"x","decision":"allow"}`, 0, nil
						}
						return `{"command":"x","decision":"deny","reason":"blocked by eval fixture"}`, 1, nil
					},
				})
				dec, err := checker.Check(context.Background(), tc.Command)
				require.NoError(t, err)
				require.NotNil(t, tc.WantDCGAllow)
				assert.Equal(t, *tc.WantDCGAllow, dec.Allow)
			default:
				t.Fatalf("unknown kind %q", tc.Kind)
			}
		})
	}
}
