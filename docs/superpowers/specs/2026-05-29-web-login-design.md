# Web Module Login Page — Design Spec

**Date**: 2026-05-29
**Status**: Approved

## Overview

Add username/password login page to the existing web module (`internal/modules/web/`). Single-user homelab scenario: credentials stored in `flowbot.yaml` config, token-based auth reused from the existing `pkg/auth/` infrastructure. All web routes require authentication; unauthenticated visitors are redirected to the login page.

## Approach

Reuse the existing token infrastructure (`auth.NewToken()` + `parameters` table + `route.GetRequestContext()`) rather than introducing JWT, bcrypt, or Redis sessions. This is the minimal-change approach that stays consistent with the project's existing auth model.

## Configuration

### `flowbot.yaml` changes

```yaml
web:
  enabled: true
  auth:
    username: "admin"
    password: "admin"
```

Password is plaintext — the YAML file is protected by filesystem permissions (`chmod 600`) on a single-user homelab. No bcrypt dependency.

### `internal/modules/web/module.go` — `configType`

```go
type AuthConfig struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type configType struct {
    Enabled bool       `json:"enabled"`
    Auth    AuthConfig `json:"auth"`
}
```

### `internal/modules/web/module.go` — `moduleHandler`

```go
type moduleHandler struct {
    module.Base
    initialized bool
    authConfig  AuthConfig
}
```

`Init()` parses the JSON config blob (already injected by server layer from `config.Bots["web"]`) and stores `authConfig` for use by login handlers.

## Routes & Middleware

### Route table

| Method | Path | Handler | Auth | Notes |
|--------|------|---------|------|-------|
| `GET` | `/service/web/login` | `loginPage` | *none* | Renders login form |
| `POST` | `/service/web/login` | `loginSubmit` | *none* | Validates credentials, sets cookie |
| `POST` | `/service/web/logout` | `logout` | *none* | Deletes token, clears cookie |
| `GET/POST/PUT/DELETE` | `/service/web/configs/*` | existing handlers | *modified* | Redirect to login if unauthenticated |

### `requireAuth()` modification

Existing `requireAuth()` returned an error for unauthenticated requests. Changed to redirect:

```go
func requireAuth(c fiber.Ctx) error {
    if route.GetRequestContext(c) != nil {
        return c.Next()
    }
    next := url.QueryEscape(string(c.Request().URI().RequestURI()))
    return c.Redirect().To("/service/web/login?next=" + next)
}
```

### Login GET (`loginPage`)

- Checks `route.GetRequestContext(c)` — if user already has a valid token cookie (authenticated), redirects to `next` (or `/service/web/configs` as fallback). No need to show login form.
- Otherwise renders `pages.LoginPage(templ.SafeURL(next), errMsg)` via Templ
- Embeds `layout/base.templ`
- Reads `?next=` from query string, stored as hidden form field in the Templ template

### Login POST (`loginSubmit`)

The form submits via HTMX (`hx-post`). Two response paths:

**Failure (wrong credentials):** Return 200 with re-rendered `pages.LoginPage` + error message. HTMX swaps the form DOM in-place, showing the error.

**Success:** Server sets `HX-Redirect` response header to the target URL. HTMX picks this up and performs a client-side `window.location` redirect (handy because the `Set-Cookie` header on the same response ensures the cookie is written before navigation).

Success steps:
1. Parse form values (`username`, `password`, `next`)
2. Compare against `authConfig.Username` / `authConfig.Password`
3. On mismatch: re-render `pages.LoginPage` with error message (200), no redirect
4. On match:
   - `token := auth.NewToken()` — generates `fb_` + 32-byte base64url token
   - `hashedToken := auth.HashToken(token)` — SHA-256 hash for DB storage
   - `params := map[string]interface{}{"token": hashedToken, "scopes": "admin:*"}`
   - `store.Database.ParameterSet(ctx, hashedToken, params)` — writes to `parameters` table
   - Set cookie: `accessToken=token; HttpOnly; Secure; SameSite=Lax; Path=/; MaxAge=86400` (24h)
   - Validate `next` is a relative path (starts with `/`) to prevent open redirect
   - Set response header: `HX-Redirect: <validated-next>` (or `/service/web/configs` as fallback)
   - Return 200 (HTMX processes the redirect header)

