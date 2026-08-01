package skills

import (
	"embed"

	"github.com/flowline-io/flowbot/cmd/cli/command"
)

//go:embed testdata/pipeline/*.yaml
var pipelineExampleFS embed.FS

const pipelineSkillTemplate = `---
name: {{.Name}}
description: >-
  {{.TriggerDescription}}
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  platform: {{.Name}}
  cli_root: {{.CLIRoot}}
---

# {{.Title}}

Use ` + "`" + `flowbot {{.CLIRoot}}` + "`" + ` for platform pipeline definitions stored in the database.
YAML is an exchange format for ` + "`" + `apply` + "`" + ` / ` + "`" + `export` + "`" + ` only — the server runs published definitions from PostgreSQL (not local files).
Prefer the workflows below; load [references/schema.md](references/schema.md) for the full YAML definition,
[references/cli.md](references/cli.md) for flags,
[references/steps.md](references/steps.md) for step fields, templates, and retry, and
[references/capabilities.md](references/capabilities.md) (index) and ` + "`" + `references/capabilities/<type>.md` + "`" + ` for every capability operation param list.
**Never invent** a capability type that is absent from the capabilities index.
Teaching examples (load via read_skill with path):
{{- range .ExampleFiles}}
- [examples/{{.}}](examples/{{.}})
{{- end}}

## Setup

1. Ensure CLI auth: ` + "`" + `flowbot login` + "`" + `
2. Set server via ` + "`" + `FLOWBOT_SERVER_URL` + "`" + ` or ` + "`" + `--server-url` + "`" + `; optional ` + "`" + `--profile` + "`" + `, ` + "`" + `--debug` + "`" + ` / ` + "`" + `-d` + "`" + `
3. {{.ScopesNote}}
4. Prefer ` + "`" + `-o json` + "`" + ` when parsing results programmatically

## Step shape

Each step uses ` + "`" + `capability` + "`" + ` + ` + "`" + `operation` + "`" + ` fields (not workflow ` + "`" + `action: capability:<type>.<op>` + "`" + `).

Load [references/steps.md](references/steps.md) for templates and retry.
Load [references/schema.md](references/schema.md) before writing any pipeline YAML.
Load [references/capabilities.md](references/capabilities.md) for the capability index; **must** open ` + "`" + `references/capabilities/<type>.md` + "`" + ` before emitting that type's operations.

## Templates

Step ` + "`" + `params` + "`" + ` use Go ` + "`" + `text/template` + "`" + ` delimiters ` + "`" + `{{"{{ }}"}}` + "`" + ` (same engine as workflows).

**Variables available in pipelines:**

| Variable | Access | Source |
|----------|--------|--------|
| Event payload | ` + "`" + `{{"{{event \"url\"}}"}}` + "`" + ` / ` + "`" + `{{"{{.Event.url}}"}}` + "`" + ` | DataEvent / ` + "`" + `pipeline run --event` + "`" + ` |
| Prior steps | ` + "`" + `{{"{{step \"name\" \"field\"}}"}}` + "`" + ` / ` + "`" + `{{"{{.Steps.name.field}}"}}` + "`" + ` | Completed step outputs |
| Env | ` + "`" + `{{"{{.Env.HOME}}"}}` + "`" + ` | Runner environment (when populated) |

Not set for pipelines: workflow-style ` + "`" + `input` + "`" + ` / ` + "`" + `.Input` + "`" + ` (use ` + "`" + `event` + "`" + ` instead).
Helpers (closed set): ` + "`" + `event` + "`" + `, ` + "`" + `step` + "`" + `, ` + "`" + `input` + "`" + `, ` + "`" + `jsonpath` + "`" + `, ` + "`" + `jsonpathExists` + "`" + `, ` + "`" + `jsonpathRaw` + "`" + `, ` + "`" + `default` + "`" + `, ` + "`" + `json` + "`" + `, ` + "`" + `len` + "`" + `, ` + "`" + `join` + "`" + `, ` + "`" + `split` + "`" + `, ` + "`" + `contains` + "`" + `, ` + "`" + `now` + "`" + `, plus Go builtins ` + "`" + `if` + "`" + `/` + "`" + `else` + "`" + `/` + "`" + `range` + "`" + `/` + "`" + `eq` + "`" + `/` + "`" + `printf` + "`" + `.
Full list: [references/steps.md](references/steps.md#templates).
**Never invent** a helper absent from that list.

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
| insufficient scope | token needs ` + "`" + `pipeline:read` + "`" + ` and/or ` + "`" + `pipeline:run` + "`" + ` |
| pipeline name is required / not found | apply first; check ` + "`" + `list` + "`" + `; draft-only pipelines are invisible to CLI |
| conflict / 409 | someone else changed the draft; re-` + "`" + `export` + "`" + `/edit and ` + "`" + `apply` + "`" + ` again |
| pipeline is disabled | set ` + "`" + `enabled: true` + "`" + ` in YAML and re-apply |
| ` + "`" + `function "…" not defined` + "`" + ` | replace with a helper from [references/steps.md](references/steps.md#templates) |
`

