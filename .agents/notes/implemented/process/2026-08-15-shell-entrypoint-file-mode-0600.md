# Agent Note: Shell entrypoint file mode 0600

Status: implemented

## Problem

`pkg/executor/runtime/shell` wrote the task entrypoint as `0700` then `chmod`ed to `0o500` so the path could be executed under the default `bash -c <path>` invocation. GSC-G302 (and the same least-privilege bar as gosec G302) expects file modes of `0600` or less; owner-execute (`0100`) fails that check. Execute-on-path was only needed because `-c` treated the path as a command to exec rather than a script to read.

## Decision

The shell runtime writes the entrypoint with mode `0o600` and does not `chmod` afterward. The default interpreter is `bash` (script path argument), not `bash -c`. Configured `executor.shell.cmd` must likewise be an interpreter that accepts a script file path (for example `/bin/bash`), not a `-c` form that would require execute bits on the file.

## Alternatives considered

- **Keep `0o500` and silence GSC-G302.** Rejected: execute-on-temp-script is unnecessary privilege once the interpreter reads the file.
- **`0o600` with `bash -c "$(cat path)"` (or stdin).** Rejected: more moving parts than passing the path to `bash`; same permission outcome.
- **Strip a trailing `-c` from configured CMD as a compat shim.** Rejected: pre-release prefers correcting the contract over hiding a footgun.

## Consequences

- Entrypoint files are owner `rw` only; no group/other bits and no execute bit.
- Operators who set `executor.shell.cmd: ["bash", "-c"]` (or equivalent) must drop `-c` or scripts will not run under `0600`.

## Verification

`go test ./pkg/executor/runtime/shell/` covers run/output capture with the default interpreter. GSC-G302 no longer flags this `Chmod` call site.
