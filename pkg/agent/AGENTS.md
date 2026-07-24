# Agent Guide

Core agent engine (Observe-Think-Act, tools, sessions, hooks). LLM via `pkg/agent/llm` (`llms.Model` only).

Full docs: [docs/agent/](../../docs/agent/README.md). Reference tool: `example/echo/`.

Product orchestration (REST/Web/platform/store) lives in `internal/server/chatagent` — do not move store/http/platform logic here.

## Entry points

Hot-path packages: `loop` / `harness` / `hooks` / `tool` / `session` / `permission` / `ctxmgr` / `model` / `transform` (`DefaultConvertToLLM`). Tools also under `coding/`, `clip/`, `notify/`, `dcg/`, `subagent/`, `sandbox/`. Eval: `eval/`.

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
