package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/command"
)

//go:embed testdata/workflow/*.yaml
var workflowExampleFS embed.FS

// stepTypeSpec documents one workflow task action type for references/steps.md.
type stepTypeSpec struct {
	Prefix      string
	Title       string
	Summary     string
	ActionForm  string
	Inputs      string // markdown: what goes into the step (action details + params)
	Outputs     string // markdown: what {{step "id" "result"}} returns
	Usage       string // markdown: how to chain / read outputs
	Notes       string
	ExampleYAML string
}

// platformSpec holds metadata for a non-capability (platform) skill.
type platformSpec struct {
	Name                     string
	Title                    string
	Description              string
	Keywords                 string
	CLIRoot                  string
	ScopesNote               string
	Workflows                []workflowSpec
	StepTypes                []stepTypeSpec
	IncludeCapabilityCatalog bool
	// Kind selects template set: "workflow" (default) or "pipeline".
	Kind       string
	CommandFn  func() *cobra.Command
	ExampleFS  embed.FS
	ExampleDir string
}

// platformSkillData is the template context for platform SKILL.md and references.
type platformSkillData struct {
	Name               string
	Title              string
	CLIRoot            string
	TriggerDescription string
	ScopesNote         string
	Operations         []opSpec
	Workflows          []workflowSpec
	StepTypes          []stepTypeSpec
	Capabilities       []capDoc
	ExampleFiles       []string
}

const platformSkillTemplate = `---
name: {{.Name}}
description: >-
  {{.TriggerDescription}}
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  platform: {{.Name}}
  cli_root: {{.CLIRoot}}
---

# {{.Title}}

Use ` + "`" + `flowbot {{.CLIRoot}}` + "`" + ` for platform workflow definitions stored in the database.
YAML is an exchange format for ` + "`" + `apply` + "`" + ` / ` + "`" + `export` + "`" + ` only — the server does not run from local files.
Prefer the workflows below; load [references/schema.md](references/schema.md) for the full YAML definition,
[references/cli.md](references/cli.md) for flags,
[references/steps.md](references/steps.md) for task action types, inputs/outputs, and usage, and
[references/capabilities.md](references/capabilities.md) (index) and ` + "`" + `references/capabilities/<type>.md` + "`" + ` for every ` + "`" + `capability:<type>.<op>` + "`" + ` param list.
**Never invent** a ` + "`" + `capability:<type>` + "`" + ` that is absent from the capabilities index.
Teaching examples (load via read_skill with path):
{{- range .ExampleFiles}}
- [examples/{{.}}](examples/{{.}})
{{- end}}

## Setup

1. Ensure CLI auth: ` + "`" + `flowbot login` + "`" + `
2. Set server via ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` or ` + "`" + `--server-url` + "`" + `; optional ` + "`" + `--profile` + "`" + `, ` + "`" + `--debug` + "`" + ` / ` + "`" + `-d` + "`" + `
3. {{.ScopesNote}}
4. Prefer ` + "`" + `-o json` + "`" + ` when parsing results programmatically

## Step types

| Prefix | Use |
|--------|-----|
{{- range .StepTypes}}
| ` + "`" + `{{.Prefix}}` + "`" + ` | {{.Summary}} |
{{- end}}

Load [references/steps.md](references/steps.md) for per-type inputs/outputs, usage, templates, and ` + "`" + `conn` + "`" + `/` + "`" + `retry` + "`" + `.
Load [references/schema.md](references/schema.md) before writing any workflow YAML.
Load [references/capabilities.md](references/capabilities.md) for the capability index; **must** open ` + "`" + `references/capabilities/<type>.md` + "`" + ` before emitting that type's actions.

## Templates

Task ` + "`" + `params` + "`" + ` use Go ` + "`" + `text/template` + "`" + ` delimiters ` + "`" + `{{"{{ }}"}}` + "`" + ` (same engine as pipelines).

**Variables available in workflows:**

| Variable | Access | Source |
|----------|--------|--------|
| Run inputs | ` + "`" + `{{"{{input \"url\"}}"}}` + "`" + ` / ` + "`" + `{{"{{input.url}}"}}` + "`" + ` / ` + "`" + `{{"{{.Input.url}}"}}` + "`" + ` | ` + "`" + `workflow run --input` + "`" + ` (keys = declared ` + "`" + `inputs` + "`" + `) |
| Prior steps | ` + "`" + `{{"{{step \"id\" \"result\"}}"}}` + "`" + ` / ` + "`" + `{{"{{.Steps.id.result}}"}}` + "`" + ` | Completed task outputs (` + "`" + `result` + "`" + ` and ` + "`" + `id` + "`" + ` hold the same payload) |

