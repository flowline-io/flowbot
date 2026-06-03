# Token Management UI + Store — Design Spec

**Date**: 2026-06-03
**Status**: Approved

## Overview

Add a token management page to the web module for listing, creating, and revoking API tokens.
Tokens remain stored as rows in the existing `parameter` table (no new schema).
The page lives at `/service/web/tokens` with a top-level "Tokens" nav link.

## Decisions

| Decision | Choice |
|----------|--------|
| Data layer | Use existing `parameter` table |
| Last used time | Track via `last_used_at` in `params` JSON, updated in auth middleware with 60s throttle |
| Nav placement | Top-level "Tokens" nav link in `base.templ` |

## Store Layer

### New methods on `Adapter` interface in `internal/store/store.go`

All three methods filter parameters where `flag` starts with `fb_` (the token prefix from `auth.NewToken()`).

#### `ListTokens(ctx context.Context) ([]model.TokenItem, error)`

Queries all parameters with `flag` LIKE `fb_%`, extracts uid/scopes/expiry/last_used from the `params` JSON column. Returns sorted by `created_at` descending. Filters out expired+unused (`last_used_at == null`) tokens older than 30 days. No pagination — homelab token count is expected to be small.

```
postgres: list tokens:
  SELECT * FROM parameter WHERE flag LIKE 'fb_%' ORDER BY created_at DESC
  Parse params JSON to extract uid, scopes, last_used_at
```

#### `CreateToken(ctx context.Context, uid types.Uid, expiresAt time.Time, scopes []string) (string, error)`

Generates a token string via `auth.NewToken()`, inserts a new `parameter` row:

```
flag       = token (plaintext, e.g. "fb_abc123...")
params     = {"uid": "user:xxx", "scopes": ["hub:apps:read"], "last_used_at": null}
expired_at = expiresAt
```

Returns the plaintext token string so the UI can display it once.

#### `RevokeToken(ctx context.Context, flag string) error`

Deletes the parameter row by `flag`. Returns `types.ErrNotFound` if no matching row.

```
DELETE FROM parameter WHERE flag = $1
```

### New type in `pkg/types/model/`

