# E2E Testing with Go-rod — Design

Date: 2026-05-29
Status: approved

## Overview

Add browser-based end-to-end tests for `internal/modules/web` using Go-rod. The tests simulate real user interactions (login, config CRUD) against a real Fiber server backed by PostgreSQL + Redis testcontainers. The six Go-rod golden principles (port isolation, context isolation, no hardcoded sleep, data-testid, failure screenshots, lifecycle defense) are enforced throughout.

## Scope

- Login flow: GET `/login`, POST `/login`, POST `/logout`
- Config CRUD: GET `/configs`, GET `/configs/new`, POST `/configs`, GET `/configs/{id}/edit`, PUT `/configs/{id}`, DELETE `/configs/{id}`
- `data-testid` attributes added to `.templ` template files in `pkg/views/`
- CI job `e2e` in `.github/workflows/testing.yml`
- Go-rod conventions documented in `AGENTS.md`

## Architecture

**Approval chosen: Approach B** — test helper creates a `fiber.App` on a dynamic port with real store (testcontainers), connected via go-rod. This avoids the cost of compiling `cmd/flowbot` while testing the actual Fiber handlers and store layer.

```
func TestMain(m *testing.M) {
    os.Exit(run(m))  // defer runs before os.Exit, containers shutdown properly
}

func run(m *testing.M) int {
    // 1. Start PostgreSQL container + Redis container (testcontainers)
    // 2. Run ent migrations on flowbot_e2e database
    // 3. Build fiber.App, mount web module routes
    // 4. net.Listen("tcp", "127.0.0.1:0") → dynamic port
    // 5. baseURL = "http://127.0.0.1:{port}"
    // 6. browser = rod.New().MustConnect()
    // defer: browser.MustClose(), app.Shutdown(), pgC.Terminate(), redisC.Terminate()
    //
    // 7. return m.Run()
    //    ├── TestLoginPage → browser.MustContext() → MustPage() → MustNavigate()
    //    ├── TestLoginSubmit → incognito session, type credentials, submit
    //    ├── TestLogout → login, logout, verify redirect
    //    ├── TestConfigsPage → cookie-injected session, navigate, wait for HTMX
    //    ├── TestConfigCreate → New Config, fill form, save, verify row
    //    ├── TestConfigUpdate → seed config, edit, save, verify change
    //    └── TestConfigDelete → seed config, click delete, verify removal
}
```

Critical: Do NOT call `os.Exit(m.Run())` directly in `TestMain`. That skips all `defer` statements and leaks testcontainers. Instead return the exit code from `run()` so all deferred cleanup executes before `os.Exit`.

## Directory Structure

```
tests/e2e/
  setup_test.go           // TestMain, global Browser, testcontainers, Fiber app on dynamic port
  login_test.go           // Login flow tests (GET /login, POST /login, POST /logout)
  config_crud_test.go     // Config CRUD tests (list, create, edit, delete)
  test-reports/           // Failure screenshots (gitignored)
```

All files use `//go:build e2e` build tag. Framework: testify with table-driven pattern, minimum 3 cases per table, `t.Run` per case.

## setup_test.go

Global state (package-level):

- `browser *rod.Browser` — one per process, shared across tests
- `baseURL string` — `http://127.0.0.1:{dynamic-port}`
- `app *fiber.App` — for Cleanup
- `pgContainer`, `redisContainer testcontainers.Container` — for Cleanup

### Lifecycle contract (principle 6)

| Resource                  | Init                              | Cleanup                                          |
| ------------------------- | --------------------------------- | ------------------------------------------------ |
| PostgreSQL container      | `TestMain` pre-run                | `TestMain` defer                                 |
| Redis container           | `TestMain` pre-run                | `TestMain` defer                                 |
| Fiber app                 | `TestMain` pre-run                | `TestMain` defer `app.Shutdown()`                |
| `rod.Browser`             | `TestMain` pre-run                | `TestMain` defer `browser.MustClose()`           |
| Per-test `BrowserContext` | Each test `browser.MustContext()` | `t.Cleanup(ctx.MustClose)`                       |
| Per-test page             | Each test via `NewPage(t)`        | `t.Cleanup(page.MustClose)`                      |
| Per-test screenshot       | `t.Cleanup` on `t.Failed()`       | `page.MustScreenshot("test-reports/{name}.png")` |

### Port isolation (principle 1)

```go
l, err := net.Listen("tcp", "127.0.0.1:0")
baseURL = "http://" + l.Addr().String()
go app.Serve(l)
```

No hardcoded ports. OS assigns a free port via port 0.

### Database isolation