Not set for workflows: ` + "`" + `event` + "`" + ` / ` + "`" + `.Event` + "`" + `, ` + "`" + `env` + "`" + ` / ` + "`" + `.Env` + "`" + `.
Helpers (closed set): ` + "`" + `input` + "`" + `, ` + "`" + `step` + "`" + `, ` + "`" + `event` + "`" + `, ` + "`" + `jsonpath` + "`" + `, ` + "`" + `jsonpathExists` + "`" + `, ` + "`" + `jsonpathRaw` + "`" + `, ` + "`" + `default` + "`" + `, ` + "`" + `json` + "`" + `, ` + "`" + `len` + "`" + `, ` + "`" + `join` + "`" + `, ` + "`" + `split` + "`" + `, ` + "`" + `contains` + "`" + `, ` + "`" + `now` + "`" + `, plus Go builtins ` + "`" + `if` + "`" + `/` + "`" + `else` + "`" + `/` + "`" + `range` + "`" + `/` + "`" + `eq` + "`" + `/` + "`" + `printf` + "`" + `.
Full list: [references/steps.md](references/steps.md#templates).
**Never invent** a helper absent from that list (no Sprig / ` + "`" + `date` + "`" + ` / other template libraries).

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
| insufficient scope | token needs ` + "`" + `workflow:read` + "`" + ` and/or ` + "`" + `workflow:run` + "`" + ` |
| workflow name is required / not found | apply first; check ` + "`" + `list` + "`" + ` |
| input validation failed | supply all required ` + "`" + `inputs` + "`" + ` with correct types |
| ` + "`" + `function "…" not defined` + "`" + ` | replace with a helper from [references/steps.md](references/steps.md#templates); do not invent helpers |
| webhook rejected | workflow must be ` + "`" + `enabled` + "`" + `; trigger needs ` + "`" + `auth.token` + "`" + ` or ` + "`" + `auth.hmac_secret` + "`" + ` |
`

const platformCLIReferenceTemplate = `# {{.Title}} CLI reference

Platform skill (not a hub capability). Root command: ` + "`" + `flowbot {{.CLIRoot}}` + "`" + `.

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

const platformStepsTemplate = `# Workflow task steps reference

Load this file when authoring or editing workflow YAML tasks. Teaching examples:
{{- range .ExampleFiles}}
- [examples/{{.}}](../examples/{{.}})
{{- end}}

For every ` + "`" + `capability:<type>.<op>` + "`" + ` param table, start at [capabilities.md](capabilities.md) and open ` + "`" + `capabilities/<type>.md` + "`" + ` **before** writing that action. Do not invent types missing from the index.

Top-level definition fields and triggers: [schema.md](schema.md).

## Shared task fields

| Field | Required | Notes |
|-------|----------|-------|
| ` + "`" + `id` + "`" + ` | yes | Unique within the workflow |
| ` + "`" + `action` + "`" + ` | yes | See action types below |
| ` + "`" + `describe` + "`" + ` | no | Human-readable label |
| ` + "`" + `params` + "`" + ` | no | Template-rendered before execution; declare matching top-level ` + "`" + `inputs` + "`" + ` when using ` + "`" + `{{"{{input.*}}"}}` + "`" + ` |
| ` + "`" + `conn` + "`" + ` | no | Upstream task ids (DAG edges; required for parallel scheduling) |
| ` + "`" + `vars` + "`" + ` | no | Optional string list (advanced; usually omit) |
| ` + "`" + `retry` + "`" + ` | no | Same shape as pipeline retry (` + "`" + `max_attempts` + "`" + `, ` + "`" + `delay` + "`" + `, ` + "`" + `backoff` + "`" + `, ` + "`" + `max_delay` + "`" + `, ` + "`" + `jitter` + "`" + `); workflows retry all errors |

With ` + "`" + `max_concurrency > 1` + "`" + `, ` + "`" + `conn` + "`" + ` drives parallel DAG scheduling. Otherwise order follows ` + "`" + `pipeline` + "`" + `.

## Templates

Workflow task ` + "`" + `params` + "`" + ` (string values) are rendered with Go ` + "`" + `text/template` + "`" + ` before the step runs.
Delimiters are ` + "`" + `{{"{{"}}` + "`" + ` and ` + "`" + `{{"}}"}}` + "`" + `. Missing keys via helpers return empty string; invalid template syntax errors.

### Available variables

Root context is ` + "`" + `TemplateData` + "`" + `: ` + "`" + `.Input` + "`" + `, ` + "`" + `.Steps` + "`" + `, ` + "`" + `.Event` + "`" + `, ` + "`" + `.Env` + "`" + `. Workflows only populate the first two.

| Variable | Populated in workflow? | How to read | Source |
|----------|------------------------|-------------|--------|
| ` + "`" + `Input` + "`" + ` | yes | ` + "`" + `{{"{{input \"name\"}}"}}` + "`" + `, ` + "`" + `{{"{{input.name}}"}}` + "`" + `, ` + "`" + `{{"{{.Input.name}}"}}` + "`" + ` | Run payload from ` + "`" + `workflow run --input` + "`" + ` / API; keys match declared top-level ` + "`" + `inputs[].name` + "`" + ` (plus defaults applied by validation) |
| ` + "`" + `Steps` + "`" + ` | yes | ` + "`" + `{{"{{step \"task_id\" \"result\"}}"}}` + "`" + `, ` + "`" + `{{"{{step \"task_id\" \"id\"}}"}}` + "`" + `, ` + "`" + `{{"{{.Steps.task_id.result}}"}}` + "`" + ` | Outputs of **already completed** tasks only. Workflow stores the same string under both ` + "`" + `result` + "`" + ` and ` + "`" + `id` + "`" + ` |
| ` + "`" + `Event` + "`" + ` | no | ` + "`" + `{{"{{event \"field\"}}"}}` + "`" + ` / ` + "`" + `{{"{{.Event.field}}"}}` + "`" + ` | Empty in workflows (pipeline DataEvent only). Do not rely on it |
| ` + "`" + `Env` + "`" + ` | no | ` + "`" + `{{"{{.Env.HOME}}"}}` + "`" + ` | Empty in workflows. Do not rely on it |

` + "`" + `{{"{{input.name}}"}}` + "`" + ` is sugar for ` + "`" + `{{"{{input \"name\"}}"}}` + "`" + `.

### Helper functions

**Closed set only.** Use helpers from this table (plus Go ` + "`" + `text/template` + "`" + ` builtins like ` + "`" + `if` + "`" + `/` + "`" + `else` + "`" + `/` + "`" + `range` + "`" + `/` + "`" + `eq` + "`" + `/` + "`" + `printf` + "`" + `). **Never invent** unlisted helpers (no Sprig, no ` + "`" + `date` + "`" + `, no invented aliases). Missing helpers fail at run time with ` + "`" + `function "…" not defined` + "`" + `.

Data accessors: ` + "`" + `input` + "`" + `, ` + "`" + `step` + "`" + `, ` + "`" + `event` + "`" + `.

| Helper | Example |
|--------|---------|
| ` + "`" + `jsonpath` + "`" + ` | ` + "`" + `{{"{{jsonpath (step \"api\" \"result\") \"data.id\"}}"}}` + "`" + ` |
| ` + "`" + `jsonpathExists` + "`" + ` | ` + "`" + `{{"{{if jsonpathExists (step \"api\" \"result\") \"error\"}}bad{{end}}"}}` + "`" + ` |
| ` + "`" + `jsonpathRaw` + "`" + ` | ` + "`" + `{{"{{json (jsonpathRaw (step \"api\" \"result\") \"items\")}}"}}` + "`" + ` |
| ` + "`" + `default` + "`" + ` | ` + "`" + `{{"{{default \"guest\" (input \"user\")}}"}}` + "`" + ` |
| ` + "`" + `json` + "`" + ` | ` + "`" + `{{"{{json (input \"meta\")}}"}}` + "`" + ` |
| ` + "`" + `len` + "`" + ` | ` + "`" + `{{"{{len (input \"tags\")}}"}}` + "`" + ` |
| ` + "`" + `join` + "`" + ` / ` + "`" + `split` + "`" + ` | ` + "`" + `{{"{{join (split (input \"tags\") \",\") \";\")}}"}}` + "`" + ` |
| ` + "`" + `contains` + "`" + ` | ` + "`" + `{{"{{if contains (input \"title\") \"ERROR\"}}alert{{end}}"}}` + "`" + ` |
| ` + "`" + `now` + "`" + ` | ` + "`" + `{{"{{now}}"}}` + "`" + ` (UTC RFC3339 string, e.g. ` + "`" + `2026-07-26T03:19:00Z` + "`" + `) |
| ` + "`" + `if` + "`" + ` / ` + "`" + `else` + "`" + ` | ` + "`" + `{{"{{if (input \"url\")}}has{{else}}missing{{end}}"}}` + "`" + ` |

YAML tip: when an expression contains quotes, wrap the param value in single quotes:

` + "```yaml" + `
params:
  description: 'Bookmark: {{"{{step \"save_bookmark\" \"result\"}}"}}'
  url: "{{"{{input.url}}"}}"
` + "```" + `

## Action types
{{- range .StepTypes}}

### {{.Title}} (` + "`" + `{{.Prefix}}` + "`" + `)

{{.Summary}}

**Action form:** ` + "`" + `{{.ActionForm}}` + "`" + `

**Inputs:**

{{.Inputs}}

**Outputs (` + "`" + `{{"{{step \"id\" \"result\"}}"}}` + "`" + `):**

{{.Outputs}}

**Usage:**

{{.Usage}}

{{- if .Notes}}

**Notes:** {{.Notes}}
{{- end}}

{{- if .ExampleYAML}}

` + "```yaml" + `
{{.ExampleYAML}}
` + "```" + `
{{- end}}
{{- end}}
`

const platformCapabilitiesIndexTemplate = `# Workflow capability actions reference

Load a capability file below when authoring ` + "`" + `capability:<type>.<operation>` + "`" + ` tasks.
Param tables are generated from each capability's ` + "`" + `OpDef.Input` + "`" + ` (same source as hub registration).

**Rules for agents:**

1. Only use capability types listed in the table below.
2. Before emitting any ` + "`" + `capability:<type>.*` + "`" + ` action, load ` + "`" + `capabilities/<type>.md` + "`" + ` and copy required params from that file.
3. Do not invent ops, param keys, or types (for example there is no ` + "`" + `capability:archive.*` + "`" + `).

## Result envelope

Every capability step stores a JSON ` + "`" + `InvokeResult` + "`" + ` string as ` + "`" + `{{"{{step \"id\" \"result\"}}"}}` + "`" + `:

| Field | Meaning |
|-------|---------|
| ` + "`" + `capability` + "`" + ` | Capability type (e.g. ` + "`" + `karakeep` + "`" + `, ` + "`" + `core` + "`" + `) |
| ` + "`" + `operation` + "`" + ` | Operation name |
| ` + "`" + `data` + "`" + ` | Domain payload (object, array, or scalar); omit when empty |
| ` + "`" + `text` + "`" + ` | Optional short human summary |
| ` + "`" + `page` + "`" + ` | Optional pagination for list ops |

**Usage patterns:**

- Field: ` + "`" + `{{"{{jsonpath (step \"save\" \"result\") \"data.url\"}}"}}` + "`" + `
- Exists: ` + "`" + `{{"{{if jsonpathExists (step \"save\" \"result\") \"data.id\"}}...{{end}}"}}` + "`" + `
- Whole JSON (rare): ` + "`" + `{{"{{step \"save\" \"result\"}}"}}` + "`" + `

Mutation ops change remote/system state; prefer read ops when exploring.

## Common ` + "`" + `data` + "`" + ` paths

| Op pattern | Typical ` + "`" + `jsonpath` + "`" + ` |
|------------|----------------------|
| ` + "`" + `karakeep.create` + "`" + ` / ` + "`" + `get` + "`" + ` | ` + "`" + `data.id` + "`" + `, ` + "`" + `data.url` + "`" + `, ` + "`" + `data.title` + "`" + ` |
| ` + "`" + `karakeep.check_url` + "`" + ` | ` + "`" + `data.exists` + "`" + `, ` + "`" + `data.id` + "`" + ` |
| ` + "`" + `karakeep.list` + "`" + ` / ` + "`" + `search` + "`" + ` | ` + "`" + `data` + "`" + ` is an array; ` + "`" + `page` + "`" + ` for cursor |
| ` + "`" + `kanboard.create_task` + "`" + ` / ` + "`" + `get_task` + "`" + ` | ` + "`" + `data.id` + "`" + `, ` + "`" + `data.title` + "`" + `, ` + "`" + `data.project_id` + "`" + ` |
| ` + "`" + `core.notify_send` + "`" + ` | ` + "`" + `data.sent` + "`" + ` |
| ` + "`" + `core.agent_run` + "`" + ` | ` + "`" + `data.reply` + "`" + ` |
| ` + "`" + `core.clip_create` + "`" + ` | ` + "`" + `data` + "`" + ` includes public URL fields |
| ` + "`" + `core.http_request` + "`" + ` | ` + "`" + `data.status` + "`" + `, ` + "`" + `data.body` + "`" + ` |
| ` + "`" + `core.run_terminal` + "`" + ` / ` + "`" + `run_code` + "`" + ` | ` + "`" + `data.output` + "`" + `, ` + "`" + `data.exit_code` + "`" + ` |
| ` + "`" + `core.kv_get` + "`" + ` | ` + "`" + `data` + "`" + ` is the stored value |

## Capabilities

| Capability | Reference |
|------------|-----------|
{{- range .Capabilities}}
| ` + "`" + `{{.Type}}` + "`" + ` | [capabilities/{{.Type}}.md](capabilities/{{.Type}}.md) — {{.Description}} |
{{- end}}
`

const platformSchemaTemplate = `# Workflow YAML schema

Canonical shape is ` + "`" + `types.WorkflowMetadata` + "`" + `. Load this file before writing or editing a definition.
Task action details: [steps.md](steps.md). Capability params: [capabilities.md](capabilities.md).

## Skeleton

` + "```yaml" + `
name: example_workflow          # required; unique
describe: "Human summary"       # recommended
enabled: true                   # false disables triggers and runs
resumable: false                # true enables checkpoint/resume
max_concurrency: 1              # 0 or 1 = sequential via pipeline order; >1 = DAG via conn

inputs:
  - name: url
    type: string                # string | number | boolean | json
    required: true
    description: "URL to process"
    # default: "https://example.com"   # optional when required is false

triggers:
  - type: manual                # see Triggers below
    enabled: true

pipeline:                       # task id order (sequential mode); also lists all tasks
  - step_a
  - step_b

tasks:
  - id: step_a
    action: capability:karakeep.create
    describe: "Save URL"
    params:
      url: "{{"{{input.url}}"}}"
  - id: step_b
    action: "mapper:"
    params:
      bookmark_id: '{{"{{jsonpath (step \"step_a\" \"result\") \"data.id\"}}"}}'
    conn:                       # required edges when max_concurrency > 1
      - step_a
` + "```" + `

## Top-level fields

| Field | Required | Notes |
|-------|----------|-------|
| ` + "`" + `name` + "`" + ` | yes | Stable identifier used by CLI ` + "`" + `get` + "`" + `/` + "`" + `run` + "`" + `/` + "`" + `delete` + "`" + ` |
| ` + "`" + `describe` + "`" + ` | no | Human-readable summary |
| ` + "`" + `enabled` + "`" + ` | no | Default false if omitted in some paths; set ` + "`" + `true` + "`" + ` for runnable workflows |
| ` + "`" + `resumable` + "`" + ` | no | Checkpoint/resume support |
| ` + "`" + `max_concurrency` + "`" + ` | no | ` + "`" + `<=1` + "`" + `: run ` + "`" + `pipeline` + "`" + ` in order; ` + "`" + `>1` + "`" + `: schedule ready tasks using ` + "`" + `conn` + "`" + ` |
| ` + "`" + `inputs` + "`" + ` | no | Declared run inputs; every ` + "`" + `{{"{{input.*}}"}}` + "`" + ` key must be listed |
| ` + "`" + `triggers` + "`" + ` | yes | At least one trigger object (use ` + "`" + `manual` + "`" + ` for CLI-only) |
| ` + "`" + `pipeline` + "`" + ` | yes | List of task ids |
| ` + "`" + `tasks` + "`" + ` | yes | Task objects; every ` + "`" + `pipeline` + "`" + ` id must exist |

## Inputs

| Field | Required | Notes |
|-------|----------|-------|
| ` + "`" + `name` + "`" + ` | yes | Key for ` + "`" + `{{"{{input.name}}"}}` + "`" + ` / run ` + "`" + `--input` + "`" + ` JSON |
| ` + "`" + `type` + "`" + ` | yes | ` + "`" + `string` + "`" + ` \| ` + "`" + `number` + "`" + ` \| ` + "`" + `boolean` + "`" + ` \| ` + "`" + `json` + "`" + ` |
| ` + "`" + `required` + "`" + ` | no | When true, run fails if missing |
| ` + "`" + `default` + "`" + ` | no | Applied when omitted |
| ` + "`" + `description` + "`" + ` | no | Documentation only |

## Triggers

| ` + "`" + `type` + "`" + ` | Purpose | Rule keys |
|---------|---------|-----------|
| ` + "`" + `manual` + "`" + ` | CLI / API ` + "`" + `workflow run` + "`" + ` | none |
| ` + "`" + `cron` + "`" + ` | Scheduled runs | ` + "`" + `rule.cron` + "`" + ` (or ` + "`" + `rule.expression` + "`" + `): standard 5-field cron or descriptor |
| ` + "`" + `webhook` + "`" + ` | HTTP ingress | ` + "`" + `rule.path` + "`" + ` (required), ` + "`" + `rule.method` + "`" + ` (GET/POST/PUT, default POST), ` + "`" + `rule.auth.token` + "`" + ` and/or ` + "`" + `rule.auth.hmac_secret` + "`" + ` (at least one), optional ` + "`" + `token_header` + "`" + `/` + "`" + `hmac_header` + "`" + `, ` + "`" + `payload` + "`" + ` (` + "`" + `raw` + "`" + `\|` + "`" + `mapped` + "`" + `), ` + "`" + `event_type` + "`" + ` |

` + "```yaml" + `
triggers:
  - type: manual
    enabled: true
  - type: cron
    enabled: true
    rule:
      cron: "0 * * * *"
  - type: webhook
    enabled: true
    rule:
      path: /hooks/my-workflow
      method: POST
      auth:
        token: "replace-me"
      payload: raw
` + "```" + `

Webhook auth: supply ` + "`" + `auth.token` + "`" + ` and/or ` + "`" + `auth.hmac_secret` + "`" + `. Defaults: token header ` + "`" + `X-Webhook-Token` + "`" + `, HMAC header ` + "`" + `X-Hub-Signature-256` + "`" + `.

## Tasks

See [steps.md](steps.md) for ` + "`" + `action` + "`" + ` prefixes, templates, ` + "`" + `conn` + "`" + `, and ` + "`" + `retry` + "`" + `.
Every ` + "`" + `capability:` + "`" + ` action must use a type from [capabilities.md](capabilities.md).

## Authoring checklist

1. Copy the skeleton; set ` + "`" + `name` + "`" + `, ` + "`" + `enabled: true` + "`" + `, and a ` + "`" + `manual` + "`" + ` trigger.
2. Declare ` + "`" + `inputs` + "`" + ` for every template key under ` + "`" + `{{"{{input.*}}"}}` + "`" + `.
3. For each capability task, open ` + "`" + `capabilities/<type>.md` + "`" + ` and fill required params only.
4. Prefer ` + "`" + `jsonpath` + "`" + ` when reading prior capability results (see Common data paths in capabilities.md).
5. If ` + "`" + `max_concurrency > 1` + "`" + `, set ` + "`" + `conn` + "`" + ` on every non-root task.
6. ` + "`" + `flowbot workflow apply --file ...` + "`" + ` then ` + "`" + `get` + "`" + ` / ` + "`" + `run` + "`" + `.
`

const platformCapabilityFileTemplate = `# ` + "`" + `{{.Type}}` + "`" + ` capability actions

{{.Description}}

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

{{- range .Ops}}

## ` + "`" + `{{.Action}}` + "`" + `

{{.Description}}{{if .Mutation}} (**mutation**){{end}}

**Inputs (params):**
{{- if .Inputs}}

| Param | Type | Required | Description |
|-------|------|----------|-------------|
{{- range .Inputs}}
| ` + "`" + `{{.Name}}` + "`" + ` | ` + "`" + `{{.Type}}` + "`" + ` | {{if .Required}}yes{{else}}no{{end}} | {{.Description}} |
{{- end}}
{{- else}}

_(none)_
{{- end}}

**Outputs:** ` + "`" + `InvokeResult` + "`" + ` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under ` + "`" + `data` + "`" + `; use ` + "`" + `text` + "`" + ` when present.

**Usage:**

` + "```yaml" + `
  - id: {{.Name}}_step
    action: {{.Action}}
{{- if .Inputs}}
    params:
{{- range .Inputs}}
      {{.Name}}: {{if eq .Type "string"}}"..."{{else if eq .Type "[]string"}}["..."]{{else if or (eq .Type "int") (eq .Type "number")}}0{{else if eq .Type "bool"}}false{{else if eq .Type "map[string]any"}}{}{{else}}...{{end}}{{if .Required}}  # required{{end}}
{{- end}}
{{- end}}
` + "```" + `
{{- end}}
`

// platformWorkflowSpec returns the platform skill definition for flowbot workflow.
func platformWorkflowSpec() platformSpec {
	return platformSpec{
		Name:        "workflow",
		Title:       "Workflow",
		CLIRoot:     "workflow",
		CommandFn:   command.WorkflowCommand,
		Description: "Manage Flowbot workflows via flowbot workflow: apply YAML definitions to the database, list/get/export/delete, run asynchronously, and inspect runs.",
		Keywords:    "workflows, workflow YAML, workflow runs, cron/webhook workflow triggers",
		ScopesNote:  "Token scopes: `workflow:read` for list/get/export/runs; `workflow:run` for apply/delete/run (run also satisfies read)",
		ExampleFS:   workflowExampleFS,
		ExampleDir:  "testdata/workflow",
		Workflows: []workflowSpec{
			{
				Title:       "Write or edit a workflow YAML",
				Description: "When the user needs a new or updated workflow definition:",
				Steps: []workflowStep{
					{Step: 1, Note: "Load references/schema.md and copy the skeleton (name, enabled, triggers, pipeline, tasks, inputs)."},
					{Step: 2, Note: "For each capability: action, open references/capabilities/<type>.md first; never invent types missing from the capabilities index."},
					{Step: 3, Note: "Use only documented template helpers from references/steps.md; never invent helpers."},
					{Step: 4, Note: "Use examples/echo_mapper.yaml, examples/save_and_track.yaml, or examples/parallel_example.yaml as starting points."},
					{Step: 5, Note: "Declare inputs for every {{input.*}} key; use jsonpath for prior capability results (see capabilities.md Common data paths)."},
					{Step: 6, Command: "flowbot workflow apply --file path/to/workflow.yaml"},
					{Step: 7, Command: "flowbot workflow get <name>"},
					{Step: 8, Note: "Optional: flowbot workflow run <name> --input '{...}' then flowbot workflow runs <name>."},
				},
			},
			{
				Title:       "Apply a definition from YAML",
				Description: "When the user already has a workflow YAML file to create or replace:",
				Steps: []workflowStep{
					{Step: 1, Note: "Validate against references/schema.md: name, pipeline, tasks, triggers, and inputs for any {{input.*}} used in params."},
					{Step: 2, Command: "flowbot workflow apply --file path/to/workflow.yaml"},
					{Step: 3, Command: "flowbot workflow get <name>"},
				},
			},
			{
				Title:       "List and inspect",
				Description: "When the user asks what workflows exist or what a workflow contains:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot workflow list"},
					{Step: 2, Command: "flowbot workflow get <name>"},
					{Step: 3, Note: "Optional: flowbot workflow export <name> -o file.yaml to round-trip YAML."},
				},
			},
			{
				Title:       "Run a workflow",
				Description: "When the user wants to execute a stored workflow:",
				Steps: []workflowStep{
					{Step: 1, Note: "Build input JSON matching declared inputs (required fields must be present)."},
					{Step: 2, Command: "flowbot workflow run <name> --input '{\"url\":\"...\",\"title\":\"...\"}'"},
					{Step: 3, Note: "Note the returned run_id (runs are asynchronous)."},
					{Step: 4, Command: "flowbot workflow runs <name>"},
				},
			},
			{
				Title:       "Delete",
				Description: "When the user wants to remove a definition (run history is kept):",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot workflow delete <name>"},
				},
			},
		},
		StepTypes:                workflowStepTypeSpecs(),
		IncludeCapabilityCatalog: true,
	}
}

