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
	Params      string
	Notes       string
	ExampleYAML string
}

// platformSpec holds metadata for a non-capability (platform) skill.
type platformSpec struct {
	Name        string
	Title       string
	Description string
	Keywords    string
	CLIRoot     string
	ScopesNote  string
	Workflows   []workflowSpec
	StepTypes   []stepTypeSpec
	CommandFn   func() *cobra.Command
	ExampleFS   embed.FS
	ExampleDir  string
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
Prefer the workflows below; load [references/cli.md](references/cli.md) for flags and
[references/steps.md](references/steps.md) for task action types and params.
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

Load [references/steps.md](references/steps.md) for params, templates, and ` + "`" + `conn` + "`" + `/` + "`" + `retry` + "`" + `.

## Templates

Task ` + "`" + `params` + "`" + ` use Go ` + "`" + `text/template` + "`" + ` delimiters ` + "`" + `{{"{{ }}"}}` + "`" + ` (same engine as pipelines).

**Variables available in workflows:**

| Variable | Access | Source |
|----------|--------|--------|
| Run inputs | ` + "`" + `{{"{{input \"url\"}}"}}` + "`" + ` / ` + "`" + `{{"{{input.url}}"}}` + "`" + ` / ` + "`" + `{{"{{.Input.url}}"}}` + "`" + ` | ` + "`" + `workflow run --input` + "`" + ` (keys = declared ` + "`" + `inputs` + "`" + `) |
| Prior steps | ` + "`" + `{{"{{step \"id\" \"result\"}}"}}` + "`" + ` / ` + "`" + `{{"{{.Steps.id.result}}"}}` + "`" + ` | Completed task outputs (` + "`" + `result` + "`" + ` and ` + "`" + `id` + "`" + ` hold the same payload) |

Not set for workflows: ` + "`" + `event` + "`" + ` / ` + "`" + `.Event` + "`" + `, ` + "`" + `env` + "`" + ` / ` + "`" + `.Env` + "`" + `. Helpers: ` + "`" + `jsonpath` + "`" + `, ` + "`" + `default` + "`" + `, ` + "`" + `json` + "`" + `, ` + "`" + `join` + "`" + `, ` + "`" + `if` + "`" + `/` + "`" + `else` + "`" + `. Full list: [references/steps.md](references/steps.md#templates).

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

## Shared task fields

| Field | Required | Notes |
|-------|----------|-------|
| ` + "`" + `id` + "`" + ` | yes | Unique within the workflow |
| ` + "`" + `action` + "`" + ` | yes | See action types below |
| ` + "`" + `describe` + "`" + ` | no | Human-readable label |
| ` + "`" + `params` + "`" + ` | no | Template-rendered before execution; declare matching top-level ` + "`" + `inputs` + "`" + ` when using ` + "`" + `{{"{{input.*}}"}}` + "`" + ` |
| ` + "`" + `conn` + "`" + ` | no | Upstream task ids (DAG edges; required for parallel scheduling) |
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

**Params:** {{.Params}}

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
					{Step: 1, Note: "Pick action types from the Step types table; load references/steps.md for params and templates."},
					{Step: 2, Note: "Use examples/echo_mapper.yaml, examples/save_and_track.yaml, or examples/parallel_example.yaml as starting points."},
					{Step: 3, Note: "Ensure name, pipeline, tasks, and inputs for any {{input.*}} used in params."},
					{Step: 4, Command: "flowbot workflow apply --file path/to/workflow.yaml"},
					{Step: 5, Command: "flowbot workflow get <name>"},
					{Step: 6, Note: "Optional: flowbot workflow run <name> --input '{...}' then flowbot workflow runs <name>."},
				},
			},
			{
				Title:       "Apply a definition from YAML",
				Description: "When the user already has a workflow YAML file to create or replace:",
				Steps: []workflowStep{
					{Step: 1, Note: "Ensure the YAML has name, pipeline, tasks, and inputs for any {{input.*}} used in params."},
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
		StepTypes: workflowStepTypeSpecs(),
	}
}

