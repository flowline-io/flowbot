# Agent Note: LLM network errors are sanitized and not retried

Status: implemented

## Problem

langchaingo replaces `http.Client` `net.Error` values with `network error: failed to reach API server` (and timeouts with a generic timeout string) so API keys and request URLs do not leak through `GenerateContent`. Chat-agent logs then show only that placeholder; `IsRetryableLLMError` did not match it, so a single ~5s dial/DNS failure aborted the turn with `stream_started=false`.

## Decision

The OpenAI-compatible transport logs the unsanitized RoundTrip error at warn (`[agent-llm] http roundtrip failed`) with method, redacted URL (no query or userinfo), duration, and `err`. `IsRetryableLLMError` treats `network error` / `failed to reach api server` as retryable so a later attempt can succeed before any stream delta.

## Alternatives considered

- **Fork langchaingo to keep `Unwrap` on sanitized errors.** Real cause would survive `errors.As`, but a fork is out of scope for this failure.
- **Return a wrapped transport error so langchaingo's type assertion misses `net.Error`.** `http.Client` still wraps RoundTrip failures as `*url.Error`, which implements `net.Error`, so sanitization still wins.
- **Log only, no retry.** Leaves a transient DNS/connect blip as a hard turn failure.

## Consequences

Operator logs include host, path, and the Go dial/DNS/TLS text. Client-facing chatagent errors stay the sanitized langchaingo string. Transient reachability failures retry up to `chat_agent.llm_retry` (default 3 attempts) when no stream delta has been delivered.

## Verification

`TestRedactHTTPURL` and `TestErrorLogTransportPassesThrough` in `pkg/agent/llm/http_client_test.go`. `TestIsRetryableLLMError` case `langchaingo sanitized network error` and `TestStreamAssistantRetry` case `retries langchaingo sanitized network error` in `pkg/agent/llm/retry_test.go`.