// workflowStepTypeSpecs returns documented workflow task action types.
func workflowStepTypeSpecs() []stepTypeSpec {
	return []stepTypeSpec{
		{
			Prefix:     "capability:",
			Title:      "Capability",
			Summary:    "Invoke a Flowbot capability operation (provider or CapCore)",
			ActionForm: "capability:<type>.<operation>",
			Inputs: `- **Action:** ` + "`" + `<type>` + "`" + ` is the hub capability ID (provider ID, or ` + "`" + `core` + "`" + `). ` + "`" + `<operation>` + "`" + ` is the op name after the last dot (e.g. ` + "`" + `karakeep.create` + "`" + `, ` + "`" + `core.notify_send` + "`" + `).
- **Params:** template-rendered KV passed to ` + "`" + `capability.Invoke` + "`" + `. Per-operation required/optional keys: load [capabilities/<type>.md](capabilities/) via the [capabilities.md](capabilities.md) index.`,
			Outputs: `JSON string of ` + "`" + `InvokeResult` + "`" + `: ` + "`" + `{"capability","operation","data","text","page",...}` + "`" + `.
- Domain payload is usually under ` + "`" + `data` + "`" + ` (object, array, or scalar). Optional ` + "`" + `text` + "`" + ` is a short human summary. List ops may include ` + "`" + `page` + "`" + `.
- CapCore shapes (examples): ` + "`" + `notify_send` + "`" + ` → ` + "`" + `data.sent` + "`" + `; ` + "`" + `agent_run` + "`" + ` → ` + "`" + `data.reply` + "`" + `; ` + "`" + `kv_get` + "`" + ` → ` + "`" + `data` + "`" + ` is the stored value; ` + "`" + `http_request` + "`" + ` → ` + "`" + `data.status` + "`" + `/` + "`" + `data.body` + "`" + `; ` + "`" + `run_*` + "`" + ` → ` + "`" + `data.output` + "`" + `/` + "`" + `data.exit_code` + "`" + `. Full per-op notes: [capabilities/core.md](capabilities/core.md).`,
			Usage: `- Read a field: ` + "`" + `{{jsonpath (step "save" "result") "data.url"}}` + "`" + `
- Check a key exists: ` + "`" + `{{if jsonpathExists (step "save" "result") "data.id"}}...{{end}}` + "`" + `
- Pass whole JSON into another param (rare): ` + "`" + `{{step "save" "result"}}` + "`" + ` (string). Prefer jsonpath for structured data.
- Authoring checklist: load [capabilities/<type>.md](capabilities/) first, copy required params, then chain with jsonpath.
- Never invent capability types or ops; only use the [capabilities.md](capabilities.md) index.
- Pre-CapCore notify/agent/clip types were removed; use ` + "`" + `capability:core.*` + "`" + ` (see docs/developer-guide/capcore-migration.md).`,
			Notes: "Do not invent param keys or types; capabilities/<type>.md is generated from OpDef.Input. Workflow wiring only forwards params — mounts/env/timeouts are not set at the workflow layer.",
			ExampleYAML: `  - id: save_bookmark
    action: capability:karakeep.create
    params:
      url: "{{input.url}}"
  - id: notify_saved
    action: capability:core.notify_send
    params:
      template_id: bookmark.saved
      channels: ["slack"]
      payload:
        url: "{{input.url}}"
        bookmark: '{{jsonpath (step "save_bookmark" "result") "data"}}'`,
		},
		{
			Prefix:     "docker:",
			Title:      "Docker",
			Summary:    "Run a container image via the Docker runtime on the workflow runner",
			ActionForm: "docker:<image>",
			Inputs: `- **Action details:** image reference after the prefix (e.g. ` + "`" + `alpine:3.20` + "`" + ` in ` + "`" + `docker:alpine:3.20` + "`" + `).
- **Params:**
  | Key | Type | Required | Meaning |
  |-----|------|----------|---------|
  | ` + "`" + `cmd` + "`" + ` | string or string list | no | Container command; overrides the image default CMD |

Workflow does **not** map mounts, env, or timeout from YAML params.`,
			Outputs: `Plain text **stdout** from the container (not JSON). Trailing newlines are kept as returned by the runtime.`,
			Usage: `- Echo into a later step: ` + "`" + `message: '{{step "run_tool" "result"}}'` + "`" + `
- Do not use ` + "`" + `jsonpath` + "`" + ` unless the container itself prints JSON; then ` + "`" + `{{jsonpath (step "run_tool" "result") "field"}}` + "`" + ` works on that string.`,
			Notes: "Requires Docker available to the server executor.",
			ExampleYAML: `  - id: run_tool
    action: docker:alpine:3.20
    params:
      cmd: ["echo", "hello"]
  - id: use_stdout
    action: "mapper:"
    params:
      from_docker: '{{step "run_tool" "result"}}'`,
		},
		{
			Prefix:     "kern:",
			Title:      "Kern",
			Summary:    "Run a container image via the kern CLI on the workflow runner (Linux only)",
			ActionForm: "kern:<image>",
			Inputs: `- **Action details:** image reference after the prefix (e.g. ` + "`" + `alpine:3.20` + "`" + ` in ` + "`" + `kern:alpine:3.20` + "`" + `).
- **Params:**
  | Key | Type | Required | Meaning |
  |-----|------|----------|---------|
  | ` + "`" + `cmd` + "`" + ` | string or string list | no | Container command; overrides the image default CMD |

Workflow does **not** map mounts, env, or timeout from YAML params.`,
			Outputs: `Plain text written to the task output file inside the box (same contract as ` + "`" + `docker:` + "`" + `). Use ` + "`" + `run` + "`" + ` scripts that write to ` + "`" + `$OUTPUT` + "`" + ` or ` + "`" + `/flowbot/stdout` + "`" + `.`,
			Usage: `- Prefer ` + "`" + `kern:` + "`" + ` when the host has the ` + "`" + `kern` + "`" + ` binary and you want a daemonless rootless box.
- Chain stdout: ` + "`" + `{{step "run_kern" "result"}}` + "`" + `.`,
			Notes: "Requires `kern` on PATH and `kern doctor` passing on Linux. No GPU, overlay networks, or workflow registry credentials (use public images or `kern login` on the host). Differs from `docker:` — see executor.kern in config.",
			ExampleYAML: `  - id: run_kern
    action: kern:alpine:3.20
    params:
      cmd: ["sh", "-c", "echo -n hello > /flowbot/stdout"]
  - id: use_stdout
    action: "mapper:"
    params:
      from_kern: '{{step "run_kern" "result"}}'`,
		},
		{
			Prefix:     "shell:",
			Title:      "Shell",
			Summary:    "Run a shell command on the workflow runner host",
			ActionForm: "shell:<command>",
			Inputs: `- **Action details:** default command after the prefix (e.g. ` + "`" + `echo hello` + "`" + `).
- **Params:**
  | Key | Type | Required | Meaning |
  |-----|------|----------|---------|
  | ` + "`" + `cmd` + "`" + ` | string | no | Replaces the command from the action details when set |`,
			Outputs: `Plain text **stdout** from the process (not JSON).`,
			Usage: `- Prefer an explicit ` + "`" + `shell:` + "`" + ` prefix over free-form actions.
- Chain stdout: ` + "`" + `{{step "echo_host" "result"}}` + "`" + `. Use jsonpath only if stdout is JSON.`,
			Notes: "Runs on the Flowbot host (or sandbox configuration of the shell runtime), not inside an arbitrary container unless the command itself uses docker.",
			ExampleYAML: `  - id: echo_host
    action: shell:echo hello
    params:
      cmd: "echo from params"
  - id: capture
    action: "mapper:"
    params:
      out: '{{step "echo_host" "result"}}'`,
		},
		{
			Prefix:     "machine:",
			Title:      "Machine (SSH)",
			Summary:    "Intended for a named remote machine via SSH runtime",
			ActionForm: "machine:<name>",
			Inputs: `- **Action details:** machine name (e.g. ` + "`" + `vm1` + "`" + `).
- **Params:** typically empty; remote target is meant to come from the machine name.`,
			Outputs: `If routed correctly, remote command stdout as text. **Current workflow routing:** ` + "`" + `DetermineRuntimeType` + "`" + ` does not select the machine engine — ` + "`" + `machine:<name>` + "`" + ` is treated like a shell command whose run string is the machine name. Prefer ` + "`" + `shell:` + "`" + ` / ` + "`" + `docker:` + "`" + ` until SSH routing is fixed.`,
			Usage: `- Do not rely on ` + "`" + `machine:` + "`" + ` in new workflows.
- Use ` + "`" + `shell:` + "`" + ` with an explicit ssh command, or ` + "`" + `docker:` + "`" + `, when remote execution is required.`,
			Notes: "Documented for completeness; avoid in production YAML until workflow selects runtime.Machine.",
			ExampleYAML: `  - id: remote_check
    action: machine:vm1`,
		},
		{
			Prefix:     "mapper:",
			Title:      "Mapper",
			Summary:    "Inline data transform: render params and marshal to JSON (no external runtime)",
			ActionForm: "mapper:",
			Inputs: `- **Action:** must be quoted in YAML: ` + "`" + `action: "mapper:"` + "`" + ` (trailing colon is otherwise invalid YAML).
- **Params:** any KV; string values support templates (` + "`" + `{{input.*}}` + "`" + `, ` + "`" + `{{step ...}}` + "`" + `). Non-string values are kept as structured data in the output object.`,
			Outputs: `JSON **object** string of the fully rendered params, e.g. ` + "`" + `{"message":"hi","tag":"demo"}` + "`" + `. There is no ` + "`" + `data` + "`" + ` wrapper — fields are top-level.`,
			Usage: `- Read a field: ` + "`" + `{{jsonpath (step "build_payload" "result") "message"}}` + "`" + `
- Use as a pure reshape step between capability/shell steps.
- See examples/echo_mapper.yaml.`,
			Notes: "Does not call executor runtimes; failures are template or JSON marshal errors only.",
			ExampleYAML: `  - id: build_payload
    action: "mapper:"
    params:
      message: "{{input.message}}"
      tag: "{{input.tag}}"
      source: cli-example
  - id: read_message
    action: "mapper:"
    params:
      echoed: '{{jsonpath (step "build_payload" "result") "message"}}'`,
		},
		{
			Prefix:     "free-form / echo",
			Title:      "Free-form and echo",
			Summary:    "Actions without a known prefix fall through to shell-style run; bare echo is a special type name",
			ActionForm: "<command> or echo",
			Inputs: `- **Action:** bare ` + "`" + `echo` + "`" + ` or any string without a known prefix (` + "`" + `capability:` + "`" + `/` + "`" + `docker:` + "`" + `/` + "`" + `shell:` + "`" + `/` + "`" + `machine:` + "`" + `/` + "`" + `mapper:` + "`" + `).
- **Params:** optional ` + "`" + `cmd` + "`" + ` (string), same override behavior as shell when treated as a shell run.`,
			Outputs: `Plain text stdout (same as shell).`,
			Usage: `- Prefer ` + "`" + `shell:` + "`" + `, ` + "`" + `docker:` + "`" + `, ` + "`" + `capability:` + "`" + `, or ` + "`" + `mapper:` + "`" + ` in new YAML.
- Avoid free-form for new workflows; keep only for legacy definitions.`,
			Notes: "A bare echo action parses as type echo with empty details; free-form strings become the run command.",
			ExampleYAML: `  - id: legacy_echo
    action: echo`,
		},
	}
}

