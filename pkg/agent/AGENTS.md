# Agent Guide

Core agent engine implementing Observe-Think-Act loops, tool orchestration, session trees, and lifecycle events. LLM calls go through `pkg/agent/llm` using langchaingo's `llms.Model`.

Full documentation: [docs/agent/](../../docs/agent/README.md) (architecture, developer guide, PlantUML).

## Structure

```
agent/
├── types.go              # AgentMessage, Context, hook types
├── config.go             # Config defaults
├── errors.go             # Domain errors
├── result/               # Result[T,E] and typed agent errors
├── env/                  # ExecutionEnv for FS/shell with Result
├── loop.go               # Stateless RunLoop / RunLoopContinue
├── agent.go              # Stateful Agent with queues and subscriptions
├── event/                # Lifecycle event stream
├── llm/                  # langchaingo adapter + fake model
├── tool/                 # Registry, schema, executor
├── session/              # Session tree + Storage interface + JSONL helpers
├── model/                # Dual-model router
├── transform/            # convertToLLM + multimodal helpers
├── ctxmgr/               # Context budget, compaction, branch summarization
├── hooks/                # Typed hook registry (on/observe/emit) bridged to loop Config
├── harness/              # High-level orchestration with hooks
└── example/echo/         # Reference echo tool
```

## Key Patterns

### Agent Loop

Use `RunLoop` for stateless runs or `agent.NewAgent` for queued steering/follow-up and subscriptions:

```go
messages, err := agent.RunLoop(ctx, []agent.AgentMessage{
    agent.NewUserMessage("hello"),
}, &agent.Context{SystemPrompt: "You are helpful."}, cfg, agent.LoopDeps{
    Model:    llmModel,
    Registry: registry,
}, stream)
```

### Tools

Register tools on `tool.Registry` and optionally restrict with `SetActive`:

```go
registry := tool.NewRegistry()
_ = registry.Register(echo.Tool{})
registry.SetActive([]string{"echo"})
```

### Session Tree

Persist via `session.Storage`; core provides JSONL marshal/unmarshal only:

```go
store := session.NewMemoryStorage()
sess := session.New(store)
_ = sess.Append(ctx, session.TreeEntry{ID: "1", Type: session.EntryMessage, Message: msg})
branch, _ := sess.GetBranch(ctx, "")
ctx := sess.BuildContext(branch)
```

### Dual Model

Use `model.Router` in `PrepareNextTurn` or harness configuration:

```go
router := model.NewRouter("gpt-4o-mini", "gpt-4o")
router.ApplyToContext(agentCtx, afterToolExecution)
```

### Context Management

Use `ctxmgr.Manager` with the harness to compact long histories and summarize branches:

```go
ctxMgr := ctxmgr.New(ctxmgr.Options{
    Model: llmModel, ModelName: "gpt-4o",
    ContextWindow: config.ContextWindowForModel("gpt-4o"),
    Settings: ctxmgr.SettingsFromConfig(config.App.ChatAgent.Compaction),
})
h := harness.New(harness.Options{ContextManager: ctxMgr, Session: sess, ...})
```

### Typed Hooks

Register on a per-run `hooks.Registry` and pass `harness.Options.Hooks`:

```go
reg := hooks.NewRegistry()
hooks.OnContext(reg, func(ctx context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
    return &hooks.ContextResult{Messages: ev.Messages}, nil
})
h := harness.New(harness.Options{Hooks: reg, AgentOptions: ...})
```

Harness bridges from `loopBaseCfg` (snapshot at `New`) via `hooks.BridgeConfig(ctx, reg, model.ApplyDefaultRouter(loopBaseCfg))` before each `Prompt`. Only `HasLoopHandlers()` triggers bridge; `Observe` does not. See [docs/agent/architecture.md](../../docs/agent/architecture.md#hooks-pkgagenthooks).

## Rules

- **langchaingo scope**: only `llms.Model` in `pkg/agent/llm`; do not use langchaingo agents/chains
- **Modules**: import `pkg/agent/llm` only for single-shot LLM tasks; do not import other `pkg/agent` packages from `internal/modules` until explicitly wired
- **Naming**: distinct from `pkg/types/agent.go` (instruct protocol) and YAML `config.agents` entries
- **Serialization**: use `sonic` for JSON/JSONL
- **Errors**: wrap with `%w`; return `ErrMaxSteps`, `ErrAborted`, `ErrToolNotFound`; hook cancel via `hooks.ErrRunCancelled`
- **Hooks**: add mutable behavior with `hooks.On*` registrars; do not extend deprecated `Harness.On(string)`
- **Result pattern**: low-level capabilities (`env`, `ctxmgr`, JSONL parse) return `result.Result[T,E]` with typed error codes; harness/session public APIs adapt to Go `error` via `result.GetOrError`; tool failures stay inline as `ToolResultMessage.IsError`
- **Tests**: table-driven unit tests (>=3 cases) + BDD in `tests/specs/agent_spec_test.go`

## Error Handling

| Layer | Pattern |
| ----- | ------- |
| `result`, `env`, `ctxmgr` internals | `result.Result[T,E]`; callers must check `IsOk()` |
| `harness`, `session` public API | Go `error`; use `errors.As` / `result.CodeOf` at integration boundaries |
| `tool.Execute` | Expected failures → `ToolResultMessage{IsError: true}`, `error` nil |
| Agent loop | Fatal failures return `error`; tool errors do not abort the turn |

Anti-patterns: returning bare `error` from compaction helpers; ignoring `!result.IsOk()`; using `panic` for expected failures.

## Testing

```bash
go test ./pkg/agent/...
go tool task test:specs   # includes agent_spec_test.go
```

Reference implementation: `pkg/agent/example/echo/`.
