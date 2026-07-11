# Agent Engine Developer Guide

How to use and extend `pkg/agent/` for multi-turn LLM runs with tools.

## Configuration

`agent.Config` (alias of `msg.Config`) controls a single loop invocation:

```go
cfg := agent.DefaultConfig()
cfg.ModelName = "gpt-4o"
cfg.ChatModel = "gpt-4o-mini"   // optional dual-model
cfg.ToolModel = "gpt-4o"
cfg.MaxSteps = 30
cfg.Temperature = 0.7
cfg.ToolExecution = agent.ToolExecutionParallel
cfg.SteeringMode = agent.QueueAll
cfg.FollowUpMode = agent.QueueAll
```

| Field | Description |
| ----- | ----------- |
| `TransformContext` | Prune or enrich messages before LLM conversion |
| `ConvertToLLM` | Defaults to `transform.DefaultConvertToLLM` |
| `PrepareNextTurn` | Refresh model/context between turns |
| `ShouldStopAfterTurn` | Early exit after a completed turn |
| `BeforeToolCall` / `AfterToolCall` | Tool interception hooks |
| `GetSteeringMessages` / `GetFollowUpMessages` | Set automatically on `Agent` via queues |

### Dual-model routing (chat agent)

Configure in `flowbot.yaml`:

```yaml
chat_agent:
  chat_model: "gpt-4o-mini"   # non-empty enables the chat agent
  tool_model: "gpt-4o"        # enables dual-model routing when set
```

When `tool_model` is set, the loop uses `chat_model` for the first LLM call and switches to `tool_model` after tool execution. Both models must be registered under `models[]` and use the same provider (v1).

For programmatic harness use, pass `harness.Options.Router`; chat agent sets `cfg.ChatModel` / `cfg.ToolModel` directly instead.

## Stateless Loop

Use when you manage context yourself and do not need queues or subscriptions:

```go
stream := agentevent.NewStream(32)
messages, err := agent.RunLoop(ctx,
    []agent.AgentMessage{agent.NewUserMessage("summarize logs")},
    &agent.Context{SystemPrompt: "You are a homelab assistant."},
    cfg,
    agent.LoopDeps{Model: llmModel, Registry: registry},
    stream,
)
```

Continue from an existing context (last message must be user or tool result):

```go
messages, err := agent.RunLoopContinue(ctx, agentCtx, cfg, deps, stream)
```

## Stateful Agent

```go
ag := agent.NewAgent(agent.Options{
    Model:    llmModel,
    Registry: registry,
    Config:   cfg,
    InitialState: &agent.Context{
        SystemPrompt: "Follow AGENTS.md conventions.",
    },
})

ag.Subscribe(func(ev agentevent.Event) error {
    if ev.Type == agentevent.TypeToolExecutionStart {
        // render progress
    }
    return nil
})

stream, _ := ag.Prompt(ctx, agent.NewUserMessage("check disk usage"))
result, _ := stream.Await(ctx)

ag.Steer(agent.NewUserMessage("also check memory"))
ag.Abort() // cancel in-flight run
```

## Implementing a Tool

```go
type MyTool struct{}

func (MyTool) Name() string        { return "disk_usage" }
func (MyTool) Description() string { return "Returns disk usage for a mount point" }
func (MyTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "path": map[string]any{"type": "string", "description": "Mount path"},
        },
        "required": []string{"path"},
    }
}

func (MyTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
    _ = onUpdate("reading stats...")
    return msg.ToolResultMessage{
        ToolCallID: id,
        Name:       "disk_usage",
        Parts:      []msg.ContentPart{msg.TextPart{Text: "72% used"}},
    }, nil
}
```

Register and optionally restrict:

```go
reg := tool.NewRegistry()
_ = reg.Register(MyTool{})
reg.SetActive([]string{"disk_usage"}) // review mode: read-only tools only
```

See reference: [pkg/agent/example/echo/](../../pkg/agent/example/echo/echo.go).

## Session Persistence

Implement `session.Storage` for your backend (PostgreSQL, file, etc.):

```go
type Storage interface {
    Append(ctx context.Context, entry session.TreeEntry) error
    GetBranch(ctx context.Context, leafID string) ([]session.TreeEntry, error)
    GetLeafID(ctx context.Context) (string, error)
    SetLeafID(ctx context.Context, id string) error
}
```

Usage:

```go
store := session.NewMemoryStorage()
sess := session.New(store)

_ = sess.Append(ctx, session.TreeEntry{
    ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("hi"),
})

branch, _ := sess.GetBranch(ctx, "")
built := session.BuildContext(branch)
agentCtx := session.ToAgentContext(built, "system prompt here")
```

