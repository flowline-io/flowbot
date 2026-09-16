package skills_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/docs/skills"
	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestEmbeddedSkillTrees(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dir        string
		extraPaths []string
	}{
		{name: "karakeep", dir: "karakeep"},
		{name: "kanboard", dir: "kanboard"},
		{name: "miniflux", dir: "miniflux"},
		{name: "memos", dir: "memos"},
		{name: "trilium", dir: "trilium"},
		{name: "fireflyiii", dir: "fireflyiii"},
		{name: "transmission", dir: "transmission"},
		{name: "nocodb", dir: "nocodb"},
		{name: "devops", dir: "devops"},
		{name: "gitea", dir: "gitea"},
		{name: "github", dir: "github"},
		{
			name: "workflow",
			dir:  "workflow",
			extraPaths: []string{
				"references/steps.md",
				"examples/echo_mapper.yaml",
				"examples/save_and_track.yaml",
				"examples/parallel_example.yaml",
			},
		},
		{
			name: "pipeline",
			dir:  "pipeline",
			extraPaths: []string{
				"references/steps.md",
				"references/schema.md",
				"examples/event_notify.yaml",
				"examples/cron_cleanup.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := fs.Stat(skills.FS, tt.dir+"/SKILL.md")
			require.NoError(t, err)
			_, err = fs.Stat(skills.FS, tt.dir+"/references/cli.md")
			require.NoError(t, err)
			for _, rel := range tt.extraPaths {
				_, err = fs.Stat(skills.FS, tt.dir+"/"+rel)
				require.NoError(t, err)
			}
		})
	}
}

func TestEmbeddedSkillFrontmatter(t *testing.T) {
	t.Parallel()
	err := fs.WalkDir(skills.FS, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || path.Base(p) != "SKILL.md" {
			return nil
		}
		raw, err := fs.ReadFile(skills.FS, p)
		require.NoError(t, err)
		fm, body, err := validate.ParseSkillMarkdown(string(raw))
		require.NoError(t, err, p)
		require.NotEmpty(t, strings.TrimSpace(body), p)
		dirName := path.Base(path.Dir(p))
		require.NoError(t, validate.SkillDocument(fm.Name, fm.Description, fm.Compatibility, dirName), p)
		return nil
	})
	require.NoError(t, err)
}

// TestSkillsRefValidate runs the official skills-ref CLI when SKILLS_REF=1 (CI).
func TestSkillsRefValidate(t *testing.T) {
	if os.Getenv("SKILLS_REF") == "" {
		t.Skip("set SKILLS_REF=1 to run skills-ref validate")
	}
	root := "."
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	npx, err := exec.LookPath("npx")
	require.NoError(t, err, "npx required when SKILLS_REF=1")

	var checked int
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		skillDir := filepath.Join(root, ent.Name())
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
			continue
		}
		checked++
		cmd := exec.Command(npx, "--yes", "skills-ref@0.1.5", "validate", skillDir)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "skills-ref validate %s\n%s", skillDir, out)
	}
	require.Positive(t, checked, "expected at least one skill directory")
}
