# Agent Note: Trajectory tool call visibility

Status: implemented

## Problem

The Trajectory log showed TOOL result rows as raw output only. Mid-turn persistence stores tool results before the assistant `tool_calls` entry, so CALL rows appeared after results (or looked like ASSISTANT). Operators could not see the method name or arguments without digging through Raw JSON, and tool-result Raw lacked `arguments`.

## Decision

`assembleTrajectory` indexes assistant tool calls first, buffers early tool results until their matching CALL row is emitted, and copies `arguments` onto the tool-result `raw` payload. CALL rows use role `call`, kind `tool_call`, and `msg.SummarizeToolCallParts` for list text. The web log chip labels CALL distinctly; TOOL previews prefix `tool_name` (and error); the inspector Preview/title surface name and arguments.

## Alternatives considered

- **Frontend-only preview of `tool_name`.** Would still leave CALL after TOOL and omit arguments on tool-result Raw.
- **Change harness persist order.** Correct for new runs only; Trajectory must stay honest for existing mid-turn trees.
- **Merge CALL+TOOL into one row.** Loses the call/result duration split used by the gantt Tools track.

## Consequences

- Trajectory JSON: `tool_call` role is `call` (was `assistant`); tool-result `raw` may include `arguments` joined from the call.
- Historical sessions without assistant `tool_calls` still show TOOL output; name comes from the result message when present.

## Verification

- `go test ./internal/server/chatagent -run TestAssembleTrajectory -count=1`
- Web: `public/js/chatagent-trajectory.js` CALL chip + `name → output` preview; inspector shows arguments on TOOL Preview/Raw.