### Logout POST (`logout`)

1. Read `accessToken` from cookie
2. If present: `hashedToken := auth.HashToken(value); store.Database.ParameterDelete(ctx, hashedToken)`
3. Clear cookie: `accessToken=; MaxAge=0`
4. Redirect 302 to `/service/web/login`

## Templ Templates

### New file: `pkg/views/pages/login.templ`

- Embeds `layout.Base("Flowbot — Login")`
- Centered card layout with username input, password input, submit button
- Hidden `<input name="next">` carrying the `next` query parameter
- `<form hx-post="/service/web/login" hx-target="this" hx-swap="outerHTML">` — HTMX handles the POST
- Error message shown via `{ .Error }` when login fails
- No Alpine.js needed — the "already logged in" check is done server-side in the handler before rendering

### Generated Go: `pkg/views/pages/login_templ.go`

Auto-generated by `go tool task templ`, committed alongside `.templ` source.

### Existing templates: no changes needed

- `layout/base.templ` — login page embeds it directly, no modification
- `pages/configs.templ`, `partials/*` — no changes

## Store Layer

### New methods in `internal/store/store.go`

```go
// ParameterSet inserts or upserts a parameter with the given ID and key-value pairs.
ParameterSet(ctx context.Context, id string, params map[string]interface{}) error

// ParameterDelete removes a parameter by its ID.
ParameterDelete(ctx context.Context, id string) error
```

Existing `ParameterGet()` already reads from the `parameters` table. The new methods enable token lifecycle management (create on login, delete on logout). Implemented via ent `Parameter` client with `OnConflict().UpdateNewValues()` for upsert semantics.

## Open Redirect Prevention

The `next` parameter in login flow is validated:

```
next 必须以 "/" 开头 === 有效
next 包含 "//" 或 ":" === 拒绝，fallback 到 /service/web/configs
```

Prevents `?next=https://evil.com/` style attacks.

## Security Considerations

| Concern | Mitigation |
|---------|------------|
| Cookie hijacking | `HttpOnly` (JS can't read), `Secure` (TLS only), `SameSite=Lax` |
| Token storage | SHA-256 hash in DB, raw token only in HttpOnly cookie |
| Open redirect | `next` validated to start with `/`, reject `//` and `:` |
| Brute force | Not addressed — single-user homelab, rate limiter (50req/10s) already in middleware stack |
| Config file exposure | `flowbot.yaml` protected by file permissions |

## Testing

### Unit tests (`internal/modules/web/module_test.go`)

All tests follow table-driven pattern with `t.Run`. Three new test functions:

| Test function | Cases |
|---|---|
| `TestLoginPage` | no cookie → 200, valid cookie → 302, missing `next` param |
| `TestLoginSubmit` | correct creds → 302 + set-cookie, wrong password → 200 + error, empty username → 200 + error, correct creds + `next` → redirect to `next`, illegal `next` → fallback, empty `next` → fallback |
| `TestLogout` | has cookie → 302 + clear cookie, no cookie → 302 |
| `TestRequireAuthRedirect` | no AuthContext → 302 to login, valid AuthContext → 200 through |

### Test helper (`internal/modules/web/test_helper_test.go`)

`testStore` extended with `ParameterSet` / `ParameterDelete` mock stubs.

### BDD integration tests (`tests/`)

Ginkgo spec for full login flow: visit configs → redirected to login → submit credentials → redirected back to configs → logout → configs redirects to login again. Requires Docker for PostgreSQL.

## Implementation Order

1. `internal/store/store.go` — add `ParameterSet` + `ParameterDelete`
2. `internal/store/store_test.go` — unit tests for new store methods
3. `internal/modules/web/module.go` — add `AuthConfig`, parse in `Init()`, pass to handlers
4. `internal/modules/web/webservice.go` — add `loginPage`, `loginSubmit`, `logout` handlers; modify `requireAuth()`
5. `pkg/views/pages/login.templ` — new Templ template
6. `go tool task templ` — generate `login_templ.go`
7. `internal/modules/web/module_test.go` — tests for handlers
8. `tests/` — BDD integration spec
9. `go tool task lint && go tool task test` — verify
