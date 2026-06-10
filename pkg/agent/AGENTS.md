# Agent Guide

Core agent engine implementing Observe-Think-Act loops, tool orchestration, session trees, and lifecycle events. LLM calls go through `pkg/agent/llm` using langchaingo's `llms.Model`.

Full documentation: [docs/agent/](../../docs/agent/README.md) (architecture, developer guide, PlantUML).

## Structure

```
agent/
├── types.go              # AgentMessage, Context, hook types
├── config.go             # Config defaults
├── errors.go             # Domain errors
├── loop.go               # Stateless RunLoop / RunLoopContinue
├── agent.go              # Stateful Agent with queues and subscriptions
├── event/                # Lifecycle event stream
├── llm/                  # langchaingo adapter + fake model
├── tool/                 # Registry, schema, executor
├── session/              # Session tree + Storage interface + JSONL helpers
├── model/                # Dual-model router
├── transform/            # convertToLLM + multimodal helpers
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

## Rules

- **langchaingo scope**: only `llms.Model` in `pkg/agent/llm`; do not use langchaingo agents/chains
- **Modules**: do not import `pkg/agent` from `internal/modules` until explicitly wired; this package is core-only
- **Naming**: distinct from `pkg/types/agent.go` (instruct protocol) and `pkg/llm/agent.go` (config lookup)
- **Serialization**: use `sonic` for JSON/JSONL
- **Errors**: wrap with `%w`; return `ErrMaxSteps`, `ErrAborted`, `ErrToolNotFound`
- **Tests**: table-driven unit tests (>=3 cases) + BDD in `tests/specs/agent_spec_test.go`

## Testing

```bash
go test ./pkg/agent/...
go tool task test:specs   # includes agent_spec_test.go
```

Reference implementation: `pkg/agent/example/echo/`.
