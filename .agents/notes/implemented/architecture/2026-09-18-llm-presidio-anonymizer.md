# Agent Note: LLM Presidio session-scoped reversible anonymization

Status: implemented

## Problem

Homelab Flowbot sends chat history, tool results, and auxiliary LLM prompts (title, compaction, clip, life) to external model providers. Those payloads often contain PII (emails, phones, CN ID numbers, names). There is no single choke point that anonymizes outbound LLM traffic while keeping agents able to call tools with real values and keeping PostgreSQL chat history readable.

## Decision

- Presidio **Analyzer only** ships as a Docker sidecar ([`deployments/presidio`](../../../../deployments/presidio/), Compose profile `pii`, GHCR `flowbot-presidio-analyzer`). English + Chinese spaCy (`en_core_web_md`, `zh_core_web_md`) plus CN phone/ID regex recognizers. Anonymizer is not deployed.
- Every real `llms.Model` from [`NewModel`](../../../../pkg/agent/llm/factory.go) is wrapped with `piiModel`. Each `GenerateContent` reads top-level `pii.enabled` so pooled models stay correct under config hot-reload. When enabled, Analyzer errors fail closed.
- Reversible placeholders (`{{PII_EMAIL_ADDRESS_1}}`) live in a process-memory table keyed by chat `session_id` (sliding TTL). [`WithPIISession`](../../../../pkg/agent/llm/pii_session.go) is set on Service.Run / CompactSession, title, and session-summary request contexts. Sessionless Complete (clip/life/judge/knowledge) uses a per-call table.
- Anonymize `human` / `ai` / `tool` text parts only; skip `system` and image parts. Restore streamed text, reasoning deltas, final content, and tool-call arguments.
- PostgreSQL keeps plaintext; only the outbound provider request is anonymized. Analyzer results are hash-cached per session.

## Alternatives considered

- **Irreversible redact / Anonymizer replace** — rejected; breaks tool arguments and multi-turn consistency.
- **Anonymizer encrypt/decrypt** — rejected; models see ciphertext, streaming decrypt is fragile.
- **HTTP RoundTrip on OpenAI transport** — rejected; Anthropic bypasses it and SSE chunks split placeholders.
- **Only wrap StreamAssistant helpers** — rejected; judge and direct GenerateContent paths leak.
- **Config under chat_agent.pii** — rejected; clip/life are not chat-agent-only.
- **Fail-open when Analyzer is down** — rejected; protection would be illusory.
- **Persist mapping in DB** — rejected; creates a second PII vault with larger blast radius.

## Consequences

- Operators opt in via `pii.enabled` plus `docker compose --profile pii`. Default stack is unchanged.
- System-prompt / context-file PII is not scanned in this change.
- In-memory maps reset on process restart; placeholder IDs may change across restarts for the same session.

## Verification

- `go test ./pkg/agent/llm/ ./pkg/config/ ./pkg/metrics/ ./internal/server/chatagent/ -count=1`
- `go tool task lint`
- Presidio `start`/`end` are treated as Unicode rune indices and mapped to UTF-8 byte offsets before slicing
- Stream restorer flushes buffered remainder back into StreamingFunc / StreamingReasoningFunc after GenerateContent
- CI: [`.github/workflows/docker-presidio.yml`](../../../../.github/workflows/docker-presidio.yml) on `presidio-v*` with en/zh `/analyze` smoke
- Operator docs: [self-hosting](../../../../docs/self-hosting.md#optional-llm-pii-presidio), [`pkg/agent/AGENTS.md`](../../../../pkg/agent/AGENTS.md)
- Unit tests that assert Analyzer call counts must use a unique session id per invocation and `resetPIISessionForTest`; a fixed id plus `-count=N` / race shards otherwise hits the session hash cache and reports `analyzeCalls=0`
