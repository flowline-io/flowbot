# Capability eval fixtures

YAML cases under this tree are loaded by `BuiltinCapabilityScenarios` / `agenteval live`.

## Layout

| Dir | Focus | Toolset | Outcomes |
| --- | ----- | ------- | -------- |
| `openqa/` | Instruction following, safety refuse, honesty, format | `echo` | text contains / contains_any + judge gold |
| `tools/` | Tool selection + **true** workspace outcomes (τ-bench style state checks) | `read_file` / `write_file` / `fs` | file asserts + text |

## Industry alignment (2026)

- **pass@k / pass^k** — reliability across trials (`agenteval live --trials`)
- **Safety** — refuse secrets, destructive shell, malware, prompt injection
- **Honesty** — admit unknown; correct false premises without sycophancy
- **Instruction following** — format / multi-constraint answers
- **Tool use** — read before inventing; write/edit files; prefer FS over shell
- **Code graders** hard-gate; LLM-as-judge is calibration-only (gold agreement)

## Smoke

`--smoke` (default) runs `DefaultSmokeCaseNames` only. Use `--smoke=false` for the full set, or `--run` regexp like `go test -run`.

## Scripts

`scripts:` drive FakeModel offline live / CI wiring. Real `--model` ignores scripts and exercises the live agent loop.
