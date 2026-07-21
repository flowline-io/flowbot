# flowbot workflow CLI reference

Platform skill (not a hub capability). Root: `flowbot workflow`.

## Commands

### apply

```bash
flowbot workflow apply --file <path.yaml>
```

Parses YAML, validates DAG and `inputs`, upserts by `name`, replaces tasks and triggers.

### list

```bash
flowbot workflow list
flowbot workflow list -o json
```

### get

```bash
flowbot workflow get <name>
flowbot workflow get <name> -o json
```

### export

```bash
flowbot workflow export <name>
flowbot workflow export <name> -o <path.yaml>
```

Reconstructs YAML from normalized DB rows.

### delete

```bash
flowbot workflow delete <name>
```

Hard-deletes the definition and related tasks/triggers. Preserves `workflow_runs`.

### run

```bash
flowbot workflow run <name> --input '<json>'
```

Starts an asynchronous run. Response includes `run_id`. Does not accept a local YAML file path.

Optional (when implemented): `--wait` to block until completion.

### runs

```bash
flowbot workflow runs <name>
flowbot workflow runs <name> -o json
```

## YAML top-level fields

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Unique identifier |
| `describe` | no | Human description |
| `enabled` | no | Default true; false disables cron/webhook |
| `resumable` | no | Checkpoint persistence |
| `max_concurrency` | no | >1 enables parallel DAG execution |
| `inputs` | no | Declared run inputs for forms/validation |
| `triggers` | no | `manual`, `cron`, `webhook` (mirror pipeline auth) |
| `pipeline` | yes | Ordered task ids |
| `tasks` | yes | Task definitions |

### inputs entry

```yaml
inputs:
  - name: url
    type: string   # string | number | boolean | json
    required: true
    description: "URL to save"
    default: ""
```

### webhook trigger rule (mirror pipeline)

Webhook triggers require `auth.token` and/or `auth.hmac_secret` in the trigger rule.
HTTP path is registered separately from pipeline webhooks; disabled workflows do not serve hooks.

## Scopes

| Scope | Allows |
|-------|--------|
| `workflow:read` | list, get, export, runs |
| `workflow:run` | apply, delete, run (+ satisfies read) |
