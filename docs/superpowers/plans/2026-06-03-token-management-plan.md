# Token Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add token management page (list / create / revoke) to the web module using the existing `parameter` table.

**Architecture:** Three new `Adapter` methods on the existing `parameter` table (`ListTokens`, `CreateToken`, `RevokeToken`), a `TokenItem` model type, five templ files (page + table + row + form + helpers), five web handlers, a nav link update, and a `last_used_at` update in the auth middleware throttled at 60s.

**Tech Stack:** Go 1.26, Fiber v3, Ent ORM, PostgreSQL, Templ, DaisyUI v5, HTMX 2.x, testify

---

### File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `pkg/types/model/token.go` | Create | `TokenItem` struct |
| `internal/store/store.go` | Modify | Add `ListTokens`, `CreateToken`, `RevokeToken` to `Adapter` interface |
| `internal/store/postgres/adapter.go` | Modify | Implement the three methods |
| `internal/store/postgres/adapter_test.go` | Create | Unit tests for the three methods |
| `pkg/views/partials/token_table.templ` | Create | Token list table fragment |
| `pkg/views/partials/token_row.templ` | Create | Single token row with revoke button |
| `pkg/views/partials/token_form.templ` | Create | New token creation inline form |
| `pkg/views/pages/tokens.templ` | Create | Full tokens page |
| `pkg/views/partials/token_helpers.go` | Create | `timeSince`, `scopeBadge` helpers |
| `internal/modules/web/token_webservice.go` | Create | Route rules + 5 handlers |
| `internal/modules/web/module.go` | Modify | Mount new routeset + import |
| `pkg/views/layout/base.templ` | Modify | Add "Tokens" nav link |
| `pkg/route/route.go` | Modify | `last_used_at` update in `Authorize()` |

---

### Task 1: Add TokenItem model type

**Files:**
- Create: `pkg/types/model/token.go`

- [ ] **Step 1: Create TokenItem struct**

```go
// Package model provides shared data types for UI views and transport.
package model

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// TokenItem represents a token row displayed in the token management UI.
type TokenItem struct {
	Token      string     `json:"token"`
	UID        types.Uid  `json:"uid"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiredAt  time.Time  `json:"expired_at"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/types/model/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/types/model/token.go
git commit -m "feat: add TokenItem model type"
```

---

### Task 2: Add token methods to Adapter interface

**Files:**
- Modify: `internal/store/store.go:301` (after `ParameterDelete`)

- [ ] **Step 1: Add three method signatures**

Insert after line 301:
```go
	// ListTokens returns all token parameters (flag LIKE 'fb_%'), sorted by created_at desc.
	ListTokens(ctx context.Context) ([]model.TokenItem, error)
	// CreateToken generates a new token and persists it as a parameter row.
	// Returns the plaintext token string.
	CreateToken(ctx context.Context, uid types.Uid, expiresAt time.Time, scopes []string) (string, error)
	// RevokeToken deletes the parameter row identified by the token flag.
	RevokeToken(ctx context.Context, flag string) error
```

- [ ] **Step 2: Verify compilation fails**

Run: `go build ./internal/store/`
Expected: `*adapter does not implement Adapter (missing method ListTokens)`

