# Store Context Propagation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `ctx context.Context` as first parameter to all data-access store methods across Adapter + 5 specialized stores.

**Architecture:** Bottom-up mechanical refactoring in 4 commits. Each method receives `ctx context.Context` as first param. Implementations replace `ctx := context.Background()` with the passed `ctx`. Callers pass their already-available context.

**Tech Stack:** Go 1.26+, Ent ORM, Fiber v3, stdlib `context` package.

---

### Task 1: RunStore and EventStore interfaces + implementations + Pipeline call sites

**Files:**

- Modify: `pkg/pipeline/engine.go:44-56` (RunStore interface)
- Modify: `pkg/pipeline/engine.go:330-377` (helper methods createRunRecord, createStepRunRecord, updateStepRunRecord, finishRunRecord)
- Modify: `pkg/pipeline/engine.go:411-472` (ResumePipeline, heartbeatLoop)
- Modify: `internal/store/event_store.go:26-87` (EventStore: AppendDataEvent, AppendEventOutbox, MarkOutboxPublished)
- Modify: `internal/store/event_store.go:90-378` (PipelineStore: all 11 methods)
- Modify: `internal/server/pipeline.go:79-81` (EventStore call sites)

- [ ] **Step 1: Add `ctx` to `RunStore` interface in `pkg/pipeline/engine.go`**

```go
// RunStore interface (add ctx)
type RunStore interface {
	CreateRun(ctx context.Context, pipelineName, eventID, eventType string) (*model.PipelineRun, error)
	UpdateRunStatus(ctx context.Context, runID int64, status model.PipelineState, errMsg string) error
	CreateStepRun(ctx context.Context, runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error)
	UpdateStepRun(ctx context.Context, stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error
	SaveCheckpoint(ctx context.Context, runID int64, data any) error
	GetIncompleteRuns(ctx context.Context) ([]*model.PipelineRun, error)
	GetCheckpoint(ctx context.Context, runID int64, target any) error
	GetRun(ctx context.Context, runID int64) (*model.PipelineRun, error)
	UpdateRunHeartbeat(ctx context.Context, runID int64) error
	HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error)
	RecordConsumption(ctx context.Context, consumerName, eventID string) error
}
```

- [ ] **Step 2: Add `ctx` to `EventStore` methods in `internal/store/event_store.go`**

```go
func (s *EventStore) AppendDataEvent(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	c := s.client.DataEvent.Create().
		SetEventID(event.EventID).
		SetEventType(event.EventType).
		SetSource(event.Source).
		SetCapability(event.Capability).
		SetOperation(event.Operation).
		SetBackend(event.Backend).
		SetApp(event.App).
		SetEntityID(event.EntityID).
		SetIdempotencyKey(event.IdempotencyKey).
		SetUID(event.UID).
		SetTopic(event.Topic).
		SetCreatedAt(time.Now())
	if event.Data != nil {
		c = c.SetData(map[string]any(event.Data))
	}
	_, err := c.Save(ctx)
	return err
}

func (s *EventStore) AppendEventOutbox(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.EventOutbox.Create().
		SetEventID(event.EventID).
		SetPayload(map[string]any{
			"event_id":        event.EventID,
			"event_type":      event.EventType,
			"source":          event.Source,
			"capability":      event.Capability,
			"operation":       event.Operation,
			"backend":         event.Backend,
			"app":             event.App,
			"entity_id":       event.EntityID,
			"idempotency_key": event.IdempotencyKey,
			"uid":             event.UID,
			"topic":           event.Topic,
		}).
		SetPublished(false).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *EventStore) MarkOutboxPublished(ctx context.Context, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.EventOutbox.Update().
		Where(eventoutbox.EventID(eventID)).
		SetPublished(true).
		Save(ctx)
	return err
}
```

- [ ] **Step 3: Add `ctx` to all `PipelineStore` methods in `internal/store/event_store.go`**

Add `ctx context.Context` as first parameter to all 11 methods and replace `ctx := context.Background()` with using the passed `ctx`:

- `UpsertDefinition(ctx context.Context, ...)`
- `CreateRun(ctx context.Context, ...)`
- `UpdateRunStatus(ctx context.Context, ...)`
- `CreateStepRun(ctx context.Context, ...)`
- `UpdateStepRun(ctx context.Context, ...)`
- `RecordConsumption(ctx context.Context, ...)`
- `HasConsumed(ctx context.Context, ...)`
- `SaveCheckpoint(ctx context.Context, ...)`
- `UpdateRunHeartbeat(ctx context.Context, ...)`
- `GetIncompleteRuns(ctx context.Context)`
- `GetCheckpoint(ctx context.Context, ...)`
- `GetRun(ctx context.Context, ...)`

