# Web Module Login Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add username/password login page to the web module, reusing existing token-based auth infrastructure.

**Architecture:** All web routes use `route.WithNotAuth()` to bypass the JSON-error-returning Authorize middleware. A new internal `authenticateWeb()` function reads `accessToken` from the cookie, validates against the `parameters` DB table, sets `RequestContext` in Locals, and redirects to login on failure. Login handlers read credentials from `flowbot.yaml` config, generate a token via `auth.NewToken()`, store it in DB via `ParameterSet()`, and set an HttpOnly cookie.

**Tech Stack:** Go, Fiber v3, Templ, HTMX, Tailwind CSS, existing ent/PostgreSQL store

---

### Task 1: Add AuthConfig to Module Config & Handler

**Files:**
- Modify: `internal/modules/web/module.go:27-50`

- [ ] **Step 1: Add AuthConfig struct and update configType**

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

- [ ] **Step 2: Add authConfig field to moduleHandler and store parsed config**

```go
type moduleHandler struct {
	initialized bool
	authConfig  AuthConfig
	module.Base
}
```

- [ ] **Step 3: Store parsed auth config in Init()**

Add after `handler.initialized = true`:
```go
handler.authConfig = config.Auth
```

- [ ] **Step 4: Add a getter for authConfig from handler**

At the bottom of `module.go`, add:
```go
func authConfig() AuthConfig {
	return handler.authConfig
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/module.go
git commit -m "feat(web): add AuthConfig to module handler"
```

---

### Task 2: Create Login Templ Template

**Files:**
- Create: `pkg/views/pages/login.templ`

- [ ] **Step 1: Write the login template**

```templ
// Package pages provides full-page Templ views.
package pages

import "github.com/flowline-io/flowbot/pkg/views/layout"

templ LoginPage(nextURL string, errorMsg string) {
	@layout.Base("Flowbot — Login") {
		<form hx-post="/service/web/login"
			hx-target="this"
			hx-swap="outerHTML"
			class="max-w-sm mx-auto mt-20 bg-white rounded-lg shadow-sm border border-gray-200 p-6">
			<h1 class="text-xl font-semibold text-gray-900 mb-6 text-center">Flowbot</h1>
			<input type="hidden" name="next" value={ nextURL }/>
			<div class="mb-4">
				<label for="username" class="block text-sm font-medium text-gray-700 mb-1">Username</label>
				<input type="text" id="username" name="username" required autocomplete="username"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"/>
			</div>
			<div class="mb-4">
				<label for="password" class="block text-sm font-medium text-gray-700 mb-1">Password</label>
				<input type="password" id="password" name="password" required autocomplete="current-password"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"/>
			</div>
			if errorMsg != "" {
				<p class="text-red-500 text-sm mb-4">{ errorMsg }</p>
			}
			<button type="submit"
				class="w-full bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700">
				Login
			</button>
		</form>
	}
}
```

- [ ] **Step 2: Generate Go code from template**

Run: `go tool task templ`
Expected: `pkg/views/pages/login_templ.go` created without errors

- [ ] **Step 3: Commit**

```bash
git add pkg/views/pages/login.templ pkg/views/pages/login_templ.go
git commit -m "feat(web): add login page Templ template"
```

---

### Task 3: Add Login/Logout Handlers + Update Auth in webservice.go

**Files:**
- Modify: `internal/modules/web/webservice.go`

- [ ] **Step 1: Add new imports**

Add to existing imports:
```go
import (
	// ... existing imports
	"time"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
)
```

- [ ] **Step 2: Add login/logout routes to webserviceRules with WithNotAuth option**

Replace the existing `webserviceRules` declaration:
```go
var webserviceRules = []webservice.Rule{
	webservice.Get("/login", loginPage, route.WithNotAuth()),
	webservice.Post("/login", loginSubmit, route.WithNotAuth()),
	webservice.Post("/logout", logout, route.WithNotAuth()),
	webservice.Get("/configs", configsPage, route.WithNotAuth()),
	webservice.Get("/configs/list", listConfigs, route.WithNotAuth()),
	webservice.Get("/configs/new", newConfigForm, route.WithNotAuth()),
	webservice.Post("/configs", createConfig, route.WithNotAuth()),
	webservice.Get("/configs/:uid/:topic/:key", getConfig, route.WithNotAuth()),
	webservice.Get("/configs/:uid/:topic/:key/edit", editConfigForm, route.WithNotAuth()),
	webservice.Put("/configs/:uid/:topic/:key", updateConfig, route.WithNotAuth()),
	webservice.Delete("/configs/:uid/:topic/:key", deleteConfig, route.WithNotAuth()),
}
```

