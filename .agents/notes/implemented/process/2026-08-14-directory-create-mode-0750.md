# Agent Note: Directory create mode 0750

Status: implemented

## Problem

`os.Mkdir` / `os.MkdirAll` used `0755` (world-traversable) in CLI, composer, eval reports, and agent workspace helpers. GSC-G301 / gosec G301 flags any directory mode greater than `0750`. The repo already created some directories as `0750` (skills, webdoc, media uploads, Docker bind mounts, CLI token dir), so the rest was an inconsistent default rather than a required world-access contract.

## Decision

New directories use mode `0750` or less (owner `rwx`, group `rx`, no world). Callers that pass a mode through `ExecutionEnv.MkdirAll` use the same cap.

Documented exception: `pkg/agent/sandbox` `cliConfigDirWorldAccessible` (`0755`) is a `chmod` fallback when `chown` to the sandbox agent user fails, so uid 1000 can still traverse an ephemeral host temp dir. That path is not a `Mkdir` default.

gosec G301 is enabled (`go tool task gosec`). G302 / G306 stay excluded (file `chmod` / `WriteFile` modes are a separate pass).

## Alternatives considered

- **Keep excluding G301.** Rejected: it hid a real least-privilege gap and let `0755` reappear next to existing `0750` call sites.
- **Default to `0700`.** Rejected: group read/traverse is useful on shared homelab accounts; `0750` is the scanner's documented ceiling and matches shipped call sites.
- **Leave agent workspace `MkdirAll` at `0755`.** Rejected: those directories are created for the running user; world traverse is not part of the workspace contract.

## Consequences

- Report output, eval sandboxes, CLI export dirs, and agent-created parent dirs are group-readable and not world-traversable.
- A new `Mkdir` / `MkdirAll` with mode `> 0750` fails `go tool task gosec` unless it is a documented exception (today: sandbox CLI config `chmod` fallback only).

## Verification

`go tool task gosec` includes G301. Production `os.MkdirAll` call sites in `pkg/agent/eval`, `cmd/composer/action/agenteval`, `cmd/cli/command/function.go`, and `internal/server/chatagent/progress.go` pass `0o750`. Root `AGENTS.md` links this note and does not restate the exception list.
