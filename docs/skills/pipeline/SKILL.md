---
name: pipeline
description: >-
  Manage Flowbot pipelines via flowbot pipeline: apply YAML definitions to the database (draft+publish), list/get/export/delete, run asynchronously with optional event payload, and inspect runs. Use when the user mentions pipelines, pipeline YAML, pipeline runs, DataEvent, cron/webhook pipeline triggers.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  platform: pipeline
  cli_root: pipeline
---

# Pipeline

Use `flowbot pipeline` for platform pipeline definitions stored in the database.
YAML is an exchange format for `apply` / `export` only — the server runs published definitions from PostgreSQL (not local files).
Prefer the workflows below; load [references/schema.md](references/schema.md) for the full YAML definition,
[references/cli.md](references/cli.md) for flags,
[references/steps.md](references/steps.md) for step fields, templates, and retry, and
[references/capabilities.md](references/capabilities.md) (index) and `references/capabilities/<type>.md` for every capability operation param list.
**Never invent** a capability type that is absent from the capabilities index.
Teaching examples (load via read_skill with path):
- [examples/cron_cleanup.yaml](examples/cron_cleanup.yaml)
- [examples/event_notify.yaml](examples/event_notify.yaml)

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Token scopes: `pipeline:read` for list/get/export/runs; `pipeline:run` for apply/delete/run (run also satisfies read)
4. Prefer `-o json` when parsing results programmatically

## Step shape

Each step uses `capability` + `operation` fields (not workflow `action: capability:<type>.<op>`).

Load [references/steps.md](references/steps.md) for templates and retry.
Load [references/schema.md](references/schema.md) before writing any pipeline YAML.
Load [references/capabilities.md](references/capabilities.md) for the capability index; **must** open `references/capabilities/<type>.md` before emitting that type's operations.

## Templates

Step `params` use Go `text/template` delimiters `{{ }}` (same engine as workflows).

**Variables available in pipelines:**

| Variable | Access | Source |
|----------|--------|--------|
| Event payload | `{{event "url"}}` / `{{.Event.url}}` | DataEvent / `pipeline run --event` |
| Prior steps | `{{step "name" "field"}}` / `{{.Steps.name.field}}` | Completed step outputs |
| Env | `{{.Env.HOME}}` | Runner environment (when populated) |

Not set for pipelines: workflow-style `input` / `.Input` (use `event` instead).
Helpers (closed set): `event`, `step`, `input`, `jsonpath`, `jsonpathExists`, `jsonpathRaw`, `default`, `json`, `len`, `join`, `split`, `contains`, `now`, plus Go builtins `if`/`else`/`range`/`eq`/`printf`.
Full list: [references/steps.md](references/steps.md#templates).
**Never invent** a helper absent from that list.

## Workflows

### Write or edit a pipeline YAML

When the user needs a new or updated pipeline definition:
1. Load references/schema.md and copy the skeleton (name, enabled, triggers, steps).
2. For each step, open references/capabilities/<type>.md first; never invent types missing from the capabilities index.
3. Use only documented template helpers from references/steps.md; prefer event/step over workflow input.
4. Use examples/event_notify.yaml or examples/cron_cleanup.yaml as starting points.
5. `flowbot pipeline apply --file path/to/pipeline.yaml`
6. `flowbot pipeline get <name>`
7. Optional: flowbot pipeline run <name> --event '{...}' then flowbot pipeline runs <name>.

### Apply a definition from YAML

When the user already has a pipeline YAML file to create or replace (publishes immediately):
1. Validate against references/schema.md: name, triggers, steps, and event/step templates.
2. `flowbot pipeline apply --file path/to/pipeline.yaml`
3. `flowbot pipeline get <name>`

### List and inspect

When the user asks what published pipelines exist:
1. `flowbot pipeline list`
2. `flowbot pipeline get <name>`
3. Optional: flowbot pipeline export <name> -o file.yaml to round-trip published YAML.

### Run a pipeline

When the user wants to execute a published pipeline manually:
1. Build optional event JSON for {{event.*}} keys (cron-only may use {}).
2. `flowbot pipeline run <name> --event '{"url":"...","title":"..."}'`
3. Note the returned run_id (runs are asynchronous).
4. `flowbot pipeline runs <name>`

### Delete

When the user wants to remove a definition (run history is deleted, including compound trigger runs):
1. `flowbot pipeline delete <name>`

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| insufficient scope | token needs `pipeline:read` and/or `pipeline:run` |
| pipeline name is required / not found | apply first; check `list`; draft-only pipelines are invisible to CLI |
| conflict / 409 | someone else changed the draft; re-`export`/edit and `apply` again |
| pipeline is disabled | set `enabled: true` in YAML and re-apply |
| `function "…" not defined` | replace with a helper from [references/steps.md](references/steps.md#templates) |