- [ ] **Step 3: Replace requireAuth with authenticateWeb and isAuthenticated helpers**

Replace the existing `requireAuth` function with these three functions:
```go
// isAuthenticated reads the accessToken cookie, validates it against the database,
// and sets RequestContext in Locals on success. Returns true if the user is authenticated.
// Does NOT redirect — callers that redirect use authenticateWeb.
func isAuthenticated(ctx fiber.Ctx) bool {
	if route.GetRequestContext(ctx) != nil {
		return true
	}
	token := ctx.Cookies("accessToken")
	if token == "" {
		return false
	}
	p, err := store.Database.ParameterGet(context.Background(), token)
	if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
		return false
	}
	paramKV := types.KV(p.Params)
	uidStr, _ := paramKV.String("uid")
	uid := types.Uid(uidStr)
	if uid.IsZero() {
		return false
	}
	topic, _ := paramKV.String("topic")
	var scopes []string
	if raw, ok := paramKV["scopes"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					scopes = append(scopes, s)
				}
			}
		case []string:
			scopes = v
		}
	}
	ctx.Locals("route:ctx", &route.RequestContext{
		UID:    uid,
		Topic:  topic,
		Param:  paramKV,
		Scopes: scopes,
	})
	return true
}

// authenticateWeb checks authentication and redirects to login on failure.
// Use in protected page handlers.
func authenticateWeb(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		return nil
	}
	return redirectToLogin(ctx)
}

// redirectToLogin sends a 302 redirect to the login page with the current URL as ?next=.
func redirectToLogin(ctx fiber.Ctx) error {
	next := string(ctx.Request().URI().RequestURI())
	nextEncoded := url.QueryEscape(next)
	return ctx.Redirect().To("/service/web/login?next=" + nextEncoded)
}
```

- [ ] **Step 4: Update all existing handlers to call authenticateWeb instead of requireAuth**

Replace all `if err := requireAuth(ctx); err != nil { return err }` with:
```go
if err := authenticateWeb(ctx); err != nil {
	return err
}
```

This affects: `configsPage`, `listConfigs`, `getConfig`, `newConfigForm`, `createConfig`, `editConfigForm`, `updateConfig`, `deleteConfig`.

- [ ] **Step 5: Add loginPage handler**

```go
func loginPage(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		next := ctx.Query("next", "/service/web/configs")
		return ctx.Redirect().To(next)
	}
	next := ctx.Query("next", "")
	ctx.Type("html")
	return pages.LoginPage(next, "").Render(context.Background(), ctx.Response().BodyWriter())
}
```

- [ ] **Step 6: Add loginSubmit handler**

```go
func loginSubmit(ctx fiber.Ctx) error {
	username := ctx.FormValue("username")
	password := ctx.FormValue("password")
	next := ctx.FormValue("next")
	cfg := authConfig()
	if username == "" || username != cfg.Username || password != cfg.Password {
		ctx.Type("html")
		return pages.LoginPage(next, "Invalid username or password").Render(context.Background(), ctx.Response().BodyWriter())
	}
	token, err := auth.NewToken()
	if err != nil {
		flog.Error("failed to generate token: %v", err)
		ctx.Type("html")
		return pages.LoginPage(next, "Internal error").Render(context.Background(), ctx.Response().BodyWriter())
	}
	uid := types.Uid("user-" + username)
	params := types.KV{
		"uid":    string(uid),
		"topic":  "web",
		"scopes": []string{"admin:*"},
	}
	expiredAt := time.Now().Add(24 * time.Hour)
	if err := store.Database.ParameterSet(context.Background(), token, params, expiredAt); err != nil {
		flog.Error("failed to store token: %v", err)
		ctx.Type("html")
		return pages.LoginPage(next, "Internal error").Render(context.Background(), ctx.Response().BodyWriter())
	}
	cookie := &fiber.Cookie{}
	cookie.SetName("accessToken")
	cookie.SetValue(token)
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(true)
	cookie.SetSameSite("Lax")
	cookie.SetPath("/")
	cookie.SetMaxAge(86400)
	ctx.Cookie(cookie)
	if next == "" || !strings.HasPrefix(next, "/") || strings.Contains(next, "//") || strings.Contains(next, ":") {
		next = "/service/web/configs"
	}
	ctx.Set("HX-Redirect", next)
	return nil
}
```

