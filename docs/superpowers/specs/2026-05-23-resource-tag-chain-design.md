# Resource Tag & Chain

Cross-app resource lineage tracking with tag propagation through pipeline execution.

## Overview

When a resource is created in one homelab app with a tag, flowbot pipeline processing creates a counterpart resource in another app. The tag is automatically propagated, and all resources sharing the same tag form a queryable resource chain.

A resource is a DataEvent entity identified by `entity_id + capability + app`. Tags are key-value pairs stored as JSON on the DataEvent.

---

## Data Model

### DataEvent extension

Add `tags` field to `pkg/types/event.go` and `internal/store/ent/schema/data_event.go`:

```go
// types.DataEvent
Tags KV `json:"tags,omitempty"`

// ent schema
field.JSON("tags", types.KV{}).Optional()
```

GIN index on `tags` for JSONB queries:

```go
index.Fields("tags").Annotations(entsql.IndexUsing("gin"))
```

### ResourceLink (new table)

`internal/store/ent/schema/resource_link.go`:

```go
type ResourceLink struct { ent.Schema }

func (ResourceLink) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("id").Immutable(),
        field.String("source_event_id").NotEmpty(),
        field.String("target_event_id").NotEmpty(),
        field.String("source_app").Default(""),
        field.String("target_app").Default(""),
        field.String("source_capability").Default(""),
        field.String("target_capability").Default(""),
        field.String("source_entity_id").Default(""),
        field.String("target_entity_id").Default(""),
        field.Int64("pipeline_run_id").Optional(),
        field.String("pipeline_name").Default(""),
        field.Time("created_at").Immutable().Default(time.Now),
    }
}

func (ResourceLink) Indexes() []ent.Index {
    return []ent.Index{
        // Downstream: find links originating from a source resource
        index.Fields("source_app", "source_entity_id"),
        // Upstream: find links targeting a specific resource
        index.Fields("target_app", "target_entity_id"),
        // Event-based link assembly
        index.Fields("source_event_id"),
        index.Fields("target_event_id"),
    }
}
```

**Unique constraint** on `(source_event_id, target_event_id)` prevents duplicate links from pipeline retries. The DAO layer uses `INSERT ... ON CONFLICT DO NOTHING` to ensure idempotency.

`types.KV` tags are stored on `data_events`, not on `resource_links`. Query flow: match DataEvent by tags via JSONB GIN → get event_ids → JOIN resource_links to assemble the chain.

### DAO layer

New DAO in `internal/store/postgres/`:
- `FindResourcesByTag(ctx, key, value string, limit int, cursor string) ([]*model.DataEvent, string, error)`
- `FindResourceLinks(ctx, eventIDs []string) ([]*model.ResourceLink, error)`
- `FindRelations(ctx, app, entityID string) (*model.ResourceRelations, error)`
- `RecordResourceLink(ctx, link *model.ResourceLink) error` — UPSERT on `(source_event_id, target_event_id)`

---

## Pipeline Engine

### Tag auto-injection

In `pkg/pipeline/engine.go`, `executeStep()` merges tags into params before `ability.Invoke` for mutation operations:

```go
if ability.IsMutation(step.Operation) && rc.Event.Tags != nil {
    renderedParams["tags"] = mergeTags(rc.Event.Tags, renderedParams["tags"])
}
```

Merge strategy: upstream tags (`rc.Event.Tags`) are the base; step-declared tags (`renderedParams["tags"]`) take priority on key collision. This allows a step to both inherit upstream tags and add its own (e.g., `"processed": "true"`). If the step does not declare any tags, upstream tags pass through unchanged.

### Resource link auto-recording

New struct in `pkg/ability/ability.go`:

```go
type ResourceMeta struct {
    EventID  string `json:"event_id"`
    EntityID string `json:"entity_id"`
    App      string `json:"app"`
}
```

Add to `InvokeResult`:

```go
type InvokeResult struct {
    // existing fields unchanged
    Resource *ResourceMeta `json:"_resource,omitempty"`
}
```

In `pkg/pipeline/engine.go`, after step success, the engine checks `result.Resource` and records a link via `RunStore`. The store interface gains `RecordResourceLink`.

### Template data

`pkg/pipeline/context.go` `templateData()` already exposes event fields. With `tags` on DataEvent, `{{.Event.tags}}` is automatically available. Users can override tags in step params if needed.

---

## Query API

Routes under `/hub/resource-chain`. New module at `internal/modules/resourcechain/`.

### GET /hub/resource-chain?key=project&value=alpha&limit=20

Query all resources sharing the specified tag key-value. Returns:

```json
{
  "tag": {"key": "project", "value": "alpha"},
  "resources": [
    {
      "entity_id": "bm-123",
      "app": "karakeep",
      "capability": "bookmark",
      "event_id": "evt-001",
      "created_at": "2026-05-23T10:00:00Z"
    }
  ],
  "links": [
    {
      "source": {"entity_id": "bm-123", "app": "karakeep", "capability": "bookmark"},
      "target": {"entity_id": "task-789", "app": "kanboard", "capability": "kanban"},
      "pipeline_name": "bookmark-to-kanban",
      "created_at": "2026-05-23T10:01:00Z"
    }
  ]
}
```

Pagination: `limit` + opaque `cursor`.

### GET /hub/resource-chain/:app/:entity_id/relations

Returns upstream (what triggered this resource) and downstream (what this resource triggered) relations. The `app` is required because `entity_id` is only unique within its own app.