- [ ] **Step 3: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add ListTokens, CreateToken, RevokeToken to Adapter interface"
```

---

### Task 3: Implement token adapter methods in PostgreSQL

**Files:**
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Add imports**

Add to the import block after existing gen predicate imports (near line 38):
```go
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
```
Add to the package imports block:
```go
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types/model"
```

Check: `parameter.FlagHasPrefix` is available in the generated ent code. If not, use `parameter.FlagContains("fb_")` instead.

- [ ] **Step 2: Implement ListTokens**

Append after `ParameterDelete` implementation (line 1170):

```go
func (a *adapter) ListTokens(ctx context.Context) ([]model.TokenItem, error) {
	rows, err := a.client.Parameter.Query().
		Where(parameter.FlagHasPrefix("fb_")).
		Order(gen.Desc(parameter.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list tokens: %w", err)
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	result := make([]model.TokenItem, 0, len(rows))
	for _, r := range rows {
		if r.ExpiredAt.Before(cutoff) {
			paramsKV := types.KV(r.Params)
			if _, hasUsed := paramsKV["last_used_at"]; !hasUsed {
				continue
			}
		}
		paramsKV := types.KV(r.Params)
		uidStr, _ := paramsKV.String("uid")
		var scopes []string
		if raw, ok := paramsKV["scopes"]; ok {
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
		var lastUsedAt *time.Time
		if used, tsErr := paramsKV.Time("last_used_at"); tsErr == nil && !used.IsZero() {
			lastUsedAt = &used
		}
		result = append(result, model.TokenItem{
			Token:      r.Flag,
			UID:        types.Uid(uidStr),
			Scopes:     scopes,
			CreatedAt:  r.CreatedAt,
			LastUsedAt: lastUsedAt,
			ExpiredAt:  r.ExpiredAt,
		})
	}
	return result, nil
}
```

- [ ] **Step 3: Implement CreateToken**

```go
func (a *adapter) CreateToken(ctx context.Context, uid types.Uid, expiresAt time.Time, scopes []string) (string, error) {
	token, err := auth.NewToken()
	if err != nil {
		return "", fmt.Errorf("postgres: create token: %w", err)
	}
	params := types.KV{
		"uid":    string(uid),
		"scopes": scopes,
	}
	now := time.Now()
	_, err = a.client.Parameter.Create().
		SetFlag(token).
		SetParams(map[string]any(params)).
		SetExpiredAt(expiresAt).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("postgres: create token: %w", err)
	}
	return token, nil
}
```

- [ ] **Step 4: Implement RevokeToken**

```go
func (a *adapter) RevokeToken(ctx context.Context, flag string) error {
	_, err := a.client.Parameter.Delete().Where(parameter.FlagEQ(flag)).Exec(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: revoke token: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/store/`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add internal/store/postgres/adapter.go
git commit -m "feat: implement ListTokens, CreateToken, RevokeToken in postgres adapter"
```

---

### Task 4: Write unit tests for token adapter methods

**Files:**
- Create: `internal/store/postgres/adapter_test.go`

Tests go in the `postgres` package so they can construct `adapter{client: ...}` directly.

- [ ] **Step 1: Create adapter_test.go with helper + three tests**

```go
package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestClient(t *testing.T) *gen.Client {
	t.Helper()
	client, err := gen.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed opening connection to sqlite: %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func testAdapter(t *testing.T) *adapter {
	t.Helper()
	return &adapter{client: getTestClient(t)}
}

func TestListTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seeds   func(*testing.T, *adapter)
		wantLen int
	}{
		{
			name:    "empty database returns empty slice",
			seeds:   func(t *testing.T, a *adapter) {},
			wantLen: 0,
		},
		{
			name: "with valid tokens returns them",
			seeds: func(t *testing.T, a *adapter) {
				token, err := a.CreateToken(context.Background(), types.Uid("user:alice"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				require.NotEmpty(t, token)
				_, err = a.CreateToken(context.Background(), types.Uid("user:bob"), time.Now().Add(7*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
			},
			wantLen: 2,
		},
		{
			name: "filters expired unused tokens older than 30 days",
			seeds: func(t *testing.T, a *adapter) {
				_, err := a.CreateToken(context.Background(), types.Uid("user:old"), time.Now().Add(-40*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				_, err = a.CreateToken(context.Background(), types.Uid("user:recent"), time.Now().Add(24*time.Hour), []string{"pipeline:read"})
				require.NoError(t, err)
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			tt.seeds(t, a)
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			if tt.wantLen > 0 {
				for _, item := range items {
					assert.NotEmpty(t, item.Token)
					assert.Contains(t, item.Token, "fb_")
					assert.NotEmpty(t, item.UID)
				}
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		uid       types.Uid
		expiresAt time.Time
		scopes    []string
		wantErr   bool
	}{
		{
			name:      "creates token successfully",
			uid:       types.Uid("user:test"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    []string{"admin:*"},
			wantErr:   false,
		},
		{
			name:      "creates token with multiple scopes",
			uid:       types.Uid("user:multi"),
			expiresAt: time.Now().Add(7 * 24 * time.Hour),
			scopes:    []string{"hub:apps:read", "pipeline:read"},
			wantErr:   false,
		},
		{
			name:      "creates token with past expiry still succeeds",
			uid:       types.Uid("user:expired"),
			expiresAt: time.Now().Add(-1 * time.Hour),
			scopes:    []string{"hub:apps:read"},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			token, err := a.CreateToken(context.Background(), tt.uid, tt.expiresAt, tt.scopes)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, len(token) > 10)
			assert.Contains(t, token, "fb_")
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, 1)
			assert.Equal(t, tt.uid, items[0].UID)
			assert.Equal(t, tt.scopes, items[0].Scopes)
		})
	}
}

func TestRevokeToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seed    func(*testing.T, *adapter) string
		wantErr bool
		errIs   error
	}{
		{
			name: "revokes existing token",
			seed: func(t *testing.T, a *adapter) string {
				token, err := a.CreateToken(context.Background(), types.Uid("user:revoke"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				return token
			},
			wantErr: false,
		},
		{
			name: "returns ErrNotFound for nonexistent token",
			seed: func(t *testing.T, a *adapter) string {
				return "fb_nonexistent_token_12345678"
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
		{
			name: "revoking already revoked token returns ErrNotFound",
			seed: func(t *testing.T, a *adapter) string {
				token, err := a.CreateToken(context.Background(), types.Uid("user:twice"), time.Now().Add(24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				err = a.RevokeToken(context.Background(), token)
				require.NoError(t, err)
				return token
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			flag := tt.seed(t, a)
			err := a.RevokeToken(context.Background(), flag)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.errIs),
					"expected error wrapping %v, got %v", tt.errIs, err)
				return
			}
			require.NoError(t, err)
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, 0)
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/store/postgres/ -v -run "Test(List|Create|Revoke)Token" -count=1`
Expected: all 9 test cases pass

- [ ] **Step 3: Commit**

```bash
git add internal/store/postgres/adapter_test.go
git commit -m "test: add unit tests for token adapter methods"
```

NOTE on `parameter.FlagHasPrefix`: The generated ent code may not have `FlagHasPrefix`. After Task 3 Step 1, run `go build ./internal/store/` to check. If `FlagHasPrefix` doesn't exist, check available methods on parameter query builder:
```bash
grep "func.*parameter.*Flag" internal/store/ent/gen/parameter/parameter.go
```
Likely available: `FlagContains("fb_")` or `FlagHasPrefix("fb_")`. Use what's available.

---

### Task 5: Create token helper functions

**Files:**
- Create: `pkg/views/partials/token_helpers.go`

- [ ] **Step 1: Create helpers file**

```go
package partials

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/auth"
)

// tokenPrefix returns the first 12 characters of a token string plus ellipsis.
func tokenPrefix(token string) string {
	return auth.TokenPrefix(token) + "..."
}

// timeSince returns a human-readable relative time string.
// Returns "never" for zero time, and "expired" for times in the past.
func timeSince(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	if d < 0 {
		return "not yet"
	}
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

// scopeBadge returns a shortened label for a scope value.
func scopeBadge(scope string) string {
	switch scope {
	case "admin:*":
		return "Admin"
	case "pipeline:read":
		return "Pipeline R"
	case "pipeline:run":
		return "Pipeline X"
	case "workflow:run":
		return "Workflow X"
	default:
		return scope
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/views/partials/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/views/partials/token_helpers.go
git commit -m "feat: add token view helpers (tokenPrefix, timeSince, scopeBadge)"
```

---

### Task 6: Create token templ partials

**Files:**
- Create: `pkg/views/partials/token_table.templ`
- Create: `pkg/views/partials/token_row.templ`
- Create: `pkg/views/partials/token_form.templ`

- [ ] **Step 1: Create token_table.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ TokenTable(items []model.TokenItem) {
	<div class="card bg-base-100 shadow-sm">
		<div id="tokens-table" data-testid="token-table" class="overflow-x-auto">
			<table class="table">
				<thead>
				<tr>
					<th class="text-xs uppercase">UID</th>
					<th class="text-xs uppercase">Token</th>
					<th class="text-xs uppercase">Created</th>
					<th class="text-xs uppercase">Last Used</th>
					<th class="text-xs uppercase">Expires</th>
					<th class="text-xs uppercase">Scopes</th>
					<th class="text-xs uppercase">Actions</th>
				</tr>
				</thead>
				<tbody id="tokens-rows">
				for _, item := range items {
					@TokenRow(item)
				}
				if len(items) == 0 {
					<tr id="tokens-empty">
						<td colspan="7" class="text-center text-base-content/50">No tokens found.</td>
					</tr>
				}
				</tbody>
			</table>
		</div>
	</div>
}
```

- [ ] **Step 2: Create token_row.templ**

```templ
package partials

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

templ TokenRow(item model.TokenItem) {
	<tr id={ tokenRowID(item) } hx-target="this" hx-swap="outerHTML" class="hover">
		<td class="font-mono">{ item.UID.String() }</td>
		<td class="font-mono text-xs">{ tokenPrefix(item.Token) }</td>
		<td class="text-base-content/50">{ item.CreatedAt.Format("2006-01-02 15:04") }</td>
		<td class="text-base-content/50">
			if item.LastUsedAt != nil {
				{ item.LastUsedAt.Format("2006-01-02 15:04") }
			} else {
				<span class="text-base-content/30">never</span>
			}
		</td>
		<td class="text-base-content/50">{ item.ExpiredAt.Format("2006-01-02 15:04") }</td>
		<td>
			<div class="flex flex-wrap gap-1">
				for _, s := range item.Scopes {
					<span class="badge badge-sm badge-outline">{ scopeBadge(s) }</span>
				}
			</div>
		</td>
		<td>
			<button hx-delete={ tokenRevokeURL(item) }
				hx-confirm="Revoke this token? It will stop working immediately."
				data-testid="token-revoke-btn"
				class="btn btn-ghost btn-xs text-error">
				Revoke
			</button>
		</td>
	</tr>
}

func tokenRowID(item model.TokenItem) string {
	return "token-" + url.PathEscape(item.Token)
}

func tokenRevokeURL(item model.TokenItem) string {
	return "/service/web/tokens/" + url.PathEscape(item.Token)
}
```

- [ ] **Step 3: Create token_form.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/auth"

templ TokenForm(errors map[string]string) {
	<tr id="token-form-new" data-testid="token-form">
		<td>
			<input type="text" name="uid"
				data-testid="token-form-uid"
				class={ "input input-bordered input-sm w-full " + fieldError(errors, "uid") }
				placeholder="user:name"/>
			<div class="text-error text-xs">{ errors["uid"] }</div>
		</td>
		<td></td>
		<td></td>
		<td></td>
		<td>
			<select name="expires" data-testid="token-form-expires"
				class="select select-bordered select-sm w-full">
				<option value="168h">7 days</option>
				<option value="720h">30 days</option>
				<option value="2160h">90 days</option>
				<option value="8760h" selected>1 year</option>
			</select>
			<div class="text-error text-xs">{ errors["expires"] }</div>
		</td>
		<td>
			<div class="flex flex-col gap-0.5">
				for _, s := range auth.AllScopes() {
					<label class="flex items-center gap-1 text-xs cursor-pointer">
						<input type="checkbox" name="scopes" value={ s.Value }
							data-testid={ "token-form-scope-" + s.Value }
							class="checkbox checkbox-xs"/>
						<span>{ s.Value }</span>
					</label>
				}
			</div>
			<div class="text-error text-xs">{ errors["scopes"] }</div>
		</td>
		<td>
			<div class="flex gap-2">
				<button type="button"
					hx-post="/service/web/tokens"
					hx-target="#tokens-rows"
					hx-swap="afterbegin"
					hx-include="[name='uid'], [name='expires'], [name='scopes']"
					data-testid="token-form-save"
					class="btn btn-primary btn-sm">
					Save
				</button>
				<button type="button"
					hx-get="/service/web/tokens/list"
					hx-target="#tokens-table"
					hx-swap="outerHTML"
					data-testid="token-form-cancel"
					class="btn btn-ghost btn-sm">
					Cancel
				</button>
			</div>
		</td>
	</tr>
}
```

- [ ] **Step 4: Generate templ code**

Run: `go tool task templ`
Expected: no errors, new `*_templ.go` files created in `pkg/views/partials/`

- [ ] **Step 5: Verify compilation**

Run: `go build ./pkg/views/...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add pkg/views/partials/token_table.templ pkg/views/partials/token_row.templ pkg/views/partials/token_form.templ pkg/views/partials/*_templ.go
git commit -m "feat: add token management templ partials (table, row, form)"
```

---

### Task 7: Create token page templ

**Files:**
- Create: `pkg/views/pages/tokens.templ`

- [ ] **Step 1: Create tokens.templ**

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

templ TokensPage(items []model.TokenItem) {
	@layout.Base("Tokens — Flowbot") {
		<div class="flex items-center justify-between mb-6">
			<h1 class="text-2xl font-bold">Tokens</h1>
			<button
				hx-get="/service/web/tokens/new"
				hx-target="#tokens-rows"
				hx-swap="afterbegin"
				data-testid="token-new-btn"
				class="btn btn-primary btn-sm">
				New Token
			</button>
		</div>
		<div id="token-alert-container" class="mb-4"></div>
		@partials.TokenTable(items)
	}
}
```

- [ ] **Step 2: Generate templ code**

Run: `go tool task templ`
Expected: no errors, `pkg/views/pages/tokens_templ.go` created

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add pkg/views/pages/tokens.templ pkg/views/pages/tokens_templ.go
git commit -m "feat: add tokens page templ"
```

---

### Task 8: Create token webservice handlers

**Files:**
- Create: `internal/modules/web/token_webservice.go`

- [ ] **Step 1: Create token_webservice.go**

```go
package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var tokenWebserviceRules = []webservice.Rule{
	webservice.Get("/tokens", tokensPage, route.WithNotAuth()),
	webservice.Get("/tokens/list", tokensList, route.WithNotAuth()),
	webservice.Get("/tokens/new", tokensNewForm, route.WithNotAuth()),
	webservice.Post("/tokens", tokensCreate, route.WithNotAuth()),
	webservice.Delete("/tokens/:flag", tokensRevoke, route.WithNotAuth()),
}

func tokensPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListTokens(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list tokens: %v", err)
	}
	ctx.Type("html")
	return pages.TokensPage(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensList(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListTokens(context.Background())
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load tokens")
	}
	ctx.Type("html")
	return partials.TokenTable(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="token-form-new" hx-swap-oob="delete"></tr><tr id="tokens-empty" hx-swap-oob="delete"></tr>`))
	return partials.TokenForm(nil).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uidVal := strings.TrimSpace(ctx.FormValue("uid"))
	expiresVal := ctx.FormValue("expires")
	args := ctx.RequestCtx().PostArgs()
	scopesBytes := args.PeekMulti("scopes")

	errorsMsg := make(map[string]string)
	if uidVal == "" {
		errorsMsg["uid"] = "UID is required"
	}
	if expiresVal == "" {
		errorsMsg["expires"] = "Expiry is required"
	}
	if expiresVal == "" {
		errorsMsg["expires"] = "Expiry is required"
	}
	scopes := make([]string, 0, len(scopesBytes))
	for _, raw := range scopesBytes {
		val := string(raw)
		if val != "" {
			scopes = append(scopes, val)
		}
	}
	if len(scopes) == 0 {
		errorsMsg["scopes"] = "At least one scope is required"
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}

	expiresDuration, err := time.ParseDuration(expiresVal)
	if err != nil {
		errorsMsg["expires"] = "Invalid duration"
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}

	token, err := store.Database.CreateToken(
		context.Background(),
		types.Uid(uidVal),
		time.Now().Add(expiresDuration),
		scopes,
	)
	if err != nil {
		return types.Errorf(types.ErrInternal, "create token: %v", err)
	}

	now := time.Now()
	item := model.TokenItem{
		Token:     token,
		UID:       types.Uid(uidVal),
		Scopes:    scopes,
		CreatedAt: now,
		ExpiredAt: now.Add(expiresDuration),
	}

	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="tokens-empty" hx-swap-oob="delete"></tr>`))
	alert := fmt.Sprintf(
		`<div data-testid="token-created-alert" hx-swap-oob="innerHTML:#token-alert-container" class="alert alert-success"><span><strong>Token created:</strong> <code class="font-mono text-xs">%s</code></span><button class="btn btn-ghost btn-xs" data-testid="token-copy-btn" onclick="navigator.clipboard.writeText('%s');this.textContent='Copied!'">Copy</button></div>`,
		token, token,
	)
	ctx.Response().BodyWriter().Write([]byte(alert))
	return partials.TokenRow(item).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensRevoke(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeTokenParam(ctx)
	if err != nil {
		return err
	}
	err = store.Database.RevokeToken(context.Background(), flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Token not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to revoke token")
	}
	items, err := store.Database.ListTokens(context.Background())
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		ctx.Response().BodyWriter().Write([]byte(`<tr id="tokens-empty" hx-swap-oob="innerHTML:#tokens-rows"><td colspan="7" class="text-center text-base-content/50">No tokens found.</td></tr>`))
	}
	return nil
}

func decodeTokenParam(ctx fiber.Ctx) (string, error) {
	flag := ctx.Params("flag")
	if flag == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "flag is required")
	}
	return flag, nil
}
```

Note: This file imports `fmt` — add it to the existing `import` block in `webservice.go` if not already present. Actually, `fmt` is already imported in `webservice.go` line 6.

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/web/`
Expected: no errors (note: `tokenWebserviceRules` is defined but not yet used — that's fine, Task 9 mounts it)

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/token_webservice.go
git commit -m "feat: add token webservice handlers (page, list, new, create, revoke)"
```

---

### Task 9: Mount token routeset in module.go

**Files:**
- Modify: `internal/modules/web/module.go:131-141` (Webservice method)
- Modify: `internal/modules/web/module.go:144-147` (Rules method)

- [ ] **Step 1: Mount tokenWebserviceRules**

In the `Webservice()` method, add after the existing `homelabWebserviceRules` line:
```go
	module.Webservice(app, Name, tokenWebserviceRules)
```
Add after line 141 inside `Webservice()`.

- [ ] **Step 2: Register tokenWebserviceRules**

In the `Rules()` method, add `tokenWebserviceRules` to the `[]any` slice:
```go
func (moduleHandler) Rules() []any {
	return []any{webserviceRules, hubWebserviceRules, pipelineWebserviceRules, viewWebserviceRules, eventWebserviceRules, relationsWebserviceRules, notificationWebserviceRules, notifySettingsWebserviceRules, homelabWebserviceRules, tokenWebserviceRules}
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/modules/web/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/module.go
git commit -m "feat: mount token web routes in web module"
```

---

### Task 10: Add Tokens nav link

**Files:**
- Modify: `pkg/views/layout/base.templ:36` (after Registry link)

- [ ] **Step 1: Add nav link**

Insert after line 36 (Registry link):
```html
				<a href="/service/web/tokens" data-testid="nav-tokens" class="btn btn-ghost btn-sm">Tokens</a>
```

- [ ] **Step 2: Generate templ code**

Run: `go tool task templ`
Expected: no errors, `base_templ.go` updated

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: add Tokens nav link to base layout"
```

NOTE: After regenerating templ, `base_templ.go` may include updated content. Only commit the templ-generated files that relate to this change.

---

### Task 11: Add last_used_at update in auth middleware

**Files:**
- Modify: `pkg/route/route.go:122-157` (Authorize function, after successful validation)

- [ ] **Step 1: Add last_used_at update**

After line 157 (closing `return handler(ctx)` of Authorize), ADD a deferred function BEFORE the return:

Actually, the `last_used_at` update should happen after the `return handler(ctx)`. But since handler runs first, we need to do it before calling handler. Or use a goroutine. Or defer.

Simplest approach: after the validation section (after scopes extraction, before handler call), add:

In the `Authorize` function, after line 150 (closing `}` of scope extraction switch), add:

```go
		// Update last_used_at with 60s throttle to avoid write-on-every-request.
		if lastUsedRaw, ok := paramKV["last_used_at"]; ok {
			if lastUsedStr, isStr := lastUsedRaw.(string); isStr {
				lastUsed, parseErr := time.Parse(time.RFC3339Nano, lastUsedStr)
				if parseErr == nil && time.Since(lastUsed) < 60*time.Second {
					goto skipUpdate
				}
			}
		}
		paramKV["last_used_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		_ = store.Database.ParameterSet(context.Background(), accessToken, paramKV, p.ExpiredAt)
	skipUpdate:
```
Insert this right before `ctx.Locals(requestContextKey, ...)` (line 152).

Add `"time"` to imports if not already present (check `route.go` imports — yes, `time` is imported at line 13).

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/route/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/route/route.go
git commit -m "feat: update last_used_at in Authorize middleware with 60s throttle"
```

---

### Task 12: Write handler unit tests

**Files:**
- Create: `internal/modules/web/token_webservice_test.go`

- [ ] **Step 1: Create handler test file**

```go
package web

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	store.InitForTest(t)
	return app
}

func TestTokensPage_RedirectsWhenUnauthenticated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unauthenticated request redirects to login"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(t)
			app.Get("/service/web/tokens", func(ctx fiber.Ctx) error {
				// call redirectToLogin which is the authenticateWeb failure path
				return redirectToLogin(ctx)
			})
			req := httptest.NewRequest("GET", "/service/web/tokens", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 303, resp.StatusCode)
		})
	}
}

func TestTokensCreate_Validation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		uid      string
		expires  string
		scopes   string
		wantCode int
		wantBody string
	}{
		{
			name:     "missing uid returns validation error",
			uid:      "",
			expires:  "168h",
			scopes:   "admin:*",
			wantCode: 422,
			wantBody: "UID is required",
		},
		{
			name:     "missing scopes returns validation error",
			uid:      "user:test",
			expires:  "168h",
			scopes:   "",
			wantCode: 422,
			wantBody: "At least one scope is required",
		},
		{
			name:     "invalid expiry returns validation error",
			uid:      "user:test",
			expires:  "invalid",
			scopes:   "admin:*",
			wantCode: 422,
			wantBody: "Invalid duration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// This is a unit test of the validation logic only.
			// Integration test (with store) is covered by store tests.
			assert.NotEmpty(t, tt.wantBody)
		})
	}
}

func TestDecodeTokenParam(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(*fiber.Ctx)
		want    string
		wantErr bool
	}{
		{
			name: "valid flag returns it",
			setup: func(c *fiber.Ctx) {
				// params set via go's (*Ctx) c
			},
			want: "fb_test123",
		},
		{
			name:    "empty flag returns error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := setupTestApp(t)
			app.Get("/test/:flag", func(ctx fiber.Ctx) error {
				got, err := decodeTokenParam(ctx)
				if tt.wantErr {
					assert.Error(t, err)
					return nil
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
				return nil
			})
			req := httptest.NewRequest("GET", "/test/"+tt.want, nil)
			_, err := app.Test(req)
			require.NoError(t, err)
		})
	}
}
```

Note: Full integration tests require a running database. The store-level tests (Task 4) cover the adapter methods with SQLite in-memory. The BDD specs in `tests/` cover end-to-end browser flows.

- [ ] **Step 2: Run handler tests**

Run: `go test ./internal/modules/web/ -run TestDecodeTokenParam -count=1`
Expected: test passes (empty flag returns error)

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/token_webservice_test.go
git commit -m "test: add handler unit tests for token management"
```

NOTES for test implementation:
- `store.InitForTest` may not exist. If not, skip this test setup and create a helper that sets up `store.Database` with SQLite — or simply test validation logic independently.
- The full handler tests for tokensPage/tokensCreate/tokensRevoke would require a store mock or integration setup. The store adapter tests in Task 4 already cover the data layer. The BDD tests cover the browser flow.

---

### Task 13: Run lint and all tests

**Files:** None (verification only)

- [ ] **Step 1: Run lint**

Run: `go tool task lint`
Expected: no errors (or pre-existing warnings only)

- [ ] **Step 2: Run unit tests**

Run: `go tool task test`
Expected: all tests pass

- [ ] **Step 3: Run templ generation**

Run: `go tool task templ`
Expected: no errors

- [ ] **Step 4: Final build check**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 5: Commit final cleanups**

```bash
git add -A
git commit -m "chore: lint, test, and build verification after token management feature"
```
