// Package skills generates SKILL.md files for CLI-invokable capabilities and
// platform skills (for example workflow).
// Operations and flags are extracted dynamically from cmd/cli/command code.
// Output follows the Agent Skills open standard (agentskills.io): lean SKILL.md
// with progressive disclosure into references/cli.md (and steps.md for workflow).
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

Use ` + "`" + `flowbot {{.CLIRoot}}` + "`" + ` for capability ` + "`" + `{{.Name}}` + "`" + `.
**CLI root is ` + "`" + `{{.CLIRoot}}` + "`" + `** — do not invent ` + "`" + `flowbot {{.Name}}` + "`" + ` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.
{{- if .LimitsNote}}

**CLI limits:** {{.LimitsNote}}
{{- end}}
{{- if .ResponseHint}}

**JSON fields:** {{.ResponseHint}}
{{- end}}

## Setup

1. Ensure CLI auth: ` + "`" + `flowbot login` + "`" + `
2. Set server via ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` or ` + "`" + `--server-url` + "`" + `; optional ` + "`" + `--profile` + "`" + `, ` + "`" + `--debug` + "`" + ` / ` + "`" + `-d` + "`" + `
3. Prefer ` + "`" + `-o json` + "`" + ` when parsing results programmatically
4. Destructive commands often need ` + "`" + `-y` + "`" + ` / ` + "`" + `--yes` + "`" + ` in non-interactive sessions — check cli.md
{{- if .ScopesNote}}
5. Token scopes: {{.ScopesNote}}
{{- end}}

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
| permission denied / 403 | token missing service scopes{{- if .ScopesNote}} ({{.ScopesNote}}){{- end}} |
| hung waiting for confirm | pass ` + "`" + `-y` + "`" + ` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use ` + "`" + `flowbot {{.CLIRoot}}` + "`" + `, not the capability id as the CLI verb |
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
	Name         string
	Title        string
	Description  string
	Keywords     string
	ScopesNote   string // e.g. service:karakeep:read / service:karakeep:write
	ResponseHint string // how to read ids from -o json
	LimitsNote   string // CLI surface gaps vs capability ops
	Workflows    []workflowSpec
	CommandFn    func() *cobra.Command
}

// metaSpecs maps each CLI-invokable capability to its skill metadata.
// Skill Name equals hub.CapabilityType; CLI paths still use domain commands
// (e.g. capability "karakeep" is invoked as "flowbot bookmark ...").
var metaSpecs = []metaSpec{
	{
		Name:         string(hub.CapKarakeep),
		Title:        "Karakeep",
		CommandFn:    command.BookmarkCommand,
		Description:  "Create, list, search, archive, and delete bookmarks via flowbot bookmark.",
		Keywords:     "bookmarks, karakeep, saved URLs, reading list, link archiving, web clippings",
		ScopesNote:   "`service:karakeep:read` / `service:karakeep:write`",
		ResponseHint: "Bookmark id is a string field `id` in `-o json` output.",
		LimitsNote:   "Tag attach/detach and health are capability ops without CLI commands; inspect tags via `bookmark get`.",
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
			{
				Title:       "Archive a bookmark",
				Description: "When a user wants to archive a saved URL:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot bookmark archive <id> -y"},
					{Step: 2, Note: "Confirm archive status from the command output."},
				},
			},
		},
	},
	{
		Name:         string(hub.CapKanboard),
		Title:        "Kanboard",
		CommandFn:    command.KanbanCommand,
		Description:  "Manage kanban boards, tasks, subtasks, timers, tags, and metadata via flowbot kanban.",
		Keywords:     "kanban, kanboard, tasks, todo, subtasks, time tracking, board columns, moving cards",
		ScopesNote:   "`service:kanboard:read` / `service:kanboard:write`",
		ResponseHint: "Task ids are integers; use `kanban list` / `get` JSON `id`. Column ids come from `kanban column list`.",
		LimitsNote:   "CLI also exposes subtask/tag/timer helpers beyond the core capability CatalogSpec ops; there is no `kanban health` command.",
		Workflows: []workflowSpec{
			{
				Title:       "Create a task with subtasks",
				Description: "When a user wants to create a well-structured task:",
				Steps: []workflowStep{
					{Step: 1, Note: "Ask for or discover project_id (do not assume 1)."},
					{Step: 2, Command: "flowbot kanban column list -p <project_id>"},
					{Step: 3, Command: "flowbot kanban create -t \"<task title>\" -d \"<description>\" -p <project_id> -c <column_id>"},
					{Step: 4, Command: "flowbot kanban subtask create <task_id> -t \"<subtask 1>\" -e <minutes>"},
					{Step: 5, Command: "flowbot kanban subtask create <task_id> -t \"<subtask 2>\" -e <minutes>"},
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
		Name:         string(hub.CapTrello),
		Title:        "Trello",
		CommandFn:    command.TrelloCommand,
		Description:  "Manage Trello boards, lists, and cards via flowbot trello.",
		Keywords:     "trello, boards, lists, cards, kanban cloud, project management",
		ScopesNote:   "`service:trello:read` / `service:trello:write`",
		ResponseHint: "Board, list, and card ids are strings; use `-o json` `id` fields.",
		Workflows: []workflowSpec{
			{
				Title:       "Create a card on a board",
				Description: "When a user wants to add a task to Trello:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot trello board list"},
					{Step: 2, Command: "flowbot trello list list <board_id>"},
					{Step: 3, Command: "flowbot trello card create --list-id <list_id> -n \"<title>\""},
				},
			},
			{
				Title:       "Review board cards",
				Description: "When a user wants to see cards on a board:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot trello card list <board_id>"},
					{Step: 2, Command: "flowbot trello card get <card_id>"},
				},
			},
		},
	},
	{
		Name:         string(hub.CapConfluence),
		Title:        "Confluence",
		CommandFn:    command.ConfluenceCommand,
		Description:  "Manage Confluence Cloud spaces and pages via flowbot confluence.",
		Keywords:     "confluence, atlassian, wiki, knowledge base, pages, spaces, cql",
		ScopesNote:   "`service:confluence:read` / `service:confluence:write`",
		ResponseHint: "Page and space ids are strings; page content uses Confluence storage XHTML.",
		Workflows: []workflowSpec{
			{
				Title:       "Create a wiki page",
				Description: "When a user wants to add a Confluence page:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot confluence space list"},
					{Step: 2, Command: "flowbot confluence page create --space-key <KEY> -t \"<title>\" -c \"<p>content</p>\""},
				},
			},
			{
				Title:       "Find and read a page",
				Description: "When a user wants to open Confluence content:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot confluence page search --cql \"text ~ '<keywords>'\""},
					{Step: 2, Command: "flowbot confluence page content <page_id>"},
				},
			},
		},
	},
	{
		Name:         string(hub.CapMiniflux),
		Title:        "Miniflux",
		CommandFn:    command.ReaderCommand,
		Description:  "Subscribe to RSS/Atom feeds and manage entries via flowbot reader.",
		Keywords:     "RSS, Atom, miniflux, feed reader, unread entries, feed subscriptions, mark as read",
		ScopesNote:   "`service:miniflux:read` / `service:miniflux:write`",
		ResponseHint: "Feed and entry ids are integers in `-o json` (`id`).",
		LimitsNote:   "No `reader refresh` or star/unstar CLI; feed fetch is server-side. Mark read/unread with `reader update-entries`.",
		Workflows: []workflowSpec{
			{
				Title:       "Subscribe to a new feed",
				Description: "When a user shares a blog or feed URL they want to follow:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot reader create -u <feed_url>"},
					{Step: 2, Command: "flowbot reader list"},
					{Step: 3, Note: "Pick the new feed id from list output (server fetches entries asynchronously)."},
					{Step: 4, Command: "flowbot reader feed-entries <feed_id> -n 5"},
					{Step: 5, Note: "Report the latest entries to the user; retry feed-entries shortly if still empty."},
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
		Name:         string(hub.CapMemos),
		Title:        "Memos",
		CommandFn:    command.MemoCommand,
		Description:  "Create, list, update, and delete memos via flowbot memo.",
		Keywords:     "memos, memo notes, scratchpad, quick notes, jotting",
		ScopesNote:   "`service:memos:read` / `service:memos:write`",
		ResponseHint: "Memo resource name (not a numeric id) is the get/delete argument; see `name` in `-o json`.",
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
		Name:         string(hub.CapTrilium),
		Title:        "Trilium",
		CommandFn:    command.TriliumCommand,
		Description:  "Create, list, search, update, and delete trilium notes via flowbot trilium.",
		Keywords:     "trilium, notes, knowledge base, note tree, personal wiki",
		ScopesNote:   "`service:trilium:read` / `service:trilium:write`",
		ResponseHint: "Note ids are strings; create requires an existing `parent_note_id`.",
		Workflows: []workflowSpec{
			{
				Title:       "Create a note under a parent",
				Description: "When a user wants to add a new trilium note:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot trilium list --limit 20"},
					{Step: 2, Note: "Choose parent_note_id from list/get (ask the user if unclear)."},
					{Step: 3, Command: "flowbot trilium create -t \"<title>\" -c \"<content>\" -p <parent_note_id>"},
					{Step: 4, Note: "Report back with the note ID."},
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
		Name:         string(hub.CapFireflyiii),
		Title:        "Firefly III",
		CommandFn:    command.FireflyiiiCommand,
		Description:  "Create Firefly III transactions and inspect instance health via flowbot fireflyiii.",
		Keywords:     "fireflyiii, firefly, finance, transactions, expenses, budgeting, accounting",
		ScopesNote:   "`service:fireflyiii:read` / `service:fireflyiii:write`",
		ResponseHint: "Transaction id is a string field `id` in create output.",
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
		Name:         string(hub.CapTransmission),
		Title:        "Transmission",
		CommandFn:    command.TransmissionCommand,
		Description:  "Add, list, stop, and remove Transmission torrents via flowbot transmission.",
		Keywords:     "transmission, torrent, magnet, download, bittorrent, seed",
		ScopesNote:   "`service:transmission:read` / `service:transmission:write`",
		ResponseHint: "Torrent ids are integers; use `list` JSON `id`.",
		Workflows: []workflowSpec{
			{
				Title:       "Add a torrent",
				Description: "When a user wants to download a magnet link or torrent URL:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot transmission add -u \"<magnet-or-http-url>\""},
					{Step: 2, Note: "Report back with the torrent ID and name."},
				},
			},
			{
				Title:       "Inspect and control downloads",
				Description: "When a user asks about current downloads or wants to stop/remove one:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot transmission list"},
					{Step: 2, Command: "flowbot transmission stop --ids <id>"},
					{Step: 3, Note: "Remove only after explicit confirmation: flowbot transmission remove --ids <id> (no -y flag; confirm with the user first)."},
				},
			},
		},
	},
	{
		Name:         string(hub.CapEmail),
		Title:        "Email",
		CommandFn:    command.EmailCommand,
		Description:  "Send and read email via SMTP/IMAP with flowbot email.",
		Keywords:     "email, mail, smtp, imap, inbox, send, unread",
		ScopesNote:   "`service:email:read` / `service:email:write`",
		ResponseHint: "Message ids are opaque strings from `list`/`search`; use `-o json` for next_cursor.",
		LimitsNote:   "No attachment download/upload in CLI; get returns text/html bodies and attachment metadata only.",
		Workflows: []workflowSpec{
			{
				Title:       "Send an email",
				Description: "When a user wants to send a message:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot email send --to \"user@example.com\" --subject \"Hello\" --text \"Body\""},
					{Step: 2, Note: "Prefer --text for plain mail; use --html when HTML is required. Confirm recipients before send."},
				},
			},
			{
				Title:       "Find and read messages",
				Description: "When a user asks to inspect inbox mail:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot email list --unseen-only"},
					{Step: 2, Command: "flowbot email get <id>"},
					{Step: 3, Note: "Mark processed mail with `flowbot email mark-read <id>` when appropriate."},
				},
			},
		},
	},
	{
		Name:         string(hub.CapNocodb),
		Title:        "NocoDB",
		CommandFn:    command.NocodbCommand,
		Description:  "Discover NocoDB bases/tables and create, list, update, or delete records via flowbot nocodb.",
		Keywords:     "nocodb, base, table, record, spreadsheet, database, airtable",
		ScopesNote:   "`service:nocodb:read` / `service:nocodb:write`",
		ResponseHint: "Use base-id / table-id / record-id strings from discover commands; `--fields` keys must match column titles.",
		Workflows: []workflowSpec{
			{
				Title:       "Discover bases and tables",
				Description: "When a user needs to find which base or table to use:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot nocodb bases"},
					{Step: 2, Command: "flowbot nocodb tables --base-id <base-id>"},
					{Step: 3, Command: "flowbot nocodb table --table-id <table-id>"},
					{Step: 4, Note: "Summarize available tables and column titles before writing records."},
				},
			},
			{
				Title:       "Read and write records",
				Description: "When a user wants to inspect or change rows in a table:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot nocodb records list --table-id <table-id>"},
					{Step: 2, Command: "flowbot nocodb records create --table-id <table-id> --fields '{\"Title\":\"value\"}'"},
					{Step: 3, Note: "Use update/delete only with an explicit record-id; prefer -o json when parsing results."},
				},
			},
		},
	},
	{
		Name:         string(hub.CapDevops),
		Title:        "DevOps",
		CommandFn:    command.DevopsCommand,
		Description:  "Query devops backends (beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx) via flowbot devops.",
		Keywords:     "devops, beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx, prometheus, loki, tempo, pyroscope, alloy, monitoring, infrastructure",
		ScopesNote:   "`service:devops:read`",
		ResponseHint: "Prefer `-o json`. Start with `devops status` to see which backends are configured.",
		LimitsNote:   "There is no `flowbot devops health`; use `flowbot devops status` for aggregate readiness, then backend-specific health/metrics commands.",
		Workflows: []workflowSpec{
			{
				Title:       "Check which backends are configured",
				Description: "When a user asks what devops tools are available:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot devops status"},
					{Step: 2, Note: "Only call subcommands for backends reported as configured."},
				},
			},
			{
				Title:       "Inspect monitoring and routing",
				Description: "When a user wants a quick infrastructure snapshot:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot devops beszel systems"},
					{Step: 2, Command: "flowbot devops uptimekuma health"},
					{Step: 3, Command: "flowbot devops traefik routers"},
					{Step: 4, Command: "flowbot devops grafana health"},
					{Step: 5, Command: "flowbot devops wakapi projects"},
					{Step: 6, Command: "flowbot devops dozzle health"},
					{Step: 7, Command: "flowbot devops netalertx totals"},
				},
			},
			{
				Title:       "Query observability backends via Grafana",
				Description: "When a user wants metrics, logs, traces, or profiles:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot devops grafana datasources"},
					{Step: 2, Command: "flowbot devops grafana query --backend prometheus --expr 'up'"},
					{Step: 3, Note: "Use backend alloy|loki|tempo|pyroscope with the matching expression language; prefer -o json for parsing."},
				},
			},
		},
	},
	{
		Name:         string(hub.CapGitea),
		Title:        "Gitea",
		CommandFn:    command.ForgeCommand,
		Description:  "Inspect forge users, repos, issues, diffs, and files via flowbot forge.",
		Keywords:     "gitea, forge, repositories, issues, commit diffs, source files, code review",
		ScopesNote:   "`service:gitea:read` / `service:gitea:write`",
		ResponseHint: "Issue index is the forge issue number argument; not the same as GitHub `number` naming in docs.",
		LimitsNote:   "CLI root is `forge` (alias may include gitea). No `forge health` command.",
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
		Name:         string(hub.CapGithub),
		Title:        "GitHub",
		CommandFn:    command.GithubCommand,
		Description:  "Inspect GitHub users, repos, issues, notifications, releases, diffs, and files via flowbot github.",
		Keywords:     "github, repositories, issues, notifications, releases, commit diffs, source files",
		ScopesNote:   "`service:github:read` / `service:github:write`",
		ResponseHint: "Issue argument is `<number>`; notification and release ids appear in `-o json`.",
		LimitsNote:   "No pull-request commands and no `github health` CLI.",
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

// platformSpecs lists non-capability (platform) skills generated alongside metaSpecs.
var platformSpecs = []platformSpec{
	platformWorkflowSpec(),
	platformPipelineSpec(),
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
func buildCLIString(path, argsUsage string, flags []flagSpec) string {
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
	ScopesNote         string
	ResponseHint       string
	LimitsNote         string
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
		ScopesNote:         meta.ScopesNote,
		ResponseHint:       meta.ResponseHint,
		LimitsNote:         meta.LimitsNote,
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
func executeTemplateFile[T any](tmpl *template.Template, path string, data T) error {
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

// SkillsAction generates SKILL.md files for all CLI-invokable capabilities and platform skills.
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

	for _, meta := range platformSpecs {
		if err := generatePlatformSkill(meta, outputDir); err != nil {
			return err
		}
	}

	_, _ = fmt.Println("SKILL.md files generated successfully")
	return nil
}
