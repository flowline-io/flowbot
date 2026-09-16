package chatagent_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestParseSkillMarkdownViaImport(t *testing.T) {
	t.Parallel()
	// ImportSkillsFromFS exercises parse through a minimal FS without requiring a database.
	tests := []struct {
		name    string
		files   map[string]string
		wantN   int
		wantErr string
	}{
		{
			name: "skips directories without skill md",
			files: map[string]string{
				"README.md": "# readme\n",
			},
			wantN: 0,
		},
		{
			name: "rejects missing frontmatter",
			files: map[string]string{
				"bad/SKILL.md": "# No frontmatter\n",
			},
			wantErr: "frontmatter",
		},
		{
			name: "rejects empty skill name",
			files: map[string]string{
				"bad/SKILL.md": "---\nname: \"\"\ndescription: x\n---\n\nBody\n",
			},
			wantErr: "non-empty",
		},
		{
			name: "rejects uppercase name",
			files: map[string]string{
				"Bad/SKILL.md": "---\nname: Bad\ndescription: demo skill for tests\n---\n\nBody\n",
			},
			wantErr: "lowercase",
		},
		{
			name: "rejects name directory mismatch",
			files: map[string]string{
				"other/SKILL.md": "---\nname: demo\ndescription: demo skill for tests\n---\n\nBody\n",
			},
			wantErr: "must match",
		},
		{
			name: "rejects consecutive hyphens",
			files: map[string]string{
				"bad--name/SKILL.md": "---\nname: bad--name\ndescription: demo skill for tests\n---\n\nBody\n",
			},
			wantErr: "consecutive",
		},
		{
			name: "rejects oversized description",
			files: map[string]string{
				"demo/SKILL.md": "---\nname: demo\ndescription: " + strings.Repeat("x", validate.SkillDescMaxLen+1) + "\n---\n\nBody\n",
			},
			wantErr: "exceeds",
		},
		{
			name: "rejects oversized compatibility",
			files: map[string]string{
				"demo/SKILL.md": "---\nname: demo\ndescription: demo skill for tests\ncompatibility: " + strings.Repeat("c", validate.SkillCompatibilityMaxLen+1) + "\n---\n\nBody\n",
			},
			wantErr: "exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{}
			for p, body := range tt.files {
				fsys[p] = &fstest.MapFile{Data: []byte(body)}
			}
			n, err := chatagent.ImportSkillsFromFS(t.Context(), fsys, ".")
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantN, n)
		})
	}
}
