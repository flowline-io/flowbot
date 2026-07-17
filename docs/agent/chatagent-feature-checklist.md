# Chat Agent Feature Checklist

Frozen acceptance boundary for chatagent product orchestration and Web UI remediation (C+T3).
Regression and PRs are judged against this list plus the linked tests. If behavior exists in code but is missing here, update this document before changing code.

Related engine docs: [README](./README.md), [Architecture](./architecture.md). Maintainer guides: `internal/server/chatagent/AGENTS.md`, `pkg/agent/AGENTS.md`.

## Dual-channel SSE protocol

Both channels use `chatagent.StreamEvent` JSON (`type` discriminator) from `internal/server/chatagent/protocol.go`.

| Channel | Message send (primary turn stream) | Live event subscribe | Subscribe filter |
| ------- | ---------------------------------- | -------------------- | ---------------- |
| REST | `POST /chatagent/sessions/:id/messages` — full SSE (`delta` / `thinking` / `tool` / … / `done`) | `GET /chatagent/sessions/:id/events` | `confirm`, `confirm_resolved`, `canceled`, `mode_change` only |
| Web | `POST /service/web/agents/:id/messages` — turn execution; UI may use fetch/SSE depending on handler | `GET /service/web/agents/:id/events` | **Same subset** (`chatagent_web_stream.go`) |

Event **shape** (`StreamEvent`) is shared. Both `/events` subscribers use the same type filter (not full deltas). Primary turn tokens come from the messages/send path (REST `StreamAPIRun`), not from `/events`.

---

## 1. Platform DM chat

| ID | Behavior | Entry | Edge cases | Tests |
| -- | -------- | ----- | ---------- | ----- |
| P-01 | Bind/create chat session for platform DM and run agent turn | `internal/server/chatagent_handler.go` → `Service.Run` | Closed session rejected; disabled agent | `chatagent` service/handler unit tests |
| P-02 | Slack: stream placeholder + `SlackStreamSink` updates | Slack platform only | Placeholder send failure → non-streaming fallback | `slack_sink_test.go`, handler stream tests |
| P-03 | Non-Slack: single reply after run completes | Other platforms | Persist assistant message failure surfaces user-visible error | handler tests |
| P-04 | Persist assistant message to store | After successful `Run` | DB error | handler tests |

## 2. REST API (`/chatagent/*`)

Auth: `ScopeChatAgentChat`. Owner checks on session-scoped routes.

| ID | Behavior | Entry | Edge cases | Tests |
| -- | -------- | ----- | ---------- | ----- |
| R-01 | Agent info | `GET /chatagent/info` | Disabled → 503 | `chatagent_http_test.go`, BDD chat HTTP |
| R-02 | List / create / close sessions | `GET\|POST /chatagent/sessions`, `DELETE …/:id` | Closed → not found for owner ops | unit + `chat_agent_chat_spec_test.go` |
| R-03 | List messages / plans | `GET …/messages`, `GET …/plans` | Empty history | unit HTTP tests |
| R-04 | Send message (full SSE stream) | `POST …/messages` | Empty text 400; run in flight 409 | `chat_agent_chat_spec_test.go` |
| R-05 | Subscribe session events | `GET …/events` | Client disconnect; buffered hub | `session_events_test.go` |
| R-06 | Confirm / cancel run | `POST …/confirm`, `POST …/cancel` | Unknown confirm 404; already resolved 409 | confirm unit + BDD plan mode |
| R-07 | Session mode get/set | `GET\|PUT …/mode` | Plan vs normal | `chat_agent_chat_spec_test.go`, plan mode tests |
| R-08 | Clear session permission grants | `DELETE …/permission-grants` | | permission session tests |
| R-09 | Permissions get/put/delete | `GET\|PUT\|DELETE /chatagent/permissions` | Invalid config | HTTP + permission tests |
| R-10 | Context usage / compact | `GET …/context`, `POST …/compact` | Compaction no-op | context_usage tests, agents page context |
| R-11 | Export session | `GET …/export` | | export tests |
| R-12 | Resolve resource URI | `GET /chatagent/resources` | `plan://`, `file://` | `chat_agent_spec_test.go` resources It |
| R-13 | Scheduled tasks CRUD + runs | `/chatagent/scheduled-tasks…` | One-shot complete; cancel | `scheduled_api_test.go`, `chat_agent_scheduled_task_spec_test.go` |

## 3. Web chat UI (`/service/web/agents/*`)

| ID | Behavior | Entry | Edge cases | Tests |
| -- | -------- | ----- | ---------- | ----- |
| W-01 | Agents home: composer + session list | `GET /service/web/agents` | Unauthenticated → login | `agents_page_spec_test.go` |
| W-02 | Create session (+ optional pending prompt) | `POST /service/web/agents` | `?prompt=` / sessionStorage pending key | agents page + JS pending prompt |
| W-03 | Chat page hydrate history | `GET /service/web/agents/:id` | Closed session | agents page |
| W-04 | Send message / cancel / confirm | Web posts under `/agents/:id/…` | Approval once/always/reject | chat BDD helpers; unit confirm |
| W-05 | Context ring + popover | `GET …/context` + JS | Token window zero | `agents_page_spec_test.go` context It |
| W-06 | Streaming markdown + tool cards + thinking | `public/js/chatagent-*.js` | Open code fence delay; tool upsert | chat BDD stream done |
| W-07 | Close session | `DELETE /service/web/agents/:id` | | agents page |

