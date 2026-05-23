# Audit Logging Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend audit logging from hub-only to cover authentication, CRUD, pipelines, workflows, webhooks, and config changes.

**Architecture:** Define an `Auditor` interface in `pkg/audit/`, implement it with existing `AuditStore` (renamed methods), inject auditor into middleware via package-level setter, into pipeline/workflow engines via constructor, and into HTTP handlers via Controller struct. All writes are synchronous, non-blocking on failure.

**Tech Stack:** Go 1.26+, ent ORM, Fiber v3, uber fx, testify, sonic

---

### Task 1: Create `pkg/audit/audit.go` (Auditor interface and types)

**Files:**

- Create: `pkg/audit/audit.go`

- [ ] **Step 1: Create the file**

```go
// Package audit provides the Auditor interface and supporting types for audit logging.
package audit

import "context"

// Auditor writes audit entries to persistent storage.
type Auditor interface {
	Record(ctx context.Context, entry Entry) error
	RecordSuccess(ctx context.Context, entry Entry) error
	RecordFailure(ctx context.Context, entry Entry, err error) error
	RecordRejected(ctx context.Context, entry Entry, reason string) error
}

// Entry represents a single audit record.
type Entry struct {
	Subject *Subject
	Action  string
	Target  Target
	Request any
}

// Subject carries authenticated actor identity.
type Subject struct {
	SubjectType string
	SubjectID   string
	UID         string
	IPAddress   string
	UserAgent   string
}

// Target identifies the resource being acted on.
type Target struct {
	Type string
	ID   string
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/audit/
```

Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add pkg/audit/audit.go
git commit -m "feat: add audit.Auditor interface and types"
```

---

### Task 2: Refactor `AuditStore` to implement `Auditor`

**Files:**

- Modify: `internal/store/audit_store.go`

- [ ] **Step 1: Replace the file content**

```go
package store

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/flog"
)

type AuditStore struct {
	client *gen.Client
}

func NewAuditStore(client *gen.Client) *AuditStore {
	return &AuditStore{client: client}
}

// Record writes an audit entry to persistent storage.
// If the store or client is nil, the call is silently skipped.
// Audit write failures are logged and do not propagate to the caller.
func (s *AuditStore) Record(ctx context.Context, entry audit.Entry) error {
	if s == nil || s.client == nil {
		return nil
	}
	actorUID := ""
	details := map[string]any{}
	if entry.Subject != nil {
		actorUID = entry.Subject.SubjectType + ":" + entry.Subject.SubjectID
		details["subject_type"] = entry.Subject.SubjectType
		details["subject_id"] = entry.Subject.SubjectID
		details["uid"] = entry.Subject.UID
		details["ip_address"] = entry.Subject.IPAddress
		details["user_agent"] = entry.Subject.UserAgent
	}
	if entry.Request != nil {
		details["request"] = entry.Request
	}
	now := time.Now()
	_, err := s.client.AuditLog.Create().
		SetAction(entry.Action).
		SetTargetType(entry.Target.Type).
		SetTargetID(entry.Target.ID).
		SetActorUID(actorUID).
		SetDetails(details).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		flog.Warn("audit write failed: %v", err)
		return nil
	}
	return nil
}

// RecordSuccess writes a success audit entry.
func (s *AuditStore) RecordSuccess(ctx context.Context, entry audit.Entry) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "success")
	return s.Record(ctx, e)
}

// RecordFailure writes a failure audit entry with the error message.
func (s *AuditStore) RecordFailure(ctx context.Context, entry audit.Entry, err error) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "failed")
	if err != nil {
		e.Request = wrapResult(e.Request, "error", err.Error())
	}
	return s.Record(ctx, e)
}

// RecordRejected writes a rejected audit entry with the reason.
func (s *AuditStore) RecordRejected(ctx context.Context, entry audit.Entry, reason string) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "rejected")
	e.Request = wrapResult(e.Request, "error", reason)
	return s.Record(ctx, e)
}

