# Agent Engine

`pkg/agent/` is Flowbot's core **Observe-Think-Act** agent runtime. It orchestrates LLM calls, tool execution, session trees, and lifecycle events. LLM transport uses [langchaingo](https://github.com/tmc/langchaingo) (`llms.Model`) via `pkg/agent/llm`.

This package is distinct from:

| Name | Location | Purpose |
| ---- | -------- | ------- |
| **Agent engine** | `pkg/agent/` | Multi-turn LLM loop with tools and session tree |
| **LLM adapter** | `pkg/agent/llm/` | langchaingo models, single-shot helpers, `chat_agent` model lookup |
| **Instruct protocol** | `pkg/types/agent.go`, `internal/server/agent.go` | Desktop agent online/offline/message protocol |
| **Chat agent config** | `chat_agent` in `flowbot.yaml` | DM chat assistant runtime and model routing |

## Documentation

| Document | Description |
| -------- | ----------- |
| [Architecture](./architecture.md) | Layering, data flows, package map, design rules |
| [Developer Guide](./developer-guide.md) | API usage, tools, sessions, hooks, testing |
| [Chat Agent Feature Checklist](./chatagent-feature-checklist.md) | Frozen product acceptance boundary (REST/Web/DM/orchestration) |
| [Chat Agent Remediation Audit](./chatagent-remediation-audit.md) | Schedule/stream duplication and `pkg/agent` boundary audit |
| [Agent Sandbox](./agent-sandbox.md) | Cloud Agent ephemeral Docker image (`flowbot-agent-sandbox`) |
| [Diagram](./agent.puml) | PlantUML component diagram (render with PlantUML) |

## Quick Start

```go
import (
    "context"

    "github.com/flowline-io/flowbot/pkg/agent"
    "github.com/flowline-io/flowbot/pkg/agent/tools/echo"
    agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
    "github.com/flowline-io/flowbot/pkg/agent/tool"
)

registry := tool.NewRegistry()
_ = registry.Register(echo.Tool{})

model := agentllm.NewFakeModel(/* scripted responses */)
ag := agent.NewAgent(agent.Options{
    Model:    model,
    Registry: registry,
    Config:   agent.DefaultConfig(),
})

stream, err := ag.Prompt(context.Background(), agent.NewUserMessage("hello"))
if err != nil {
    return err
}
result, err := stream.Await(context.Background())
_ = result
```

## Source Layout

```
pkg/agent/
├── facade.go, types.go           # Public facade (re-exports loop + msg aliases)
├── msg/                          # Shared domain types
├── result/ event/                # Result pattern + lifecycle events
├── loop/                         # Observe-Think-Act + stateful Agent
├── llm/ tool/ session/ model/ transform/ hooks/
├── ctxmgr/ harness/ subagent/    # Orchestration
├── env/ sandbox/ permission/ approval/ dcg/
├── tools/coding/                 # Engine coding tools
├── tools/echo/                   # Reference tool
├── eval/
└── AGENTS.md
```

Product tools (clip, notify) live under `internal/server/chatagent/tools/`.

## Testing

```bash
go test ./pkg/agent/...
go tool task test:specs   # includes tests/specs/agent_spec_test.go
```

Maintainer reference: [pkg/agent/AGENTS.md](../../pkg/agent/AGENTS.md).