- [ ] **Step 4: Update pipeline engine callers in `pkg/pipeline/engine.go`**

The engine already has `ctx` available in all its methods. Pass it through:

- `e.store.HasConsumed(ctx, ...)` instead of `e.store.HasConsumed(...)`
- `e.store.RecordConsumption(ctx, ...)` instead of `e.store.RecordConsumption(...)`
- `e.store.SaveCheckpoint(ctx, runID, cp)` instead of `e.store.SaveCheckpoint(runID, cp)`
- `e.store.CreateRun(ctx, name, eventID, eventType)` instead of `e.store.CreateRun(name, eventID, eventType)`
- `e.store.CreateStepRun(ctx, runID, ...)` instead of `e.store.CreateStepRun(runID, ...)`
- `e.store.UpdateStepRun(ctx, stepRunID, ...)` instead of `e.store.UpdateStepRun(stepRunID, ...)`
- `e.store.UpdateRunStatus(ctx, runID, ...)` instead of `e.store.UpdateRunStatus(runID, ...)`
- `e.store.UpdateRunHeartbeat(ctx, runID)` instead of `e.store.UpdateRunHeartbeat(runID)`
- `e.store.GetRun(ctx, runID)` instead of `e.store.GetRun(runID)`
- `e.store.GetCheckpoint(ctx, runID, cp)` instead of `e.store.GetCheckpoint(runID, cp)`
- `e.store.GetIncompleteRuns(ctx)` instead of `e.store.GetIncompleteRuns()`
- `e.store.SaveCheckpoint(ctx, runID, cp)` instead of `e.store.SaveCheckpoint(runID, cp)` (in ResumePipeline)

For `heartbeatLoop`, the goroutine already has `ctx` — pass it:

```go
if err := e.store.UpdateRunHeartbeat(ctx, runID); err != nil {
```

- [ ] **Step 5: Update EventStore callers in `internal/server/pipeline.go:79-81`**

The `ability.SetEventEmitter` callback already receives `ctx context.Context`:

```go
eventStore := store.NewEventStore(store.Database.GetDB().(*store.Client))
_ = eventStore.AppendDataEvent(ctx, dataEvent)
_ = eventStore.AppendEventOutbox(ctx, dataEvent)
```

- [ ] **Step 6: Verify build compiles**

```bash
go build ./pkg/pipeline/...
go build ./internal/store/...
go build ./internal/server/
```

- [ ] **Step 7: Commit**

```bash
git add pkg/pipeline/engine.go internal/store/event_store.go internal/server/pipeline.go
git commit -m "store: add ctx to RunStore, EventStore, PipelineStore"
```

---

### Task 2: AuditStore + HubStore interfaces + implementations + call sites

**Files:**

- Modify: `internal/store/audit_store.go:34-112` (Write, Success, Rejected, Failed)
- Modify: `internal/store/hub_store.go:28-79` (SaveHomelabApps)
- Modify: `internal/server/hub.go:167` (AuditStore call site + writeLifecycleAudit)
- Modify: `internal/server/homelab.go:42-44` (HubStore call site)

- [ ] **Step 1: Add `ctx` to `AuditStore` methods in `internal/store/audit_store.go`**

