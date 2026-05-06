// Package skills generates SKILL.md files for CLI-invokable capabilities.
// Operations and flags are extracted dynamically from cmd/cli/command code.
package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/flowline-io/flowbot/cmd/cli/command"
	"github.com/urfave/cli/v3"
)

// skillTemplate is the SKILL.md template following the docs/skill_spec.md format.
const skillTemplate = `---
name: {{.Name}}
description: >
  {{.Description}}
  Make sure to use this skill whenever the user mentions {{.Keywords}}.
---

# Flowbot {{.Title}}

{{.Description}}

## Prerequisites

- The ` + "`" + `flowbot` + "`" + ` CLI must be installed and logged in (` + "`" + `flowbot login` + "`" + `).
- The Flowbot server must be running and reachable.
- Global flags: ` + "`" + `--server-url` + "`" + ` (server address), ` + "`" + `--profile` + "`" + ` (config profile), ` + "`" + `--debug` + "`" + ` (enable debug logging).

## Global Flags Reference

| Flag | Shorthand | Type | Description |
|------|-----------|------|-------------|
| ` + "`" + `--server-url` + "`" + ` | | string | Flowbot server URL (or set ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` env var) |
| ` + "`" + `--profile` + "`" + ` | | string | Configuration profile name |
| ` + "`" + `--debug` + "`" + ` | ` + "`" + `-d` + "`" + ` | bool | Enable debug mode |

## Common Output Options

Most commands support ` + "`" + `--output` + "`" + ` / ` + "`" + `-o` + "`" + ` to choose between ` + "`" + `table` + "`" + ` (default, human-readable) and ` + "`" + `json` + "`" + ` (structured) output.

## Operations
{{- range .Operations}}

### {{.Title}}

**Command:** ` + "`" + `{{.CLI}}` + "`" + `
{{- if .Description}}
{{.Description}}
{{- end}}
{{- if .Args}}

**Positional Arguments:**{{- range .Args}}
- ` + "`" + `<{{.}}>` + "`" + `{{- end}}
{{- end}}
{{- if .Flags}}

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
{{- range .Flags}}
| ` + "`" + `--{{.Name}}` + "`" + ` | {{if .Shorthand}}` + "`" + `-{{.Shorthand}}` + "`" + `{{end}} | {{.Type}} | {{if .Required}}yes{{else}}no{{end}} | {{.Description}} |
{{- end}}
{{- end}}
{{- if .Examples}}

**Examples:**
{{- range .Examples}}
- ` + "`" + `{{.}}` + "`" + `
{{- end}}
{{- end}}

---
{{- end}}

## Common Workflows
{{- range .Workflows}}

### {{.Title}}

{{.Description}}
{{range .Steps}}
{{.Step}}. ` + "`" + `{{.Command}}` + "`" + `
{{- end}}
{{end}}

## Troubleshooting

- **"not logged in"**: Run ` + "`" + `flowbot login` + "`" + ` first.
- **"server URL is required"**: Set ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` env var or use ` + "`" + `--server-url` + "`" + ` flag.
- **Empty results**: Check the server is running and you have access to the requested resources.
`

// flagSpec describes a CLI flag extracted from cli.Flag.
type flagSpec struct {
	Name        string
	Shorthand   string
	Type        string
	Required    bool
	Description string
}

// opSpec describes a single CLI operation extracted from a *cli.Command leaf.
type opSpec struct {
	Title       string
	CLI         string
	Description string
	Args        []string
	Flags       []flagSpec
	Examples    []string
}

// workflowStep is a single step in a workflow.
type workflowStep struct {
	Step    int
	Command string
}

// workflowSpec describes a multi-step workflow.
type workflowSpec struct {
	Title       string
	Description string
	Steps       []workflowStep
}

// metaSpec holds contextual information not derivable from CLI command structs.
type metaSpec struct {
	Name        string
	Title       string
	Description string
	Keywords    string
	Workflows   []workflowSpec
	CommandFn   func() *cli.Command
}

