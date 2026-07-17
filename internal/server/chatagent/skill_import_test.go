package chatagent_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

func TestParseSkillMarkdownViaImport(t *testing.T) {
	t.Parallel()
	// ImportSkillsFromFS exercises parse through a minimal FS without requiring a database.
	tests := []struct {
		name    string
		files   map[string]string
		wantN   int
		wantErr bool
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
			wantErr: true,
		},
		{
			name: "rejects empty skill name",
			files: map[string]string{
				"bad/SKILL.md": "---\nname: \"\"\ndescription: x\n---\n\nBody\n",
			},
			wantErr: true,
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
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantN, n)
		})
	}
}