```go
func (s *AuditStore) Write(ctx context.Context, entry AuditEntry) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	_, err := s.client.AuditLog.Create().
		SetAction(entry.Action).
		SetTargetType(entry.ResourceType).
		SetTargetID(entry.ResourceName).
		SetActorUID(entry.ActorType + ":" + entry.ActorID).
		SetDetails(map[string]any{
			"actor_type":    entry.ActorType,
			"actor_id":      entry.ActorID,
			"uid":           entry.UID,
			"topic":         entry.Topic,
			"action":        entry.Action,
			"resource_type": entry.ResourceType,
			"resource_name": entry.ResourceName,
			"request":       map[string]any(entry.Request),
			"result":        entry.Result,
			"error":         entry.Error,
			"ip_address":    entry.IPAddress,
			"user_agent":    entry.UserAgent,
		}).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *AuditStore) Success(ctx context.Context, actorType, actorID, uid, topic, action, resourceType, resourceName, ip, ua string) error {
	return s.Write(ctx, AuditEntry{
		ActorType: actorType, ActorID: actorID, UID: uid, Topic: topic,
		Action: action, ResourceType: resourceType, ResourceName: resourceName,
		Result: "success", IPAddress: ip, UserAgent: ua,
	})
}

func (s *AuditStore) Rejected(ctx context.Context, actorType, actorID, uid, topic, action, resourceType, resourceName, reason, ip, ua string) error {
	return s.Write(ctx, AuditEntry{
		ActorType: actorType, ActorID: actorID, UID: uid, Topic: topic,
		Action: action, ResourceType: resourceType, ResourceName: resourceName,
		Result: "rejected", Error: reason, IPAddress: ip, UserAgent: ua,
	})
}

func (s *AuditStore) Failed(ctx context.Context, actorType, actorID, uid, topic, action, resourceType, resourceName, errMsg, ip, ua string) error {
	return s.Write(ctx, AuditEntry{
		ActorType: actorType, ActorID: actorID, UID: uid, Topic: topic,
		Action: action, ResourceType: resourceType, ResourceName: resourceName,
		Result: "failed", Error: errMsg, IPAddress: ip, UserAgent: ua,
	})
}
```

- [ ] **Step 2: Add `ctx` to `HubStore` in `internal/store/hub_store.go`**

```go
func (s *HubStore) SaveHomelabApps(ctx context.Context, apps []homelab.App) error {
```

Replace `ctx := context.Background()` with using the passed `ctx` parameter.

- [ ] **Step 3: Update all AuditStore call sites**

In `internal/server/hub.go`, the `writeLifecycleAudit` method needs to accept and pass `ctx`. Find the method and update:

Search for `writeLifecycleAudit` in `hub.go` and add `ctx context.Context` parameter:

```go
func (c *Controller) writeLifecycleAudit(ctx context.Context, appName, action, result, errMsg string) {
```

Each call site already has `ctx` via `fiber.Ctx.Context()`:

```go
c.writeLifecycleAudit(ctx.Context(), app.Name, "hub.apps.start", "failed", err.Error())
```

Update AuditStore instantiation to pass ctx:

```go
auditStore := store.NewAuditStore(store.Database.GetDB().(*store.Client))
_ = auditStore.Write(ctx, store.AuditEntry{...})
```

Similarly update `Success`/`Rejected`/`Failed` calls to pass `ctx`.

- [ ] **Step 4: Update HubStore call site in `internal/server/homelab.go:42-44`**

Find the `SaveHomelabApps` call. It runs in a goroutine with `context.Background()` — keep it as-is:

```go
store.NewHubStore(client).SaveHomelabApps(context.Background(), apps)
```

- [ ] **Step 5: Commit**

```bash
git add internal/store/audit_store.go internal/store/hub_store.go internal/server/hub.go internal/server/homelab.go
git commit -m "store: add ctx to AuditStore, HubStore"
```

---

### Task 3: WorkflowRunStore interface + implementations + Workflow call sites

**Files:**

- Modify: `pkg/workflow/persistence.go:20-30` (WorkflowRunStore interface)
- Modify: `internal/store/workflow_store.go:26-201` (all 9 methods)
- Modify: `pkg/workflow/workflow.go` (all call sites in Runner methods)
- Modify: `pkg/workflow/scheduler.go` (call sites in parallel execution)
- Modify: `internal/modules/workflow/webservice.go:48` (Execute call, passes context.Background())

- [ ] **Step 1: Add `ctx` to `WorkflowRunStore` interface in `pkg/workflow/persistence.go`**

```go
type WorkflowRunStore interface {
	CreateRun(ctx context.Context, workflowName, workflowFile, triggerType string, triggerInfo, inputParams model.JSON) (*model.WorkflowRun, error)
	UpdateRunStatus(ctx context.Context, runID int64, status model.WorkflowRunState, errMsg string) error
	CreateStepRun(ctx context.Context, runID int64, stepID, stepName, action, actionType string, params model.JSON, attempt int) (*model.WorkflowStepRun, error)
	UpdateStepRun(ctx context.Context, stepRunID int64, status model.WorkflowRunState, result model.JSON, errMsg string, attempt int) error
	SaveCheckpoint(ctx context.Context, runID int64, data any) error
	GetIncompleteRuns(ctx context.Context) ([]*model.WorkflowRun, error)
	GetCheckpoint(ctx context.Context, runID int64, target any) error
	GetRun(ctx context.Context, runID int64) (*model.WorkflowRun, error)
	UpdateRunHeartbeat(ctx context.Context, runID int64) error
}
```