func wrapResult(request any, key, value string) map[string]any {
	m := map[string]any{key: value}
	if request != nil {
		if existing, ok := request.(map[string]any); ok {
			for k, v := range existing {
				if _, exists := m[k]; !exists {
					m[k] = v
				}
			}
		}
	}
	return m
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/store/
```

Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/store/audit_store.go
git commit -m "refactor: AuditStore implements audit.Auditor interface"
```

---

### Task 3: Write `AuditStore` unit tests

**Files:**

- Create: `internal/store/audit_store_test.go`

- [ ] **Step 1: Write the test file**

```go
package store

import (
	"context"
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/stretchr/testify/assert"
)

func TestAuditStore_ImplementsAuditor(t *testing.T) {
	t.Parallel()
	var _ audit.Auditor = (*AuditStore)(nil)
}

func TestAuditStore_NilSafe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		store *AuditStore
	}{
		{name: "nil store", store: nil},
		{name: "zero store", store: &AuditStore{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.Record(context.Background(), audit.Entry{
				Action: "test.action",
				Target: audit.Target{Type: "test", ID: "1"},
			})
			assert.NoError(t, err)
		})
	}
}

func TestAuditStore_RecordSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.action"}, wantErr: false},
		{name: "zero store", store: &AuditStore{}, entry: audit.Entry{Action: "test.action"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordSuccess(context.Background(), tt.entry)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_RecordFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		err     error
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.fail"}, err: errors.New("boom"), wantErr: false},
		{name: "zero store", store: &AuditStore{}, entry: audit.Entry{Action: "test.fail"}, err: errors.New("boom"), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordFailure(context.Background(), tt.entry, tt.err)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_RecordRejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		reason  string
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.deny"}, reason: "nope", wantErr: false},
		{name: "zero store", store: &AuditStore{}, entry: audit.Entry{Action: "test.deny"}, reason: "nope", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordRejected(context.Background(), tt.entry, tt.reason)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_SubjectExtraction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		subject *audit.Subject
	}{
		{
			name: "full subject",
			subject: &audit.Subject{
				SubjectType: "user",
				SubjectID:   "owner",
				UID:         "auth0|123",
				IPAddress:   "10.0.0.1",
				UserAgent:   "test/1.0",
			},
		},
		{
			name:    "nil subject",
			subject: nil,
		},
		{
			name: "system pipeline",
			subject: &audit.Subject{
				SubjectType: "pipeline",
				SubjectID:   "system:pipeline",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &AuditStore{}
			err := store.Record(context.Background(), audit.Entry{
				Subject: tt.subject,
				Action:  "test.action",
				Target:  audit.Target{Type: "test", ID: "1"},
			})
			assert.NoError(t, err)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test ./internal/store/ -run TestAuditStore -v
```

Expected: all tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/store/audit_store_test.go
git commit -m "test: add AuditStore auditor interface tests"
```

---

### Task 4: Add audit to route `Authorize` middleware

**Files:**

- Modify: `pkg/route/route.go`

- [ ] **Step 1: Add SetAuditor and modify Authorize**

Replace `pkg/route/route.go` content (additions inline):

After the existing imports, add:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

After the existing constants, add the SetAuditor function:

```go
var routeAuditor audit.Auditor

// SetAuditor sets the global auditor used for auth event recording.
func SetAuditor(a audit.Auditor) {
	routeAuditor = a
}
```

Modify the `Authorize` function. Replace lines 94-147 (the Authorize function) with:

```go
func Authorize(authLevel AuthLevel, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if authLevel == NoAuth {
			return handler(ctx)
		}

		var r http.Request
		if err := fasthttpadaptor.ConvertRequest(ctx.RequestCtx(), &r, true); err != nil {
			auditAuthReject(ctx, "auth.token.validate.fail", "request conversion failed", "error", err.Error())
			return protocol.ErrNotAuthorized.Wrap(err)
		}

		accessToken := GetAccessToken(&r)
		if accessToken == "" {
			auditAuthReject(ctx, "auth.token.validate.fail", "token", "missing token")
			return protocol.ErrNotAuthorized.New("Missing token")
		}

		p, err := store.Database.ParameterGet(context.Background(), accessToken)
		if err != nil || p.ID <= 0 || p.IsExpired() {
			auditAuthReject(ctx, "auth.token.validate.fail", "token", "invalid or expired")
			return protocol.ErrNotAuthorized.New("parameter error")
		}

		paramKV := types.KV(p.Params)
		topic, _ := paramKV.String("topic")
		uidStr, _ := paramKV.String("uid")
		uid := types.Uid(uidStr)

		if uid.IsZero() {
			auditAuthReject(ctx, "auth.token.validate.fail", "token", "uid empty")
			return protocol.ErrNotAuthorized.New("uid empty")
		}

		var scopes []string
		if raw, ok := paramKV["scopes"]; ok {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						scopes = append(scopes, s)
					}
				}
			case []string:
				scopes = v
			}
		}

		ctx.Locals(requestContextKey, &RequestContext{
			UID:    uid,
			Topic:  topic,
			Param:  paramKV,
			Scopes: scopes,
		})

		return handler(ctx)
	}
}

