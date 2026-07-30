# Agent Guide

Core agent engine (Observe-Think-Act, tools, sessions, hooks). LLM via `pkg/agent/llm` (`llms.Model` only).

Full docs: [docs/agent/](../../docs/agent/README.md). Reference tool: `tools/echo/`.

Product orchestration (REST/Web/platform/store) lives in `internal/server/chatagent` — do not move store/http/platform logic here. Product tools that depend on hub/capability/notify live under `internal/server/chatagent/tools/` (e.g. clip, notify).

## Structure

```
agent/
├── facade.go, types.go, doc.go   # Public facade (msg aliases + loop re-exports)
├── msg/                          # Core message/context/error types (canonical definitions)
├── result/                       # Result[T,E], typed errors, overflow helpers
├── event/                        # Lifecycle event stream
├── loop/                         # Observe-Think-Act + stateful Agent runtime
├── llm/                          # langchaingo adapter, retry, fake model
├── tool/                         # Registry, schema, executor, ValidateArgs, FormatToolError
├── session/                      # Session tree + Storage interface + JSONL helpers
├── model/                        # Model catalog and dual-model router
├── transform/                    # convertToLLM + multimodal helpers
├── ctxmgr/                       # Context budget, compaction, branch summarization
├── hooks/                        # Typed hook registry (on/observe/emit) bridged to loop Config
├── harness/                      # High-level orchestration with hooks + overflow degrade
├── permission/                   # Tool permission evaluation, forms, session/scheduled policies
├── dcg/                          # Destructive Command Guard for run_terminal / run_code
├── subagent/                     # Subagent orchestration
├── env/                          # ExecutionEnv for FS/shell with Result
├── sandbox/                      # Opt-in Docker ExecutionEnv for shell/code
├── tools/
│   ├── coding/                   # Code/FS/web/terminal tools
│   └── echo/                     # Reference echo tool
└── eval/                         # FakeModel harness eval scenarios
```

## Entry points

Hot-path packages: `loop` / `harness` / `hooks` / `tool` / `session` / `permission` / `ctxmgr` / `model` / `transform` (`DefaultConvertToLLM`). Engine tools under `tools/coding/`, `tools/echo/`; also `dcg/`, `subagent/`, `sandbox/`. Eval: `eval/`.

External callers may keep importing `pkg/agent` for types (`AgentMessage`, `NewAgent`, `RunLoop`). Subpackages must not import the parent `pkg/agent` facade — use `msg` / `loop` instead. `ctxmgr` depends only on the `StatefulAgent` seam (`State` / `ApplyState`).

## Non-obvious rules

- **langchaingo**: only `llms.Model` in `pkg/agent/llm` — no agents/chains.
- **Modules**: prefer `pkg/agent/llm` for single-shot LLM. Web may import already-wired packages (`permission`; tests: `model`/`msg`/`session`); do not import other `pkg/agent` packages from modules until wired.
- Distinct from `pkg/types/agent.go` (instruct) and YAML `chat_agent` config.
- JSON/JSONL: `sonic`. Metrics: `metrics.Agent()` — low-cardinality labels (`status`, `model`, `tool`, `level`); never `session_id`.
- LLM retry only before first stream delta (`ErrStreamStarted`). Tool expected failures → `ToolResultMessage{IsError: true}` + `FormatToolError`.
- Result pattern: `env` / `ctxmgr` / JSONL parse return `result.Result[T,E]`; harness/session public APIs use Go `error`. Hook cancel: `hooks.ErrRunCancelled`.
- Harness bridges hooks via `hooks.BridgeConfig` only when `HasLoopHandlers()` (not Observe-only).

## Testing

```bash
go test ./pkg/agent/...
go test ./pkg/agent/eval/...
```

Path-only moves of product tools (`clip`/`notify` → `internal/server/chatagent/tools/`) keep existing package unit tests; no BDD update is required when behavior is unchanged.
