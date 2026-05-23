# Audit Logging Expansion

**Date**: 2026-05-21
**Status**: Draft
**Author**: Flowbot

## Problem

Audit logging (`audit_logs` table) is currently written only by the hub lifecycle controller in `internal/server/hub.go` for app start/stop/restart/pull/update operations. Authentication events, CRUD mutations, configuration changes, pipeline executions, and webhook invocations produce no audit trail. Additionally, actor tracking is hardcoded to `actor_uid = "token:"` with empty SubjectID/IP/UA fields, discarding rich request context available from `auth.Context`.

## Solution

Define an `Auditor` interface implemented by the existing `AuditStore`. Inject the interface into middleware, pipeline/workflow engines, and HTTP handlers. Each component emits semantically-named audit events at key decision points (success, failure, rejection). The `audit_logs` table schema remains unchanged.

### Scope

- Authentication: all auth requests, token create/revoke, scope denial, token validation failure
- CRUD: Homelab Apps, Pipeline/Workflow definitions, configuration changes, webhook registrations
- Pipelines/Workflows: start, completion, failure
- Webhooks: reception, failure
- Existing hub lifecycle audit: enriched with proper actor/subject tracking
- Write-only extension; no read/query API in this iteration
- Synchronous writes with best-effort semantics (audit failure logs a warning, does not block the caller)

### Components

| Component  | File                            | Purpose                                                                 |
| ---------- | ------------------------------- | ----------------------------------------------------------------------- |
| Auditor    | `pkg/audit/audit.go`            | Interface: `Record`, `RecordSuccess`, `RecordFailure`, `RecordRejected` |
| Target     | `pkg/audit/audit.go`            | Struct: `Type` + `ID` for resource identification                       |
| Subject    | `pkg/audit/audit.go`            | Struct: actor identity extracted from `auth.Context`                    |
| AuditStore | `internal/store/audit_store.go` | Concrete implementation of `Auditor` (existing, refactored)             |
| Route      | `pkg/route/route.go`            | `Authorize` middleware writes auth-failure audits                       |
| Pipeline   | `pkg/pipeline/engine.go`        | Pipeline engine writes start/complete/fail audits                       |
| Workflow   | `pkg/workflow/workflow.go`      | Workflow engine writes start/complete/fail audits                       |
| Handlers   | `internal/server/`              | CRUD handlers write mutation audits                                     |

### Auditor Interface

```go
package audit

type Auditor interface {
    Record(ctx context.Context, entry Entry) error
    RecordSuccess(ctx context.Context, entry Entry) error
    RecordFailure(ctx context.Context, entry Entry, err error) error
    RecordRejected(ctx context.Context, entry Entry, reason string) error
}

type Entry struct {
    Subject *Subject
    Action  string
    Target  Target
    Request any
    Result  string
    Error   string
}

type Subject struct {
    SubjectType string
    SubjectID   string
    UID         string
    IPAddress   string
    UserAgent   string
}

type Target struct {
    Type string
    ID   string
}
```

`AuditStore` directly implements `Auditor` — no adapter layer needed.

### Action Naming Convention

Hierarchical, dot-separated: `{domain}.{resource}.{operation}`.

| Domain        | Actions                                                                                                                                                                                                         |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Auth          | `auth.token.create`, `auth.token.revoke`, `auth.token.validate.fail`, `auth.scope.deny`                                                                                                                         |
| CRUD          | `app.create`, `app.delete`, `app.update`, `pipeline.create`, `pipeline.update`, `pipeline.delete`, `workflow.create`, `workflow.update`, `workflow.delete`, `webhook.create`, `webhook.delete`, `config.change` |
| Pipeline/Flow | `pipeline.start`, `pipeline.complete`, `pipeline.fail`, `workflow.start`, `workflow.complete`, `workflow.fail`                                                                                                  |
| Webhook       | `webhook.receive`, `webhook.receive.fail`                                                                                                                                                                       |

### Data Model

The `audit_logs` table (Ent schema) is unchanged:

| Column        | Source                                                          |
| ------------- | --------------------------------------------------------------- |
| `action`      | `Entry.Action`                                                  |
| `target_type` | `Target.Type`                                                   |
| `target_id`   | `Target.ID`                                                     |
| `actor_uid`   | `"{Subject.SubjectType}:{Subject.SubjectID}"`                   |
| `details`     | JSON: subject metadata, IP, UA, request snapshot, error, result |

`details` JSON structure:

```json
{
  "subjectType": "user",
  "subjectID": "owner",
  "uid": "auth0|xxx",
  "ipAddress": "192.168.1.1",
  "userAgent": "curl/8.0",
  "request": {},
  "error": "token expired",
  "result": "failed"
}
```

### Subject Extraction

