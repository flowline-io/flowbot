---
name: workflow
description: >-
  Manage Flowbot workflows via flowbot workflow: apply YAML definitions to the database, list/get/export/delete, run asynchronously, and inspect runs. Use when the user mentions workflows, workflow YAML, workflow runs, cron/webhook workflow triggers.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  platform: workflow
  cli_root: workflow
---

# Workflow

Use `flowbot workflow` for platform workflow definitions stored in the database.
YAML is an exchange format for `apply` / `export` only — the server does not run from local files.
Prefer the workflows below; load [references/cli.md](references/cli.md) for flags and
[references/steps.md](references/steps.md) for task action types and params.
Teaching examples (load via read_skill with path):
- [examples/echo_mapper.yaml](examples/echo_mapper.yaml)
- [examples/parallel_example.yaml](examples/parallel_example.yaml)
- [examples/save_and_track.yaml](examples/save_and_track.yaml)

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Token scopes: `workflow:read` for list/get/export/runs; `workflow:run` for apply/delete/run (run also satisfies read)
4. Prefer `-o json` when parsing results programmatically

## Step types

| Prefix | Use |
|--------|-----|
| `capability:` | Invoke a Flowbot capability operation |
| `docker:` | Run a container image via the Docker runtime |
| `shell:` | Run a shell command on the workflow runner host |
| `machine:` | Run on a named remote machine via SSH runtime |
| `mapper:` | Inline data transform: render params and marshal to JSON (no external runtime) |
| `free-form / echo` | Actions without a known prefix fall through to shell-style run; bare echo is a special type name |

Load [references/steps.md](references/steps.md) for params, templates, and `conn`/`retry`.

## Templates

Task `params` use Go `text/template` delimiters `{{ }}` (same engine as pipelines).

**Variables available in workflows:**

| Variable | Access | Source |
|----------|--------|--------|
| Run inputs | `{{input "url"}}` / `{{input.url}}` / `{{.Input.url}}` | `workflow run --input` (keys = declared `inputs`) |
| Prior steps | `{{step "id" "result"}}` / `{{.Steps.id.result}}` | Completed task outputs (`result` and `id` hold the same payload) |

Not set for workflows: `event` / `.Event`, `env` / `.Env`. Helpers: `jsonpath`, `default`, `json`, `join`, `if`/`else`. Full list: [references/steps.md](references/steps.md#templates).

## Workflows

### Write or edit a workflow YAML

When the user needs a new or updated workflow definition:
1. Pick action types from the Step types table; load references/steps.md for params and templates.
2. Use examples/echo_mapper.yaml, examples/save_and_track.yaml, or examples/parallel_example.yaml as starting points.
3. Ensure name, pipeline, tasks, and inputs for any {{input.*}} used in params.
4. `flowbot workflow apply --file path/to/workflow.yaml`
5. `flowbot workflow get <name>`
6. Optional: flowbot workflow run <name> --input '{...}' then flowbot workflow runs <name>.

### Apply a definition from YAML

When the user already has a workflow YAML file to create or replace:
1. Ensure the YAML has name, pipeline, tasks, and inputs for any {{input.*}} used in params.
2. `flowbot workflow apply --file path/to/workflow.yaml`
3. `flowbot workflow get <name>`

### List and inspect

When the user asks what workflows exist or what a workflow contains:
1. `flowbot workflow list`
2. `flowbot workflow get <name>`
3. Optional: flowbot workflow export <name> -o file.yaml to round-trip YAML.

### Run a workflow

When the user wants to execute a stored workflow:
1. Build input JSON matching declared inputs (required fields must be present).
2. `flowbot workflow run <name> --input '{"url":"...","title":"..."}'`
3. Note the returned run_id (runs are asynchronous).
4. `flowbot workflow runs <name>`

### Delete

When the user wants to remove a definition (run history is kept):
1. `flowbot workflow delete <name>`

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| insufficient scope | token needs `workflow:read` and/or `workflow:run` |
| workflow name is required / not found | apply first; check `list` |
| input validation failed | supply all required `inputs` with correct types |
| webhook rejected | workflow must be `enabled`; trigger needs `auth.token` or `auth.hmac_secret` |
