# Agent Note: chat_agent.enabled master switch

Status: implemented

## Problem

Chat agent on/off was implied by a non-empty `chat_agent.chat_model`. Operators could not keep a model configured while leaving the agent off, and `core.agent_run` still ran whenever a pipeline runner was wired even when the agent was notionally disabled.

## Decision

- Add `chat_agent.enabled` as `*bool`.
  - `nil`: legacy — non-empty `chat_model` enables; startup logs a compat warning.
  - `false`: agent off even when `chat_model` is set.
  - `true`: requires non-empty `chat_model` (validated at load).
- `config.ChatAgentEnabled()` is the single gate for product surfaces and `core.agent_run` / `core.agent_health`.
- When off, `agent_run` returns `unavailable` with an explicit disabled message.

## Alternatives considered

- Keep `chat_model != ""` as the only switch. Rejected: cannot park a model while the agent is off, and the enable signal stays implicit.
- Require `enabled` immediately with no nil legacy path. Rejected: would break every existing yaml that only sets `chat_model` before 1.0.

## Consequences

- Reference yaml documents `enabled`; omitting it while setting `chat_model` is Deprecated until 1.0.
- `core.agent_health` no longer reports healthy solely because a runner is registered.

## Verification

- `go test ./pkg/config ./pkg/capability/core -count=1`
- BDD: `tests/specs/chat_agent_spec_test.go` covers `agent_run` unavailable when `enabled: false`.
