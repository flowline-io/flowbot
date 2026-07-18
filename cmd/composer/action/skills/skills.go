// Package skills generates SKILL.md files for CLI-invokable capabilities.
// Operations and flags are extracted dynamically from cmd/cli/command code.
// Output follows the Agent Skills open standard (agentskills.io): lean SKILL.md
// with progressive disclosure into references/cli.md.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/flowline-io/flowbot/cmd/cli/command"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// maxDescriptionLen is the agentskills.io limit for the description frontmatter field.
const maxDescriptionLen = 1024

// skillTemplate is the lean SKILL.md body (instructions + workflows).
// Full CLI reference lives in references/cli.md for progressive disclosure.
const skillTemplate = `---
name: {{.Name}}
description: >-
  {{.TriggerDescription}}
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: {{.Name}}
  cli_root: {{.CLIRoot}}
---

# {{.Title}}

Use ` + "`" + `flowbot {{.CLIRoot}}` + "`" + ` for capability ` + "`" + `{{.Name}}` + "`" + `. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: ` + "`" + `flowbot login` + "`" + `
2. Set server via ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` or ` + "`" + `--server-url` + "`" + `; optional ` + "`" + `--profile` + "`" + `, ` + "`" + `--debug` + "`" + ` / ` + "`" + `-d` + "`" + `
3. Prefer ` + "`" + `-o json` + "`" + ` when parsing results programmatically

## Workflows
{{- range .Workflows}}

### {{.Title}}

{{.Description}}
{{- range .Steps}}
{{.Step}}. {{if .Command}}` + "`" + `{{.Command}}` + "`" + `{{else}}{{.Note}}{{end}}
{{- end}}
{{- end}}

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | ` + "`" + `flowbot login` + "`" + ` |
| server URL is required | set ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` or pass ` + "`" + `--server-url` + "`" + ` |
| empty results | confirm server health and capability access scopes |
`

// cliReferenceTemplate is the on-demand CLI command reference.
const cliReferenceTemplate = `# {{.Title}} CLI reference

Capability ` + "`" + `{{.Name}}` + "`" + `. Root command: ` + "`" + `flowbot {{.CLIRoot}}` + "`" + `.

Global flags: ` + "`" + `--server-url` + "`" + `, ` + "`" + `--profile` + "`" + `, ` + "`" + `--debug` + "`" + ` / ` + "`" + `-d` + "`" + `. Most commands accept ` + "`" + `-o table|json` + "`" + ` (omitted below).

## Commands
{{- range .Operations}}

### {{.Title}}

` + "`" + `{{.CLI}}` + "`" + `
{{- if .Description}}

{{.Description}}
{{- end}}
{{- if .Flags}}

{{formatFlags .Flags}}
{{- end}}
{{- end}}
`

// flagSpec describes a CLI flag extracted from pflag.Flag.
type flagSpec struct {
	Name        string
	Shorthand   string
	Type        string
	Required    bool
	Description string
}

// opSpec describes a single CLI operation extracted from a *cobra.Command leaf.
type opSpec struct {
	Title       string
	CLI         string
	Description string
	Flags       []flagSpec
}

// workflowStep is a single step in a workflow.
// Set Command for a CLI invocation (rendered in backticks) or Note for prose.
type workflowStep struct {
	Step    int
	Command string
	Note    string
}

// workflowSpec describes a multi-step workflow.
type workflowSpec struct {
	Title       string
	Description string
	Steps       []workflowStep
}

// metaSpec holds contextual information not derivable from CLI command structs.
// Name must be the hub.CapabilityType string (provider ID), not the CLI domain name.
type metaSpec struct {
	Name        string
	Title       string
	Description string
	Keywords    string
	Workflows   []workflowSpec
	CommandFn   func() *cobra.Command
}