const pipelineStepsTemplate = `# Pipeline steps reference

Load this file when authoring or editing pipeline YAML steps. Teaching examples:
{{- range .ExampleFiles}}
- [examples/{{.}}](../examples/{{.}})
{{- end}}

For every capability operation param table, start at [capabilities.md](capabilities.md) and open ` + "`" + `capabilities/<type>.md` + "`" + ` **before** writing that step. Do not invent types missing from the index.

Top-level definition fields and triggers: [schema.md](schema.md).

## Shared step fields

| Field | Required | Notes |
|-------|----------|-------|
| ` + "`" + `name` + "`" + ` | yes | Unique within the pipeline |
| ` + "`" + `capability` + "`" + ` | yes | Capability type (e.g. ` + "`" + `core` + "`" + `, ` + "`" + `karakeep` + "`" + `) |
| ` + "`" + `operation` + "`" + ` | yes | Operation name on that capability |
| ` + "`" + `params` + "`" + ` | no | Template-rendered before execution |
| ` + "`" + `retry` + "`" + ` | no | ` + "`" + `max_attempts` + "`" + `, ` + "`" + `delay` + "`" + `, ` + "`" + `backoff` + "`" + `, ` + "`" + `max_delay` + "`" + `, ` + "`" + `jitter` + "`" + `, ` + "`" + `retry_on` + "`" + ` |

## Templates

Pipeline step ` + "`" + `params` + "`" + ` (string values) are rendered with Go ` + "`" + `text/template` + "`" + ` before the step runs.
Delimiters are ` + "`" + `{{"{{"}}` + "`" + ` and ` + "`" + `{{"}}"}}` + "`" + `.

### Available variables

| Variable | Populated? | How to read | Source |
|----------|------------|-------------|--------|
| ` + "`" + `Event` + "`" + ` | yes | ` + "`" + `{{"{{event \"field\"}}"}}` + "`" + ` / ` + "`" + `{{"{{.Event.field}}"}}` + "`" + ` | DataEvent payload or ` + "`" + `pipeline run --event` + "`" + ` JSON |
| ` + "`" + `Steps` + "`" + ` | yes | ` + "`" + `{{"{{step \"step_name\" \"field\"}}"}}` + "`" + ` | Outputs of **already completed** steps |
| ` + "`" + `Env` + "`" + ` | sometimes | ` + "`" + `{{"{{.Env.HOME}}"}}` + "`" + ` | Process environment on the runner |
| ` + "`" + `Input` + "`" + ` | no | — | Workflow-only; do not rely on it in pipelines |

### Helper functions

**Closed set only.** Missing helpers fail at run time with ` + "`" + `function "…" not defined` + "`" + `.

| Helper | Example |
|--------|---------|
| ` + "`" + `event` + "`" + ` | ` + "`" + `{{"{{event \"url\"}}"}}` + "`" + ` |
| ` + "`" + `step` + "`" + ` | ` + "`" + `{{"{{step \"fetch\" \"count\"}}"}}` + "`" + ` |
| ` + "`" + `jsonpath` + "`" + ` | ` + "`" + `{{"{{jsonpath (step \"api\" \"result\") \"data.id\"}}"}}` + "`" + ` |
| ` + "`" + `default` + "`" + ` | ` + "`" + `{{"{{default \"guest\" (event \"user\")}}"}}` + "`" + ` |
| ` + "`" + `json` + "`" + ` / ` + "`" + `len` + "`" + ` / ` + "`" + `join` + "`" + ` / ` + "`" + `split` + "`" + ` / ` + "`" + `contains` + "`" + ` / ` + "`" + `now` + "`" + ` | see workflow steps docs for shapes |

YAML tip: when an expression contains quotes, wrap the param value in single quotes:

` + "```yaml" + `
params:
  title: 'Item: {{"{{event \"title\"}}"}}'
  url: "{{"{{event.url}}"}}"
` + "```" + `
`