A fixed database `flowbot_e2e` is used (no parallel test processes in e2e). Tables are truncated once in `TestMain` after migrations and before tests run.

Additionally, a `ResetDB(t *testing.T)` helper truncates all tables at the start of each CRUD test case. This prevents data state bleeding between test cases within a table-driven test. Call `ResetDB(t)` at the top of each `t.Run` for any test that modifies database state (create, update, delete).```go
ResetDB := func(t \*testing.T) {
if err := store.Database.Reset(); err != nil {
t.Fatalf("reset db: %v", err)
}
}

````

### Context isolation (principle 2)

Each test function creates its own `BrowserContext` (incognito) via `browser.MustContext()`. Cookie, localStorage, and session state are isolated per test case.

### Helper functions

```go
// NewPage creates a new incognito page with cleanup and failure screenshot.
func NewPage(t testing.TB) *rod.Page

// URL returns the full URL for a given path.
func URL(path string) string

// ResetDB truncates all tables before each CRUD test case to prevent state bleeding.
func ResetDB(t *testing.T)

// loginAsAdmin logs in via UI flow (used only in login_test.go).
func loginAsAdmin(t *testing.T) *rod.Page

// loginViaCookie injects an accessToken cookie directly, skipping UI login (used in CRUD tests).
func loginViaCookie(t *testing.T) *rod.Page

// seedConfig creates a config directly via store adapter.
func seedConfig(t *testing.T, uid, topic, key string, value any) int64
````

### Programmatic login (cookie injection)

UI login (POST /login) is tested exhaustively in `login_test.go`. For CRUD tests in `config_crud_test.go`, authentication is done via direct cookie injection using `page.MustSetCookies()`. This avoids repeating the slow UI login flow in every CRUD test case.

```go
func loginViaCookie(t *testing.T) *rod.Page {
    page := NewPage(t)
    token := generateTestToken()  // or store.ParameterSet with token + SetCookies
    page.MustSetCookies(&proto.NetworkCookieParam{
        Name:  "accessToken",
        Value: token,
        URL:   baseURL,
    })
    return page
}
```

## Login Flow Tests (`login_test.go`)

### TestLoginPage

Cases: render login form, form is present, no error initially.

Selectors: `[data-testid="login-form"]`, `[data-testid="login-username"]`, `[data-testid="login-password"]`, `[data-testid="login-submit"]`.

### TestLoginSubmit

Cases:

- valid credentials (set via store seed or env) → redirected to /configs, `accessToken` cookie set
- invalid username → error message shown via `[data-testid="login-error"]`, still on login page
- empty credentials → error message shown

Assertions: After `page.MustWaitStable()`, check for presence/absence of elements and cookie state.

### Credential seeding

The web module uses token-based authentication stored in `store.ParameterGet/Set`. Before tests run, `TestMain` seeds a test token:

```go
// In run(), before m.Run():
token := "e2e-test-token"
store.Database.ParameterSet(ctx, "web", "access_token", token)
store.Database.ParameterSet(ctx, "web", "username_" + token, "admin")
```

### TestLogout

1. Call `loginAsAdmin(t)` to get authenticated session
2. Navigate to any page with nav, click `[data-testid="nav-logout"]`
3. `MustWaitStable()` — verify redirected to login page and no `accessToken` cookie

## Config CRUD Tests (`config_crud_test.go`)

### TestConfigsPage

1. `loginViaCookie(t)`
2. Navigate to `/configs`
3. `page.MustWaitRequestIdle()` — wait for HTMX-loaded config table
4. Wait for `[data-testid="configs-table"]` to appear
5. If empty DB → "No configs found." text present; if seeded → rows present

### TestConfigCreate

Cases:

- Create a string config → row appears with correct values
- Create a numeric config → JSON value renders correctly
- Create without required fields → error messages on form fields

Flow: `ResetDB(t)` → `loginViaCookie(t)` → click `[data-testid="configs-new"]` → fill `[data-testid="config-uid"]`, `[data-testid="config-topic"]`, `[data-testid="config-key"]`, `[data-testid="config-value"]` → click `[data-testid="config-save"]` → `page.MustWaitRequestIdle()` → verify new row in table.

### TestConfigUpdate

1. `ResetDB(t)` + `seedConfig(...)` via store
2. `loginViaCookie(t)` → `/configs`
3. Click `[data-testid="config-edit"]` on the seeded row
4. `page.MustWaitRequestIdle()` — wait for edit form HTMX swap
5. Form replaces row with `[data-testid="config-value"]` pre-filled
6. Modify value → click `[data-testid="config-save"]`
7. `page.MustWaitRequestIdle()` — wait for PUT to complete
8. Row updates with new value