// generatePlatformSkill writes SKILL.md, references, and examples for one platform skill.
func generatePlatformSkill(meta platformSpec, outputDir string) error {
	if meta.CommandFn == nil {
		return fmt.Errorf("platform skill %q: CommandFn is required", meta.Name)
	}

	dirPath := filepath.Join(outputDir, meta.Name)
	if err := os.MkdirAll(filepath.Join(dirPath, "references"), 0o750); err != nil {
		return fmt.Errorf("create directory %s: %w", dirPath, err)
	}
	if err := os.MkdirAll(filepath.Join(dirPath, "examples"), 0o750); err != nil {
		return fmt.Errorf("create examples directory: %w", err)
	}

	rootCmd := meta.CommandFn()
	cliRoot := meta.CLIRoot
	if cliRoot == "" {
		cliRoot = rootCmd.Name()
	}

	exampleFiles, err := copyEmbeddedExamples(meta.ExampleFS, meta.ExampleDir, filepath.Join(dirPath, "examples"))
	if err != nil {
		return err
	}

	data := platformSkillData{
		Name:               meta.Name,
		Title:              meta.Title,
		CLIRoot:            cliRoot,
		TriggerDescription: buildTriggerDescription(meta.Description, meta.Keywords),
		ScopesNote:         meta.ScopesNote,
		Operations:         extractOperations(rootCmd, cliRoot),
		Workflows:          meta.Workflows,
		StepTypes:          meta.StepTypes,
		ExampleFiles:       exampleFiles,
	}
	if meta.IncludeCapabilityCatalog {
		data.Capabilities = workflowCapabilityCatalog()
	}

	kind := meta.Kind
	if kind == "" {
		kind = "workflow"
	}
	if err := writePlatformSkillFiles(dirPath, data, meta.IncludeCapabilityCatalog, kind); err != nil {
		return err
	}
	for _, name := range exampleFiles {
		_, _ = fmt.Printf("  generated: %s\n", filepath.Join(dirPath, "examples", name))
	}
	return nil
}