- [ ] **Step 7: Add logout handler**

```go
func logout(ctx fiber.Ctx) error {
	token := ctx.Cookies("accessToken")
	if token != "" {
		if err := store.Database.ParameterDelete(context.Background(), token); err != nil {
			flog.Error("failed to delete token on logout: %v", err)
		}
	}
	cookie := &fiber.Cookie{}
	cookie.SetName("accessToken")
	cookie.SetHTTPOnly(true)
	cookie.SetPath("/")
	cookie.SetMaxAge(0)
	ctx.Cookie(cookie)
	return ctx.Redirect().To("/service/web/login")
}
```

- [ ] **Step 8: Add strings import if not present**

Verify `strings` is in imports (used by `strings.HasPrefix`, `strings.Contains`). Add if missing.

- [ ] **Step 9: Run lint to check compilation**

```bash
go tool task lint
```
Expected: no errors from web module files.

- [ ] **Step 10: Commit**

```bash
git add internal/modules/web/webservice.go
git commit -m "feat(web): add login/logout handlers, refactor auth for web routes"
```

---

### Task 4: Update testStore and Add Tests

**Files:**
- Modify: `internal/modules/web/test_helper_test.go`
- Modify: `internal/modules/web/module_test.go`

- [ ] **Step 1: Add ParameterSet and ParameterDelete to testStore**

Add fields to `testStore`:
```go
type testStore struct {
	store.Adapter
	configs      []model.ConfigItem
	configErr    error
	setConfigFn  func(uid types.Uid, topic, key string, value types.KV) error
	getConfigFn  func(uid types.Uid, topic, key string) (types.KV, error)
	delConfigFn  func(uid types.Uid, topic, key string) error
	paramGetFn   func(ctx context.Context, flag string) (gen.Parameter, error)
	paramSetFn   func(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	paramDelFn   func(ctx context.Context, flag string) error
}
```

Update `ParameterGet` to use `paramGetFn` if set:
```go
func (s *testStore) ParameterGet(ctx context.Context, flag string) (gen.Parameter, error) {
	if s.paramGetFn != nil {
		return s.paramGetFn(ctx, flag)
	}
	return gen.Parameter{
		ID:        1,
		Flag:      "test-token",
		Params:    map[string]any{"uid": "testuser", "topic": "test"},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}
```

Add new methods:
```go
func (s *testStore) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	if s.paramSetFn != nil {
		return s.paramSetFn(ctx, flag, params, expiredAt)
	}
	return nil
}
func (s *testStore) ParameterDelete(ctx context.Context, flag string) error {
	if s.paramDelFn != nil {
		return s.paramDelFn(ctx, flag)
	}
	return nil
}
```

- [ ] **Step 2: Update setupTestApp to initialize auth config for tests**

```go
func setupTestApp() (*fiber.App, *testStore) {
	ts := &testStore{}
	store.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts
}
```

- [ ] **Step 3: Update existing configs tests to use cookie auth**

Replace all `?accessToken=test` query params with cookie. Example for `TestConfigsPage`:
```go
req := httptest.NewRequest(http.MethodGet, "/service/web/configs", nil)
req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
```

Apply this pattern to ALL existing test functions: `TestConfigsPage`, `TestListConfigs`, `TestDeleteConfig`, `TestGetConfig`.

- [ ] **Step 4: Add TestLoginPage**