```go
type TokenItem struct {
    Token      string     `json:"token"`
    UID        types.Uid  `json:"uid"`
    Scopes     []string   `json:"scopes"`
    CreatedAt  time.Time  `json:"created_at"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty"`
    ExpiredAt  time.Time  `json:"expired_at"`
}
```

### Error handling

- `ListTokens`: wraps DB errors as `fmt.Errorf("postgres: list tokens: %w", err)`
- `CreateToken`: wraps insert errors; returns `types.ErrAlreadyExists` on constraint violation (should not happen with random tokens)
- `RevokeToken`: returns `types.ErrNotFound` when flag not found; `types.ErrInternal` on DB errors

### Transactions

`CreateToken` and `RevokeToken` are single-query operations — no multi-step writes that need a transaction.

## Auth Middleware

### `pkg/route/route.go` `Authorize()`

After successful token validation (existing `ParameterGet` + `ParameterIsExpired` check), update `params.last_used_at` on the parameter row:

1. Parse existing `params` JSON
2. Check if `last_used_at` was set more than 60 seconds ago (or is null)
3. If yes, set `last_used_at = time.Now()` and persist via `ParameterSet`
4. If no, skip the write (throttle to avoid write-on-every-request)

This gate means at most one write per 60 seconds per token — negligible overhead.

### Token prefix display

When displaying tokens in the list, show only the first 12 characters (prefix) of the token string to avoid exposing full tokens. The full token is only shown once at creation time. This matches the existing `auth.TokenPrefix()` helper.

## Web Routes

New `tokenWebserviceRules` routeset in `internal/modules/web/`, mounted alongside other routesets in `module.go`.

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | `/service/web/tokens` | `tokensPage` | `WithNotAuth()` | Full page |
| GET | `/service/web/tokens/list` | `tokensList` | `WithNotAuth()` | Table fragment |
| GET | `/service/web/tokens/new` | `tokensNewForm` | `WithNotAuth()` | Form fragment |
| POST | `/service/web/tokens` | `tokensCreate` | `WithNotAuth()` | Create token |
| DELETE | `/service/web/tokens/:flag` | `tokensRevoke` | `WithNotAuth()` | Revoke token |

Routes use `WithNotAuth()` because the web module's `authenticateWeb()` middleware already handles the access token cookie for all `/service/web/*` routes.

### Handler behaviors

**`tokensPage`**: Calls `store.Database.ListTokens(ctx)`, renders `pages.TokensPage(data)` with full base layout.

**`tokensList`**: Calls `store.Database.ListTokens(ctx)`, renders `partials.TokenTable(data)` as an HTML fragment. Used for HTMX refresh after create/revoke.

**`tokensNewForm`**: Renders `partials.TokenForm(nil)` with empty form. Any `?uid` query param pre-fills the UID field.

**`tokensCreate`**: Parses form fields (`uid`, `scopes[]`, `expires`), calls `store.Database.CreateToken(ctx, ...)`. On success, renders the token row AND an alert div showing the full token with a "copy" button (single display). On validation error, re-renders the form with `errors` map.

**`tokensRevoke`**: Parses `:flag` from URL, calls `store.Database.RevokeToken(ctx, flag)`. On success, returns empty response with `hx-swap-oob` to remove the row.

### Scope selection

Scopes are presented as checkboxes grouped by domain. Available scopes come from the existing `pkg/auth/scope.go` constants:

```
admin:     admin:*
hub:       hub:apps:read, hub:apps:write, hub:capabilities:read, hub:health:read
service:   service:*:read, service:*:write
pipeline:  pipeline:read, pipeline:run, pipeline:write
workflow:  workflow:run
```

### Expiry selection

A `<select>` with presets: 7 days, 30 days, 90 days, 1 year, custom (shows a date input). Server receives the computed `expiresAt` timestamp.

### Validation

- `uid` required, non-empty
- `scopes` at least one checked
- `expires` valid future timestamp
- Errors render the form inline: `fieldError(errors, "uid")` returns `"border-red-500"`

## Templ Files

| File | Purpose |
|------|---------|
| `pkg/views/pages/tokens.templ` | Full page: `<h1>Tokens</h1>`, "New Token" button, calls `@partials.TokenTable(items)` |
| `pkg/views/partials/token_table.templ` | `<table>` with `<thead>`: UID, Token, Created, Last Used, Expires, Scopes, Actions. Renders `TokenRow` per item, or `EmptyState` if `len(items)==0` |
| `pkg/views/partials/token_row.templ` | `<tr>`: uid text, masked token prefix (`prefix + "..."`), relative time (via `timeSince()` helper), scope `<span class="badge">` list, Revoke button with `hx-confirm` + `hx-delete` |
| `pkg/views/partials/token_form.templ` | `<tr>` form row: UID text input, scope checkboxes in a grid, expiry select + optional date input, Save + Cancel buttons. On create success, the server renders: the new `token_row` for the table + an OOB `<div>` alert with the full token + copy button |

### Data attributes

All interactive elements use `data-testid` for testability:
- Table: `data-testid="token-table"`
- New button: `data-testid="token-new-btn"`
- Form row: `data-testid="token-form"`
- UID input: `data-testid="token-form-uid"`
- Scope checkboxes: `data-testid="token-form-scope-{value}"`
- Save button: `data-testid="token-form-save"`
- Cancel button: `data-testid="token-form-cancel"`
- Revoke button: `data-testid="token-revoke-btn"`
- Token alert: `data-testid="token-created-alert"`
- Copy button: `data-testid="token-copy-btn"`

### Helpers

Add helper functions in `pkg/views/partials/helpers.go` or a new `token_helpers.go`:

- `timeSince(t time.Time) string` — returns relative time like "2 hours ago", "3 days ago", "never"
- `scopeBadge(scope string) string` — maps scope value to a readable label ("admin:*" -> "Admin")

## Nav Bar

### `pkg/views/layout/base.templ`

Add a "Tokens" `<a>` link in the navigation bar, placed alphabetically among existing links (after Notifications, before Workflows if present):

```html
<a href="/service/web/tokens" class="btn btn-ghost btn-sm">
    <span>Tokens</span>
</a>
```

## Implementation Order

1. `pkg/types/model/model.go` — add `TokenItem` struct
2. `internal/store/store.go` — add `ListTokens`, `CreateToken`, `RevokeToken` to `Adapter` interface
3. `internal/store/postgres/adapter.go` — implement the three methods
4. `internal/store/store_test.go` — unit tests for new methods
5. `pkg/views/partials/` — add `token_table.templ`, `token_row.templ`, `token_form.templ`
6. `pkg/views/pages/tokens.templ` — add full page
7. `pkg/views/partials/token_helpers.go` — add `timeSince`, `scopeBadge`
8. `go tool task templ` — generate Go code from templates
9. `internal/modules/web/` — add `tokenWebserviceRules` + handlers in `token_webservice.go`
10. `internal/modules/web/module.go` — mount `tokenWebserviceRules` in `Webservice()`
11. `pkg/views/layout/base.templ` — add "Tokens" nav link
12. `go tool task templ` — regenerate
13. `pkg/route/route.go` — add `last_used_at` update in `Authorize()`
14. `pkg/route/route_test.go` — test for last_used_at update
15. `go tool task lint && go tool task test` — verify

## Testing

### Unit tests (table-driven, `*_test.go` co-located)

**Store tests** (`internal/store/store_test.go`):
- `TestListTokens`: 3+ cases — empty, with tokens, with expired tokens
- `TestCreateToken`: 3+ cases — success, duplicate uid, past expiry
- `TestRevokeToken`: 3+ cases — success, not found, already revoked

**Handler tests** (`internal/modules/web/token_webservice_test.go`):
- `TestTokensPage`: renders with tokens list
- `TestTokensCreate`: creates token, returns token in response
- `TestTokensCreateValidation`: missing uid, missing scopes
- `TestTokensRevoke`: success, not found

### BDD specs (Ginkgo v2 + Gomega, `tests/`)

`Describe("Token Management")`: visit tokens page, create token, see token once, revoke token with confirmation, verify token no longer works.
