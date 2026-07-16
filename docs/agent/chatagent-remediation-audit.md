## Chatagent Remediation Audit (WS-B/WS-C)

This document captures evidence-based audit notes used by the remediation plan.
If there is no concrete duplication/harm, we prefer **no code movement** and only
document the boundary.

Related acceptance boundary: `docs/agent/chatagent-feature-checklist.md`.

---

### Appendix A — Schedule / stream duplication audit (WS-B)

#### Schedule cluster (current)

- **Files**:
  - `internal/server/chatagent/scheduler.go`
  - `internal/server/chatagent/schedule.go`
  - `internal/server/chatagent/schedule_helpers.go`
  - `internal/server/chatagent/schedule_tool.go`
  - `internal/server/chatagent/scheduled_api.go`
  - `internal/server/chatagent/scheduled_run.go`
  - `internal/server/chatagent/scheduled_delivery.go`
  - `internal/server/chatagent/schedule_errors.go`

#### Findings

- **Create/update arg parsing**: `parseCreateScheduleArgs` (helpers) is used by `ScheduleTaskTool.Execute` (tool). The validation path is already single-sourced via `ValidateScheduleInput` + `ParseRunAt`.
- **DB + scheduler coordination**: `persistScheduledTask` rolls back DB state if scheduler registration fails; behavior is correct and covered by existing tests. No duplicate code found to consolidate safely without changing behavior.
- **Recommendation**: **No merge performed**. The current split is functional (helpers vs tool registry vs scheduler lifecycle). Any refactor here should be driven by a failing test or measurable coupling issue.

#### Stream cluster (current)

- **Files**:
  - `internal/server/chatagent/api_stream.go`
  - `internal/server/chatagent/event_stream.go`
  - `internal/server/chatagent/stream_coalescer.go`
  - `internal/server/chatagent/event_sink.go`
  - `internal/server/chatagent/progress.go`
  - `internal/server/chatagent/sink.go`

#### Findings

- **Streaming roles are distinct**:
  - `event_stream.go` converts `pkg/agent/event` to product `StreamEvent`.
  - `api_stream.go` owns HTTP SSE writing and run lifecycle management for REST `POST …/messages`.
  - `stream_coalescer.go` batches deltas; not duplicated elsewhere.
  - `event_sink.go` provides bounded buffering semantics (critical vs droppable).
- **Observer filter duplication** (REST/Web `/events`): consolidated into `chatagent.IsObserverStreamEvent` and used by both `internal/server/chatagent_http_sessions.go` and `internal/modules/web/chatagent_web_stream.go`.
- **Recommendation**: **No merge performed** beyond the shared observer filter. Further merging risks making the streaming path harder to reason about and test.

---

### Appendix B — `chatagent` ↔ `pkg/agent` boundary audit (WS-C)

#### Current dependency shape

`internal/server/chatagent` is a **product orchestration layer**. It imports `pkg/agent` primitives (loop/harness/tools/session/permission) and binds them to:

- Flowbot persistence (`internal/store`)
- REST + Web SSE protocols
- Platform sinks (e.g. Slack streaming)
- Scheduled tasks and delivery context
- User permission configs stored as ConfigData

Import hotspots (non-test, approximate):

- `pkg/agent/msg`, `coding`, `tool`, `hooks`, `permission`, `session`, `harness`, `ctxmgr`, `event`, `llm`, `subagent`, `memory`, `env`, `sandbox`.

#### Findings

- **No store leakage into `pkg/agent`**: `internal/store` is only used inside `internal/server/chatagent` (correct).
- **No obvious duplicated pure logic** between `chatagent` and `pkg/agent` that should be moved today, based on a quick structural audit. Most logic in `chatagent` is product-specific (sessions, permissions storage, SSE protocol, scheduled task records).
- The only boundary cleanup applied in this remediation is **additive + product-layer**:
  - `chatagent.IsObserverStreamEvent` consolidates duplicated filter logic between REST and Web `/events` subscribers (both are product endpoints).

#### Recommendation

- Keep WS-C as **documentation-only** for this iteration: write `internal/server/chatagent/AGENTS.md` and add cross-links in `pkg/agent/AGENTS.md` and `internal/modules/web/AGENTS.md`.
- Only move code into `pkg/agent` if a future change shows true duplication across multiple products (not just chatagent), and has tests demonstrating identical semantics.

