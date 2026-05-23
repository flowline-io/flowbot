# Merge store/model into ent/schema Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate `internal/store/model/` package by moving standalone types into `ent/schema/` and replacing Ent-backed DTOs with `gen.*` types directly.

**Architecture:** Move enums, JSON helpers, and standalone structs to `ent/schema/types.go`. Delete ~50 Ent-backed DTO files. Update Adapter interface (59 methods) and postgres adapter to use `gen.*` types. Update ~20 consumer packages to use `gen.*` + `schema.*`.

**Tech Stack:** Go, Ent ORM, PostgreSQL

---

### Task 1: Create `ent/schema/types.go` with all standalone types

**Files:**
- Create: `internal/store/ent/schema/types.go`
- Modify: None yet

- [ ] **Step 1: Write ent/schema/types.go** containing all enum types from model/types.go, JSON/IDList from model/json.go, ResourceRelations/ResourceRef from model/resource_link.go, helper structs (Node, Edge, etc.), and model.go methods. Package name `schema`.

See the complete content inline below.

- [ ] **Step 2: Verify it compiles** - `go build ./internal/store/ent/schema/...`

### Task 2: Update store.go Adapter interface

**Files:**
- Modify: `internal/store/store.go`

- [ ] Replace all `model.*` types in the Adapter interface with `gen.*` types
- [ ] Replace `model.JSON` with `schema.JSON` in store types (PipelineStore, WorkflowRunStore, etc.)
- [ ] Remove `model` import, add `gen` alias reference, add `schema` import paths

### Task 3: Update postgres adapter

**Files:**
- Modify: `internal/store/postgres/adapter.go`
- Modify: `internal/store/postgres/adapter_test.go`

- [ ] Remove all `entXxxToModel` conversion functions
- [ ] Return `*gen.Xxx` directly instead of `*model.Xxx`
- [ ] Drop `model` import, keep `gen` + `schema` imports

### Task 4: Update store.go internal store types

**Files:**
- Modify: `internal/store/store.go` (PipelineStore, WorkflowRunStore, EventStore, etc.)

- [ ] Replace model type references with gen.* + schema.*
- [ ] Update conversion functions (genWorkflowRunToModel, etc.) or remove them

### Task 5: Update all consumers

**Files:** (see consumer list in analysis)
- `pkg/pipeline/engine.go`, `pkg/pipeline/event_handler.go`
- `pkg/workflow/workflow.go`, `pkg/workflow/scheduler.go`, `pkg/workflow/persistence.go`
- `pkg/recovery/recovery.go`
- `internal/server/func.go`, `internal/server/router.go`, `internal/server/module.go`, `internal/server/fx.go`
- `internal/server/pipeline.go`, `internal/server/homelab.go`
- `pkg/event/action.go`, `pkg/event/action_test.go`
- `pkg/module/module.go`
- `pkg/page/layout.go`, `pkg/page/component/*.go`
- `pkg/types/msg.go`
- `internal/modules/hub/webservice.go`, `internal/modules/hub/module.go`
- `internal/modules/workflow/webservice.go`
- `internal/modules/bookmark/command.go`
- `internal/platforms/platforms.go`
- `cmd/composer/action/admin/admin.go`
- `tests/specs/pipeline_spec_test.go`, `tests/specs/event_spec_test.go`

- [ ] Change all `model.*` imports to `schema.*` for enums
- [ ] Change DTO usage to `gen.*` types
- [ ] State comparisons: `model.PipelineStart` → `int(schema.PipelineStart)`

### Task 6: Delete model/ package

**Files:**
- Delete: `internal/store/model/*.go`

- [ ] Delete all files in model/ directory
- [ ] Remove model reference from `internal/store/AGENTS.md`

### Task 7: Final verification

- [ ] `go build ./...` compiles
- [ ] `go tool task lint` passes
- [ ] `go test ./internal/store/...` passes
- [ ] `go test ./...` passes (all unit tests)
