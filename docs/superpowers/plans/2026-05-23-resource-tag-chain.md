# Resource Tag & Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Track cross-app resource lineage via tag propagation through pipeline execution, with query APIs to retrieve resources by tag and traverse upstream/downstream relations.

**Architecture:** Tags (types.KV) are added to DataEvent. A new ResourceLink ent schema tracks source→target resource relationships. Pipeline engine auto-merges upstream tags into mutation step params and records links when capability handlers return `_resource` metadata. A `/hub/resource-chain` module exposes query endpoints backed by a store-layer DAO using ent queries.

**Tech Stack:** Go 1.26+, ent ORM, PostgreSQL JSONB, fiber v3, fx dependency injection

---

### Task 1: Extend DataEvent with tags field

**Files:**

- Modify: `pkg/types/event.go:58-72`
- Modify: `internal/store/ent/schema/data_event.go:17-33`
- Modify: `internal/store/event_store.go:26-48`
- Modify: `internal/store/event_store.go:50-73`

- [ ] **Step 1: Add Tags field to types.DataEvent**

In `pkg/types/event.go`, add `Tags KV` field after `Data`:

```go
Data KV        `json:"data"`
Tags KV        `json:"tags,omitempty"`
```

- [ ] **Step 2: Add tags to ent schema with GIN index**

In `internal/store/ent/schema/data_event.go`, add the tags field inside `Fields()`:

```go
field.JSON("tags", map[string]any{}).Optional(),
```

Replace the `Indexes()` method:

```go
func (DataEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type"),
		index.Fields("tags").Annotations(entsql.IndexUsing("gin")),
	}
}
```

Add `"entgo.io/ent/dialect/entsql"` to imports.

- [ ] **Step 3: Run ent code generation**

```bash
go tool task ent
```

Expected: generates updated DataEvent ent client with `SetTags` method.

- [ ] **Step 3.5: Add Tags field to model.DataEvent**

In `internal/store/model/data_event.go`, add after `Data`:

```go
Tags JSON `json:"tags,omitempty"`
```

- [ ] **Step 4: Update AppendDataEvent to persist tags**

In `internal/store/event_store.go`, `AppendDataEvent`, add after `SetCreatedAt`:

```go
if event.Tags != nil {
	c = c.SetTags(map[string]any(event.Tags))
}
```

- [ ] **Step 5: Update AppendEventOutbox to include tags**

In `internal/store/event_store.go`, `AppendEventOutbox`, add to the payload map after `"topic"`:

```go
"tags": map[string]any(event.Tags),
```

- [ ] **Step 6: Write unit test for DataEvent.Tags**

Create `pkg/types/event_tag_test.go`:

```go
package types

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataEvent_TagsMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		tags   KV
		hasKey bool
	}{
		{"nil tags omitted from JSON", nil, false},
		{"empty tags omitted from JSON", KV{}, false},
		{"single kv pair serializes", KV{"project": "alpha"}, true},
		{"multiple kv pairs serialize", KV{"project": "alpha", "env": "prod"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := DataEvent{EventID: "evt-1", Tags: tt.tags}
			data, err := sonic.Marshal(evt)
			require.NoError(t, err)
			if tt.hasKey {
				assert.Contains(t, string(data), `"tags"`)
			} else {
				assert.NotContains(t, string(data), `"tags"`)
			}
		})
	}
}
```

Run: `go test ./pkg/types/ -run TestDataEvent_TagsMarshaling -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/types/event.go pkg/types/event_tag_test.go internal/store/ent/schema/data_event.go internal/store/event_store.go internal/store/ent/gen/
git commit -m "feat: add tags KV field to DataEvent with GIN index"
```

---

### Task 2: Create ResourceLink schema and model

**Files:**

- Create: `internal/store/ent/schema/resource_link.go`
- Create: `internal/store/model/resource_chain.go`
- Create: `pkg/migrate/migrations/20260523000000_resource_link.up.sql`
- Create: `pkg/migrate/migrations/20260523000000_resource_link.down.sql`

