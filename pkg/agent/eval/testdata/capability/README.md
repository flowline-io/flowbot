# Capability eval fixtures

YAML cases under this tree are loaded by `BuiltinCapabilityScenarios` / `agenteval live`.

## Layout

| Dir | Focus | Toolset | Outcomes |
| --- | ----- | ------- | -------- |
| `openqa/` | Instruction following, safety, honesty, format, reasoning | `none` / `echo` | text contains / contains_any + judge gold |
| `tools/` | Tool use + **true** workspace outcomes | `read_file` / `write_file` / `fs` / `coding` | file asserts + text |

`coding` toolset = read + write + glob_files + grep_files (+ echo).

Use `toolset: none` when the case forbids tools — do not register `echo` and also ban it.

## Difficulty ladder

Set `difficulty: easy|medium|hard` on each case (default **easy**).

| Level | Intent |
| --- | --- |
| easy | Single-step greet/read/write; clear refuse |
| medium | Multi-constraint, distractors, multi-file edits |
| hard | Conflicting sources, glob/grep discovery, subtle safety, nested schema |

```bash
# medium + hard (recommended for optimization A/B — avoids smoke saturation)
go run ./cmd/composer agenteval live --model SUBJECT --judge-model JUDGE --judge-fake=false --difficulty medium+

# hard only
go run ./cmd/composer agenteval live --model SUBJECT --difficulty hard --smoke=false
```

`--difficulty` skips `--smoke`. Combine with `--run` (AND).

## Industry alignment (2026)

- **pass@k / pass^k** — reliability across trials
- **Safety** — refuse secrets, destructive shell, malware, injection, subtle exfil
- **Honesty / instruction following** — unknown, false premises, priority overrides, JSON schema
- **Tool use** — grounding, conflicting files, glob/grep then edit
- **capability_index** — reliability + judge quality (needs real `--judge-model`)

## Smoke

`--smoke` (default) runs `DefaultSmokeCaseNames` only (easy OpenQA). Prefer `--difficulty medium+` for scoring headroom.
