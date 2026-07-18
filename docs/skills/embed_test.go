package skills_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/docs/skills"
)

func TestEmbeddedSkillTrees(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dir  string
	}{
		{name: "karakeep", dir: "karakeep"},
		{name: "kanboard", dir: "kanboard"},
		{name: "miniflux", dir: "miniflux"},
		{name: "memos", dir: "memos"},
		{name: "trilium", dir: "trilium"},
		{name: "fireflyiii", dir: "fireflyiii"},
		{name: "gitea", dir: "gitea"},
		{name: "github", dir: "github"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := fs.Stat(skills.FS, tt.dir+"/SKILL.md")
			require.NoError(t, err)
			_, err = fs.Stat(skills.FS, tt.dir+"/references/cli.md")
			require.NoError(t, err)
		})
	}
}