// workflowStepTypeSpecs returns documented workflow task action types.
func workflowStepTypeSpecs() []stepTypeSpec {
	return []stepTypeSpec{
		{
			Prefix:     "capability:",
			Title:      "Capability",
			Summary:    "Invoke a Flowbot capability operation",
			ActionForm: "capability:<type>.<operation>",
			Params:     "KV object passed to the capability after template render. Keys depend on the operation — use the matching capability skill (e.g. karakeep, kanboard) for field details; do not invent provider-specific keys.",
			Notes:      "Example: capability:karakeep.create with params.url. See examples/save_and_track.yaml.",
			ExampleYAML: `  - id: save_bookmark
    action: capability:karakeep.create
    params:
      url: "{{input.url}}"`,
		},
		{
			Prefix:     "docker:",
			Title:      "Docker",
			Summary:    "Run a container image via the Docker runtime",
			ActionForm: "docker:<image>",
			Params:     "Optional `cmd` (string or string list) overrides the container command.",
			Notes:      "Image is taken from the action details (e.g. docker:alpine:3.20).",
			ExampleYAML: `  - id: run_tool
    action: docker:alpine:3.20
    params:
      cmd: ["echo", "hello"]`,
		},
		{
			Prefix:     "shell:",
			Title:      "Shell",
			Summary:    "Run a shell command on the workflow runner host",
			ActionForm: "shell:<command>",
			Params:     "Optional `cmd` (string) replaces the command from the action details.",
			Notes:      "Prefer explicit shell: prefix over free-form actions.",
			ExampleYAML: `  - id: echo_host
    action: shell:echo hello
    params:
      cmd: "echo from params"`,
		},
		{
			Prefix:     "machine:",
			Title:      "Machine (SSH)",
			Summary:    "Run on a named remote machine via SSH runtime",
			ActionForm: "machine:<name>",
			Params:     "Typically empty; remote target comes from the machine name in the action.",
			Notes:      "Requires the machine runtime to be configured on the server.",
			ExampleYAML: `  - id: remote_check
    action: machine:vm1`,
		},
		{
			Prefix:     "mapper:",
			Title:      "Mapper",
			Summary:    "Inline data transform: render params and marshal to JSON (no external runtime)",
			ActionForm: "mapper:",
			Params:     "Any KV; values support templates. The rendered object is stored as the step result JSON string.",
			Notes:      "Quote the action in YAML (`action: \"mapper:\"`) because a trailing colon is otherwise invalid. See examples/echo_mapper.yaml.",
			ExampleYAML: `  - id: build_payload
    action: "mapper:"
    params:
      message: "{{input.message}}"
      tag: "{{input.tag}}"`,
		},
		{
			Prefix:     "free-form / echo",
			Title:      "Free-form and echo",
			Summary:    "Actions without a known prefix fall through to shell-style run; bare echo is a special type name",
			ActionForm: "<command> or echo",
			Params:     "Same optional `cmd` override behavior as shell when treated as a shell run.",
			Notes:      "Prefer shell:, docker:, capability:, or mapper: in new YAML. A bare echo action parses as type echo with empty details; free-form strings become the run command. Avoid relying on free-form for new workflows.",
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

	funcs := newTemplateFuncs()
	skillTmpl, err := template.New("platform_skill").Funcs(funcs).Parse(platformSkillTemplate)
	if err != nil {
		return fmt.Errorf("parse platform skill template: %w", err)
	}
	cliTmpl, err := template.New("platform_cli").Funcs(funcs).Parse(platformCLIReferenceTemplate)
	if err != nil {
		return fmt.Errorf("parse platform cli template: %w", err)
	}
	stepsTmpl, err := template.New("platform_steps").Funcs(funcs).Parse(platformStepsTemplate)
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
	for _, name := range exampleFiles {
		_, _ = fmt.Printf("  generated: %s\n", filepath.Join(dirPath, "examples", name))
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