```json
{
  "entity_id": "bm-123",
  "app": "karakeep",
  "upstream": [],
  "downstream": [
    {
      "target": {"entity_id": "task-789", "app": "kanboard"},
      "pipeline_name": "bookmark-to-kanban"
    }
  ]
}
```

---

## Capability Convention

### For new capability create handlers

Each mutation handler should:
1. Read `params["tags"]` (types.KV) and include it in the emitted DataEvent
2. Populate `result.Resource` with the new resource's identity

Example (in descriptor invokeCreate):

```go
func invokeCreate(svc Service) ability.Invoker {
    return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
        title, err := ability.RequiredString(params, "title")
        if err != nil {
            return nil, err
        }
        tags, _ := params["tags"].(types.KV)
        item, err := svc.CreateItem(ctx, title, tags)
        if err != nil {
            return nil, err
        }
        eventID := emitDataEvent(ctx, "example.item.created", item, tags)
        return &ability.InvokeResult{
            Data: item,
            Resource: &ability.ResourceMeta{
                EventID:  eventID,
                EntityID: item.ID,
                App:      getAppName(),
            },
        }, nil
    }
}
```

### Provider layer

No interface changes. `tags` is flowbot-internal metadata. If an external service supports native tags, the provider's create method receives tags through existing request types and passes them to the API. No provider interface change required.

### Backward compatibility

Existing capability handlers that do not populate `result.Resource` continue to work — the pipeline engine skips link recording when `Resource` is nil.

---

## Example Updates

### ability/example/descriptor.go

Update `invokeCreate` to demonstrate reading `tags` from params and populating `result.Resource`:

```go
func invokeCreate(svc Service) ability.Invoker {
    return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
        title, err := ability.RequiredString(params, "title")
        if err != nil {
            return nil, err
        }
        tags, _ := params["tags"].(types.KV)
        item, err := svc.CreateItem(ctx, title, tags)
        if err != nil {
            return nil, err
        }
        return &ability.InvokeResult{
            Data: item,
            Resource: &ability.ResourceMeta{
                EventID:  "", // populated by real capability event emitter
                EntityID: item.ID,
                App:      backend,
            },
        }, nil
    }
}
```

### ability/example/example/adapter.go

Update `CreateItem` signature to accept optional tags:

```go
func (a *Adapter) CreateItem(ctx context.Context, title string, _ types.KV) (*ability.Host, error) {
```

### ability/example/interface.go

Update `Service` interface:

```go
type Service interface {
    // ...
    CreateItem(ctx context.Context, title string, tags types.KV) (*ability.Host, error)
    // ...
}
```

### internal/modules/example/webservice.go

Update `createExampleItem` to pass tags from request body:

```go
func createExampleItem(ctx fiber.Ctx) error {
    var body struct {
        Title string    `json:"title"`
        Tags  types.KV  `json:"tags,omitempty"`
    }
    // ...
    res, err := ability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleCreate,
        map[string]any{"title": body.Title, "tags": body.Tags})
    // ...
}
```

---

## Error Handling

| Level | Scenario | Behavior |
|-------|----------|----------|
| Pipeline engine | `result.Resource` is nil | Skip link recording, no error |
| Pipeline engine | `recordResourceLink` duplicate (retry) | UPSERT ON CONFLICT DO NOTHING, silent |
| Capability handler | `params["tags"]` missing or wrong type | Ignore, create resource without tags |
| DAO | JSONB query fails | Return `types.ErrInternal` |
| API | Missing `key` or `value` param | Return 400 with protocol error code |
| API | Tag not found | Return empty result (not 404) |

---

## Testing Strategy

### Unit tests (table-driven TDD)

| Package | Focus |
|---------|-------|
| `pkg/types` | `DataEvent.Tags` marshal/unmarshal, tag match helpers |
| `pkg/pipeline` | Auto tag merge into params (no-step-tags, step-override, merge-collision), `recordResourceLink` call, `Resource` nil branch, idempotent link recording on retry |
| `pkg/ability` | `InvokeResult.Resource` field passthrough |
| `internal/modules/resourcechain` | Input validation, empty results, pagination cursor |

### BDD integration tests (Ginkgo)

End-to-end: create tagged resource → pipeline creates downstream resource → API query returns chain with both resources and a link.

### Existing capability migration

No forced migration. Each capability can be updated independently to populate `result.Resource` and read `params["tags"]`. Missing `Resource` is silently ignored.

---

## Files Changed

```
internal/store/ent/schema/data_event.go        # add tags JSON field + GIN index
internal/store/ent/schema/resource_link.go      # NEW - resource link table
internal/store/model/resource_chain.go          # NEW - model types
internal/store/postgres/resource_chain_dao.go   # NEW - DAO queries
pkg/types/event.go                              # add Tags KV field to DataEvent
pkg/pipeline/context.go                         # expose tags in template data
pkg/pipeline/engine.go                          # auto-inject tags + record resource link
pkg/ability/ability.go                          # add ResourceMeta + Resource field to InvokeResult
pkg/ability/invoke.go                           # no signature change
pkg/ability/operations.go                       # IsMutation already exists

internal/modules/resourcechain/                 # NEW - query API module
  module.go
  webservice.go

# Example updates (reference implementation)
pkg/ability/example/descriptor.go               # demonstrate tags+Resource in invokeCreate
pkg/ability/example/interface.go                # CreateItem accepts tags
pkg/ability/example/example/adapter.go          # CreateItem passes tags
internal/modules/example/webservice.go          # create handler reads tags from body
```