// metaSpecs defines all capabilities with their CLI command factories.
var metaSpecs = []metaSpec{
	{
		Name:     "homelab-bookmark",
		Title:    "Bookmark",
		CommandFn: command.BookmarkCommand,
		Description: "Manage bookmarks via the Flowbot CLI. Create, list, search, archive, and tag bookmarks stored in the Flowbot server.",
		Keywords:    "bookmarks, saving URLs, link collection, web clippings, reading list, tagging URLs, URL archiving, checking saved links",
		Workflows: []workflowSpec{
			{
				Title:       "Save a URL from a chat message",
				Description: "When a user shares a URL they want to save:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot bookmark check-url -u <url>"},
					{Step: 2, Command: "flowbot bookmark create -u <url>"},
					{Step: 3, Command: "Report back with the bookmark details including the assigned ID."},
				},
			},
			{
				Title:       "Find and review bookmarks",
				Description: "When a user wants to find previously saved content:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot bookmark search -q \"<keywords>\" --limit 10"},
					{Step: 2, Command: "flowbot bookmark get <id>"},
					{Step: 3, Command: "Present the bookmark details to the user."},
				},
			},
		},
	},
	{
		Name:     "homelab-kanban",
		Title:    "Kanban",
		CommandFn: command.KanbanCommand,
		Description: "Manage kanban boards and tasks via the Flowbot CLI. Create, update, move, and search tasks. Manage subtasks with time tracking, tags, columns, and metadata.",
		Keywords:    "kanban, task management, project management, kanban board, todo list, task tracking, issue tracking, subtasks, time tracking, moving cards, board columns, task tags",
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
					{Step: 4, Command: "Summarize task status, subtask completion, and suggest next actions."},
				},
			},
		},
	},
	{
		Name:     "homelab-reader",
		Title:    "Reader",
		CommandFn: command.ReaderCommand,
		Description: "Manage RSS and Atom feed subscriptions via the Flowbot CLI. Add feeds, list entries, mark items read/unread, star entries, and manage feed lifecycle.",
		Keywords:    "RSS feeds, RSS reader, feed reader, news feeds, Atom feeds, subscribing to blogs, reading feeds, feed management, marking read, starring articles",
		Workflows: []workflowSpec{
			{
				Title:       "Subscribe to a new feed",
				Description: "When a user shares a blog or feed URL they want to follow:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot reader create -u <feed_url>"},
					{Step: 2, Command: "flowbot reader refresh <feed_id>"},
					{Step: 3, Command: "flowbot reader feed-entries <feed_id> -n 5"},
					{Step: 4, Command: "Report the latest entries to the user."},
				},
			},
			{
				Title:       "Catch up on unread entries",
				Description: "When a user wants to see what's new across all feeds:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot reader entries -s unread -n 20"},
					{Step: 2, Command: "Present the entries in a readable format."},
					{Step: 3, Command: "If the user wants to mark as read: flowbot reader update-entries -i <ids> -s read"},
				},
			},
		},
	},
}

// extractOperations walks a *cli.Command tree recursively and returns all
// leaf-level operations with their full CLI path, flags, and metadata.
func extractOperations(cmd *cli.Command, pathPrefix string) []opSpec {
	var ops []opSpec

	if len(cmd.Commands) == 0 {
		// Leaf command: extract as an operation.
		if cmd.Name == "" {
			return nil
		}
		flags := extractFlags(cmd.Flags)
		cliPath := pathPrefix
		if cliPath == "" {
			cliPath = cmd.Name
		}
		op := opSpec{
			Title:       zeroDefault(cmd.Usage, cmd.Name),
			CLI:         buildCLIString(cliPath, cmd.ArgsUsage, flags),
			Description: strings.TrimSpace(cmd.Description),
			Args:        parseArgs(cmd.ArgsUsage),
			Flags:       flags,
		}
		ops = append(ops, op)
	} else {
		// Container command: recurse into subcommands.
		for _, sub := range cmd.Commands {
			subPath := pathPrefix
			if subPath == "" {
				subPath = cmd.Name + " " + sub.Name
			} else {
				subPath += " " + sub.Name
			}
			ops = append(ops, extractOperations(sub, subPath)...)
		}
	}

	return ops
}

