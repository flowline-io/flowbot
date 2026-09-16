# Agent Note: LLM generate errors include model name

Status: implemented

## Problem

When dual-model chat agent routing switches from `chat_model` to `tool_model`, a provider failure (for example HTTP 402) was wrapped as `agent llm: generate content: …` with no model name. Operators and the UI error surface looked like the selected chat model failed even when the failing call used a different tool model.

## Decision

Wrap LLM generate / complete / empty-response failures with `model=<name>` in `pkg/agent/llm` (`mapGenerateError`, `runCompletion`, `generateWithModel`, assemble empty response). The chat-agent run path already surfaces `result.Err.Error()` to clients and logs, so the model appears without a second product-layer rewrite.

## Alternatives considered

- **Annotate only in chat-agent service logs** — lost for UI / SSE error messages and non-chat callers of `StreamAssistant` / `Complete`.
- **Change the UI model picker to also set tool_model** — unrelated to error clarity; dual routing remains a server/YAML default.

## Consequences

Error strings look like `agent llm: generate content model=mimo-v2.5: API returned unexpected status code: 402: …`. Callers that match exact prefixes must allow the `model=` segment; `errors.Is` / `%w` wrapping is unchanged for the underlying cause.

## Verification

`go test ./pkg/agent/llm -run TestStreamAssistant_GenerateErrorIncludesModel` asserts the wrapped error contains the failing model name.
