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
| [Local CLI Gateway](./local-cli-gateway.md) | Homelab ↔ `cmd/gateway` Cursor worker (pull model; not notification gateway) |
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

## Agent evaluation

`pkg/agent/eval` scores agent runs with code graders (CI) and optional model graders (capability).

| Suite | Purpose | How to run |
| ----- | ------- | ---------- |
| **regression** | Near-100% FakeModel / deterministic gates | `go test ./pkg/agent/eval/...` or `go tool task agent:eval` (CI: [Agent Eval](../../.github/workflows/agent-eval.yml)) |
| **capability** | Layered L1/L2/L3 climb; multi-trial Pass@1 + appendix `pass@k` / `pass^k` | `go tool task agent:eval:live` (local/nightly; not a PR hard gate) |

### Fake vs true outcome

- **FakeModel regression** exercises grader wiring and **proxy outcomes** (completion, forbidden tools, max steps). Scripts already choose the tool path, so tool *order* is soft by default (`ExpectedTools`); hard tool gates are `RequiredTools` coverage and `ForbiddenTools`.
- **True outcomes** use isolated workspaces (e.g. `write_file` + file asserts) or live models. Workspaces go through the eval `Sandbox` interface (default `WorkspaceSandbox`; Docker deferred).
- **Product policy** outcomes live in `internal/server/chatagent/eval` (permission.Evaluator / DCG with fixtures)—not inside `pkg/agent/eval`.

### Graders and gates

- CI hard gates: required/forbidden tools, args, outcome asserts, completion, max steps. LLM-as-judge never blocks PRs and does **not** enter Total.
- Live: default `k=3` trials; case PASS still requires all trials pass; **Pass@1** = successful trials / (cases×k) enters L3.
- OpenQA smoke ≤5 tasks; gold files under `pkg/agent/eval/testdata/capability/openqa/`. Judge–gold **agreement** = fraction of dimensions with `|judge−gold|≤1` on a 1–5 scale (skip score `0` / Unknown).
- Capability fixtures: `openqa/` + `tools/` + `repair/`. Default `--smoke` uses `DefaultSmokeCaseNames` (excludes repair/system); `--smoke=false` runs the full set.

### Scorecard (replaces capability_index)

```
Total = 100 * (0.2*L1 + 0.5*L2 + 0.3*L3)
L1 = compliance rate (forbidden tools + text guardrails; not tool selection)
L2 = (ToolCallAcc + RepairRate) / 2   # RepairRate from tier=repair only; else ToolCallAcc
L3 = mean(Pass@1, LatencyScore?, TokenScore?)  # omit dims when actual==0
```

Appendix: judge quality, `pass@k` / `pass^k`, natural repair rate. Wilson CI on Pass@1/RepairRate; `--repeats N` (N≥2) for Total CI. Budgets: `--latency-budget-ms` (default 8000), `--token-budget` (default 6000). Historical `capability_index` trends are discontinued.

### Graduation

When a capability task is stably near 100% `pass^k` and criteria are unambiguous, promote it into the regression suite (code graders only). Keep capability hard enough to avoid early saturation.

### CLI artifacts

```bash
go tool task agent:eval            # → tmp/agent_eval/regression_*.json
go tool task agent:eval:live       # FakeModel offline live path
go run ./cmd/composer agenteval live --model SUBJECT --judge-model JUDGE --judge-fake=false
go run ./cmd/composer agenteval compare --baseline ... --candidate ...
go run ./cmd/composer agenteval export --report ... --out tmp/agent_eval/drafts
go run ./cmd/composer agenteval report --dir tmp/agent_eval   # HTML overview + detail pages
```

Flags: `--cases`, `--out`, `--trials`, `--smoke`, `--model`, `--judge-model`, `--config`, `--run`, `--difficulty`, `--tier`, `--latency-budget-ms`, `--token-budget`, `--repeats`.

```bash
go run ./cmd/composer agenteval live --model SUBJECT --judge-model JUDGE --judge-fake=false --difficulty medium+
go run ./cmd/composer agenteval live --model SUBJECT --tier repair --smoke=false
```

Compare optimization A/B with scorecard deltas (`total` / L1 / L2 / L3):

```bash
go run ./cmd/composer agenteval compare --baseline before.json --candidate after.json
```

### Phase 2

Harness reliability report is available through `agenteval harness` (runs through `pkg/agent/harness`), and Docker sandbox mode can be selected via `--sandbox=docker`.
For current eval fixtures, Docker isolation primarily applies to tools that execute commands (`Exec` path).
Examples:

```bash
go run ./cmd/composer agenteval live --model SUBJECT --run openqa_greet
go run ./cmd/composer agenteval live --model SUBJECT --run 'refuse|explain'
go run ./cmd/composer agenteval run --run echo_happy
```

Reports are local/CI files only (no DB). Failed cases include a transcript summary—read that before changing agent or grader.

Maintainer reference: [pkg/agent/AGENTS.md](../../pkg/agent/AGENTS.md).