// metaSpecs maps each CLI-invokable capability to its skill metadata.
// Skill Name equals hub.CapabilityType; CLI paths still use domain commands
// (e.g. capability "karakeep" is invoked as "flowbot bookmark ...").
var metaSpecs = []metaSpec{
	{
		Name:        string(hub.CapKarakeep),
		Title:       "Karakeep",
		CommandFn:   command.BookmarkCommand,
		Description: "Create, list, search, archive, and delete bookmarks via flowbot bookmark.",
		Keywords:    "bookmarks, karakeep, saved URLs, reading list, link archiving, web clippings",
		Workflows: []workflowSpec{
			{
				Title:       "Save a URL from a chat message",
				Description: "When a user shares a URL they want to save:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot bookmark check-url -u <url>"},
					{Step: 2, Command: "flowbot bookmark create -u <url>"},
					{Step: 3, Note: "Report back with the bookmark details including the assigned ID."},
				},
			},
			{
				Title:       "Find and review bookmarks",
				Description: "When a user wants to find previously saved content:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot bookmark search -q \"<keywords>\" --limit 10"},
					{Step: 2, Command: "flowbot bookmark get <id>"},
					{Step: 3, Note: "Present the bookmark details to the user."},
				},
			},
		},
	},
	{
		Name:        string(hub.CapKanboard),
		Title:       "Kanboard",
		CommandFn:   command.KanbanCommand,
		Description: "Manage kanban boards, tasks, subtasks, timers, tags, and metadata via flowbot kanban.",
		Keywords:    "kanban, kanboard, tasks, todo, subtasks, time tracking, board columns, moving cards",
		Workflows: []workflowSpec{
			{
				Title:       "Create a task with subtasks",
				Description: "When a user wants to create a well-structured task:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot kanban column list -p 1"},
					{Step: 2, Command: "flowbot kanban create -t \"<task title>\" -d \"<description>\" -p 1 -c <column_id>"},
					{Step: 3, Command: "flowbot kanban subtask create <task_id> -t \"<subtask 1>\" -e <minutes>"},
					{Step: 4, Command: "flowbot kanban subtask create <task_id> -t \"<subtask 2>\" -e <minutes>"},
				},
			},
			{
				Title:       "Review and triage tasks",
				Description: "When reviewing the current board state:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot kanban list -s active"},
					{Step: 2, Command: "flowbot kanban get <task_id>"},
					{Step: 3, Command: "flowbot kanban subtask list <task_id>"},
					{Step: 4, Note: "Summarize task status, subtask completion, and suggest next actions."},
				},
			},
		},
	},
	{
		Name:        string(hub.CapMiniflux),
		Title:       "Miniflux",
		CommandFn:   command.ReaderCommand,
		Description: "Subscribe to RSS/Atom feeds and manage entries via flowbot reader.",
		Keywords:    "RSS, Atom, miniflux, feed reader, unread entries, starring articles, feed subscriptions",
		Workflows: []workflowSpec{
			{
				Title:       "Subscribe to a new feed",
				Description: "When a user shares a blog or feed URL they want to follow:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot reader create -u <feed_url>"},
					{Step: 2, Command: "flowbot reader refresh <feed_id>"},
					{Step: 3, Command: "flowbot reader feed-entries <feed_id> -n 5"},
					{Step: 4, Note: "Report the latest entries to the user."},
				},
			},
			{
				Title:       "Catch up on unread entries",
				Description: "When a user wants to see what's new across all feeds:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot reader entries -s unread -n 20"},
					{Step: 2, Note: "Present the entries in a readable format."},
					{Step: 3, Note: "If the user wants to mark as read, run: flowbot reader update-entries -i <ids> -s read"},
				},
			},
		},
	},
	{
		Name:        string(hub.CapMemos),
		Title:       "Memos",
		CommandFn:   command.MemoCommand,
		Description: "Create, list, update, and delete memos via flowbot memo.",
		Keywords:    "memos, memo notes, scratchpad, quick notes, jotting",
		Workflows: []workflowSpec{
			{
				Title:       "Capture a quick note",
				Description: "When a user wants to save a short memo:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot memo create -c \"<content>\""},
					{Step: 2, Note: "Report back with the memo name."},
				},
			},
			{
				Title:       "Review recent memos",
				Description: "When a user wants to browse or open memos:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot memo list --limit 20"},
					{Step: 2, Command: "flowbot memo get <name>"},
					{Step: 3, Note: "Present the memo content to the user."},
				},
			},
		},
	},
	{
		Name:        string(hub.CapTrilium),
		Title:       "Trilium",
		CommandFn:   command.TriliumCommand,
		Description: "Create, list, search, update, and delete trilium notes via flowbot trilium.",
		Keywords:    "trilium, notes, knowledge base, note tree, personal wiki",
		Workflows: []workflowSpec{
			{
				Title:       "Create a note under a parent",
				Description: "When a user wants to add a new trilium note:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot trilium create -t \"<title>\" -c \"<content>\" -p <parent_note_id>"},
					{Step: 2, Note: "Report back with the note ID."},
				},
			},
			{
				Title:       "Find and read a note",
				Description: "When a user wants to search and open trilium notes:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot trilium search -q \"<keywords>\""},
					{Step: 2, Command: "flowbot trilium get <id>"},
					{Step: 3, Command: "flowbot trilium content get <id>"},
					{Step: 4, Note: "Present the note content to the user."},
				},
			},
		},
	},
	{
		Name:        string(hub.CapFireflyiii),
		Title:       "Firefly III",
		CommandFn:   command.FireflyiiiCommand,
		Description: "Create Firefly III transactions and inspect instance health via flowbot fireflyiii.",
		Keywords:    "fireflyiii, firefly, finance, transactions, expenses, budgeting, accounting",
		Workflows: []workflowSpec{
			{
				Title:       "Record an expense",
				Description: "When a user wants to log a withdrawal or purchase:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot fireflyiii create -t withdrawal --date <YYYY-MM-DD> -a <amount> -m \"<description>\" --source-name \"<account>\" --destination-name \"<payee>\""},
					{Step: 2, Note: "Report back with the transaction ID. Source and destination must each use --*-id or --*-name."},
				},
			},
			{
				Title:       "Check Firefly III connectivity",
				Description: "When verifying the finance backend:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot fireflyiii health"},
					{Step: 2, Command: "flowbot fireflyiii about"},
					{Step: 3, Note: "Summarize version and health status."},
				},
			},
		},
	},
	{
		Name:        string(hub.CapGitea),
		Title:       "Gitea",
		CommandFn:   command.ForgeCommand,
		Description: "Inspect forge users, repos, issues, diffs, and files via flowbot forge.",
		Keywords:    "gitea, forge, repositories, issues, commit diffs, source files, code review",
		Workflows: []workflowSpec{
			{
				Title:       "Inspect a repository issue",
				Description: "When a user asks about a forge issue:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot forge issues <owner> -s open -n 10"},
					{Step: 2, Command: "flowbot forge issue <owner> <repo> <index>"},
					{Step: 3, Note: "Summarize the issue for the user."},
				},
			},
			{
				Title:       "Review a commit change",
				Description: "When a user wants to inspect a commit:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot forge diff <owner> <repo> <commit-id>"},
					{Step: 2, Command: "flowbot forge file <owner> <repo> <commit-id> <file-path>"},
					{Step: 3, Note: "Explain the relevant changes."},
				},
			},
		},
	},
	{
		Name:        string(hub.CapGithub),
		Title:       "GitHub",
		CommandFn:   command.GithubCommand,
		Description: "Inspect GitHub users, repos, issues, notifications, releases, diffs, and files via flowbot github.",
		Keywords:    "github, repositories, issues, notifications, releases, pull requests, commit diffs",
		Workflows: []workflowSpec{
			{
				Title:       "Triage open issues",
				Description: "When a user wants to review GitHub issues:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot github issues <owner> -s open -n 10"},
					{Step: 2, Command: "flowbot github issue <owner> <repo> <number>"},
					{Step: 3, Note: "Summarize the issue for the user."},
				},
			},
			{
				Title:       "Check notifications and releases",
				Description: "When a user wants an activity overview:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot github notifications -n 20"},
					{Step: 2, Command: "flowbot github releases <owner> <repo> -n 5"},
					{Step: 3, Note: "Present a concise summary."},
				},
			},
		},
	},
}