- [ ] **Step 2: Add `ctx` to all `WorkflowRunStore` implementation methods in `internal/store/workflow_store.go`**

For each method, add `ctx context.Context` as first parameter. Remove `ctx := context.Background()`. Pass `ctx` to all Ent calls:

- `CreateRun(ctx context.Context, ...)` — replace `ctx := context.Background()` at line 30 with passed `ctx`
- `UpdateRunStatus(ctx context.Context, ...)` — replace line 53
- `CreateStepRun(ctx context.Context, ...)` — replace line 70
- `UpdateStepRun(ctx context.Context, ...)` — replace line 96
- `SaveCheckpoint(ctx context.Context, ...)` — replace line 118
- `GetIncompleteRuns(ctx context.Context)` — replace line 138
- `GetCheckpoint(ctx context.Context, ...)` — replace line 158
- `GetRun(ctx context.Context, ...)` — replace line 181
- `UpdateRunHeartbeat(ctx context.Context, ...)` — replace line 196

- [ ] **Step 3: Update all Workflow call sites in `pkg/workflow/workflow.go`**

Add `ctx` as first argument to all `r.store.Xxx(...)` calls:

In `executeWithRunRecord` (line 232):

```go
run, err = r.store.CreateRun(ctx, wf.Name, workflowFile, triggerType, nil, inputJSON)
```

In `runSequential` (line 283):

```go
_ = r.store.UpdateRunStatus(ctx, run.ID, model.WorkflowRunDone, "")
```

In `executeSequentialStep` (line 313):

```go
stepRun, err = r.store.CreateStepRun(ctx, run.ID, stepID, wt.Describe, wt.Action, info.Type, model.JSON(params), 1)
```

In `executeSequentialMapperStep` (line 352):

```go
_ = r.store.UpdateStepRun(ctx, stepRun.ID, model.WorkflowRunDone, resultJSON, "", 1)
```

In `executeSequentialExecutorStep` (line 420):

```go
_ = r.store.UpdateStepRun(ctx, stepRun.ID, model.WorkflowRunDone, resultJSON, "", attempt)
```

In `saveCheckpoint` (line 436):

```go
if cerr := r.store.SaveCheckpoint(ctx, run.ID, &cp); cerr != nil {
```

Note: `saveCheckpoint` does NOT have ctx — the calling function does. Add ctx parameter to `saveCheckpoint` function:

```go
func saveCheckpoint(ctx context.Context, stepIndex int, r *Runner, wf types.WorkflowMetadata, results map[string]string, input types.KV, run *model.WorkflowRun) {
```

In `ResumeWorkflow` (line 450, 469, 505, 535, 550, 581, 625):

```go
run, err := r.store.GetRun(ctx, runID) // line 450 needs `ctx` — use `context.Background()` or thread the ctx from caller
// ... other calls
_ = r.store.UpdateRunStatus(ctx, runID, model.WorkflowRunDone, "")
```

**Important**: `ResumeWorkflow` doesn't have `ctx` parameter. Add `ctx context.Context` to it:

```go
func (r *Runner) ResumeWorkflow(ctx context.Context, runID int64) error {
```

And replace `context.Background()` at line 490:

```go
hbCtx, cancelHeartbeat = context.WithCancel(ctx)
```

Similarly, the `resume*` step functions need `ctx`:

```go
func (r *Runner) executeResumeStep(ctx context.Context, ...) error {
```

And the executeResumeExecutorStep at line 607:

```go
ctx := context.Background() // REPLACE with passed ctx parameter
```

In `heartbeat` (line 657):

```go
_ = r.store.UpdateRunHeartbeat(ctx, runID) // already has ctx from goroutine
```

In `failRun` (line 636):

```go
_ = r.store.UpdateRunStatus(ctx, run.ID, model.WorkflowRunFailed, err.Error())
```

Add `ctx context.Context` parameter.

In `failStep` (line 643):

```go
_ = r.store.UpdateStepRun(ctx, stepRun.ID, model.WorkflowRunFailed, nil, err.Error(), attempt)
```

Add `ctx context.Context` parameter.

In `runWithRetry` (line 683):

```go
_ = r.store.UpdateStepRun(ctx, stepRun.ID, model.WorkflowRunRunning, nil, err.Error(), attempt)
```

Already has `ctx` parameter.

- [ ] **Step 4: Update all Workflow call sites in `pkg/workflow/scheduler.go`**

