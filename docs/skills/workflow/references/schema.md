# Workflow YAML schema

Canonical shape is `types.WorkflowMetadata`. Load this file before writing or editing a definition.
Task action details: [steps.md](steps.md). Capability params: [capabilities.md](capabilities.md).

## Skeleton

```yaml
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
      url: "{{input.url}}"
  - id: step_b
    action: "mapper:"
    params:
      bookmark_id: '{{jsonpath (step "step_a" "result") "data.id"}}'
    conn:                       # required edges when max_concurrency > 1
      - step_a
```

## Top-level fields

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Stable identifier used by CLI `get`/`run`/`delete` |
| `describe` | no | Human-readable summary |
| `enabled` | no | Default false if omitted in some paths; set `true` for runnable workflows |
| `resumable` | no | Checkpoint/resume support |
| `max_concurrency` | no | `<=1`: run `pipeline` in order; `>1`: schedule ready tasks using `conn` |
| `inputs` | no | Declared run inputs; every `{{input.*}}` key must be listed |
| `triggers` | yes | At least one trigger object (use `manual` for CLI-only) |
| `pipeline` | yes | List of task ids |
| `tasks` | yes | Task objects; every `pipeline` id must exist |

## Inputs

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Key for `{{input.name}}` / run `--input` JSON |
| `type` | yes | `string` \| `number` \| `boolean` \| `json` |
| `required` | no | When true, run fails if missing |
| `default` | no | Applied when omitted |
| `description` | no | Documentation only |

## Triggers

| `type` | Purpose | Rule keys |
|---------|---------|-----------|
| `manual` | CLI / API `workflow run` | none |
| `cron` | Scheduled runs | `rule.cron` (or `rule.expression`): standard 5-field cron or descriptor |
| `webhook` | HTTP ingress | `rule.path` (required), `rule.method` (GET/POST/PUT, default POST), `rule.auth.token` and/or `rule.auth.hmac_secret` (at least one), optional `token_header`/`hmac_header`, `payload` (`raw`\|`mapped`), `event_type` |

```yaml
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
```

Webhook auth: supply `auth.token` and/or `auth.hmac_secret`. Defaults: token header `X-Webhook-Token`, HMAC header `X-Hub-Signature-256`.

## Tasks

See [steps.md](steps.md) for `action` prefixes, templates, `conn`, and `retry`.
Every `capability:` action must use a type from [capabilities.md](capabilities.md).

## Authoring checklist

1. Copy the skeleton; set `name`, `enabled: true`, and a `manual` trigger.
2. Declare `inputs` for every template key under `{{input.*}}`.
3. For each capability task, open `capabilities/<type>.md` and fill required params only.
4. Prefer `jsonpath` when reading prior capability results (see Common data paths in capabilities.md).
5. If `max_concurrency > 1`, set `conn` on every non-root task.
6. `flowbot workflow apply --file ...` then `get` / `run`.
