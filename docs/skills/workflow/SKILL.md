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
Prefer the workflows below; load [references/schema.md](references/schema.md) for the full YAML definition,
[references/cli.md](references/cli.md) for flags,
[references/steps.md](references/steps.md) for task action types, inputs/outputs, and usage, and
[references/capabilities.md](references/capabilities.md) (index) and `references/capabilities/<type>.md` for every `capability:<type>.<op>` param list.
**Never invent** a `capability:<type>` that is absent from the capabilities index.
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
| `capability:` | Invoke a Flowbot capability operation (provider or CapCore) |
| `docker:` | Run a container image via the Docker runtime on the workflow runner |
| `shell:` | Run a shell command on the workflow runner host |
| `machine:` | Intended for a named remote machine via SSH runtime |
| `mapper:` | Inline data transform: render params and marshal to JSON (no external runtime) |
| `free-form / echo` | Actions without a known prefix fall through to shell-style run; bare echo is a special type name |

Load [references/steps.md](references/steps.md) for per-type inputs/outputs, usage, templates, and `conn`/`retry`.
Load [references/schema.md](references/schema.md) before writing any workflow YAML.
Load [references/capabilities.md](references/capabilities.md) for the capability index; **must** open `references/capabilities/<type>.md` before emitting that type's actions.

## Templates

Task `params` use Go `text/template` delimiters `{{ }}` (same engine as pipelines).

**Variables available in workflows:**

| Variable | Access | Source |
|----------|--------|--------|
| Run inputs | `{{input "url"}}` / `{{input.url}}` / `{{.Input.url}}` | `workflow run --input` (keys = declared `inputs`) |
| Prior steps | `{{step "id" "result"}}` / `{{.Steps.id.result}}` | Completed task outputs (`result` and `id` hold the same payload) |

Not set for workflows: `event` / `.Event`, `env` / `.Env`.
Helpers (closed set): `input`, `step`, `event`, `jsonpath`, `jsonpathExists`, `jsonpathRaw`, `default`, `json`, `len`, `join`, `split`, `contains`, `now`, plus Go builtins `if`/`else`/`range`/`eq`/`printf`.
Full list: [references/steps.md](references/steps.md#templates).
**Never invent** a helper absent from that list (no Sprig / `date` / other template libraries).

## Workflows

### Write or edit a workflow YAML

When the user needs a new or updated workflow definition:
1. Load references/schema.md and copy the skeleton (name, enabled, triggers, pipeline, tasks, inputs).
2. For each capability: action, open references/capabilities/<type>.md first; never invent types missing from the capabilities index.
3. Use only documented template helpers from references/steps.md; never invent helpers.
4. Use examples/echo_mapper.yaml, examples/save_and_track.yaml, or examples/parallel_example.yaml as starting points.
5. Declare inputs for every {{input.*}} key; use jsonpath for prior capability results (see capabilities.md Common data paths).
6. `flowbot workflow apply --file path/to/workflow.yaml`
7. `flowbot workflow get <name>`
8. Optional: flowbot workflow run <name> --input '{...}' then flowbot workflow runs <name>.

### Apply a definition from YAML

When the user already has a workflow YAML file to create or replace:
1. Validate against references/schema.md: name, pipeline, tasks, triggers, and inputs for any {{input.*}} used in params.
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
| `function "…" not defined` | replace with a helper from [references/steps.md](references/steps.md#templates); do not invent helpers |
| webhook rejected | workflow must be `enabled`; trigger needs `auth.token` or `auth.hmac_secret` |
