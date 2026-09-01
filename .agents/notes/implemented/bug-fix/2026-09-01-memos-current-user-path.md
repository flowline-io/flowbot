# Agent Note: Memos current-user health path across 0.25 and 0.26

Status: implemented

## Problem

Capability health (`memos.health`) calls `GetCurrentUser`, which only requested `GET /api/v1/auth/me`. Homelab Memos 0.25.x (`neosmemo/memos:stable` at 0.25.3) does not register that route. The instance returns gRPC-gateway `404 {"code":5,"message":"Not Found"}`, so healthz marks memos unhealthy even when list/create still work.

0.25.x exposes the same user payload at `GET /api/v1/auth/sessions/current`. 0.26+ replaced that RPC with `GET /api/v1/auth/me`. Operators cannot upgrade Memos in lockstep with Flowbot.

Stripping a mistaken `/api/v1` endpoint suffix ([capability-health-probes](2026-09-01-capability-health-probes.md)) does not fix this: a correctly configured origin still 404s on `/auth/me`.

## Decision

`GetCurrentUser` tries `GET /api/v1/auth/me` first. On HTTP 404 only, it retries `GET /api/v1/auth/sessions/current`. Other statuses (401, 500, network errors) are not retried so a bad token on 0.26+ stays a hard failure.

This is a recorded shim for an external consumer that cannot move in the same change ([foundation-over-shims](../process/2026-08-13-pre-release-foundation-over-shims.md)): homelab Memos 0.25.x.

## Alternatives considered

- **Call only `/api/v1/auth/sessions/current`.** Rejected: 0.26+ dropped that RPC; health would fail after a Memos upgrade.
- **Health-check `GET /api/v1/instance/profile` or public `GET /api/v1/memos`.** Rejected: those succeed without a token, so they would not catch an expired or missing access token.
- **Require operators to upgrade to Memos 0.26+.** Rejected: `neosmemo/memos:stable` is still 0.25.3 in this homelab, and health should not depend on a major Memos bump.

## Consequences

- One extra round trip on 0.25.x health checks (404 then session GET).
- 0.26+ keeps a single request.
- Memo CRUD paths are unchanged; this note does not claim 0.25/0.26 request-body compatibility for create/update.

## Verification

- `go test ./pkg/providers/memos -run 'TestGetCurrentUser'`
- Unauthenticated probe of a 0.25.3 instance: `/api/v1/auth/me` → 404 gRPC Not Found; `/api/v1/auth/sessions/current` → 401 (route exists).
