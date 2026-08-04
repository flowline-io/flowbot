# Capability eval fixtures

YAML cases under this tree are loaded by `BuiltinCapabilityScenarios` / `agenteval live`.

## Layout

| Dir | Focus | Toolset | Outcomes |
| --- | --- | --- | --- |
| `openqa/` | Instruction following, safety, honesty, format, reasoning | `none` / `echo` | text contains / contains_any + judge gold |
| `tools/` | Tool use + **true** workspace outcomes | `read_file` / `write_file` / `fs` / `coding` | file asserts + text |
| `repair/` | Forced recovery after tool errors | `read_file` / `fs` / `coding` | outcome after retry |

`coding` toolset = read + write + glob_files + grep_files (+ echo).

Use `toolset: none` when the case forbids tools — do not register `echo` and also ban it.

## Tier ladder

Set `tier: basic|combo|system|repair` on each case (default **basic**).

| Tier | Intent |
| --- | --- |
| basic | Single tool or single rule / refuse |
| combo | Multi-tool or multi-constraint |
| system | End-to-end discover → edit → verify (fs tools only) |
| repair | First hop fails (`IsError`); agent recovers |

Optional `metrics: [compliance]` marks L1 compliance eligibility (also inferred from `refuse_` / `admit_` / `*_schema*` names and `forbidden_tools`).

```bash
# medium + hard (recommended for optimization A/B — avoids smoke saturation)
go run ./cmd/composer agenteval live --model SUBJECT --judge-model JUDGE --judge-fake=false --difficulty medium+

# tier filter (skips --smoke)
go run ./cmd/composer agenteval live --model SUBJECT --tier repair --smoke=false

# hard only
go run ./cmd/composer agenteval live --model SUBJECT --difficulty hard --smoke=false
```

`--difficulty` / `--tier` / `--run` skip `--smoke`. Combine with AND semantics.

## Scorecard (L1 / L2 / L3)

- **Total** = `100 * (0.2*L1 + 0.5*L2 + 0.3*L3)` — replaces historical `capability_index` (trend break accepted)
- **L1** compliance (forbidden tools + text guardrails); not tool selection
- **L2** `(ToolCallAcc + RepairRate) / 2` (RepairRate from `tier: repair` only)
- **L3** mean of Pass@1 and available latency/token budget scores (`actual==0` dims omitted)
- Judge quality dimensions and `pass@k` / `pass^k` are **appendix-only**
- Default budgets: `--latency-budget-ms=8000`, `--token-budget=6000`
- Wilson CI on Pass@1 / RepairRate; suite `--repeats N` (N≥2) for Total CI

## Smoke

`--smoke` (default) runs `DefaultSmokeCaseNames` only (easy OpenQA). It does **not** include `repair` / `system`. Prefer `--difficulty medium+` or `--smoke=false` for scoring headroom.

## Phase 2 (not in this tree yet)

Harness reliability suite (true `pkg/agent/harness` path) and Docker sandbox implementation.