Add `ctx` to all `r.store.Xxx(...)` calls in `runParallel`, `executeParallelStep`, `runParallelResume`, etc. Each of these methods already receives `ctx context.Context`.

- [ ] **Step 5: Update `internal/modules/workflow/webservice.go:48`**

```go
if err := runner.Execute(ctx.Context(), *wf, types.KV(body.Params), body.File); err != nil {
```

Replace `context.Background()` with `ctx.Context()` (Fiber context).

- [ ] **Step 6: Verify build compiles**

```bash
go build ./pkg/workflow/...
go build ./internal/store/...
go build ./internal/modules/workflow/...
go build ./internal/server/
```

- [ ] **Step 7: Commit**

```bash
git add pkg/workflow/persistence.go pkg/workflow/workflow.go pkg/workflow/scheduler.go internal/store/workflow_store.go internal/modules/workflow/webservice.go
git commit -m "store: add ctx to WorkflowRunStore"
```

---

### Task 4: Postgres adapter — add ctx to all 86 data-access methods

**Files:**

- Modify: `internal/store/postgres/adapter.go` (lines ~120-1959 — all method signatures)

- [ ] **Step 1: Add `ctx context.Context` to every data-access method signature**

For every method in `adapter` struct (except `Open`, `Close`, `IsOpen`, `GetName`, `Stats`, `GetDB`), add `ctx context.Context` as the first parameter. Replace each `ctx := context.Background()` with the passed `ctx`.

Example patterns (apply to all methods):

```go
// Before:
func (a *adapter) UserCreate(user *model.User) error {
    ctx := context.Background()
    ...

// After:
func (a *adapter) UserCreate(ctx context.Context, user *model.User) error {
    ...
```

All 86 data-access methods follow the same mechanical pattern. The `context` import is already present at the top of the file.

**List of methods to modify (all in `internal/store/postgres/adapter.go`):**

- UserCreate, UserGet, UserGetAll, FirstUser, UserDelete, UserUpdate
- FileStartUpload, FileFinishUpload, FileGet, FileDeleteUnused
- GetUsers, GetUserById, GetUserByFlag
- CreatePlatformUser, GetPlatformUsersByUserId, GetPlatformUserByFlag, UpdatePlatformUser
- GetPlatformChannelByFlag, GetPlatformChannelsByPlatformIds, GetPlatformChannelsByChannelId, CreatePlatformChannel
- CreatePlatformChannelUser, GetPlatformChannelUsersByUserFlag
- GetMessage, GetMessageByPlatform, GetMessagesBySession, CreateMessage
- GetBot, GetBotByName, CreateBot, UpdateBot, DeleteBot, GetBots
- GetPlatform, GetPlatformByName, GetPlatforms, CreatePlatform
- GetChannel, GetChannelByName, CreateChannel, UpdateChannel, DeleteChannel, GetChannels
- DataSet, DataGet, DataList, DataDelete
- ConfigSet, ConfigGet, ListConfigByPrefix, ConfigDelete
- OAuthSet, OAuthGet, OAuthGetAvailable
- FormSet, FormGet, PageSet, PageGet
- BehaviorSet, BehaviorGet, BehaviorList, BehaviorIncrease
- ParameterSet, ParameterGet, ParameterDelete
- CreateInstruct, ListInstruct, UpdateInstruct
- ListWebhook, CreateWebhook, UpdateWebhook, DeleteWebhook, IncreaseWebhookCount, GetWebhookBySecret, GetWebhookByUidAndFlag
- CreateCounter, IncreaseCounter, DecreaseCounter, ListCounter, GetCounter, GetCounterByFlag
- GetAgents, GetAgentByHostid, CreateAgent, UpdateAgentLastOnlineAt, UpdateAgentOnlineDuration

- [ ] **Step 2: Verify build compiles (against old Adapter interface — will compile)**

```bash
go build ./internal/store/postgres/
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/postgres/adapter.go
git commit -m "store: add ctx to postgres adapter data-access methods"
```

---

### Task 5: Adapter interface — add ctx to all 86 data-access methods

**Files:**

- Modify: `internal/store/store.go:150-274` (Adapter interface)

- [ ] **Step 1: Add `ctx context.Context` to every data-access method in the Adapter interface**

This is purely mechanical. `context` is already imported at line 4.

All data-access methods get `ctx context.Context` as first param. Lifecycle methods (`Open`, `Close`, `IsOpen`, `GetName`, `Stats`, `GetDB`) remain unchanged.