func auditAuthReject(ctx fiber.Ctx, action, targetType, reason ...string) {
	if routeAuditor == nil {
		return
	}
	ip := ctx.IP()
	ua := string(ctx.Request().Header.UserAgent())
	_ = routeAuditor.RecordRejected(context.Background(), audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "token",
			IPAddress:   ip,
			UserAgent:   ua,
		},
		Action: action,
		Target: audit.Target{Type: targetType},
	}, strings.Join(reason, ": "))
}
```

- [ ] **Step 2: Modify `RequireScope` to audit scope denials**

Replace the `RequireScope` function with:

```go
func RequireScope(scope string, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		scopes := GetScopes(ctx)
		if !auth.HasScope(scopes, scope) {
			auditScopeDeny(ctx, scope)
			return protocol.ErrAccessDenied.New("insufficient scope: " + scope)
		}
		return handler(ctx)
	}
}

func auditScopeDeny(ctx fiber.Ctx, scope string) {
	if routeAuditor == nil {
		return
	}
	rc := GetRequestContext(ctx)
	uid := ""
	if rc != nil {
		uid = string(rc.UID)
	}
	ip := ctx.IP()
	ua := string(ctx.Request().Header.UserAgent())
	_ = routeAuditor.RecordRejected(context.Background(), audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "token",
			UID:         uid,
			IPAddress:   ip,
			UserAgent:   ua,
		},
		Action: "auth.scope.deny",
		Target: audit.Target{Type: "scope"},
	}, "required: "+scope)
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./pkg/route/
```

Expected: no output (success)

- [ ] **Step 4: Commit**

```bash
git add pkg/route/route.go
git commit -m "feat: add auth audit logging to Authorize and RequireScope middleware"
```

---

### Task 5: Write route auth audit tests

**Files:**

- Create: `pkg/route/route_test.go`

- [ ] **Step 1: Write the test file**

```go
package route

import (
	"context"
	"io"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/audit"
)

type mockAuditor struct {
	entries []audit.Entry
}

func (m *mockAuditor) Record(_ context.Context, entry audit.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditor) RecordSuccess(_ context.Context, entry audit.Entry) error {
	return m.Record(nil, entry)
}

func (m *mockAuditor) RecordFailure(_ context.Context, entry audit.Entry, _ error) error {
	return m.Record(nil, entry)
}

func (m *mockAuditor) RecordRejected(_ context.Context, entry audit.Entry, _ string) error {
	return m.Record(nil, entry)
}

func TestAuthorize_AuditNoAuditor(t *testing.T) {
	t.Parallel()
	SetAuditor(nil)
	app := fiber.New()
	app.Get("/test", Authorize(0, func(c fiber.Ctx) error {
		return c.SendString("ok")
	}))
	req, _ := app.Test(io.NopCloser, nil).Get("/test")
	assert.Equal(t, 200, req.StatusCode())
}

func TestAuthorize_AuditTokenMissing(t *testing.T) {
	t.Parallel()
	m := &mockAuditor{}
	SetAuditor(m)
	app := fiber.New()
	app.Get("/test", Authorize(0, func(c fiber.Ctx) error {
		return c.SendString("ok")
	}))
	req, _ := app.Test(io.NopCloser, nil).Get("/test")
	status := req.StatusCode()
	assert.True(t, status == 401 || status == 403, "expected 401/403, got %d", status)
	require.Len(t, m.entries, 1)
	assert.Equal(t, "auth.token.validate.fail", m.entries[0].Action)
	assert.Equal(t, "token", m.entries[0].Target.Type)
}

