# Agent Note: Fiber 3.7 CSRF storage key prefix

Status: implemented

## Problem

Upgrading `github.com/gofiber/fiber/v3` from 3.6.1 to 3.7.0 makes CSRF-protected `/service/web` mutations return 403 in unit and BDD tests that seed tokens via `AttachCSRFForTest` / `seedCSRFToken`. Matching cookie + header (or form) pairs that passed on 3.6.1 fail because Fiber no longer finds the seeded key in Storage.

## Decision

Fiber 3.7.0 prefixes CSRF Storage keys with the unexported `storagePrefix` (`"csrf:"`) in `middleware/csrf/storage_manager.go`. Test seeding writes the same prefixed key (`csrfStoragePrefix+token`) so middleware `GetWithContext` hits the entry. Production tokens created by the middleware already use the prefix.

## Alternatives considered

- **Drop custom Storage and let Fiber own an in-memory default.** Would remove seed/prefix coupling, but loses the shared store used across test app instances and `AttachCSRFForTest`.
- **Issue tokens only through a real GET that runs CSRF middleware.** Correct but slower and harder to share a fixed token across many httptest helpers.
- **Pin Fiber at 3.6.1.** Avoids the break; blocks fasthttp/security fixes in 3.7.0.

## Consequences

`seedCSRFToken` and Fiber's Storage contract stay coupled to the `"csrf:"` prefix string. If Fiber renames the prefix again, CSRF unit tests (`TestCSRFMiddleware`, `TestEnsureCSRFCookieSetsCookie`) go red before production login does.

## Verification

`go test ./internal/modules/web/ -run 'TestCSRF|TestAttachCSRFForTest|TestAgentKnowledgeCreateAuthenticated' -count=1` passes on Fiber v3.7.0. Matching-header / matching-form CSRF cases return 200; reuse-of-seeded-cookie still keeps the same token value.