// extractOperations walks a *cobra.Command tree recursively and returns all
// leaf-level operations with their full CLI path, flags, and metadata.
func extractOperations(cmd *cobra.Command, pathPrefix string) []opSpec {
	if skipCommand(cmd) {
		return nil
	}

	var ops []opSpec

	if !cmd.HasSubCommands() {
		if cmd.Name() == "" {
			return nil
		}
		flags := extractFlags(cmd.Flags())
		cliPath := pathPrefix
		if cliPath == "" {
			cliPath = cmd.Name()
		}
		argsUsage := extractArgsFromUse(cmd.Use)
		op := opSpec{
			Title:       firstNonEmpty(cmd.Short, cmd.Name()),
			CLI:         buildCLIString(cliPath, argsUsage, flags),
			Description: strings.TrimSpace(cmd.Long),
			Flags:       flags,
		}
		ops = append(ops, op)
	} else {
		for _, sub := range cmd.Commands() {
			if skipCommand(sub) {
				continue
			}
			subPath := pathPrefix
			if subPath == "" {
				subPath = cmd.Name() + " " + sub.Name()
			} else {
				subPath += " " + sub.Name()
			}
			ops = append(ops, extractOperations(sub, subPath)...)
		}
	}

	return ops
}

// skipCommand reports whether cmd should be omitted from generated skill docs.
func skipCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	if cmd.Hidden || cmd.Name() == "help" || cmd.IsAdditionalHelpTopicCommand() {
		return true
	}
	return false
}

