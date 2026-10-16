# Agent Note: CLI agent output

Status: implemented

## Problem

External agents drive flowbot through the CLI, and the generated skill docs are the calling surface. Success and failure were free-form text, every error exited 1, and several destructive commands blocked on stdin. An agent could not tell a missing argument from a missing resource, or a conflict from a retryable outage.

## Decision

`--output` / `-o` is a root persistent flag with values `table` (default) and `json`. The format is not implied by a non-terminal stdin. `--yes` / `-y` and `--no-input` are root persistent flags. Non-terminal stdin implies `--no-input`. A pseudo-terminal still prompts. `pipeline export` and `workflow export` take the YAML path on `--file`.

The envelope, exit codes, and confirmation rule have one home: `cliOutputSection` in [skills.go](../../../../cmd/composer/action/skills/skills.go). Generated CLI references embed that section. Choices that are easy to reopen: exit 7 is `confirmation_required` (not usage), exit 6 is the only unchanged retry, `conflict` stays exit 5 with `retryable: true`, confirmation stays on commands that already prompted, and `version` exits 0 with `server_error` when the server cannot be read. `--help` stays cobra text. `ToToolError` lives in `pkg/toolerr`. CLI-only `cancelled` stays in the CLI exit layer.

## Alternatives considered

- **Map "needs confirmation" to exit 2.** Rejected: that code is usage, and the table is frozen, so agents could not later tell the two states apart.
- **Use exit 7 only in v1.1.** Rejected: the code has to be stable in v1.0 even though the interactive protocol can grow later.
- **Match MCP success shapes (`items`, `trace`).** Rejected: the CLI calling surface is one envelope. Trace stays on MCP.
- **Default `--output` to json when stdin is not a terminal.** Rejected: pipes such as `flowbot hub apps | grep` would change with the terminal.
- **Put `ToToolError` in `pkg/mcp`.** Rejected: `pkg/types` imports `pkg/config`, which imports `pkg/mcp`. See [MCP tool contract](../architecture/2026-10-16-mcp-tool-contract.md).
- **Import `internal/server/mcp` from the CLI.** Rejected: that links the server MCP package into the CLI binary.

## Consequences

- Existing `-o json` callers that parsed a bare array or object now parse `data`.
- `pipeline export -o path.yaml` is `pipeline export --file path.yaml`.
- Skill references embed `cliOutputSection`.
- Table mode keeps the existing sentence. `-o json` writes that command's API value under `data`. Id-only deletes use `{"id":"..."}`. Boolean health checks use `{"healthy":...}`. Trilium content get keeps `{id,content}`. MCP config JSON is the snippet object.

## Verification

`go test ./pkg/toolerr/ ./cmd/cli/utils/ ./cmd/cli/command/ ./internal/server/mcp/ -count=1` covers the shared error object, exit codes, the success envelope, an empty page, a JSON object, YAML export, a document body, the stderr error envelope, leaf inheritance of `--output`, version's `server_error`, confirmation exit 7, and login's JSON stdout.
