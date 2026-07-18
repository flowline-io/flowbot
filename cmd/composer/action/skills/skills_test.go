package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/cmd/cli/command"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestExtractArgsFromUse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		use  string
		want string
	}{
		{name: "no args", use: "list", want: ""},
		{name: "one required arg", use: "get <id>", want: "<id>"},
		{name: "two required args", use: "update <task_id> <subtask_id>", want: "<task_id> <subtask_id>"},
		{name: "optional arg kept", use: "get <task_id> [name]", want: "<task_id> [name]"},
		{name: "variadic arg kept", use: "set <task_id> <name=value>...", want: "<task_id> <name=value>..."},
		{name: "flags placeholder skipped", use: "list [flags]", want: ""},
		{name: "plain words skipped", use: "create something", want: ""},
		{name: "feed entries id", use: "feed-entries <feed-id>", want: "<feed-id>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractArgsFromUse(tt.use)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSplitArgTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    []string
		wantNil bool
	}{
		{name: "empty", input: "", wantNil: true},
		{name: "one required", input: "<id>", want: []string{"<id>"}},
		{name: "optional preserved", input: "<task_id> [name]", want: []string{"<task_id>", "[name]"}},
		{name: "variadic preserved", input: "<task_id> <name=value>...", want: []string{"<task_id>", "<name=value>..."}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitArgTokens(tt.input)
			if tt.wantNil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildCLIString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		path      string
		argsUsage string
		flags     []flagSpec
		want      string
	}{
		{
			name:      "no args no flags",
			path:      "bookmark list",
			argsUsage: "",
			flags:     nil,
			want:      "flowbot bookmark list",
		},
		{
			name:      "with args",
			path:      "bookmark get",
			argsUsage: "<id>",
			flags:     nil,
			want:      "flowbot bookmark get <id>",
		},
		{
			name:      "optional and variadic args",
			path:      "kanban metadata get",
			argsUsage: "<task_id> [name]",
			flags:     nil,
			want:      "flowbot kanban metadata get <task_id> [name]",
		},
		{
			name:      "mixed required and optional",
			path:      "bookmark create",
			argsUsage: "",
			flags: []flagSpec{
				{Name: "url", Shorthand: "u", Type: "string", Required: true, Description: "Bookmark URL"},
				{Name: "output", Shorthand: "o", Type: "string", Required: false, Description: "Format"},
			},
			want: "flowbot bookmark create --url <url> [flags]",
		},
		{
			name:      "with required bool flag and optional",
			path:      "archive",
			argsUsage: "<id>",
			flags: []flagSpec{
				{Name: "force", Shorthand: "f", Type: "bool", Required: true, Description: "Force"},
				{Name: "output", Shorthand: "o", Type: "string", Required: false, Description: "Format"},
			},
			want: "flowbot archive <id> --force [flags]",
		},
		{
			name: "all required no optional",
			path: "kanban set",
			flags: []flagSpec{
				{Name: "name", Type: "string", Required: true},
			},
			want: "flowbot kanban set --name <name>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildCLIString(tt.path, tt.argsUsage, tt.flags)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{name: "prefers first", a: "hello", b: "fallback", want: "hello"},
		{name: "falls back when empty", a: "", b: "fallback", want: "fallback"},
		{name: "both empty", a: "", b: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, firstNonEmpty(tt.a, tt.b))
		})
	}
}

func TestExtractFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		setup         func(*pflag.FlagSet)
		wantNames     []string
		wantRequired  map[string]bool
		wantShorthand map[string]string
		wantType      map[string]string
	}{
		{
			name: "skips output and keeps others",
			setup: func(fs *pflag.FlagSet) {
				fs.StringP("output", "o", "table", "Output format")
				fs.StringP("url", "u", "", "Bookmark URL")
				fs.String("name", "", "Tag name")
				fs.BoolP("force", "f", false, "Force")
			},
			wantNames:     []string{"url", "name", "force"},
			wantRequired:  map[string]bool{"url": false, "name": false, "force": false},
			wantShorthand: map[string]string{"url": "u", "force": "f"},
			wantType:      map[string]string{"url": "string", "name": "string", "force": "bool"},
		},
		{
			name: "marks MarkFlagRequired as required",
			setup: func(fs *pflag.FlagSet) {
				fs.StringP("url", "u", "", "Bookmark URL")
				fs.StringP("query", "q", "", "Search query")
				_ = cobra.MarkFlagRequired(fs, "url")
			},
			wantNames:    []string{"url", "query"},
			wantRequired: map[string]bool{"url": true, "query": false},
			wantType:     map[string]string{"url": "string", "query": "string"},
		},
		{
			name: "default flags are not required",
			setup: func(fs *pflag.FlagSet) {
				fs.StringP("title", "t", "", "Title")
				fs.IntP("limit", "n", 20, "Limit")
			},
			wantNames:    []string{"title", "limit"},
			wantRequired: map[string]bool{"title": false, "limit": false},
			wantType:     map[string]string{"title": "string", "limit": "int"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			tt.setup(fs)

			result := extractFlags(fs)
			gotNames := make([]string, 0, len(result))
			byName := make(map[string]flagSpec, len(result))
			for _, f := range result {
				gotNames = append(gotNames, f.Name)
				byName[f.Name] = f
			}
			require.ElementsMatch(t, tt.wantNames, gotNames)

			for name, wantReq := range tt.wantRequired {
				f, ok := byName[name]
				require.True(t, ok, "missing flag %s", name)
				require.Equal(t, wantReq, f.Required, "flag %s required", name)
			}
			for name, wantSH := range tt.wantShorthand {
				require.Equal(t, wantSH, byName[name].Shorthand, "flag %s shorthand", name)
			}
			for name, wantType := range tt.wantType {
				require.Equal(t, wantType, byName[name].Type, "flag %s type", name)
			}
		})
	}
}

func TestExtractOperations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		build     func() *cobra.Command
		wantCLI   []string
		checkFlag map[string]map[string]bool // cli path -> flag name -> required
	}{
		{
			name: "nested path and optional args from kanban metadata",
			build: func() *cobra.Command {
				return command.KanbanCommand()
			},
			wantCLI: []string{
				"flowbot kanban metadata get <task_id> [name]",
				"flowbot kanban metadata set <task_id> <name=value>...",
			},
		},
		{
			name: "bookmark create marks url required",
			build: func() *cobra.Command {
				return command.BookmarkCommand()
			},
			wantCLI: []string{"flowbot bookmark create --url <url>"},
			checkFlag: map[string]map[string]bool{
				"flowbot bookmark create --url <url>": {"url": true},
			},
		},
		{
			name: "skips hidden and help commands",
			build: func() *cobra.Command {
				root := &cobra.Command{Use: "demo", Short: "Demo"}
				root.AddCommand(&cobra.Command{
					Use:   "visible",
					Short: "Visible leaf",
					RunE:  func(*cobra.Command, []string) error { return nil },
				})
				root.AddCommand(&cobra.Command{
					Use:    "secret",
					Short:  "Hidden leaf",
					Hidden: true,
					RunE:   func(*cobra.Command, []string) error { return nil },
				})
				root.InitDefaultHelpCmd()
				return root
			},
			wantCLI: []string{"flowbot demo visible"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := tt.build()
			ops := extractOperations(root, root.Name())

			byCLI := make(map[string]opSpec, len(ops))
			gotCLIs := make([]string, 0, len(ops))
			for _, op := range ops {
				byCLI[op.CLI] = op
				gotCLIs = append(gotCLIs, op.CLI)
			}

			for _, want := range tt.wantCLI {
				require.Contains(t, gotCLIs, want)
			}
			for cli, flags := range tt.checkFlag {
				op, ok := byCLI[cli]
				require.True(t, ok, "missing op %s", cli)
				flagByName := make(map[string]flagSpec, len(op.Flags))
				for _, f := range op.Flags {
					flagByName[f.Name] = f
				}
				for name, wantReq := range flags {
					f, ok := flagByName[name]
					require.True(t, ok, "missing flag %s on %s", name, cli)
					require.Equal(t, wantReq, f.Required)
				}
			}
			if tt.name == "skips hidden and help commands" {
				require.Len(t, ops, 1)
				require.Equal(t, "flowbot demo visible", ops[0].CLI)
			}
		})
	}
}

func TestSkipCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{name: "nil", cmd: nil, want: true},
		{name: "runnable leaf", cmd: &cobra.Command{Use: "list", RunE: func(*cobra.Command, []string) error { return nil }}, want: false},
		{name: "hidden", cmd: &cobra.Command{Use: "secret", Hidden: true}, want: true},
		{name: "help", cmd: &cobra.Command{Use: "help"}, want: true},
		{name: "non-runnable help topic", cmd: &cobra.Command{Use: "topic"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, skipCommand(tt.cmd))
		})
	}
}

func TestBuildTriggerDescription(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		what     string
		keywords string
		want     string
	}{
		{
			name:     "what and when",
			what:     "Create and list bookmarks via flowbot bookmark.",
			keywords: "bookmarks, karakeep",
			want:     "Create and list bookmarks via flowbot bookmark. Use when the user mentions bookmarks, karakeep.",
		},
		{
			name:     "adds period before when",
			what:     "Manage memos via flowbot memo",
			keywords: "memos, notes",
			want:     "Manage memos via flowbot memo. Use when the user mentions memos, notes.",
		},
		{
			name:     "what only",
			what:     "Do the thing.",
			keywords: "",
			want:     "Do the thing.",
		},
		{
			name:     "truncates over limit by runes",
			what:     strings.Repeat("你", maxDescriptionLen),
			keywords: "触发",
			want:     strings.Repeat("你", maxDescriptionLen-3) + "...",
		},
		{
			name:     "truncates ascii over limit",
			what:     strings.Repeat("a", maxDescriptionLen),
			keywords: "trigger",
			want:     strings.Repeat("a", maxDescriptionLen-3) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildTriggerDescription(tt.what, tt.keywords)
			require.Equal(t, tt.want, got)
			require.LessOrEqual(t, len([]rune(got)), maxDescriptionLen)
		})
	}
}

func TestFormatFlagsCompact(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		flags []flagSpec
		want  string
	}{
		{name: "empty", flags: nil, want: ""},
		{
			name: "required with shorthand",
			flags: []flagSpec{
				{Name: "url", Shorthand: "u", Type: "string", Required: true, Description: "Bookmark URL"},
			},
			want: "Flags: `--url` (`-u`) string, required — Bookmark URL",
		},
		{
			name: "multiple flags",
			flags: []flagSpec{
				{Name: "limit", Shorthand: "n", Type: "int", Required: false, Description: "Max items"},
				{Name: "force", Type: "bool", Required: true, Description: "Force"},
			},
			want: "Flags: `--limit` (`-n`) int — Max items; `--force` bool, required — Force",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, formatFlagsCompact(tt.flags))
		})
	}
}