`AuditStore.Record()` extracts actor identity from `auth.FromContext(ctx)` and maps fields to `audit.Subject`. If `Entry.Subject` is explicitly set (e.g., system pipelines), it takes precedence. If neither is available, `actor_uid` defaults to `":"` and metadata fields are empty.

Mapping: `auth.Context.SubjectType` → `audit.Subject.SubjectType`, `auth.Context.SubjectID` → `audit.Subject.SubjectID`, `auth.Context.UID` → `audit.Subject.UID`, `auth.Context.IPAddress` → `audit.Subject.IPAddress`, `auth.Context.UserAgent` → `audit.Subject.UserAgent`.

### Injection Points

```
cmd/ flowbot main
  │
  ├── entClient ─► auditStore := store.NewAuditStore(client)
  │
  ├── route.NewRouter(auditStore, ...)
  │     Authorize middleware:
  │       ├── token missing/invalid → auditor.RecordRejected("auth.token.validate.fail", ...)
  │       └── scope deny             → auditor.RecordRejected("auth.scope.deny", ...)
  │
  ├── pipeline.NewEngine(auditStore, ...)
  │     Run():
  │       ├── start    → auditor.Record("pipeline.start", ...)
  │       ├── complete → auditor.RecordSuccess("pipeline.complete", ...)
  │       └── fail     → auditor.RecordFailure("pipeline.fail", ...)
  │
  ├── workflow.NewEngine(auditStore, ...)
  │     Run():
  │       ├── start    → auditor.Record("workflow.start", ...)
  │       ├── complete → auditor.RecordSuccess("workflow.complete", ...)
  │       └── fail     → auditor.RecordFailure("workflow.fail", ...)
  │
  ├── HTTP handlers (hub, pipeline, workflow, webhook, config)
  │     CRUD success → auditor.RecordSuccess("app.create", ...)
  │     CRUD failure → auditor.RecordFailure("app.create", ...)
  │
  └── webhook handler
        receive success → auditor.RecordSuccess("webhook.receive", ...)
        receive fail    → auditor.RecordFailure("webhook.receive.fail", ...)
```

### Error Handling

Audit write failures are logged via `flog.Warn` and do not block the caller. The request/operation continues normally. `AuditStore` remains nil-safe: if the store is nil (init failure), writes are silently skipped.

### Edge Cases

| Scenario                 | Behavior                                                   |
| ------------------------ | ---------------------------------------------------------- |
| `AuditStore` is nil      | Silent skip (existing behavior)                            |
| No `auth.Context` in ctx | Empty `actor_uid = ":"`, no metadata in details            |
| Pipeline with system ctx | Uses `SystemPipelineContext()` → `actor_uid = "pipeline:"` |
| Workflow with system ctx | Uses `SystemWorkflowContext()` → `actor_uid = "workflow:"` |
| Concurrent writes        | Ent connection pool, no special handling needed            |
| Empty Target             | Allowed; `target_type` and `target_id` are empty strings   |

### Testing

#### Unit Tests (`*_test.go` co-located)

| File                                 | Coverage                                                                                 |
| ------------------------------------ | ---------------------------------------------------------------------------------------- |
| `internal/store/audit_store_test.go` | `AuditStore` satisfies `Auditor`; Subject extraction; nil-safe; details JSON correctness |
| `pkg/route/route_test.go`            | `Authorize` writes auth-failure audits; scope-denial audits                              |
| `pkg/pipeline/engine_test.go`        | Engine calls Auditor on start/complete/fail                                              |
| `pkg/workflow/workflow_test.go`      | Engine calls Auditor on start/complete/fail                                              |

#### BDD Acceptance Tests (`tests/`)

- Authentication: failed validation, scope denial, token create/revoke
- CRUD: app create/update/delete, pipeline definition changes, config changes, webhook registration
- Pipelines: start, complete, failure
- Webhooks: reception, processing failure

### Refactoring Impact

| Existing code              | Change                                                       |
| -------------------------- | ------------------------------------------------------------ |
| `AuditStore.Success()`     | Rename to `RecordSuccess()`, adjust callers                  |
| `AuditStore.Failed()`      | Rename to `RecordFailure()`, adjust callers                  |
| `AuditStore.Rejected()`    | Rename to `RecordRejected()`, adjust callers                 |
| `AuditStore.Write()`       | Remove from `Auditor` interface, keep internal               |
| `internal/server/hub.go`   | Replace direct `AuditStore` calls with injected `Auditor`    |
| `pkg/pipeline/engine.go`   | Add `auditor audit.Auditor` field to `PipelineEngine` struct |
| `pkg/workflow/workflow.go` | Add `auditor audit.Auditor` field to `WorkflowEngine` struct |
| `pkg/route/route.go`       | Add `auditor audit.Auditor` field, wire into `Authorize`     |
| `cmd/` entry points        | Wire auditStore into all four injection points               |