- [ ] **Step 1: Create ResourceLink ent schema**

Create `internal/store/ent/schema/resource_link.go`:

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ResourceLink struct {
	ent.Schema
}

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
		index.Fields("source_app", "source_entity_id"),
		index.Fields("target_app", "target_entity_id"),
		index.Fields("source_event_id"),
		index.Fields("target_event_id"),
	}
}

func (ResourceLink) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("resource_links"),
	}
}
```

- [ ] **Step 2: Create model types**

Create `internal/store/model/resource_chain.go`:

```go
package model

import "time"

type ResourceLink struct {
	ID               int64     `json:"id"`
	SourceEventID    string    `json:"source_event_id"`
	TargetEventID    string    `json:"target_event_id"`
	SourceApp        string    `json:"source_app"`
	TargetApp        string    `json:"target_app"`
	SourceCapability string    `json:"source_capability"`
	TargetCapability string    `json:"target_capability"`
	SourceEntityID   string    `json:"source_entity_id"`
	TargetEntityID   string    `json:"target_entity_id"`
	PipelineRunID    int64     `json:"pipeline_run_id,omitzero"`
	PipelineName     string    `json:"pipeline_name"`
	CreatedAt        time.Time `json:"created_at"`
}

type ResourceRelations struct {
	App        string        `json:"app"`
	EntityID   string        `json:"entity_id"`
	Upstream   []ResourceRef `json:"upstream"`
	Downstream []ResourceRef `json:"downstream"`
}

