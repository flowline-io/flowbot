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
    ContextWindow: config.ContextWindowForModel("gpt-4o"),
    Settings:      ctxmgr.SettingsFromConfig(config.App.ChatAgent.Compaction),
    SystemPrompt:  systemPrompt,
})

h := harness.New(harness.Options{
    AgentOptions:   agent.Options{Model: llmModel, Registry: registry},
    Session:        sess,
    ContextManager: ctxMgr,
    SystemPrompt:   systemPrompt,
    ModelName:      "gpt-4o",
})
```

Configure per-model context windows and compaction thresholds in `flowbot.yaml`:

```yaml
models:
  - provider: openai
    model_names: ["gpt-4o"]
    context_windows:
      gpt-4o: 128000

chat_agent:
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

Higher-level orchestration with hooks and automatic session append:

```go
h := harness.New(harness.Options{
    AgentOptions: agent.Options{Model: llmModel, Registry: registry},
    Session:      sess,
    Router:       model.NewRouter("gpt-4o-mini", "gpt-4o"),
    SystemPrompt: "You are Flowbot.",
    ModelName:    "gpt-4o-mini",
})

h.On("before_agent_start", func(ctx context.Context, ev harness.HookEvent) error {
    return nil
})

_ = h.RegisterTool(echo.Tool{})
stream, _ := h.Prompt(ctx, agent.NewUserMessage("echo test"))
```

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
6. Add table-driven tests (≥3 cases) and BDD coverage when behavior is user-visible

## Future Integration Points

Not implemented in the core library (planned for upper layers):

- REST/SSE endpoints in `internal/server`
- Wiring `config.agents` YAML to `agent.Config`
- Pipeline/workflow steps that invoke `agent.RunLoop`
- Compaction is implemented in `pkg/agent/ctxmgr` and wired through `harness.Options.ContextManager`
- Skills (pi harness advanced features)
