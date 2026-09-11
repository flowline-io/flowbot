# Agent Note: Webhook query-token migration UX

Status: implemented

## Problem

Pipeline/function/workflow webhook auth no longer accepts `?token=` ([security hardening](../bug-fix/2026-09-11-security-audit-hardening.md)). Callers that still use query tokens get opaque 401s; the UI still had stale “or query token” copy in places, and operators lacked a clear curl example or audit trail for the deprecated pattern.

## Decision

Ship a thin migration layer on top of the hard cut:

1. UI labels state Header/HMAC only (pipeline drawer/card hints via `webhookAuthHint`, function call links, workflow trigger rows).
2. One-click curl examples send `X-Webhook-Token` and/or HMAC (`X-Hub-Signature-256`) headers and never embed `?token=` (pipeline, workflow, function).
3. Any request that still carries `?token=` records `webhook.auth.query_token_deprecated` via `route.WarnLegacyWebhookQueryToken` (pipeline, workflow, function). The token value is never logged or stored.

## Alternatives considered

- Soft-accept query tokens with a deprecation window — rejected; same foundation-over-shims rationale as the hardening note.
- Log-only warnings without audit rows — rejected; audit is the durable ops signal for spotting stuck integrations.

## Consequences

- Function editor auth hint no longer mentions query tokens.
- Workflow trigger table shows auth hint + copy-curl alongside the URL.
- Ops can filter audit logs on `webhook.auth.query_token_deprecated` to find callers that need migration.

## Verification

- `go test` covers `route.WarnLegacyWebhookQueryToken`, workflow curl/auth-hint helpers, and pipeline editor CSP markers for the auth hint and curl button.
- Package tests for pipeline/function webhook auth continue to assert query tokens are unauthorized.