const pipelineSchemaTemplate = `# Pipeline YAML schema

Canonical shape is ` + "`" + `pipeline.EditorDefinition` + "`" + ` (DB published YAML). Load this file before writing or editing a definition.
Step details: [steps.md](steps.md). Capability params: [capabilities.md](capabilities.md).

` + "```yaml" + `
name: example_pipeline
description: "Human-readable description"
enabled: true
resumable: false
triggers:
  - type: event
    enabled: true
    event: demo.item.created
  - type: cron
    enabled: false
    cron: "0 3 * * *"
    cron_timeout: "30m"
steps:
  - name: notify
    capability: core
    operation: notify_send
    params:
      template_id: "demo.new_item"
      channels: ["slack"]
      payload:
        title: '{{"{{event \"title\"}}"}}'
` + "```" + `

## Top-level fields

| Field | Required | Notes |
|-------|----------|-------|
| ` + "`" + `name` + "`" + ` | yes | Unique; validated by ` + "`" + `pipeline.ValidateName` + "`" + ` |
| ` + "`" + `description` + "`" + ` | no | Human-readable |
| ` + "`" + `enabled` + "`" + ` | no | Published disabled pipelines appear in ` + "`" + `list` + "`" + ` but cannot ` + "`" + `run` + "`" + ` / load into the engine |
| ` + "`" + `resumable` + "`" + ` | no | Checkpoint + restart recovery |
| ` + "`" + `triggers` + "`" + ` | yes | Array of event / cron / webhook triggers |
| ` + "`" + `steps` + "`" + ` | yes | Ordered capability invocations |

## Triggers

| Type | Fields |
|------|--------|
| ` + "`" + `event` + "`" + ` | ` + "`" + `event` + "`" + ` (DataEvent.EventType) |
| ` + "`" + `cron` + "`" + ` | ` + "`" + `cron` + "`" + `, optional ` + "`" + `cron_timeout` + "`" + ` |
| ` + "`" + `webhook` + "`" + ` | ` + "`" + `webhook.path` + "`" + `, auth token and/or hmac_secret |

## Authoring checklist

1. Copy the skeleton; set ` + "`" + `name` + "`" + ` and ` + "`" + `enabled: true` + "`" + `.
2. Add at least one enabled trigger.
3. For each step, open ` + "`" + `capabilities/<type>.md` + "`" + ` and fill required params.
4. Use ` + "`" + `event` + "`" + ` / ` + "`" + `step` + "`" + ` helpers only from [steps.md](steps.md).
5. ` + "`" + `flowbot pipeline apply --file ...` + "`" + ` then ` + "`" + `get` + "`" + ` / ` + "`" + `run` + "`" + `.
`

const pipelineCapabilitiesIndexTemplate = `# Pipeline capability operations reference

Load a capability file below when authoring pipeline steps with ` + "`" + `capability` + "`" + ` + ` + "`" + `operation` + "`" + `.
Param tables are generated from each capability's ` + "`" + `OpDef.Input` + "`" + ` (same source as hub registration).

**Rules for agents:**

1. Only use capability types listed in the table below.
2. Before emitting any operation, load ` + "`" + `capabilities/<type>.md` + "`" + ` and copy required params from that file.
3. Do not invent ops, param keys, or types.
4. Pipeline YAML uses fields ` + "`" + `capability` + "`" + ` and ` + "`" + `operation` + "`" + ` (not workflow ` + "`" + `action: capability:<type>.<op>` + "`" + `).

## Result envelope

Capability steps expose an ` + "`" + `InvokeResult` + "`" + `-shaped payload readable via ` + "`" + `{{"{{step \"name\" \"…\"}}"}}` + "`" + ` / ` + "`" + `jsonpath` + "`" + `.

## Capabilities

| Capability | Reference |
|------------|-----------|
{{- range .Capabilities}}
| ` + "`" + `{{.Type}}` + "`" + ` | [capabilities/{{.Type}}.md](capabilities/{{.Type}}.md) — {{.Description}} |
{{- end}}
`