func TestAuthorize_AuditRequireScopeDeny(t *testing.T) {
	t.Parallel()
	m := &mockAuditor{}
	SetAuditor(m)
	app := fiber.New()
	app.Get("/test", RequireScope("admin:test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	}))
	req, _ := app.Test(io.NopCloser, nil).Get("/test")
	status := req.StatusCode()
	assert.True(t, status == 401 || status == 403, "expected 401/403, got %d", status)
	require.Len(t, m.entries, 1)
	assert.Equal(t, "auth.scope.deny", m.entries[0].Action)
}

func TestAuditor_SetAndNil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		auditor audit.Auditor
	}{
		{name: "set mock", auditor: &mockAuditor{}},
		{name: "set nil", auditor: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			SetAuditor(tt.auditor)
			// Verify setting works without panic
		})
	}
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./pkg/route/ -run TestAuthorize_Audit -v
go test ./pkg/route/ -run TestAuditor -v
```

Expected: all tests PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/route/route_test.go
git commit -m "test: add route auth audit tests"
```

---

### Task 6: Add auditor field to Pipeline Engine

**Files:**

- Modify: `pkg/pipeline/engine.go`

- [ ] **Step 1: Add import and modify Engine struct and NewEngine**

Add to imports:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

Modify the `Engine` struct (around line 58):

```go
type Engine struct {
	defs            []Definition
	store           RunStore
	auditor         audit.Auditor
	pipelineMetrics *metrics.PipelineCollector
	eventMetrics    *metrics.EventCollector
	handler         func(ctx context.Context, event types.DataEvent) error
}
```

Modify `NewEngine` (around line 66):

```go
func NewEngine(defs []Definition, store RunStore, auditor audit.Auditor, pc *metrics.PipelineCollector, ec *metrics.EventCollector) *Engine {
	e := &Engine{
		defs:            defs,
		store:           store,
		auditor:         auditor,
		pipelineMetrics: pc,
		eventMetrics:    ec,
	}
	e.handler = e.handleEvent
	return e
}
```

- [ ] **Step 2: Add audit calls in executePipeline**

Modify `executePipeline` to audit start/complete/fail. Insert at line 107 (after `runStart := time.Now()`):

```go
	runStart := time.Now()

	e.auditPipelineEvent(ctx, def.Name, "pipeline.start", event.EventID, event.EventType)

	alreadyDone, err := e.checkDedupAndRecord(ctx, def.Name, event.EventID, event.EventType)
```

And at line 145-150, modify `finishRunRecord` section:

```go
	e.finishRunRecord(ctx, runID, failed, finalErr)

	if finalErr != nil {
		e.auditPipelineEvent(ctx, def.Name, "pipeline.fail", event.EventID, event.EventType)
		return finalErr
	}
	e.auditPipelineEvent(ctx, def.Name, "pipeline.complete", event.EventID, event.EventType)
	return nil
```

- [ ] **Step 3: Add auditPipelineEvent helper method**

Add to `engine.go` (before `checkDedupAndRecord`):

```go
func (e *Engine) auditPipelineEvent(ctx context.Context, pipelineName, action, eventID, eventType string) {
	if e.auditor == nil {
		return
	}
	_ = e.auditor.Record(ctx, audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "pipeline",
			SubjectID:   "system:pipeline",
		},
		Action: action,
		Target: audit.Target{Type: "pipeline", ID: pipelineName},
		Request: map[string]any{
			"event_id":   eventID,
			"event_type": eventType,
		},
	})
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./pkg/pipeline/
```

Expected: no output (success)

- [ ] **Step 5: Commit**

```bash
git add pkg/pipeline/engine.go
git commit -m "feat: add audit logging to pipeline engine start/complete/fail"
```

---

### Task 7: Write pipeline audit tests

**Files:**

- Modify: `pkg/pipeline/pipeline_test.go` (append tests)

- [ ] **Step 1: Add mock auditor and tests**

Append to `pkg/pipeline/pipeline_test.go`:

```go
type mockAuditor struct {
	entries []audit.Entry
}

func (m *mockAuditor) Record(_ context.Context, entry audit.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockAuditor) RecordSuccess(_ context.Context, entry audit.Entry) error { return m.Record(nil, entry) }
func (m *mockAuditor) RecordFailure(_ context.Context, entry audit.Entry, _ error) error { return m.Record(nil, entry) }
func (m *mockAuditor) RecordRejected(_ context.Context, entry audit.Entry, _ string) error { return m.Record(nil, entry) }

func TestEngine_Audit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		pipelineName  string
		event         types.DataEvent
		expectActions []string
	}{
		{
			name:         "audit start and complete",
			pipelineName: "audit-pl",
			event:        types.DataEvent{EventID: "evt1", EventType: "test.event"},
			expectActions: []string{"pipeline.start", "pipeline.complete"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &mockAuditor{}
			defs := []Definition{
				{
					Name:    tt.pipelineName,
					Enabled: true,
					Trigger: Trigger{Event: "test.event"},
					Steps:   []Step{},
				},
			}
			e := NewEngine(defs, nil, m, noopPC, noopEC)
			_ = e.Handler()(context.Background(), tt.event)
			require.Len(t, m.entries, len(tt.expectActions))
			for i, expected := range tt.expectActions {
				assert.Equal(t, expected, m.entries[i].Action)
				assert.Equal(t, "pipeline", m.entries[i].Target.Type)
				assert.Equal(t, tt.pipelineName, m.entries[i].Target.ID)
			}
		})
	}
}

func TestNewEngine_WithAuditor(t *testing.T) {
	t.Parallel()
	m := &mockAuditor{}
	noopPC := metrics.NewPipelineCollector(nil)
	noopEC := metrics.NewEventCollector(nil)
	tests := []struct {
		name    string
		auditor audit.Auditor
	}{
		{name: "with auditor", auditor: m},
		{name: "with nil auditor", auditor: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(nil, nil, tt.auditor, noopPC, noopEC)
			assert.NotNil(t, e)
		})
	}
}
```

Also add the imports:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

Note: also add `context` import if not already present.

- [ ] **Step 2: Run tests**

```bash
go test ./pkg/pipeline/ -run TestEngine_Audit -v
go test ./pkg/pipeline/ -run TestNewEngine_WithAuditor -v
```

Expected: all tests PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/pipeline/pipeline_test.go
git commit -m "test: add pipeline audit tests"
```

---

### Task 8: Add auditor field to Workflow Runner

**Files:**

- Modify: `pkg/workflow/workflow.go`

- [ ] **Step 1: Add import and modify Runner struct and constructor**

Add to imports:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

Modify the `Runner` struct (around line 146):

```go
type Runner struct {
	engines      map[string]*executor.Engine
	store        WorkflowRunStore
	auditor      audit.Auditor
	metrics      *metrics.WorkflowCollector
	workflowFile string
	triggerType  string
}
```

Modify `NewRunnerWithStore` (around line 161):

```go
func NewRunnerWithStore(store WorkflowRunStore, auditor audit.Auditor, wc *metrics.WorkflowCollector, workflowFile, triggerType string) *Runner {
	return &Runner{
		engines: map[string]*executor.Engine{
			runtime.Capability: executor.New(runtime.Capability),
			runtime.Shell:      executor.New(runtime.Shell),
			runtime.Docker:     executor.New(runtime.Docker),
			runtime.Machine:    executor.New(runtime.Machine),
		},
		store:        store,
		auditor:      auditor,
		metrics:      wc,
		workflowFile: workflowFile,
		triggerType:  triggerType,
	}
}
```

Fix `NewRunner` (around line 155) to pass nil auditor:

```go
func NewRunner() *Runner {
	return NewRunnerWithStore(nil, nil, nil, "", "")
}
```

- [ ] **Step 2: Add audit calls and helper**

Add this helper method after the `Runner` struct definition:

```go
func (r *Runner) auditWorkflowEvent(ctx context.Context, wfName, action string) {
	if r.auditor == nil {
		return
	}
	_ = r.auditor.Record(ctx, audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "workflow",
			SubjectID:   "system:workflow",
		},
		Action: action,
		Target: audit.Target{Type: "workflow", ID: wfName},
	})
}
```

- [ ] **Step 3: Add audit calls in runSequential and fail paths**

In `runSequential` (around line 253), add audit at the start:

```go
func (r *Runner) runSequential(ctx context.Context, wf types.WorkflowMetadata, input types.KV, taskMap map[string]types.WorkflowTask, run *model.WorkflowRun, cancelHeartbeat context.CancelFunc) error {
	start := time.Now()
	r.auditWorkflowEvent(ctx, wf.Name, "workflow.start")
```

In `runSequential` completion path (around line 279-285):

```go
	if r.store != nil && run != nil {
		if cancelHeartbeat != nil {
			cancelHeartbeat()
		}
		_ = r.store.UpdateRunStatus(ctx, run.ID, model.WorkflowRunDone, "")
	}

	r.auditWorkflowEvent(ctx, wf.Name, "workflow.complete")
	return nil
```

In `runSequential` failure path (around line 273):

```go
		if err := r.executeSequentialStep(ctx, stepID, taskMap, wf, results, input, run); err != nil {
			r.failRun(ctx, run, cancelHeartbeat, err)
			r.auditWorkflowEvent(ctx, wf.Name, "workflow.fail")
			runErr = err
			return runErr
		}
```

- [ ] **Step 4: Also audit workflow.fail in failRun**

Modify `failRun` at around line 634:

```go
func (r *Runner) failRun(ctx context.Context, run *model.WorkflowRun, cancelHeartbeat context.CancelFunc, err error) {
	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}
	if r.store != nil && run != nil {
		_ = r.store.UpdateRunStatus(ctx, run.ID, model.WorkflowRunFailed, err.Error())
	}
	// Note: workflow.fail audit is already recorded in runSequential before calling failRun
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go build ./pkg/workflow/
```

Expected: no output (success)

- [ ] **Step 6: Commit**

```bash
git add pkg/workflow/workflow.go
git commit -m "feat: add audit logging to workflow runner start/complete/fail"
```

---

### Task 9: Write workflow audit tests

**Files:**

- Modify: `pkg/workflow/workflow_test.go` (append tests)

- [ ] **Step 1: Check the existing test file structure first**

```bash
head -50 pkg/workflow/workflow_test.go
```

- [ ] **Step 2: Append audit tests**

Append to `pkg/workflow/workflow_test.go`:

```go
type mockAuditor struct {
	entries []audit.Entry
}

func (m *mockAuditor) Record(_ context.Context, entry audit.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockAuditor) RecordSuccess(_ context.Context, entry audit.Entry) error { return m.Record(nil, entry) }
func (m *mockAuditor) RecordFailure(_ context.Context, entry audit.Entry, _ error) error { return m.Record(nil, entry) }
func (m *mockAuditor) RecordRejected(_ context.Context, entry audit.Entry, _ string) error { return m.Record(nil, entry) }

func TestRunner_Audit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		wfName        string
		expectActions []string
	}{
		{
			name:          "audit start and complete",
			wfName:        "test-wf",
			expectActions: []string{"workflow.start", "workflow.complete"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &mockAuditor{}
			runner := NewRunnerWithStore(nil, m, nil, "", "")
			assert.NotNil(t, runner)
		})
	}
}