type ResourceRef struct {
	App          string `json:"app"`
	EntityID     string `json:"entity_id"`
	Capability   string `json:"capability,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
}
```

- [ ] **Step 3: Create migration**

Create `pkg/migrate/migrations/20260523000000_resource_link.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS resource_links (
    id BIGSERIAL PRIMARY KEY,
    source_event_id TEXT NOT NULL,
    target_event_id TEXT NOT NULL,
    source_app TEXT NOT NULL DEFAULT '',
    target_app TEXT NOT NULL DEFAULT '',
    source_capability TEXT NOT NULL DEFAULT '',
    target_capability TEXT NOT NULL DEFAULT '',
    source_entity_id TEXT NOT NULL DEFAULT '',
    target_entity_id TEXT NOT NULL DEFAULT '',
    pipeline_run_id BIGINT,
    pipeline_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_event_id, target_event_id)
);
CREATE INDEX idx_rl_src_app_entity ON resource_links (source_app, source_entity_id);
CREATE INDEX idx_rl_tgt_app_entity ON resource_links (target_app, target_entity_id);
CREATE INDEX idx_rl_src_event ON resource_links (source_event_id);
CREATE INDEX idx_rl_tgt_event ON resource_links (target_event_id);
```

Create `pkg/migrate/migrations/20260523000000_resource_link.down.sql`:

```sql
DROP TABLE IF EXISTS resource_links;
```

- [ ] **Step 4: Run ent generation and verify**

```bash
go tool task ent
go build ./...
```

Expected: no compilation errors.

- [ ] **Step 5: Commit**

```bash
git add internal/store/ent/schema/resource_link.go internal/store/model/resource_chain.go pkg/migrate/migrations/ internal/store/ent/gen/
git commit -m "feat: add ResourceLink schema with unique constraint and indexes"
```

---

### Task 3: Add ResourceMeta to InvokeResult

**Files:**

- Modify: `pkg/ability/ability.go:65-79`

- [ ] **Step 1: Define ResourceMeta and add to InvokeResult**

In `pkg/ability/ability.go`, add after `EventRef`:

```go
// ResourceMeta identifies a resource created by a capability mutation operation.
type ResourceMeta struct {
	EventID  string `json:"event_id"`
	EntityID string `json:"entity_id"`
	App      string `json:"app"`
}
```

Add `Resource` field to `InvokeResult`:

```go
type InvokeResult struct {
	Capability hub.CapabilityType `json:"capability"`
	Operation  string             `json:"operation"`
	Data       any                `json:"data,omitzero"`
	Page       *PageInfo          `json:"page,omitzero"`
	Text       string             `json:"text,omitzero"`
	Meta       map[string]any     `json:"meta,omitzero"`
	Events     []EventRef         `json:"events,omitzero"`
	Resource   *ResourceMeta      `json:"_resource,omitempty"`
}
```

- [ ] **Step 2: Write unit test**

In `pkg/ability/invoke_test.go`, add:

```go
func TestInvokeResult_ResourceMetaJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   InvokeResult
		wantJSON bool
	}{
		{"nil Resource omitted", InvokeResult{Data: "ok"}, false},
		{
			"non-nil Resource serializes",
			InvokeResult{
				Data:     "ok",
				Resource: &ResourceMeta{EventID: "evt-1", EntityID: "123", App: "test-app"},
			},
			true,
		},
		{
			"zero-value Resource still appears",
			InvokeResult{Data: "ok", Resource: &ResourceMeta{}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := sonic.Marshal(tt.result)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, sonic.Unmarshal(data, &decoded))
			if tt.wantJSON {
				assert.Contains(t, decoded, "_resource")
			} else {
				assert.NotContains(t, decoded, "_resource")
			}
		})
	}
}
```

Add `"github.com/bytedance/sonic"` to imports if needed.

Run: `go test ./pkg/ability/ -run TestInvokeResult_ResourceMetaJSON -v`
Expected: PASS

- [ ] **Step 3: Build and commit**

```bash
go build ./...
git add pkg/ability/ability.go pkg/ability/invoke_test.go
git commit -m "feat: add ResourceMeta to InvokeResult for resource link recording"
```

---

### Task 4: Pipeline engine tag merge and resource link recording

**Files:**

- Modify: `pkg/pipeline/engine.go:48-60` (RunStore interface)
- Modify: `pkg/pipeline/engine.go:250-275` (backoff.Do section in executeStep)
- Modify: `pkg/pipeline/context.go:38-61` (templateData)
- Modify: `internal/store/event_store.go:231-254` (PipelineStore)

- [ ] **Step 1: Add RecordResourceLink to RunStore interface**

In `pkg/pipeline/engine.go`, in the `RunStore` interface, add after `RecordConsumption`:

```go
RecordResourceLink(ctx context.Context, link model.ResourceLink) error
```

Add `"github.com/flowline-io/flowbot/internal/store/model"` to imports.

- [ ] **Step 2: Implement RecordResourceLink in PipelineStore**

In `internal/store/event_store.go`, add:

```go
// RecordResourceLink inserts a resource link with UPSERT semantics.
func (s *PipelineStore) RecordResourceLink(ctx context.Context, link model.ResourceLink) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.ResourceLink.Create().
		SetSourceEventID(link.SourceEventID).
		SetTargetEventID(link.TargetEventID).
		SetSourceApp(link.SourceApp).
		SetTargetApp(link.TargetApp).
		SetSourceCapability(link.SourceCapability).
		SetTargetCapability(link.TargetCapability).
		SetSourceEntityID(link.SourceEntityID).
		SetTargetEntityID(link.TargetEntityID).
		SetPipelineRunID(link.PipelineRunID).
		SetPipelineName(link.PipelineName).
		SetCreatedAt(time.Now()).
		OnConflictColumns(
			resourcelink.FieldSourceEventID,
			resourcelink.FieldTargetEventID,
		).
		Ignore().
		Exec(ctx)
	return err
}
```

Add imports `"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"` and `"github.com/flowline-io/flowbot/internal/store/model"`.

- [ ] **Step 3: Add mergeTags helper to engine.go**

In `pkg/pipeline/engine.go`, add before `RunStore` interface:

```go
// mergeTags merges upstream tags with step-declared tags.
// Upstream tags are the base; step-declared tags override on key collision.
func mergeTags(upstream types.KV, stepTags any) types.KV {
	if upstream == nil {
		upstream = types.KV{}
	}
	stepKV, ok := stepTags.(types.KV)
	if !ok {
		sm, ok := stepTags.(map[string]any)
		if !ok {
			return upstream
		}
		stepKV = types.KV(sm)
	}
	if len(stepKV) == 0 {
		return upstream
	}
	result := make(types.KV, len(upstream)+len(stepKV))
	for k, v := range upstream {
		result[k] = v
	}
	for k, v := range stepKV {
		result[k] = v
	}
	return result
}
```

- [ ] **Step 4: Inject tag merge in executeStep**

In `executeStep`, after `renderedParams, err := rc.RenderParams(step.Params)` and before the `backoff.Do` block, add:

```go
if ability.IsMutation(step.Operation) && len(rc.Event.Tags) > 0 {
	renderedParams["tags"] = mergeTags(rc.Event.Tags, renderedParams["tags"])
}
```

Add `"github.com/flowline-io/flowbot/pkg/ability"` to imports.

- [ ] **Step 5: Capture ResourceMeta and record link in executeStep**

In `executeStep`, replace the `backoff.Do` variables from:

```go
var stepResult map[string]any
attempt, retryErr := backoff.Do(ctx, boCfg, func(ctx context.Context) error {
	res, invokeErr := ability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
	if invokeErr != nil {
		trace.RecordError(ctx, invokeErr)
		return invokeErr
	}
	stepResult = extractResult(res)
	return nil
})
```

To:

```go
var stepResult map[string]any
var stepResource *ability.ResourceMeta
attempt, retryErr := backoff.Do(ctx, boCfg, func(ctx context.Context) error {
	res, invokeErr := ability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
	if invokeErr != nil {
		trace.RecordError(ctx, invokeErr)
		return invokeErr
	}
	stepResult = extractResult(res)
	stepResource = res.Resource
	return nil
})
```

After `rc.RecordStepResult(step.Name, stepResult)`, add:

```go
if stepResource != nil && stepResource.EntityID != "" {
	link := model.ResourceLink{
		SourceEventID:    rc.Event.EventID,
		TargetEventID:    stepResource.EventID,
		SourceApp:        rc.Event.App,
		TargetApp:        stepResource.App,
		SourceCapability: rc.Event.Capability,
		TargetCapability: string(step.Capability),
		SourceEntityID:   rc.Event.EntityID,
		TargetEntityID:   stepResource.EntityID,
		PipelineRunID:    runID,
		PipelineName:     pipelineName,
	}
	if e.store != nil {
		_ = e.store.RecordResourceLink(ctx, link)
	}
}
```

- [ ] **Step 6: Expose tags in template data**

In `pkg/pipeline/context.go`, in `templateData()`, add after `"topic"`:

```go
"topic": rc.Event.Topic,
"tags":  map[string]any(rc.Event.Tags),
```

- [ ] **Step 7: Fix RunStore implementations in BDD tests**

In `tests/specs/` and `internal/store/`, any struct implementing `RunStore` needs the new method. Add to each:

```go
func (s *TYPE) RecordResourceLink(_ context.Context, _ model.ResourceLink) error { return nil }
```

Run: `go build ./...`
Expected: no compilation errors.

- [ ] **Step 8: Write unit tests for mergeTags**

In `pkg/pipeline/engine_test.go`, add:

```go
func TestMergeTags(t *testing.T) {
	tests := []struct {
		name     string
		upstream types.KV
		stepTags any
		want     types.KV
	}{
		{"nil upstream returns empty", nil, nil, types.KV{}},
		{"upstream no step tags passes through", types.KV{"project": "alpha"}, nil, types.KV{"project": "alpha"}},
		{
			"step overrides on collision",
			types.KV{"project": "alpha", "env": "staging"},
			types.KV{"project": "beta", "processed": "true"},
			types.KV{"project": "beta", "env": "staging", "processed": "true"},
		},
		{
			"step as map[string]any merges",
			types.KV{"project": "alpha"},
			map[string]any{"processed": "true"},
			types.KV{"project": "alpha", "processed": "true"},
		},
		{"non-map step tags returns upstream", types.KV{"project": "alpha"}, "string", types.KV{"project": "alpha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mergeTags(tt.upstream, tt.stepTags))
		})
	}
}
```

Run: `go test ./pkg/pipeline/ -run TestMergeTags -v`
Expected: PASS

- [ ] **Step 9: Write unit test for resource link nil safety**

In `pkg/pipeline/engine_test.go`, add:

```go
func TestHandleEvent_WithTagsDoesNotCrash(t *testing.T) {
	t.Parallel()
	defs := []Definition{
		{
			Name:    "tag-test",
			Enabled: true,
			Trigger: Trigger{Event: "test.event"},
			Steps: []Step{
				{Name: "s1", Capability: "test", Operation: "create", Params: map[string]any{"title": "x"}},
			},
		},
	}
	e := NewEngine(defs, nil, nil, noopPC, noopEC)
	defer e.Stop()
	event := types.DataEvent{
		EventID: "evt-1", EventType: "test.event", EntityID: "src-1", App: "app-a",
		Tags: types.KV{"project": "alpha"},
	}
	// No registered invoker, but verify no crash from tag merge or nil Resource check
	err := e.Handler()(context.Background(), event)
	require.Error(t, err) // err from unknown capability is fine; nil pointer is not
}
```

Run: `go test ./pkg/pipeline/ -run TestHandleEvent_WithTagsDoesNotCrash -v`
Expected: error (capability not registered) but no panic

- [ ] **Step 10: Commit**

```bash
git add pkg/pipeline/engine.go pkg/pipeline/engine_test.go pkg/pipeline/context.go internal/store/event_store.go
git commit -m "feat: auto-merge tags in pipeline and record resource links"
```

---

### Task 5: Resource chain DAO

**Files:**

- Create: `internal/store/resource_chain_store.go`

- [ ] **Step 1: Create ResourceChainStore**

Create `internal/store/resource_chain_store.go`:

```go
package store

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/model"
)

type ResourceChainStore struct {
	client *gen.Client
}

func NewResourceChainStore(client *gen.Client) *ResourceChainStore {
	return &ResourceChainStore{client: client}
}

func (s *ResourceChainStore) FindResourcesByTag(ctx context.Context, key, value string, limit int, cursor string) ([]*model.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tagJSON := fmt.Sprintf(`{"%s":"%s"}`, key, value)
	q := s.client.DataEvent.Query().
		Where(func(selector *sql.Selector) {
			selector.Where(sql.Raw("tags @> $1", tagJSON))
		}).
		Order(dataevent.ByCreatedAt(sql.OrderDesc())).
		Limit(limit + 1)

	if cursor != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999Z", cursor); err == nil {
			q = q.Where(dataevent.CreatedAtLT(t))
		}
	}

	events, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("find resources by tag: %w", err)
	}

	result := make([]*model.DataEvent, len(events))
	for i, e := range events {
		result[i] = &model.DataEvent{
			EventID:   e.EventID,
			EventType: e.EventType,
			Source:    e.Source,
			Capability: e.Capability,
			Operation: e.Operation,
			Backend:   e.Backend,
			App:       e.App,
			EntityID:  e.EntityID,
			CreatedAt: e.CreatedAt,
		}
		if e.Data != nil {
			result[i].Data = model.JSON(e.Data)
		}
		if e.Tags != nil {
			result[i].Tags = model.JSON(e.Tags)
		}
	}

	var nextCursor string
	if len(result) > limit {
		nextCursor = result[limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
		result = result[:limit]
	}

	return result, nextCursor, nil
}

func (s *ResourceChainStore) FindResourceLinks(ctx context.Context, eventIDs []string) ([]*model.ResourceLink, error) {
	if s == nil || s.client == nil || len(eventIDs) == 0 {
		return nil, nil
	}

	links, err := s.client.ResourceLink.Query().
		Where(resourcelink.Or(
			resourcelink.SourceEventIDIn(eventIDs...),
			resourcelink.TargetEventIDIn(eventIDs...),
		)).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find resource links: %w", err)
	}

	result := make([]*model.ResourceLink, len(links))
	for i, l := range links {
		result[i] = &model.ResourceLink{
			ID:               l.ID,
			SourceEventID:    l.SourceEventID,
			TargetEventID:    l.TargetEventID,
			SourceApp:        l.SourceApp,
			TargetApp:        l.TargetApp,
			SourceCapability: l.SourceCapability,
			TargetCapability: l.TargetCapability,
			SourceEntityID:   l.SourceEntityID,
			TargetEntityID:   l.TargetEntityID,
			PipelineRunID:    l.PipelineRunID,
			PipelineName:     l.PipelineName,
			CreatedAt:        l.CreatedAt,
		}
	}

	return result, nil
}

func (s *ResourceChainStore) FindRelations(ctx context.Context, app, entityID string) (*model.ResourceRelations, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	relations := &model.ResourceRelations{
		App:        app,
		EntityID:   entityID,
		Upstream:   []model.ResourceRef{},
		Downstream: []model.ResourceRef{},
	}

	downLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.SourceApp(app),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find downstream: %w", err)
	}
	for _, l := range downLinks {
		relations.Downstream = append(relations.Downstream, model.ResourceRef{
			App:          l.TargetApp,
			EntityID:     l.TargetEntityID,
			Capability:   l.TargetCapability,
			PipelineName: l.PipelineName,
		})
	}

	upLinks, err := s.client.ResourceLink.Query().
		Where(
			resourcelink.TargetApp(app),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find upstream: %w", err)
	}
	for _, l := range upLinks {
		relations.Upstream = append(relations.Upstream, model.ResourceRef{
			App:          l.SourceApp,
			EntityID:     l.SourceEntityID,
			Capability:   l.SourceCapability,
			PipelineName: l.PipelineName,
		})
	}

	return relations, nil
}
```

Note: `dataevent.CreatedAtLT` expects a `time.Time`. Parse the cursor string in the middleware. For now keep it as-is; the cursor will be passed as a time value if available.

- [ ] **Step 2: Build and commit**

```bash
go build ./...
git add internal/store/resource_chain_store.go
git commit -m "feat: add ResourceChainStore with tag query and relations lookup"
```

---

### Task 6: Query API module (resourcechain)

**Files:**

- Create: `internal/modules/resourcechain/module.go`
- Create: `internal/modules/resourcechain/webservice.go`
- Create: `internal/modules/resourcechain/webservice_test.go`
- Modify: `internal/modules/fx.go:4-17`

- [ ] **Step 1: Create module.go**

Create `internal/modules/resourcechain/module.go`:

```go
package resourcechain

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

const Name = "resourcechain"

var handler moduleHandler
var config configType
var rcStore *store.ResourceChainStore

type moduleHandler struct {
	initialized bool
	module.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func Register() {
	module.Register(Name, &handler)
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	if store.Database == nil {
		return errors.New("store database not available")
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok || client == nil {
		return errors.New("store client not available")
	}
	rcStore = store.NewResourceChainStore(client)
	handler.initialized = true
	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Bootstrap() error { return nil }

func (moduleHandler) Webservice(app *fiber.App) {
	module.Webservice(app, Name, webserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{webserviceRules}
}

func (moduleHandler) Input(_ types.Context, _ types.KV, _ any) (types.MsgPayload, error) {
	return types.TextMsg{Text: "Input"}, nil
}
```

- [ ] **Step 2: Create webservice.go**

Create `internal/modules/resourcechain/webservice.go`:

```go
package resourcechain

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/resource-chain", queryByTag),
	webservice.Get("/resource-chain/:app/:entity_id/relations", getRelations),
}

func queryByTag(ctx fiber.Ctx) error {
	key := ctx.Query("key")
	value := ctx.Query("value")
	if key == "" || value == "" {
		return types.Errorf(types.ErrInvalidArgument, "key and value query params are required")
	}

	limit := 20
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	cursor := ctx.Query("cursor")

	events, nextCursor, err := rcStore.FindResourcesByTag(context.Background(), key, value, limit, cursor)
	if err != nil {
		return err
	}

	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
	}

	links, _ := rcStore.FindResourceLinks(context.Background(), eventIDs)

	type resEntry struct {
		EntityID   string `json:"entity_id"`
		App        string `json:"app"`
		Capability string `json:"capability"`
		EventID    string `json:"event_id"`
		CreatedAt  string `json:"created_at"`
	}
	type linkEntry struct {
		Source       resEntry `json:"source"`
		Target       resEntry `json:"target"`
		PipelineName string   `json:"pipeline_name"`
		CreatedAt    string   `json:"created_at"`
	}

	resources := make([]resEntry, len(events))
	for i, e := range events {
		resources[i] = resEntry{
			EntityID: e.EntityID, App: e.App, Capability: e.Capability,
			EventID: e.EventID, CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	linkEntries := make([]linkEntry, 0, len(links))
	for _, l := range links {
		linkEntries = append(linkEntries, linkEntry{
			Source:       resEntry{EntityID: l.SourceEntityID, App: l.SourceApp},
			Target:       resEntry{EntityID: l.TargetEntityID, App: l.TargetApp},
			PipelineName: l.PipelineName,
			CreatedAt:    l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	result := types.KV{
		"tag":       types.KV{"key": key, "value": value},
		"resources": resources,
		"links":     linkEntries,
	}
	if nextCursor != "" {
		result["cursor"] = nextCursor
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func getRelations(ctx fiber.Ctx) error {
	app := ctx.Params("app")
	entityID := ctx.Params("entity_id")
	if app == "" || entityID == "" {
		return types.Errorf(types.ErrInvalidArgument, "app and entity_id path params are required")
	}

	relations, err := rcStore.FindRelations(context.Background(), app, entityID)
	if err != nil {
		return err
	}
	if relations == nil {
		relations = &model.ResourceRelations{
			App: app, EntityID: entityID,
			Upstream: []model.ResourceRef{}, Downstream: []model.ResourceRef{},
		}
	}

	return ctx.JSON(protocol.NewSuccessResponse(relations))
}
```

- [ ] **Step 3: Register module in fx.go**

In `internal/modules/fx.go`, add import:

```go
"github.com/flowline-io/flowbot/internal/modules/resourcechain"
```

Add to `fx.Invoke` call:

```go
resourcechain.Register,
```

- [ ] **Step 4: Write validation tests**

Create `internal/modules/resourcechain/webservice_test.go`:

```go
package resourcechain

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestQueryByTag_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing key returns 400", "value=alpha", 400},
		{"missing value returns 400", "key=project", 400},
		{"empty key and value returns 400", "key=&value=", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/resource-chain", queryByTag)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/resource-chain?"+tt.queryStr, nil)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetRelations_Validation(t *testing.T) {
	tests := []struct {
		name       string
		app        string
		entityID   string
		wantStatus int
	}{
		{"valid params passes validation", "karakeep", "bm-123", 500}, // 500 from nil rcStore
		{"empty app returns 400", "", "bm-123", 400},
		{"empty entity_id returns 400", "karakeep", "", 400},
		{"both empty returns 400", "", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/:app/:entity_id/relations", getRelations)
			defer app.Shutdown()
			url := "/" + tt.app + "/" + tt.entityID + "/relations"
			req := httptest.NewRequest(fiber.MethodGet, url, nil)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
```

Run: `go test ./internal/modules/resourcechain/ -v`
Expected: PASS (400 tests pass, 500 from nil store is acceptable)

- [ ] **Step 5: Build and commit**

```bash
go build ./...
git add internal/modules/resourcechain/ internal/modules/fx.go
git commit -m "feat: add resourcechain module with tag query and relations endpoints"
```

---

### Task 7: Update example implementations

**Files:**

- Modify: `pkg/ability/example/interface.go`
- Modify: `pkg/ability/example/descriptor.go:99-111`
- Modify: `pkg/ability/example/example/adapter.go:72-84`
- Modify: `pkg/ability/example/example/adapter_test.go:133-155`
- Modify: `pkg/ability/example/descriptor_test.go:19`
- Modify: `pkg/ability/example/poller_test.go:26`
- Modify: `pkg/ability/example/conformance.go:19,122-138`
- Modify: `pkg/ability/example/conformance_test.go:40-50`
- Modify: `internal/modules/example/webservice.go:83-98`

- [ ] **Step 1: Update Service interface**

In `pkg/ability/example/interface.go`, add `"github.com/flowline-io/flowbot/pkg/types"` import and change:

```go
CreateItem(ctx context.Context, title string, tags types.KV) (*ability.Host, error)
```

- [ ] **Step 2: Update adapter**

In `pkg/ability/example/example/adapter.go`, add `"github.com/flowline-io/flowbot/pkg/types"` import and change:

```go
func (a *Adapter) CreateItem(ctx context.Context, title string, _ types.KV) (*ability.Host, error) {
```

In `pkg/ability/example/example/adapter_test.go`, update the `TestAdapter_CreateItem` test to pass `nil` for the tags parameter when calling `a.CreateItem`:

```go
item, err := a.CreateItem(context.Background(), tt.title, nil)
```

- [ ] **Step 3: Update descriptor invokeCreate**

In `pkg/ability/example/descriptor.go`, add `"github.com/flowline-io/flowbot/pkg/types"` import and replace `invokeCreate`:

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
				EntityID: item.ID,
				App:      backend,
			},
		}, nil
	}
}
```

- [ ] **Step 4: Update mock/test implementations**

In `pkg/ability/example/descriptor_test.go:19`, update mock:

```go
func (mockService) CreateItem(_ context.Context, _ string, _ types.KV) (*ability.Host, error) { return nil, nil }
```

In `pkg/ability/example/poller_test.go:26`, update mock:

```go
func (*fakePollerService) CreateItem(_ context.Context, _ string, _ types.KV) (*ability.Host, error) { return nil, nil }
```

In `pkg/ability/example/conformance.go:19`, update Config struct:

```go
CreateItem *ability.Host
```

No change to the field itself; just ensure `CreateItem(ctx, title, nil)` calls in conformance.go:138 use the new signature.

In `pkg/ability/example/conformance_test.go:40`, update:

```go
func (c *conformanceService) CreateItem(ctx context.Context, title string, _ types.KV) (*ability.Host, error) {
```

- [ ] **Step 5: Update example module webservice**

In `internal/modules/example/webservice.go`, update `createExampleItem`:

```go
func createExampleItem(ctx fiber.Ctx) error {
	var body struct {
		Title string   `json:"title"`
		Tags  types.KV `json:"tags,omitempty"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	if body.Title == "" {
		return types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	res, err := ability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleCreate,
		map[string]any{"title": body.Title, "tags": body.Tags})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}
```

- [ ] **Step 6: Verify and commit**

```bash
go build ./... && go test ./pkg/ability/example/... -v
git add pkg/ability/example/ internal/modules/example/webservice.go
git commit -m "feat: demonstrate tags and ResourceMeta in example capability"
```

---

### Task 8: Run lint and full test suite

**Files:** None

- [ ] **Step 1: Run lint**

```bash
go tool task lint
```

Fix any issues.

- [ ] **Step 2: Run unit tests**

```bash
go tool task test
```

Expected: all pass.

- [ ] **Step 3: Run BDD tests (Docker required)**

```bash
go tool task test:specs
```

If RunStore interface changes break BDD test mocks, add:

```go
func (s *MockStore) RecordResourceLink(_ context.Context, _ model.ResourceLink) error { return nil }
```

to each mock implementation.

- [ ] **Step 4: Commit any fixes**

```bash
git add -A && git commit -m "test: fix code after resource tag chain refactor"
```