Branch rollback:

```go
_ = sess.MoveTo(ctx, "previous-entry-id", "Summary of abandoned branch...")
```

With `harness.Options.ContextManager` set, `Harness.MoveTo` generates branch summaries automatically when the summary argument is empty.

## Context Management

Wire compaction through the harness for production chat flows:

```go
ctxMgr := ctxmgr.New(ctxmgr.Options{
    Model:         llmModel,
    ModelName:     "gpt-4o",
    ContextWindow: config.ChatAgentContextWindow(),
    Settings:      ctxmgr.SettingsFromConfig(config.App.ChatAgent.Compaction),
    SystemPrompt:  systemPrompt,
})
```

Register models in `flowbot.yaml` and add catalog entries in `pkg/agent/model/catalog.go` for accurate context limits. Unknown model names use the default 128000-token budget.

Configure compaction thresholds in `flowbot.yaml`:

```yaml
chat_agent:
  chat_model: "gpt-4o"
  compaction:
    enabled: true
    reserve_tokens: 16384
    keep_recent_tokens: 20000
```

JSONL export (caller writes bytes to disk or DB):

```go
data, _ := session.SerializeSession(entries)
entries, _ := session.DeserializeSession(data)
```

## Harness

Higher-level orchestration with session persistence and hook bridging:

```go
h := harness.New(harness.Options{
    AgentOptions: agent.Options{Model: llmModel, Registry: registry},
    Session:      sess,
    Router:       model.NewRouter("gpt-4o-mini", "gpt-4o"),
    SystemPrompt: "You are Flowbot.",
    ModelName:    "gpt-4o-mini",
    Hooks:        reg, // optional; see Typed Hooks below
})

_ = h.RegisterTool(echo.Tool{})
stream, _ := h.Prompt(ctx, agent.NewUserMessage("echo test"))
```

`Harness.WaitIdle(ctx)` blocks until the run finishes persisting session entries. `Harness.Hooks()` returns the registry for late registration (prefer registering before the first `Prompt`).

## Typed Hooks (`pkg/agent/hooks`)

Process-local extension points (aligned with pi-agent harness hooks). One `hooks.Registry` per harness run; register handlers before `Prompt`.

```go
import (
    "context"

    "github.com/flowline-io/flowbot/pkg/agent/hooks"
)

reg := hooks.NewRegistry()

// Mutable: runs before loop; bridged outside loop Config
hooks.OnBeforeAgentStart(reg, func(ctx context.Context, ev hooks.BeforeAgentStartEvent) (*hooks.BeforeAgentStartResult, error) {
    prompt := ev.SystemPrompt + "\nExtra instructions."
    return &hooks.BeforeAgentStartResult{SystemPrompt: &prompt}, nil
})

// Mutable: each LLM request; composed into Config.TransformContext
hooks.OnContext(reg, func(ctx context.Context, ev hooks.ContextEvent) (*hooks.ContextResult, error) {
    return &hooks.ContextResult{Messages: ev.Messages}, nil
})

// Mutable: tool gate; composed into Config.BeforeToolCall
hooks.OnToolCall(reg, func(ctx context.Context, ev hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
    if ev.ToolCall.Name == "dangerous" {
        return &hooks.ToolCallResult{Block: true, Reason: "not allowed"}, nil
    }
    return nil, nil
})

// Mutable: patch tool output; Terminate stops the inner loop
hooks.OnToolResult(reg, func(ctx context.Context, ev hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
    return nil, nil
})

// Read-only: harness lifecycle notifications
hooks.Observe(reg, func(ctx context.Context, ev hooks.ObservationEvent) error {
    // ev.Type: hooks.EventSavePoint, EventContextUsage, EventModelUpdate, ...
    return nil
})
```

Pass the registry into harness:

```go
h := harness.New(harness.Options{Hooks: reg, /* ... */})
```

### Registrar reference

| Function | Event constant | Result type | Effect |
| -------- | -------------- | ----------- | ------ |
| `OnBeforeAgentStart` | `EventBeforeAgentStart` | `BeforeAgentStartResult` | Replace prompts; `Cancel` → `hooks.ErrRunCancelled` |
| `OnContext` | `EventContext` | `ContextResult` | Replace message list before LLM |
| `OnToolCall` | `EventToolCall` | `ToolCallResult` | `Block` skips tool execution |
| `OnToolResult` | `EventToolResult` | `ToolResultResult` | Patch `Parts` / `IsError`; `Terminate` ends run |
| `Observe` | (various) | — | No loop impact |
| `OnObservation` | filter by type | — | Convenience wrapper over `Observe` |

