package chatagent

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestCollectSkillAuxFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		files     map[string]string
		skillDir  string
		wantPaths []string
		wantErr   bool
	}{
		{
			name: "references md and examples yaml",
			files: map[string]string{
				"workflow/SKILL.md":                     "---\nname: workflow\ndescription: x\n---\n\nBody\n",
				"workflow/references/cli.md":            "# cli\n",
				"workflow/references/steps.md":          "# steps\n",
				"workflow/examples/echo_mapper.yaml":    "name: echo\n",
				"workflow/examples/save_and_track.yaml": "name: save\n",
			},
			skillDir: "workflow",
			wantPaths: []string{
				"examples/echo_mapper.yaml",
				"examples/save_and_track.yaml",
				"references/cli.md",
				"references/steps.md",
			},
		},
		{
			name: "skips skill md and non text aux",
			files: map[string]string{
				"demo/SKILL.md":              "---\nname: demo\ndescription: x\n---\n\nBody\n",
				"demo/references/cli.md":     "# cli\n",
				"demo/examples/note.txt":     "plain\n",
				"demo/scripts/run.sh":        "#!/bin/sh\n",
				"demo/references/ignore.bin": "\x00\x01",
			},
			skillDir:  "demo",
			wantPaths: []string{"references/cli.md"},
		},
		{
			name: "missing optional dirs is ok",
			files: map[string]string{
				"bare/SKILL.md": "---\nname: bare\ndescription: x\n---\n\nBody\n",
			},
			skillDir:  "bare",
			wantPaths: []string{},
		},
		{
			name: "yml extension accepted under examples",
			files: map[string]string{
				"w/SKILL.md":            "---\nname: w\ndescription: x\n---\n\nBody\n",
				"w/examples/sample.yml": "name: sample\n",
			},
			skillDir:  "w",
			wantPaths: []string{"examples/sample.yml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{}
			for p, body := range tt.files {
				fsys[p] = &fstest.MapFile{Data: []byte(body)}
			}
			got, err := collectSkillAuxFiles(fsys, tt.skillDir)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			paths := make([]string, 0, len(got))
			for _, f := range got {
				paths = append(paths, f.RelPath)
			}
			require.Equal(t, tt.wantPaths, paths)
		})
	}
}

func TestIsBundledSkillAuxFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rel  string
		want bool
	}{
		{name: "references md", rel: "references/cli.md", want: true},
		{name: "examples yaml", rel: "examples/echo_mapper.yaml", want: true},
		{name: "examples yml", rel: "examples/x.yml", want: true},
		{name: "skill md skipped", rel: "SKILL.md", want: false},
		{name: "txt skipped", rel: "examples/note.txt", want: false},
		{name: "shell skipped", rel: "scripts/run.sh", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isBundledSkillAuxFile(tt.rel))
		})
	}
}