func TestNewRunnerWithStore_DefaultAuditor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		auditor audit.Auditor
	}{
		{name: "nil auditor", auditor: nil},
		{name: "mock auditor", auditor: &mockAuditor{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRunnerWithStore(nil, tt.auditor, nil, "", "")
			assert.NotNil(t, r)
		})
	}
}
```

Add imports:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

- [ ] **Step 3: Run tests**

```bash
go test ./pkg/workflow/ -run TestRunner_Audit -v
go test ./pkg/workflow/ -run TestNewRunnerWithStore_DefaultAuditor -v
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/workflow/workflow_test.go
git commit -m "test: add workflow audit tests"
```

---

### Task 10: Wire auditor in Fx dependency injection

**Files:**

- Modify: `internal/server/fx.go`

- [ ] **Step 1: Add auditor provider and route wiring**

In `fx.go`, add the `newAuditor` provider function and wire it into the `fx.Provide` block. Also add the `setRouteAuditor` invoke.

First, add to imports (ensure `audit` package is imported):

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/route"
)
```

Add `newAuditor` to the `fx.Provide` block:

```go
	fx.Provide(
		config.NewConfig,
		cache.NewCache,
		rdb.NewClient,
		cache.NewRedisStore,
		search.NewClient,
		event.NewRouter,
		event.NewSubscriber,
		event.NewPublisher,
		slack.NewDriver,
		trace.NewTracerProvider,
		newController,
		newDatabaseAdapter,
		newHTTPServer,
		newAuditor,  // <-- add this line
	),
```

Add `setRouteAuditor` to the `fx.Invoke` block BEFORE `handleRoutes`:

```go
	fx.Invoke(
		setServerCacheStore,
		setModuleServerCacheStore,
		setModuleCacheStore,
		setBookmarkCacheStore,
		setReaderCacheStore,
		setKanbanCacheStore,
		setGiteaCacheStore,
		setRouteAuditor,   // <-- add this line (must be before handleRoutes)
		handleRoutes,
		handleEvents,
		handleModules,
		handlePlatform,
		initPipeline,
		RunServer,
		profiling.NewProfiler,
	),
```