Handlers receive the same `context.Context` passed to `Harness.Prompt` (respects cancellation during bridged loop hooks).

### Direct loop use (without harness)

For tests or custom callers, bridge manually:

```go
routed := model.ApplyDefaultRouter(cfg)
cfg = hooks.BridgeConfig(ctx, reg, routed)
```

Or set `Config.TransformContext` / `BeforeToolCall` / `AfterToolCall` directly on `agent.Config` without a registry.

### Chat agent integration

`internal/server/chatagent` wires observational hooks per session run:

```go
reg := hooks.NewRegistry()
RegisterHooks(reg, ChatHookDeps{SessionID: req.SessionID})
harness.New(harness.Options{Hooks: reg, /* ... */})
```

`RegisterHooks` logs `context_usage` and `save_point` at debug level. Add product hooks by extending `RegisterHooks` or registering on `reg` before `harness.New`.

## LLM Provider Setup

Map flowbot YAML models to langchaingo:

```go
model, name, err := agentllm.NewModel(ctx, "gpt-4o") // model name from config.models
```

Supported providers (via `pkg/agent/llm` provider constants): OpenAI, OpenAI-compatible, Anthropic, Gemini.

For tests, use `agentllm.NewFakeModel` with scripted `ResponseScript` entries (text and/or tool calls).

## Multimodal Input

```go
parts := transform.ProcessAttachments([]transform.Attachment{
    {MIMEType: "image/png", URL: "https://example.com/chart.png"},
})
msg := agent.NewUserMessageWithParts(parts...)
```

Attachments are converted to langchaingo `ImageURLContent` or `BinaryContent` during `ConvertToLLM`.

## Testing

### Unit tests (table-driven, testify)

Location: `pkg/agent/**/*_test.go`. Minimum three cases per table per [TDD spec](../testing/tdd-specs.md).

```bash
go test ./pkg/agent/...
```

Use `agentllm.NewFakeModel` — never call real APIs in unit tests.

### BDD acceptance

Location: [tests/specs/agent_spec_test.go](../../tests/specs/agent_spec_test.go) — full prompt → tool → response flow with fake model (no Docker LLM required).

```bash
go tool task test:specs
```

## Extension Checklist

When adding features to `pkg/agent/`:

1. Shared types go in `pkg/agent/msg/` if multiple subpackages need them
2. Avoid import cycles: subpackages must not import root `agent`
3. langchaingo stays inside `pkg/agent/llm/`
4. Database or file I/O stays outside core — use interfaces (`session.Storage`)
5. Update [architecture.md](./architecture.md) and [pkg/agent/AGENTS.md](../../pkg/agent/AGENTS.md)
6. Product hooks: use `pkg/agent/hooks` registrars on a per-run `Registry`
7. Add table-driven tests (≥3 cases) and BDD coverage when behavior is user-visible

## Error Handling (Result Pattern)

Flowbot mirrors pi-agent's layered error model:

1. **Low-level** (`pkg/agent/result`, `env`, `ctxmgr`, JSONL parse): return `result.Result[T,E]` with stable typed codes (`compaction`, `timeout`, `not_found`, …). Callers must branch on `IsOk()` — do not ignore failures.
2. **Boundary** (`harness`, `session` public methods): adapt Result to Go `error` via `result.GetOrError` or `normalizeHarnessError`.
3. **Tool layer**: expected runtime failures become `ToolResultMessage{IsError: true}`; `Execute` returns `(result, nil)`.
4. **Loop layer**: LLM, hook, and transport failures abort the run; tool errors do not.

```go
compactResult := ctxmgr.RunCompaction(ctx, model, name, prep)
if !compactResult.IsOk() {
    return result.GetOrError(compactResult)
}
```

Use `result.CodeOf(err)` or `errors.As` at HTTP/chat integration boundaries instead of string matching.

## Future Integration Points

Not implemented in the core library (planned for upper layers):

- Provider payload hooks (`before_provider_request`) — reserved in `hooks/events.go`, not yet implemented
- Session compact/tree hooks (`session_before_compact`) — second phase; requires ctxmgr callback wiring

Already wired in product layers:

- REST/SSE chat agent in `internal/server` (`/chatagent/*`)
- Pipeline `agent.run` steps (`capability: agent`, `operation: run`) with template-rendered `prompt` and ephemeral sessions
- `chat_agent` YAML → `agent.Config` (models, retry, sensors, ability_tools, sandbox)
- Compaction via `pkg/agent/ctxmgr` and `harness.Options.ContextManager`
- LLM retry, agent metrics/OTel, path sensors, progress artifact, ability tools, opt-in sandbox