// extractFlags converts []cli.Flag into []flagSpec.
// Flag metadata is extracted via separate interfaces (RequiredFlag, DocGenerationFlag)
// since the base Flag interface does not cover all methods.
// The common --output flag is skipped since it is documented globally.
func extractFlags(flags []cli.Flag) []flagSpec {
	var result []flagSpec
	for _, f := range flags {
		names := f.Names()
		if len(names) == 0 {
			continue
		}
		// Skip the ubiquitous --output flag.
		if names[0] == "output" {
			continue
		}

		shorthand := ""
		if len(names) > 1 {
			shorthand = names[1]
		}

		required := false
		if rf, ok := f.(cli.RequiredFlag); ok {
			required = rf.IsRequired()
		}

		typeName := ""
		usage := ""
		if df, ok := f.(cli.DocGenerationFlag); ok {
			typeName = df.TypeName()
			usage = df.GetUsage()
		}

		result = append(result, flagSpec{
			Name:        names[0],
			Shorthand:   shorthand,
			Type:        typeName,
			Required:    required,
			Description: usage,
		})
	}
	return result
}

// buildCLIString constructs the CLI command reference string.
func buildCLIString(path string, argsUsage string, flags []flagSpec) string {
	cmd := "flowbot " + path
	if argsUsage != "" {
		cmd += " " + argsUsage
	}
	// Append required flags inline.
	for _, fl := range flags {
		if fl.Required {
			cmd += " --" + fl.Name
			if fl.Type != "bool" {
				cmd += " <" + fl.Name + ">"
			}
		}
	}
	// Indicate optional flags exist.
	hasOptional := false
	for _, fl := range flags {
		if !fl.Required {
			hasOptional = true
			break
		}
	}
	if hasOptional {
		cmd += " [flags]"
	}
	return cmd
}

// parseArgs splits an ArgsUsage string like "<id>" or "<task_id> <subtask_id>"
// into individual arg names.
func parseArgs(argsUsage string) []string {
	if argsUsage == "" {
		return nil
	}
	raw := strings.ReplaceAll(argsUsage, "<", "")
	raw = strings.ReplaceAll(raw, ">", "")
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil
	}
	return parts
}

// zeroDefault returns a if non-empty, otherwise b.
func zeroDefault(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// SkillsAction generates SKILL.md files for all CLI-invokable capabilities.
func SkillsAction(_ context.Context, cmd *cli.Command) error {
	outputDir := cmd.String("output")
	if outputDir == "" {
		outputDir = "./docs/skills"
	}

	tmpl, err := template.New("skill").Parse(skillTemplate)
	if err != nil {
		return fmt.Errorf("parse skill template: %w", err)
	}

	for _, meta := range metaSpecs {
		dirPath := filepath.Join(outputDir, meta.Name)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dirPath, err)
		}

		// Extract operations from the live CLI command tree.
		rootCmd := meta.CommandFn()
		operations := extractOperations(rootCmd, rootCmd.Name)

		data := struct {
			Name        string
			Title       string
			Description string
			Keywords    string
			Operations  []opSpec
			Workflows   []workflowSpec
		}{
			Name:        meta.Name,
			Title:       meta.Title,
			Description: meta.Description,
			Keywords:    meta.Keywords,
			Operations:  operations,
			Workflows:   meta.Workflows,
		}

		filePath := filepath.Join(dirPath, "SKILL.md")
		f, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("create file %s: %w", filePath, err)
		}

		if err := tmpl.Execute(f, data); err != nil {
			_ = f.Close()
			return fmt.Errorf("execute template for %s: %w", meta.Name, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close file %s: %w", filePath, err)
		}

		_, _ = fmt.Printf("  generated: %s\n", filePath)
	}

	_, _ = fmt.Println("SKILL.md files generated successfully")
	return nil
}