Add the two new helper functions at the bottom of fx.go:

```go
// newAuditor creates an audit.Auditor from the global store database.
// Returns nil if the database is not yet initialized.
func newAuditor() audit.Auditor {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewAuditStore(client)
}

// setRouteAuditor injects the global auditor into the route package
// for auth failure audit logging in the Authorize middleware.
func setRouteAuditor(a audit.Auditor) {
	route.SetAuditor(a)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/server/
```

Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/server/fx.go
git commit -m "feat: wire audit.Auditor into Fx dependency injection"
```

---

### Task 11: Replace hub lifecycle audit with Auditor

**Files:**

- Modify: `internal/server/hub.go`
- Modify: `internal/server/router.go` (Controller struct + handleRoutes)

- [ ] **Step 1: Add auditor field to Controller struct**

In `internal/server/router.go`, modify the `Controller` struct:

```go
type Controller struct {
	driver         protocol.Driver
	tailchatDriver protocol.Driver
	auditor        audit.Auditor  // <-- add this field
}
```

Add import:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

- [ ] **Step 2: Modify newController to accept auditor**

```go
func newController(driver protocol.Driver, cfg *config.Type, storeAdapter store.Adapter, auditor audit.Auditor) *Controller {
	return &Controller{
		driver:         driver,
		tailchatDriver: tailchat.NewDriver(cfg, storeAdapter),
		auditor:        auditor,
	}
}
```

(Note: Fx auto-injects `audit.Auditor` from the `newAuditor` provider added in Task 10.)

- [ ] **Step 3: Modify handleRoutes signature to pass Controller**

```go
func handleRoutes(a *fiber.App, ctl *Controller) {
```

This is already the signature; no change needed to the function declaration. But change method calls from `(*Controller).hubAppStart` to `ctl.hubAppStart` etc. Looking at the existing code, it already uses `ctl.` — good.

- [ ] **Step 4: rewrite hub.go writeLifecycleAudit to use controller.auditor**

Replace `writeLifecycleAudit` method and all 12 call sites. In `internal/server/hub.go`:

Delete the `writeLifecycleAudit` method (lines 165-180).

Replace all calls:

- `c.writeLifecycleAudit(ctx.Context(), app.Name, "hub.apps.start", "failed", err.Error())`
  → `c.writeLifecycleAudit(ctx.Context(), app.Name, "hub.apps.start", "failed", err.Error())`

Actually, let's keep the method but change its implementation:

Replace the method body:

```go
func (c *Controller) writeLifecycleAudit(ctx context.Context, appName, action, result, errMsg string) {
	if c.auditor == nil {
		return
	}
	entry := audit.Entry{
		Action: action,
		Target: audit.Target{Type: "app", ID: appName},
	}
	switch result {
	case "success":
		_ = c.auditor.RecordSuccess(ctx, entry)
	case "failed":
		_ = c.auditor.RecordFailure(ctx, entry, fmt.Errorf("%s", errMsg))
	case "rejected":
		_ = c.auditor.RecordRejected(ctx, entry, errMsg)
	}
}
```

Also update the import: remove `store` import from hub.go (clean up unused). Add `audit` import.

- [ ] **Step 5: Also audit doWebhook in router.go**

Modify `doWebhook` in `internal/server/router.go` (around line 408). Add audit at the end of the function:

After line 474 `return ctx.JSON(payload)`, wrap with audit. Actually, modify the flow:

```go
func (c *Controller) doWebhook(ctx fiber.Ctx) error {
	flag := ctx.Params("flag")
	// ... existing code ...

	payload, err := botHandler.Webhook(typesCtx, data)
	if err != nil {
		c.auditWebhook(ctx, "webhook.receive.fail", flag, err)
		return protocol.ErrFlagError.Wrap(err)
	}

	// ... existing code (increase count) ...

	c.auditWebhook(ctx, "webhook.receive", flag, nil)
	return ctx.JSON(payload)
}

func (c *Controller) auditWebhook(ctx fiber.Ctx, action, flag string, err error) {
	if c.auditor == nil {
		return
	}
	entry := audit.Entry{
		Action: action,
		Target: audit.Target{Type: "webhook", ID: flag},
	}
	if err != nil {
		_ = c.auditor.RecordFailure(ctx.Context(), entry, err)
	} else {
		_ = c.auditor.RecordSuccess(ctx.Context(), entry)
	}
}
```

- [ ] **Step 6: Verify it compiles**

```bash
go build ./internal/server/
```

Expected: no output (success)

- [ ] **Step 7: Commit**

```bash
git add internal/server/hub.go internal/server/router.go
git commit -m "feat: replace direct AuditStore calls with injected Auditor in hub and webhook handlers"
```

---

### Task 12: Pass auditor to Pipeline and Workflow engines

**Files:**

- Modify: `internal/server/pipeline.go`
- Modify: `internal/modules/workflow/webservice.go`

- [ ] **Step 1: Pass auditor to NewEngine in initPipeline**

In `internal/server/pipeline.go`, add `auditor audit.Auditor` to the `initPipeline` function parameters. Fx auto-injects it via the `newAuditor` provider.

Change the function signature at line 25:

```go
func initPipeline(
	_ fx.Lifecycle,
	cfg *config.Type,
	router *message.Router,
	subscriber message.Subscriber,
	pc *metrics.PipelineCollector,
	ec *metrics.EventCollector,
	ac *metrics.AbilityCollector,
	auditor audit.Auditor,  // <-- add; Fx auto-injects
) error {
```

At line 47, pass auditor to NewEngine:

```go
	engine := pipeline.NewEngine(pipelineDefs, runStore, auditor, pc, ec)
```

Add import:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

- [ ] **Step 2: Pass auditor to NewRunnerWithStore in workflow webservice**

In `internal/modules/workflow/webservice.go`, pass auditor when creating runner. Need to get the auditor.

Looking at the code, the store client is obtained from `store.Database.GetDB()`. We need auditor similarly. Since `AuditStore` is available through store package:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/audit"
)
```

At line 39, replace:

```go
	runner := workflowpkg.NewRunner()
	var runStore workflowpkg.WorkflowRunStore
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			runStore = store.NewWorkflowRunStore(client)
		}
		runner = workflowpkg.NewRunnerWithStore(runStore, nil, body.File, "manual")
	}
```

With:

```go
	runner := workflowpkg.NewRunner()
	var runStore workflowpkg.WorkflowRunStore
	var auditor audit.Auditor
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			runStore = store.NewWorkflowRunStore(client)
			auditor = store.NewAuditStore(client)
		}
		runner = workflowpkg.NewRunnerWithStore(runStore, auditor, nil, body.File, "manual")
	}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/server/
go build ./internal/modules/workflow/
```

Expected: no output (success)

- [ ] **Step 4: Commit**

```bash
git add internal/server/pipeline.go internal/modules/workflow/webservice.go
git commit -m "feat: wire auditor into pipeline engine and workflow runner"
```

---

### Task 13: Update existing tests for new signatures

**Files:**

- Modify: `pkg/pipeline/pipeline_test.go` (TestNewEngine)
- Modify: `pkg/workflow/workflow_test.go` (NewRunner calls)
- Modify: `tests/specs/pipeline_spec_test.go` (NewEngine calls)

- [ ] **Step 1: Fix TestNewEngine in pipeline_test.go**

Find `TestNewEngine` at around line 668. Replace `NewEngine(tt.defs, tt.store, noopPC, noopEC)` with `NewEngine(tt.defs, tt.store, nil, noopPC, noopEC)`.

- [ ] **Step 2: Fix BDD pipeline spec tests**

In `tests/specs/pipeline_spec_test.go`, search for `NewEngine` calls and add `nil` as the third argument.

- [ ] **Step 3: Run all existing pipeline and workflow tests**

```bash
go test ./pkg/pipeline/ -v
go test ./pkg/workflow/ -v
```

Expected: all existing tests PASS

- [ ] **Step 4: Run full test suite**

```bash
go tool task test
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/pipeline/pipeline_test.go pkg/workflow/workflow_test.go tests/specs/pipeline_spec_test.go
git commit -m "test: update tests for new auditor constructor parameter"
```

---

### Task 14: Final verification and lint

**Files:** none (verification only)

- [ ] **Step 1: Run lint**

```bash
go tool task lint
```

Expected: no errors

- [ ] **Step 2: Run all unit tests**

```bash
go tool task test
```

Expected: all PASS

- [ ] **Step 3: Run full build**

```bash
go tool task build
```

Expected: success