### TestConfigDelete

1. `ResetDB(t)` + `seedConfig(...)` via store
2. `loginViaCookie(t)` → `/configs`
3. Set up dialog handler BEFORE clicking delete:
   ```go
   wait, handle := page.MustHandleDialog()
   go page.MustElement(`[data-testid="config-delete"]`).MustClick()
   wait()            // blocks until dialog appears
   handle(true, "")  // accept (OK) the confirm dialog
   ```
4. `page.MustWaitRequestIdle()` — wait for HTMX DELETE to complete
5. Assert row removed from table

## data-testid Additions

18 markers across 6 `.templ` files in `pkg/views/`:

| Template                      | Element                   | `data-testid`     |
| ----------------------------- | ------------------------- | ----------------- |
| `pages/login.templ`           | `<form>`                  | `login-form`      |
| `pages/login.templ`           | `<input name="username">` | `login-username`  |
| `pages/login.templ`           | `<input name="password">` | `login-password`  |
| `pages/login.templ`           | `<button type="submit">`  | `login-submit`    |
| `pages/login.templ`           | Error `<p>`               | `login-error`     |
| `pages/configs.templ`         | "New Config" button       | `configs-new`     |
| `pages/configs.templ`         | "Refresh" button          | `configs-refresh` |
| `partials/config_table.templ` | Table container `<div>`   | `configs-table`   |
| `partials/config_row.templ`   | "Edit" button             | `config-edit`     |
| `partials/config_row.templ`   | "Delete" button           | `config-delete`   |
| `partials/config_form.templ`  | `<input name="uid">`      | `config-uid`      |
| `partials/config_form.templ`  | `<input name="topic">`    | `config-topic`    |
| `partials/config_form.templ`  | `<input name="key">`      | `config-key`      |
| `partials/config_form.templ`  | `<textarea name="value">` | `config-value`    |
| `partials/config_form.templ`  | "Save" button             | `config-save`     |
| `partials/config_form.templ`  | "Cancel" button           | `config-cancel`   |
| `layout/base.templ`           | Nav "Configs" link        | `nav-configs`     |
| `layout/base.templ`           | "Logout" button           | `nav-logout`      |

Go-rod selectors exclusively use `[data-testid="xxx"]` — no Tailwind class selectors (principle 4).

## CI Integration

### New job in `.github/workflows/testing.yml`

```yaml
e2e:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v6
    - uses: actions/setup-go@v6
      with:
        go-version: "^1.26"
        cache: true
    - run: go mod download
    - run: go tool task test:e2e
    - uses: actions/upload-artifact@v7
      if: failure()
      with:
        name: e2e-screenshots
        path: tests/e2e/test-reports/
```

### New task in `taskfile.yaml`

```yaml
test:e2e:
  desc: Run browser-based end-to-end tests
  cmds:
    - go tool gotestsum --raw-command -- go test -v -tags e2e -count=1 ./tests/e2e/...
```

The `e2e` job runs in parallel with `race` and `bdd`. No explicit Docker setup in job steps — testcontainers-go handles PostgreSQL and Redis internally.

## Go-rod Dependencies

Add to `go.mod`:

- `github.com/go-rod/rod` (v0.116.2, already in go.sum as transitive)
- Chromium browser: go-rod auto-downloads via `rod/lib/launcher` or system-installed `chromium-browser` in CI

## AGENTS.md Additions

New section appended after the existing table, documenting the six golden rules plus location, framework, backend, commands, and screenshots conventions.

The full Go-rod section:

```markdown
## E2E Testing (Go-rod)

| Rule                | Principle                                                                                   |
| ------------------- | ------------------------------------------------------------------------------------------- |
| Port isolation      | `net.Listen("tcp", "127.0.0.1:0")` — never hardcode ports                                   |
| Context isolation   | `browser.MustContext()` per test — incognito session per case                               |
| No hardcoded sleep  | `page.MustWaitRequestIdle()` / `page.MustWaitStable()` — Go-rod auto-wait                   |
| HTMX wait strategy  | Prefer `MustWaitRequestIdle()` after Ajax-triggering actions; wait for injected elements    |
| Dialog handling     | `page.MustHandleDialog()` with goroutine — never `MustAcceptDialog()` alone                 |
| Test identifiers    | `data-testid` attributes on `.templ` elements — no CSS class selectors                      |
| Failure screenshots | `t.Cleanup` + `page.MustScreenshot("test-reports/...")` — artifact upload                   |
| Lifecycle defense   | `run(m)` wrapper in `TestMain` — `os.Exit(run(m))` ensures defer runs                       |
| Programmatic login  | Cookie injection via `page.MustSetCookies()` for CRUD tests; UI login only in login_test.go |
| DB isolation        | `ResetDB(t)` at top of each CRUD `t.Run` to prevent state bleeding                          |
| Location            | `tests/e2e/` with `//go:build e2e` tag                                                      |
| Framework           | testify table-driven, min 3 cases per table, `t.Run` per case                               |
| Backend             | testcontainers (PostgreSQL + Redis) with real migrations                                    |
| Server              | `fiber.App` on dynamic port, not full `cmd/flowbot` binary                                  |
| Commands            | `go tool task test:e2e` local, CI job `e2e` in `testing.yml`                                |
| Screenshots         | `tests/e2e/test-reports/` — gitignored, uploaded as CI artifacts on failure                 |
```

## Testing Strategy

### Test-driven development (TDD)

1. Write test first (red) — define the expected behavior as a failing test
2. Implement the minimum code to pass (green) — add `data-testid` to templates, write test helpers
3. Refactor (refactor) — clean up, follow patterns

### Table-driven pattern

All test functions follow:

```go
func TestXxx(t *testing.T) {
    tests := []struct {
        name string
        // inputs
        // expected outputs
    }{
        {name: "happy path", ...},
        {name: "error case 1", ...},
        {name: "error case 2", ...},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

### No hardcoded sleep (principle 3)

Use Go-rod's built-in auto-wait mechanisms. For HTMX interactions specifically:

- **Preferred**: `page.MustWaitRequestIdle()` / `WaitRequestIdle()` — waits for ALL in-flight network requests (including HTMX Ajax calls) to complete. This is the most reliable strategy for HTMX-driven UIs.
- **Alternative**: `page.MustWaitStable()` — DOM has settled, no new frames. Good for full-page navigations but may pass too early for HTMX partial swaps.
- **Element-based**: `page.MustElement("[data-testid=xxx]")` — auto-waits for an element to appear in DOM. Use for asserting HTMX swap results.
- **Request-level**: `page.MustWaitRequestIdle()` with a 300ms window returns a wait function that blocks until all pending requests settle.

```go
// Pattern: wait for network idle after HTMX-triggered action
page.MustElement(`[data-testid="config-save"]`).MustClick()
page.MustWaitRequestIdle()  // blocks until HTMX Ajax POST completes
// Now assert: new row is visible, etc.
```

Never use `time.Sleep`.

### HTMX wait strategy (principle 3 — extended)

HTMX swaps are Ajax-driven and DOM changes can be subtle. The recommended patterns are:

1. **Network-idle after click** — for actions that trigger HTMX requests:

   ```go
   page.MustElement(`[data-testid="config-edit"]`).MustClick()
   page.MustWaitRequestIdle()
   ```

2. **Element-appearance after swap** — for waiting on specific HTMX-injected content:

   ```go
   page.MustElementR("[data-testid='configs-table'] tr", "some-value")
   ```

3. **Request-idle callback** — for chaining async operations:
   ```go
   wait := page.MustWaitRequestIdle()
   page.MustElement(`[data-testid="config-delete"]`).MustClick()
   wait()
   ```

## Files Modified

| File                                    | Change                                        |
| --------------------------------------- | --------------------------------------------- |
| `.github/workflows/testing.yml`         | Add `e2e` parallel job                        |
| `taskfile.yaml`                         | Add `test:e2e` task                           |
| `AGENTS.md`                             | Add E2E Testing (Go-rod) section              |
| `.gitignore`                            | Add `tests/e2e/test-reports/`                 |
| `go.mod`                                | Add `github.com/go-rod/rod` direct dependency |
| `pkg/views/pages/login.templ`           | Add 5 `data-testid` markers                   |
| `pkg/views/pages/configs.templ`         | Add 2 `data-testid` markers                   |
| `pkg/views/partials/config_table.templ` | Add 1 `data-testid` marker                    |
| `pkg/views/partials/config_row.templ`   | Add 2 `data-testid` markers                   |
| `pkg/views/partials/config_form.templ`  | Add 6 `data-testid` markers                   |
| `pkg/views/layout/base.templ`           | Add 2 `data-testid` markers                   |

## Files Created

| File                            | Purpose                                      |
| ------------------------------- | -------------------------------------------- |
| `tests/e2e/setup_test.go`       | TestMain, testcontainers, Browser, Fiber app |
| `tests/e2e/login_test.go`       | Login flow tests                             |
| `tests/e2e/config_crud_test.go` | Config CRUD tests                            |