// extractFlags converts *pflag.FlagSet into []flagSpec.
// Flag metadata is extracted from pflag attributes.
// The common --output flag is skipped since it is documented globally.
func extractFlags(flagSet *pflag.FlagSet) []flagSpec {
	var result []flagSpec
	flagSet.VisitAll(func(f *pflag.Flag) {
		if f.Name == "output" {
			return
		}

		required := false
		if f.Annotations != nil {
			if v, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok && len(v) > 0 && v[0] == "true" {
				required = true
			}
		}

		result = append(result, flagSpec{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			Type:        f.Value.Type(),
			Required:    required,
			Description: f.Usage,
		})
	})
	return result
}

// extractArgsFromUse extracts positional argument placeholders from cobra's Use string.
// Keeps required (<...>), optional ([...]), and variadic (<...>)... tokens; skips [flags].
// Returns a space-separated string like "<id>", "<task_id> [name]", or "<task_id> <name=value>...".
func extractArgsFromUse(use string) string {
	parts := strings.Fields(use)
	if len(parts) <= 1 {
		return ""
	}
	var args []string
	for _, p := range parts[1:] {
		if p == "[flags]" {
			continue
		}
		if strings.HasPrefix(p, "<") || strings.HasPrefix(p, "[") {
			args = append(args, p)
		}
	}
	return strings.Join(args, " ")
}

// buildCLIString constructs the CLI command reference string.
func buildCLIString(path string, argsUsage string, flags []flagSpec) string {
	var cmd strings.Builder
	_, _ = cmd.WriteString("flowbot " + path)
	if argsUsage != "" {
		_, _ = cmd.WriteString(" " + argsUsage)
	}
	for _, fl := range flags {
		if fl.Required {
			_, _ = cmd.WriteString(" --" + fl.Name)
			if fl.Type != "bool" {
				_, _ = cmd.WriteString(" <" + fl.Name + ">")
			}
		}
	}
	hasOptional := false
	for _, fl := range flags {
		if !fl.Required {
			hasOptional = true
			break
		}
	}
	if hasOptional {
		_, _ = cmd.WriteString(" [flags]")
	}
	return cmd.String()
}

// splitArgTokens splits an argsUsage string into display tokens without altering brackets.
func splitArgTokens(argsUsage string) []string {
	if argsUsage == "" {
		return nil
	}
	parts := strings.Fields(argsUsage)
	if len(parts) == 0 {
		return nil
	}
	return parts
}

// firstNonEmpty returns a if non-empty, otherwise b.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// buildTriggerDescription builds the agentskills description (WHAT + WHEN), capped at maxDescriptionLen runes.
func buildTriggerDescription(what, keywords string) string {
	what = strings.TrimSpace(what)
	keywords = strings.TrimSpace(keywords)
	var b strings.Builder
	_, _ = b.WriteString(what)
	if keywords != "" {
		if !strings.HasSuffix(what, ".") {
			_, _ = b.WriteString(".")
		}
		_, _ = b.WriteString(" Use when the user mentions ")
		_, _ = b.WriteString(keywords)
		_, _ = b.WriteString(".")
	}
	desc := b.String()
	runes := []rune(desc)
	if len(runes) <= maxDescriptionLen {
		return desc
	}
	return string(runes[:maxDescriptionLen-3]) + "..."
}