## 4. Permissions UI

| ID | Behavior | Entry | Edge cases | Tests |
| -- | -------- | ----- | ---------- | ----- |
| Q-01 | Permissions page render | `GET /service/web/chatagent-permissions` | | `chatagent_permissions_webservice_test.go` |
| Q-02 | Save form / JSON / reset | `POST …`, `POST …/reset` | Field validation errors | same |
| Q-03 | REST permissions parity | `/chatagent/permissions` | Defaults when empty | BDD permissions It |

## 5. Skills / Memory / Subagents / Scheduled (Web admin)

| ID | Behavior | Entry | Edge cases | Tests |
| -- | -------- | ----- | ---------- | ----- |
| A-01 | Skills CRUD + files | `/service/web/agent-skills…` | Enable/disable | agent skills webservice tests / pages |
| A-02 | Memory file list/read/write | `/service/web/agent-memory…` | Max file bytes | memory webservice tests |
| A-03 | Subagents CRUD + tasks | `/service/web/agent-subagents…` | Seed defaults on enable | subagent specs/tests |
| A-04 | Scheduled tasks list/detail/state | `/service/web/agent-scheduled-tasks…` | Pause/resume | `agent_scheduled_tasks_page_spec_test.go` |
| A-05 | Session inspect UI | `/service/web/agent-sessions…` | Entry payload, events, confirm | `agent_sessions_page_spec_test.go` |

## 6. Orchestration capabilities

| ID | Behavior | Entry | Edge cases | Tests |
| -- | -------- | ----- | ---------- | ----- |
| O-01 | Interactive `Service.Run` / `RunAPI` | `service.go` | Session lock; harness pool reuse | `service_test.go`, BDD |
| O-02 | Plan mode blocks writes until confirm/normal | mode + permission hooks | Return to normal allows write | `chat_agent_spec_test.go` plan It |
| O-03 | Confirm gate (once / always / reject) | `confirm.go` | Pattern suggest | confirm tests + BDD |
| O-04 | Harness pool TTL / config hash refresh | `harness_pool.go` | Evict on close/abort | harness-related tests |
| O-05 | Session title generation | `title.go` | LLM disabled for tests | `title_test.go`, BDD title wait helpers |
| O-06 | Manual + automatic compaction | `CompactSession`, ctxmgr | | compaction / context tests |
| O-08 | Pipeline agent step (ephemeral) | `pipeline_run.go` / `RunPipelineAgent` | Tools/skills allowlist; memory default off | `pipeline_run_test.go`, `ephemeral_run_test.go` |
| O-09 | Scheduled autonomous run + delivery | `scheduled_run.go`, scheduler | Isolated session; permission policy | `chat_agent_scheduled_task_spec_test.go`, scheduled_* tests |
| O-10 | Skills tool / memory tool / subagent task tool | registry + tools | Allowlists for subagents | skills/memory/subagent tests |
| O-11 | Sensors / progress / usage recording | sensors, progress, usage_record | | unit tests |
| O-12 | Prompt cache | `prompt_cache.go` | Invalidation on config change | `prompt_cache_test.go` |

## 7. Out of scope / non-goals for this remediation

- Rewriting `pkg/agent` loop/harness core
- Merging `/chatagent` and `/service/web/agents` route prefixes
- Desktop instruct protocol (`pkg/types/agent.go` / `internal/server/agent.go`)
- New product features beyond cleanup

---

## QA self-test (manual)

Use after each vertical slice and before freeze sign-off.

1. Platform: send one DM (Slack if available) — streaming or final reply, message persisted.
2. Web: create session from `/service/web/agents`, send a turn, see SSE done and history hydrate.
3. Approval: trigger a gated tool — Approve once; repeat with Always; Reject path.
4. Plan mode: set plan → blocked write → switch normal → write works.
5. Compact: call compact on a long session; context ring updates.
6. Scheduled: create one-shot task, wait for completed + run row.
7. Skills: toggle/enable a skill and confirm agent can `read_skill`.
8. Memory: read/write via UI or tool; respect max size.
9. Subagent: run a task tool delegation; progress/tool events appear.
10. Permissions: save form, reset to defaults; session grants clear.
11. Export: download/export session JSONL or documented format.
12. Title: first user message eventually updates session title (or test-disabled path).
13. REST smoke: `POST /chatagent/sessions/:id/messages` SSE reaches `done`; `GET …/events` receives hub events when a run publishes.

---

## Appendix A — Schedule / stream duplication audit (WS-B)

See [chatagent-remediation-audit.md](./chatagent-remediation-audit.md#appendix-a--schedule--stream-duplication).

## Appendix B — `chatagent` ↔ `pkg/agent` boundary audit (WS-C)

See [chatagent-remediation-audit.md](./chatagent-remediation-audit.md#appendix-b--chatagent--pkgagent-boundary).

Maintainer guides: `internal/server/chatagent/AGENTS.md`, `internal/modules/web/AGENTS.md` (chatagent JS guardrails).

## Warranty note

Regressions introduced directly by this remediation are in scope for fix within two weeks of checklist freeze. Checklist gaps discovered later require updating this document before code changes.
