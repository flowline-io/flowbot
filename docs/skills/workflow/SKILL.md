---
name: workflow
description: >-
  Manage Flowbot workflows via flowbot workflow: apply YAML definitions to the
  database, list/get/export/delete, run asynchronously, and inspect runs. Also
  covers the Automate → Workflows web UI (list, detail DAG/YAML, run history).
  Use when the user mentions workflows, workflow YAML, workflow runs,
  cron/webhook workflow triggers, or Automate → Workflows.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  platform: workflow
  cli_root: workflow
---

# Workflow

Use `flowbot workflow` for platform workflow definitions stored in the database.
YAML is an exchange format for `apply` / `export` only — the server does not run
from local files. Prefer the workflows below; load [references/cli.md](references/cli.md)
when you need a flag or response field not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Token scopes: `workflow:read` for list/get/export/runs; `workflow:run` for apply/delete/run (run also satisfies read)
4. Prefer `-o json` when parsing results programmatically

## Workflows

### Apply a definition from YAML

When the user has a workflow YAML file to create or replace:

1. Ensure the YAML has `name`, `pipeline`, `tasks`, and `inputs` for any `{{input.*}}` used in params
2. `flowbot workflow apply --file path/to/workflow.yaml`
3. Confirm with `flowbot workflow get <name>`

### List and inspect

When the user asks what workflows exist or what a workflow contains:

1. `flowbot workflow list`
2. `flowbot workflow get <name>`
3. Optional: `flowbot workflow export <name> -o file.yaml` to round-trip YAML

### Run a workflow

When the user wants to execute a stored workflow:

1. Build input JSON matching declared `inputs` (required fields must be present)
2. `flowbot workflow run <name> --input '{"url":"...","title":"..."}'`
3. Note the returned `run_id` (runs are asynchronous)
4. `flowbot workflow runs <name>` to check status

### Delete

When the user wants to remove a definition (run history is kept):

1. `flowbot workflow delete <name>`

### Web UI

When the user wants a browser UI instead of CLI:

1. Open `/service/web/workflows` (Automate → Workflows)
2. List shows Last Run and Enable/Disable
3. Detail Overview: Inputs, Triggers, Execution DAG, Run now; YAML tab for exported text
4. Runs page: expand a run for step Input/Output/Error

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| insufficient scope | token needs `workflow:read` and/or `workflow:run` |
| workflow name is required / not found | apply first; check `list` |
| input validation failed | supply all required `inputs` with correct types |
| webhook rejected | workflow must be `enabled`; trigger needs `auth.token` or `auth.hmac_secret` |