func writePlatformSkillFiles(dirPath string, data platformSkillData, withCapabilities bool, kind string) error {
	funcs := newTemplateFuncs()
	skillBody := platformSkillTemplate
	stepsBody := platformStepsTemplate
	schemaBody := platformSchemaTemplate
	capsIndexBody := platformCapabilitiesIndexTemplate
	capFileBody := platformCapabilityFileTemplate
	if kind == "pipeline" {
		skillBody = pipelineSkillTemplate
		stepsBody = pipelineStepsTemplate
		schemaBody = pipelineSchemaTemplate
		capsIndexBody = pipelineCapabilitiesIndexTemplate
		capFileBody = pipelineCapabilityFileTemplate
	}

	skillTmpl, err := template.New("platform_skill").Funcs(funcs).Parse(skillBody)
	if err != nil {
		return fmt.Errorf("parse platform skill template: %w", err)
	}
	cliTmpl, err := template.New("platform_cli").Funcs(funcs).Parse(platformCLIReferenceTemplate)
	if err != nil {
		return fmt.Errorf("parse platform cli template: %w", err)
	}
	stepsTmpl, err := template.New("platform_steps").Funcs(funcs).Parse(stepsBody)
	if err != nil {
		return fmt.Errorf("parse platform steps template: %w", err)
	}

	skillPath := filepath.Join(dirPath, "SKILL.md")
	if err := executeTemplateFile(skillTmpl, skillPath, data); err != nil {
		return fmt.Errorf("write %s: %w", skillPath, err)
	}
	cliPath := filepath.Join(dirPath, "references", "cli.md")
	if err := executeTemplateFile(cliTmpl, cliPath, data); err != nil {
		return fmt.Errorf("write %s: %w", cliPath, err)
	}
	stepsPath := filepath.Join(dirPath, "references", "steps.md")
	if err := executeTemplateFile(stepsTmpl, stepsPath, data); err != nil {
		return fmt.Errorf("write %s: %w", stepsPath, err)
	}

	_, _ = fmt.Printf("  generated: %s\n", skillPath)
	_, _ = fmt.Printf("  generated: %s\n", cliPath)
	_, _ = fmt.Printf("  generated: %s\n", stepsPath)

	if !withCapabilities {
		return nil
	}
	if err := writePlatformSchemaFile(dirPath, funcs, schemaBody); err != nil {
		return err
	}
	return writePlatformCapabilitiesFile(dirPath, data, funcs, capsIndexBody, capFileBody)
}