```go
func TestLoginPage(t *testing.T) {
	tests := []struct {
		name          string
		cookieToken   string
		queryNext     string
		wantStatus    int
		wantLocation  string
		wantContains  string
	}{
		{
			name:         "no cookie renders login form",
			wantStatus:   http.StatusOK,
			wantContains: "Login",
		},
		{
			name:         "with valid cookie redirects to configs",
			cookieToken:  "valid-test-token",
			wantStatus:   http.StatusFound,
			wantLocation: "/service/web/configs",
		},
		{
			name:         "with valid cookie and next param redirects to next",
			cookieToken:  "valid-test-token",
			queryNext:    "/service/web/configs?foo=bar",
			wantStatus:   http.StatusFound,
			wantLocation: "/service/web/configs?foo=bar",
		},
		{
			name:   "empty next defaults to configs",
			queryNext: "",
			wantStatus:   http.StatusOK,
			wantContains: "Login",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			url := "/service/web/login"
			if tt.queryNext != "" {
				url += "?next=" + url.QueryEscape(tt.queryNext)
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantLocation != "" {
				loc := resp.Header.Get("Location")
				if loc != tt.wantLocation {
					t.Errorf("want location %q, got %q", tt.wantLocation, loc)
				}
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}
```

- [ ] **Step 5: Add TestLoginSubmit**

```go
func TestLoginSubmit(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		nextVal      string
		wantStatus   int
		wantContains string
		wantHXRedirect string
		wantCookie   bool
	}{
		{
			name:         "correct credentials returns redirect",
			username:     "admin",
			password:     "admin",
			wantStatus:   http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookie:   true,
		},
		{
			name:         "wrong password shows error",
			username:     "admin",
			password:     "wrong",
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
			wantCookie:   false,
		},
		{
			name:         "wrong username shows error",
			username:     "nobody",
			password:     "admin",
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
			wantCookie:   false,
		},
		{
			name:         "empty username shows error",
			username:     "",
			password:     "admin",
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
			wantCookie:   false,
		},
		{
			name:         "correct credentials with valid next redirects",
			username:     "admin",
			password:     "admin",
			nextVal:      "/service/web/configs?page=2",
			wantStatus:   http.StatusOK,
			wantHXRedirect: "/service/web/configs?page=2",
			wantCookie:   true,
		},
		{
			name:         "correct credentials with external next falls back",
			username:     "admin",
			password:     "admin",
			nextVal:      "https://evil.com",
			wantStatus:   http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookie:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			var storedFlag string
			var storedParams types.KV
			ts.paramSetFn = func(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
				storedFlag = flag
				storedParams = params
				return nil
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			form := url.Values{}
			form.Set("username", tt.username)
			form.Set("password", tt.password)
			if tt.nextVal != "" {
				form.Set("next", tt.nextVal)
			}
			req := httptest.NewRequest(http.MethodPost, "/service/web/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantHXRedirect != "" {
				got := resp.Header.Get("HX-Redirect")
				if got != tt.wantHXRedirect {
					t.Errorf("want HX-Redirect %q, got %q", tt.wantHXRedirect, got)
				}
			}
			if tt.wantCookie {
				found := false
				for _, c := range resp.Header.Values("Set-Cookie") {
					if strings.Contains(c, "accessToken=") {
						found = true
					}
				}
				if !found {
					t.Error("expected accessToken cookie to be set")
				}
				if storedFlag == "" {
					t.Error("expected token to be stored in DB")
				}
				if uid, _ := storedParams.String("uid"); uid != "user-admin" {
					t.Errorf("expected uid user-admin, got %q", uid)
				}
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}
```

- [ ] **Step 6: Add TestLogout**

```go
func TestLogout(t *testing.T) {
	tests := []struct {
		name       string
		cookieToken string
		wantStatus int
		wantLocation string
		wantCookieClear bool
	}{
		{
			name:       "logout with valid cookie clears it",
			cookieToken: "some-token",
			wantStatus: http.StatusFound,
			wantLocation: "/service/web/login",
			wantCookieClear: true,
		},
		{
			name:       "logout without cookie still redirects",
			cookieToken: "",
			wantStatus: http.StatusFound,
			wantLocation: "/service/web/login",
			wantCookieClear: false,
		},
		{
			name:       "logout deletes parameter from DB",
			cookieToken: "token-to-delete",
			wantStatus: http.StatusFound,
			wantLocation: "/service/web/login",
			wantCookieClear: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			deletedFlag := ""
			ts.paramDelFn = func(ctx context.Context, flag string) error {
				deletedFlag = flag
				return nil
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodPost, "/service/web/logout", nil)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantLocation != "" {
				loc := resp.Header.Get("Location")
				if loc != tt.wantLocation {
					t.Errorf("want location %q, got %q", tt.wantLocation, loc)
				}
			}
			if tt.wantCookieClear {
				if tt.cookieToken != "" && deletedFlag == "" {
					t.Error("expected token to be deleted from DB")
				}
			}
		})
	}
}
```

