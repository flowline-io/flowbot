# Pipeline YAML schema

Canonical shape is `pipeline.EditorDefinition` (DB published YAML). Load this file before writing or editing a definition.
Step details: [steps.md](steps.md). Capability params: [capabilities.md](capabilities.md).

```yaml
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
        title: '{{event "title"}}'
```

## Top-level fields

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Unique; validated by `pipeline.ValidateName` |
| `description` | no | Human-readable |
| `enabled` | no | Published disabled pipelines appear in `list` but cannot `run` / load into the engine |
| `resumable` | no | Checkpoint + restart recovery |
| `triggers` | yes | Array of event / cron / webhook triggers |
| `steps` | yes | Ordered capability invocations |

## Triggers

| Type | Fields |
|------|--------|
| `event` | `event` (DataEvent.EventType) |
| `cron` | `cron`, optional `cron_timeout` |
| `webhook` | `webhook.path`, auth token and/or hmac_secret |

## Authoring checklist

1. Copy the skeleton; set `name` and `enabled: true`.
2. Add at least one enabled trigger.
3. For each step, open `capabilities/<type>.md` and fill required params.
4. Use `event` / `step` helpers only from [steps.md](steps.md).
5. `flowbot pipeline apply --file ...` then `get` / `run`.