const pipelineCapabilityFileTemplate = `# ` + "`" + `{{.Type}}` + "`" + ` capability operations

{{.Description}}

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

{{- range .Ops}}

## ` + "`" + `{{.Name}}` + "`" + `

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

**Usage:**

` + "```yaml" + `
  - name: {{.Name}}_step
    capability: {{$.Type}}
    operation: {{.Name}}
{{- if .Inputs}}
    params:
{{- range .Inputs}}
      {{.Name}}: {{if eq .Type "string"}}"..."{{else if eq .Type "[]string"}}["..."]{{else if or (eq .Type "int") (eq .Type "number")}}0{{else if eq .Type "bool"}}false{{else if eq .Type "map[string]any"}}{}{{else}}...{{end}}{{if .Required}}  # required{{end}}
{{- end}}
{{- end}}
` + "```" + `
{{- end}}
`

// platformPipelineSpec returns the platform skill definition for flowbot pipeline.
func platformPipelineSpec() platformSpec {
	return platformSpec{
		Name:                     "pipeline",
		Title:                    "Pipeline",
		CLIRoot:                  "pipeline",
		Kind:                     "pipeline",
		CommandFn:                command.PipelineCommand,
		Description:              "Manage Flowbot pipelines via flowbot pipeline: apply YAML definitions to the database (draft+publish), list/get/export/delete, run asynchronously with optional event payload, and inspect runs.",
		Keywords:                 "pipelines, pipeline YAML, pipeline runs, DataEvent, cron/webhook pipeline triggers",
		ScopesNote:               "Token scopes: `pipeline:read` for list/get/export/runs; `pipeline:run` for apply/delete/run (run also satisfies read)",
		ExampleFS:                pipelineExampleFS,
		ExampleDir:               "testdata/pipeline",
		IncludeCapabilityCatalog: true,
		Workflows: []workflowSpec{
			{
				Title:       "Write or edit a pipeline YAML",
				Description: "When the user needs a new or updated pipeline definition:",
				Steps: []workflowStep{
					{Step: 1, Note: "Load references/schema.md and copy the skeleton (name, enabled, triggers, steps)."},
					{Step: 2, Note: "For each step, open references/capabilities/<type>.md first; never invent types missing from the capabilities index."},
					{Step: 3, Note: "Use only documented template helpers from references/steps.md; prefer event/step over workflow input."},
					{Step: 4, Note: "Use examples/event_notify.yaml or examples/cron_cleanup.yaml as starting points."},
					{Step: 5, Command: "flowbot pipeline apply --file path/to/pipeline.yaml"},
					{Step: 6, Command: "flowbot pipeline get <name>"},
					{Step: 7, Note: "Optional: flowbot pipeline run <name> --event '{...}' then flowbot pipeline runs <name>."},
				},
			},
			{
				Title:       "Apply a definition from YAML",
				Description: "When the user already has a pipeline YAML file to create or replace (publishes immediately):",
				Steps: []workflowStep{
					{Step: 1, Note: "Validate against references/schema.md: name, triggers, steps, and event/step templates."},
					{Step: 2, Command: "flowbot pipeline apply --file path/to/pipeline.yaml"},
					{Step: 3, Command: "flowbot pipeline get <name>"},
				},
			},
			{
				Title:       "List and inspect",
				Description: "When the user asks what published pipelines exist:",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot pipeline list"},
					{Step: 2, Command: "flowbot pipeline get <name>"},
					{Step: 3, Note: "Optional: flowbot pipeline export <name> -o file.yaml to round-trip published YAML."},
				},
			},
			{
				Title:       "Run a pipeline",
				Description: "When the user wants to execute a published pipeline manually:",
				Steps: []workflowStep{
					{Step: 1, Note: "Build optional event JSON for {{event.*}} keys (cron-only may use {})."},
					{Step: 2, Command: "flowbot pipeline run <name> --event '{\"url\":\"...\",\"title\":\"...\"}'"},
					{Step: 3, Note: "Note the returned run_id (runs are asynchronous)."},
					{Step: 4, Command: "flowbot pipeline runs <name>"},
				},
			},
			{
				Title:       "Delete",
				Description: "When the user wants to remove a definition (run history is deleted, including compound trigger runs):",
				Steps: []workflowStep{
					{Step: 1, Command: "flowbot pipeline delete <name>"},
				},
			},
		},
	}
}
