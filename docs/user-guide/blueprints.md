# Pipeline blueprints

Parameterized pipeline templates. Official templates ship in the Flowbot binary. You can also paste YAML into the Web library. There is no remote URL / GitHub / Gist fetch.

Source: `pkg/pipeline/blueprints/`, `pkg/pipeline/blueprint.go`

## Library

Web UI: `/service/web/blueprints`. Official cards and imported cards share the same list.

| Action | Official | Imported |
| --- | --- | --- |
| View YAML | yes | yes |
| Instantiate | if required capabilities are registered | same |
| Download | yes | yes |
| Replace YAML | no | yes (same `id`) |
| Delete | no | yes, only when no linked pipelines remain |

Missing capabilities: the card stays visible and Instantiate is disabled, with a link to Hub.

## Document shape

`flowbot pipeline apply` rejects this file. Instantiate it instead.

```yaml
kind: pipeline_blueprint
id: webhook_notify
title: Incoming webhook notification
description: Notify when a pipeline webhook is invoked.
requires:
  capabilities: []
inputs:
  - name: webhook_path
    type: string
    required: true
    description: Webhook path segment (for example hooks/notify)
  - name: notify_channel
    type: notify_channel
    required: true
    description: Notification channels
  - name: template_id
    type: string
    required: true
    description: Notification template ID from Notify settings
definition:
  description: Notify on incoming webhook
  resumable: true
  triggers:
    - type: webhook
      enabled: true
      webhook:
        path: "{{inputs.webhook_path}}"
  steps:
    - name: notify
      capability: core
      operation: notify_send
      params:
        template_id: "{{inputs.template_id}}"
        channels: "{{inputs.notify_channel}}"
```

Input types: `string`, `number`, `boolean`, `notify_channel`. Placeholders are `{{inputs.name}}` only. `capability` and `operation` are fixed by the author.

## Instances

Creating a pipeline from a blueprint asks for a unique pipeline name, input values, and an optional enable switch (default off). The instance stays linked: you can change inputs and the enable flag; steps are read-only until **Take control**, which is irreversible. `flowbot pipeline apply` also refuses to overwrite a linked instance.

Template updates never apply automatically. The instance shows an Update action when the catalog hash differs. Confirming rematerializes with current inputs; incompatible required inputs fail the whole update.

CLI: `flowbot blueprint list|import|instantiate|update|take-control`.