// formatFlagsCompact renders flags as a single dense line for reference docs.
func formatFlagsCompact(flags []flagSpec) string {
	if len(flags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(flags))
	for _, f := range flags {
		var s strings.Builder
		_, _ = s.WriteString("`--")
		_, _ = s.WriteString(f.Name)
		_, _ = s.WriteString("`")
		if f.Shorthand != "" {
			_, _ = s.WriteString(" (`-")
			_, _ = s.WriteString(f.Shorthand)
			_, _ = s.WriteString("`)")
		}
		_, _ = s.WriteString(" ")
		_, _ = s.WriteString(f.Type)
		if f.Required {
			_, _ = s.WriteString(", required")
		}
		if f.Description != "" {
			_, _ = s.WriteString(" — ")
			_, _ = s.WriteString(f.Description)
		}
		parts = append(parts, s.String())
	}
	return "Flags: " + strings.Join(parts, "; ")
}

// skillData is the template context shared by SKILL.md and references/cli.md.
type skillData struct {
	Name               string
	Title              string
	CLIRoot            string
	TriggerDescription string
	Operations         []opSpec
	Workflows          []workflowSpec
}

// newTemplateFuncs returns template helpers used by skill generators.
func newTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatFlags": formatFlagsCompact,
	}
}

// generateSkill writes SKILL.md and references/cli.md for one capability.
func generateSkill(meta metaSpec, outputDir string, skillTmpl, refTmpl *template.Template) error {
	dirPath := filepath.Join(outputDir, meta.Name)
	if err := os.MkdirAll(filepath.Join(dirPath, "references"), 0o750); err != nil {
		return fmt.Errorf("create directory %s: %w", dirPath, err)
	}

	rootCmd := meta.CommandFn()
	cliRoot := rootCmd.Name()
	data := skillData{
		Name:               meta.Name,
		Title:              meta.Title,
		CLIRoot:            cliRoot,
		TriggerDescription: buildTriggerDescription(meta.Description, meta.Keywords),
		Operations:         extractOperations(rootCmd, cliRoot),
		Workflows:          meta.Workflows,
	}

	skillPath := filepath.Join(dirPath, "SKILL.md")
	if err := executeTemplateFile(skillTmpl, skillPath, data); err != nil {
		return fmt.Errorf("write %s: %w", skillPath, err)
	}

	refPath := filepath.Join(dirPath, "references", "cli.md")
	if err := executeTemplateFile(refTmpl, refPath, data); err != nil {
		return fmt.Errorf("write %s: %w", refPath, err)
	}

	_, _ = fmt.Printf("  generated: %s\n", skillPath)
	_, _ = fmt.Printf("  generated: %s\n", refPath)
	return nil
}

// executeTemplateFile creates path and executes tmpl with data.
func executeTemplateFile(tmpl *template.Template, path string, data skillData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	execErr := tmpl.Execute(f, data)
	closeErr := f.Close()
	if execErr != nil {
		return execErr
	}
	return closeErr
}

// SkillsAction generates SKILL.md files for all CLI-invokable capabilities.
func SkillsAction(cmd *cobra.Command, _ []string) error {
	outputDir, err := cmd.Flags().GetString("output")
	if err != nil {
		return fmt.Errorf("get output flag: %w", err)
	}
	if outputDir == "" {
		outputDir = "./docs/skills"
	}

	funcs := newTemplateFuncs()
	skillTmpl, err := template.New("skill").Funcs(funcs).Parse(skillTemplate)
	if err != nil {
		return fmt.Errorf("parse skill template: %w", err)
	}
	refTmpl, err := template.New("cli_ref").Funcs(funcs).Parse(cliReferenceTemplate)
	if err != nil {
		return fmt.Errorf("parse cli reference template: %w", err)
	}

	for _, meta := range metaSpecs {
		if err := generateSkill(meta, outputDir, skillTmpl, refTmpl); err != nil {
			return err
		}
	}

	_, _ = fmt.Println("SKILL.md files generated successfully")
	return nil
}