func TestGenerateSkillProgressiveDisclosure(t *testing.T) {
	t.Parallel()

	byName := make(map[string]metaSpec, len(metaSpecs))
	for _, m := range metaSpecs {
		byName[m.Name] = m
	}

	tests := []struct {
		name  string
		skill string
	}{
		{name: "karakeep", skill: "karakeep"},
		{name: "kanboard", skill: "kanboard"},
		{name: "github", skill: "github"},
	}

	funcs := newTemplateFuncs()
	skillTmpl, err := template.New("skill").Funcs(funcs).Parse(skillTemplate)
	require.NoError(t, err)
	refTmpl, err := template.New("cli_ref").Funcs(funcs).Parse(cliReferenceTemplate)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta, ok := byName[tt.skill]
			require.True(t, ok)

			dir := t.TempDir()
			require.NoError(t, generateSkill(meta, dir, skillTmpl, refTmpl))

			skillBody, err := os.ReadFile(filepath.Join(dir, meta.Name, "SKILL.md"))
			require.NoError(t, err)
			skill := string(skillBody)
			require.Contains(t, skill, "name: "+meta.Name)
			require.Contains(t, skill, "Use when the user mentions")
			require.Contains(t, skill, "references/cli.md")
			require.Contains(t, skill, "## Workflows")
			require.NotContains(t, skill, "## Commands")
			require.Less(t, strings.Count(skill, "\n"), 120, "SKILL.md should stay lean")

			refBody, err := os.ReadFile(filepath.Join(dir, meta.Name, "references", "cli.md"))
			require.NoError(t, err)
			ref := string(refBody)
			require.Contains(t, ref, "## Commands")
			require.Contains(t, ref, "flowbot ")
		})
	}
}

func TestMetaSpecsUseCapabilityIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		skill     string
		wantCap   string
		wantTitle string
		wantCLI   string
	}{
		{name: "karakeep", skill: "karakeep", wantCap: string(hub.CapKarakeep), wantTitle: "Karakeep", wantCLI: "bookmark"},
		{name: "kanboard", skill: "kanboard", wantCap: string(hub.CapKanboard), wantTitle: "Kanboard", wantCLI: "kanban"},
		{name: "miniflux", skill: "miniflux", wantCap: string(hub.CapMiniflux), wantTitle: "Miniflux", wantCLI: "reader"},
		{name: "memos", skill: "memos", wantCap: string(hub.CapMemos), wantTitle: "Memos", wantCLI: "memo"},
		{name: "trilium", skill: "trilium", wantCap: string(hub.CapTrilium), wantTitle: "Trilium", wantCLI: "trilium"},
		{name: "fireflyiii", skill: "fireflyiii", wantCap: string(hub.CapFireflyiii), wantTitle: "Firefly III", wantCLI: "fireflyiii"},
		{name: "transmission", skill: "transmission", wantCap: string(hub.CapTransmission), wantTitle: "Transmission", wantCLI: "transmission"},
		{name: "gitea", skill: "gitea", wantCap: string(hub.CapGitea), wantTitle: "Gitea", wantCLI: "forge"},
		{name: "github", skill: "github", wantCap: string(hub.CapGithub), wantTitle: "GitHub", wantCLI: "github"},
	}

	byName := make(map[string]metaSpec, len(metaSpecs))
	for _, m := range metaSpecs {
		byName[m.Name] = m
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, ok := byName[tt.skill]
			require.True(t, ok, "missing metaSpec for skill %q", tt.skill)
			require.Equal(t, tt.wantCap, m.Name)
			require.Equal(t, tt.wantTitle, m.Title)
			root := m.CommandFn()
			require.NotNil(t, root)
			require.Equal(t, tt.wantCLI, root.Name())
		})
	}

	t.Run("every metaSpec name is a known capability id", func(t *testing.T) {
		t.Parallel()
		known := map[string]struct{}{
			string(hub.CapKarakeep):     {},
			string(hub.CapKanboard):     {},
			string(hub.CapMiniflux):     {},
			string(hub.CapMemos):        {},
			string(hub.CapFireflyiii):   {},
			string(hub.CapTransmission): {},
			string(hub.CapGitea):        {},
			string(hub.CapGithub):       {},
			string(hub.CapTrilium):      {},
			string(hub.CapNotify):       {},
			string(hub.CapAgent):        {},
			string(hub.CapExample):      {},
		}
		for _, m := range metaSpecs {
			_, ok := known[m.Name]
			require.True(t, ok, "metaSpec name %q is not a hub.CapabilityType", m.Name)
		}
	})
}