func writePlatformSchemaFile(dirPath string, funcs template.FuncMap, schemaBody string) error {
	schemaTmpl, err := template.New("platform_schema").Funcs(funcs).Parse(schemaBody)
	if err != nil {
		return fmt.Errorf("parse platform schema template: %w", err)
	}
	schemaPath := filepath.Join(dirPath, "references", "schema.md")
	if err := executeTemplateFile(schemaTmpl, schemaPath, struct{}{}); err != nil {
		return fmt.Errorf("write %s: %w", schemaPath, err)
	}
	_, _ = fmt.Printf("  generated: %s\n", schemaPath)
	return nil
}

func writePlatformCapabilitiesFile(dirPath string, data platformSkillData, funcs template.FuncMap, indexBody, fileBody string) error {
	capsDir := filepath.Join(dirPath, "references", "capabilities")
	if err := os.MkdirAll(capsDir, 0o750); err != nil {
		return fmt.Errorf("create capabilities directory: %w", err)
	}

	indexTmpl, err := template.New("platform_capabilities_index").Funcs(funcs).Parse(indexBody)
	if err != nil {
		return fmt.Errorf("parse platform capabilities index template: %w", err)
	}
	indexPath := filepath.Join(dirPath, "references", "capabilities.md")
	if err := executeTemplateFile(indexTmpl, indexPath, data); err != nil {
		return fmt.Errorf("write %s: %w", indexPath, err)
	}
	_, _ = fmt.Printf("  generated: %s\n", indexPath)

	fileTmpl, err := template.New("platform_capability_file").Funcs(funcs).Parse(fileBody)
	if err != nil {
		return fmt.Errorf("parse platform capability file template: %w", err)
	}
	for _, cap := range data.Capabilities {
		capPath := filepath.Join(capsDir, cap.Type+".md")
		if err := executeTemplateFile(fileTmpl, capPath, cap); err != nil {
			return fmt.Errorf("write %s: %w", capPath, err)
		}
		_, _ = fmt.Printf("  generated: %s\n", capPath)
	}
	return nil
}

// copyEmbeddedExamples copies yaml files from fs root into destDir and returns sorted basenames.
func copyEmbeddedExamples(efs fs.FS, root, destDir string) ([]string, error) {
	entries, err := fs.ReadDir(efs, root)
	if err != nil {
		return nil, fmt.Errorf("read embedded examples %s: %w", root, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		src := filepath.ToSlash(filepath.Join(root, e.Name()))
		data, readErr := fs.ReadFile(efs, src)
		if readErr != nil {
			return nil, fmt.Errorf("read embedded %s: %w", src, readErr)
		}
		dest := filepath.Join(destDir, e.Name())
		if writeErr := os.WriteFile(dest, data, 0o640); writeErr != nil {
			return nil, fmt.Errorf("write example %s: %w", dest, writeErr)
		}
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no yaml examples under %s", root)
	}
	slices.Sort(names)
	return names, nil
}
