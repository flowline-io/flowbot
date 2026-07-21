package chatagent_test

import (
	"archive/zip"
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

func TestImportSkillsFromZip(t *testing.T) {
	t.Parallel()
	validSkill := "---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n\nBody\n"
	tests := []struct {
		name    string
		data    []byte
		wantN   int
		wantErr string
	}{
		{
			name:  "zip without skill md imports nothing",
			data:  mustZip(t, map[string]string{"readme.txt": "hi\n"}),
			wantN: 0,
		},
		{
			name:    "rejects empty archive",
			data:    nil,
			wantErr: "empty zip",
		},
		{
			name:    "rejects invalid zip bytes",
			data:    []byte("not-a-zip"),
			wantErr: "zip",
		},
		{
			name:    "rejects zip slip path",
			data:    mustZip(t, map[string]string{"../evil/SKILL.md": validSkill}),
			wantErr: "illegal path",
		},
		{
			name:    "rejects skill zip when store unavailable",
			data:    mustZip(t, map[string]string{"demo/SKILL.md": validSkill}),
			wantErr: "skill store unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n, err := chatagent.ImportSkillsFromZip(t.Context(), tt.data)
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

func TestImportSkillsFromFSFindsRootSkill(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"SKILL.md": &fstest.MapFile{Data: []byte("---\nname: root-skill\ndescription: Root\n---\n\nBody\n")},
	}
	n, err := chatagent.ImportSkillsFromFS(t.Context(), fsys, ".")
	require.Error(t, err)
	require.Contains(t, err.Error(), "skill store unavailable")
	require.Equal(t, 0, n)
}

// mustZip builds an in-memory zip archive from path→content pairs.
func mustZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}