```go
type Adapter interface {
	// General (unchanged)
	Open(storeConfig config.StoreType) error
	Close() error
	IsOpen() bool
	GetName() string
	Stats() any
	GetDB() any

	// User management
	UserCreate(ctx context.Context, user *model.User) error
	UserGet(ctx context.Context, uid types.Uid) (*model.User, error)
	// ... (all 86 methods get ctx)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/store/store.go
git commit -m "store: add ctx to Adapter interface"
```

---

### Task 6: Update all 97 call sites across all files

**Files (23 total — update all `store.Database.Xxx(...)` calls):**

Call sites are in these files (grouped by context source):

**HTTP handlers (fiber.Ctx)**:

- Modify: `internal/server/router.go` — Use `c.Context()` for all `store.Database.Xxx(...)` calls
- Modify: `internal/server/func.go` — Already has `ctx` (from protocol handler) — pass `ctx.Context()` or `ctx` directly
- Modify: `internal/server/module.go` — Already has `c` (fiber.Ctx) in some calls; others use types.Context

**HTTP handlers (types.Context)**:

- `internal/modules/webhook/command.go` — calls receive `ctx *types.Context`, pass `ctx.Context`
- `internal/modules/github/command.go` — same
- `internal/modules/github/cron.go` — same
- `internal/modules/server/cron.go` — `GetAgents()` has no ctx, use `context.Background()`
- `internal/modules/server/command.go` — pass `ctx.Context`
- `internal/modules/notify/command.go` — pass `ctx.Context`
- `internal/modules/notify/form.go` — pass `ctx.Context`

**Core packages (types.Context)**:

- `pkg/event/action.go` — pass `ctx.Context`
- `pkg/module/module.go` — pass `ctx.Context`
- `pkg/route/route.go` — call sites may use `context.Background()` for cron-like behavior
- `pkg/notify/notify.go` — pass `ctx.Context()`
- `pkg/media/fs/filesys.go` — pass `ctx.Context`
- `pkg/media/minio/minio.go` — pass `ctx.Context`
- `pkg/types/ruleset/cron/cron.go:132` — `store.Database.GetUsers()` runs in cron context, use `context.Background()`
- `pkg/types/ruleset/page/page.go:30` — pass `ctx.Context()`

- [ ] **Step 1: Update all call sites mechanically**

For each `store.Database.Xxx(args...)` call, add `ctx` as the first argument:

- In HTTP handlers: `store.Database.UserGet(c.Context(), uid)`
- In handlers with `types.Context`: `store.Database.OAuthGet(ctx.Context, ctx.AsUser, ctx.Topic, Name)` — note: `types.Context` has a `.Context` field of type `context.Context`
- In fire-and-forget (cron, heartbeat, event emission): `store.Database.GetAgents(context.Background())`
- In `pkg/route/route.go`: `store.Database.ParameterGet(context.Background(), accessToken)` (auth middleware, no request ctx available)

Read `pkg/types/context.go` to verify the field name for the embedded context (likely `.Context` as shown in existing code).

- [ ] **Step 2: Build and fix compilation errors**

```bash
go build ./... 2>&1 | head -200
```

The compiler will catch every missed call site. Fix incrementally until clean.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "store: update all call sites to pass ctx"
```

---

### Task 7: Verify

- [ ] **Step 1: Build all binaries**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 2: Run vet**

```bash
go vet ./...
```

Expected: clean.

- [ ] **Step 3: Run lint**

```bash
go tool task lint
```

Expected: clean.

- [ ] **Step 4: Run unit tests**

```bash
go tool task test
```

Expected: all pass.

- [ ] **Step 5: Run BDD specs** (if Docker available)

```bash
go tool task test:specs
```

---

### Fire-and-Forget Context Decisions

Specific call sites that use `context.Background()` intentionally:

| Location                              | Method                                 | Reason                               |
| ------------------------------------- | -------------------------------------- | ------------------------------------ |
| `pkg/types/ruleset/cron/cron.go:132`  | `GetUsers()`                           | Cron goroutine, no request ctx       |
| `internal/modules/server/cron.go:160` | `GetAgents()`                          | Cron goroutine                       |
| `internal/server/homelab.go:43`       | `SaveHomelabApps()`                    | Startup goroutine                    |
| `pkg/route/route.go:108,180`          | `ParameterGet()`                       | Auth middleware, pre-request         |
| `internal/server/pipeline.go:79-81`   | `AppendDataEvent`, `AppendEventOutbox` | Event emission has ctx from callback |

All other call sites pass the caller's context.