- [ ] **Step 7: Add TestAuthenticateWebRedirect**

```go
func TestAuthenticateWebRedirect(t *testing.T) {
	tests := []struct {
		name         string
		cookieToken  string
		paramGetFn   func(ctx context.Context, flag string) (gen.Parameter, error)
		wantStatus   int
		wantBodyContains string
	}{
		{
			name:        "valid token allows access to configs",
			cookieToken: "valid-token",
			paramGetFn: func(ctx context.Context, flag string) (gen.Parameter, error) {
				return gen.Parameter{
					ID:        1,
					Flag:      flag,
					Params:    map[string]any{"uid": "user-admin", "topic": "web", "scopes": []any{"admin:*"}},
					ExpiredAt: time.Now().Add(time.Hour),
				}, nil
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "Configs",
		},
		{
			name:             "no cookie redirects to login",
			cookieToken:      "",
			paramGetFn:       nil, // not called, short-circuited by empty cookie check
			wantStatus:       http.StatusFound,
			wantBodyContains: "",
		},
		{
			name:        "invalid token in cookie redirects to login",
			cookieToken: "bad-token",
			paramGetFn: func(ctx context.Context, flag string) (gen.Parameter, error) {
				return gen.Parameter{}, types.ErrNotFound
			},
			wantStatus:       http.StatusFound,
			wantBodyContains: "",
		},
		{
			name:        "expired token redirects to login",
			cookieToken: "expired-token",
			paramGetFn: func(ctx context.Context, flag string) (gen.Parameter, error) {
				return gen.Parameter{
					ID:        2,
					Flag:      flag,
					Params:    map[string]any{"uid": "user-admin", "topic": "web", "scopes": []any{"admin:*"}},
					ExpiredAt: time.Now().Add(-time.Hour),
				}, nil
			},
			wantStatus:       http.StatusFound,
			wantBodyContains: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.paramGetFn != nil {
				ts.paramGetFn = tt.paramGetFn
			} else {
				ts.paramGetFn = func(ctx context.Context, flag string) (gen.Parameter, error) {
					return gen.Parameter{}, types.ErrNotFound
				}
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs", nil)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBodyContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBodyContains) {
					t.Errorf("want body containing %q", tt.wantBodyContains)
				}
			}
		})
	}
}
```

- [ ] **Step 8: Add missing imports to module_test.go**

Add `time` and `net/url` to imports. Add `"github.com/flowline-io/flowbot/internal/store/ent/gen"` if not already present.

- [ ] **Step 9: Run tests**

```bash
go test ./internal/modules/web/ -v -count=1
```
Expected: all tests PASS

- [ ] **Step 10: Run lint**

```bash
go tool task lint
```
Expected: no errors

- [ ] **Step 11: Commit**

```bash
git add internal/modules/web/test_helper_test.go internal/modules/web/module_test.go
git commit -m "test(web): add login, logout, and auth redirect tests"
```

---

### Task 5: Update flowbot.yaml Configuration

**Files:**
- Modify: `flowbot.yaml`

- [ ] **Step 1: Add auth section under web config**

In `flowbot.yaml`, find the `web:` section and update it:
```yaml
  - name: web
    enabled: true
    auth:
      username: "admin"
      password: "admin"
```

If the web section uses the short format `web.enabled: true`, convert it to the expanded `bots:` list format matching how other modules are configured. Check the existing bots section first.

- [ ] **Step 2: Run lint to verify**

```bash
go tool task lint
```

- [ ] **Step 3: Commit**

```bash
git add flowbot.yaml
git commit -m "config: add web auth credentials"
```

---

### Task 6: Run Full Verification

- [ ] **Step 1: Run all unit tests for web module**

```bash
go test ./internal/modules/web/ -v -count=1
```
Expected: all tests PASS

- [ ] **Step 2: Run full project lint**

```bash
go tool task lint
```
Expected: no errors

- [ ] **Step 3: Run all project tests**

```bash
go tool task test
```
Expected: all tests PASS

- [ ] **Step 4: Verify Templ generation is up to date**

```bash
go tool task templ
```
Expected: no changes (already regenerated in Task 2)
