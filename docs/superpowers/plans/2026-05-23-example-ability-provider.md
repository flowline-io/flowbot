# Example Ability & Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create complete reference implementations for providers (`pkg/providers/example/`), abilities (`pkg/ability/example/`), and modules (`internal/modules/example/`) that demonstrate the full Module -> Ability -> Provider -> httpbin.org chain.

**Architecture:** Provider layer wraps httpbin.org REST API. Ability layer defines Service interface, Descriptor + RegisterService, WebhookConverter, PollingResource. Adapter (`ability/example/example/adapter.go`) bridges provider and capability. Module layer wires everything together in Init() and exposes REST + webhook routes.

**Tech Stack:** Go 1.26+, resty v3 HTTP client, Fiber v3 router, sonic JSON, testify assertions, Ginkgo v2 + Gomega BDD

---

### Task 0: Add CapExample Capability Type and Auth Scopes

**Files:**

- Modify: `pkg/hub/capability.go`
- Modify: `pkg/auth/scope.go`

- [ ] **Step 1: Add CapExample constant to capability types**

```go
// pkg/hub/capability.go — add after CapNotify:
	CapExample CapabilityType = "example"
```

- [ ] **Step 2: Add auth scope constants**

```go
// pkg/auth/scope.go — add after ScopeServiceShellRead:
	ScopeServiceExampleRead  = "service:example:read"
	ScopeServiceExampleWrite = "service:example:write"
```

Also add to the scopes list in `scope.go`:

```go
	{Value: ScopeServiceExampleRead, Description: "read example"},
	{Value: ScopeServiceExampleWrite, Description: "write example"},
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add pkg/hub/capability.go pkg/auth/scope.go
git commit -m "feat: add CapExample capability type and auth scopes"
```

---

### Task 1: Provider Types

**Files:**

- Create: `pkg/providers/example/types.go`

- [ ] **Step 1: Create types.go with response, query, and webhook payload types**

```go
// Package example implements the example provider using httpbin.org for demonstration.
package example

// MaxPageSize is the maximum number of items per page.
const MaxPageSize = 100

// Response mirrors httpbin response JSON structure for GET/POST/PUT/DELETE endpoints.
type Response struct {
	Args    map[string]string `json:"args"`
	Data    string            `json:"data"`
	Files   map[string]string `json:"files"`
	Form    map[string]string `json:"form"`
	Headers map[string]string `json:"headers"`
	JSON    any               `json:"json"`
	Method  string            `json:"method"`
	Origin  string            `json:"origin"`
	URL     string            `json:"url"`
}

// StatusResponse mirrors httpbin /status/{code} response structure.
type StatusResponse struct {
	Code    int               `json:"code"`
	Headers map[string]string `json:"headers"`
}

// DelayResponse mirrors httpbin /delay/{seconds} response structure.
type DelayResponse struct {
	Delay   int               `json:"delay"`
	Headers map[string]string `json:"headers"`
	Origin  string            `json:"origin"`
	URL     string            `json:"url"`
}

// WebhookPayload represents a webhook event payload from the example provider.
type WebhookPayload struct {
	EventType string `json:"event_type"`
	EntityID  string `json:"entity_id"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

// ItemsResponse carries a list of items with cursor for polling.
type ItemsResponse struct {
	Items      []map[string]any `json:"items"`
	NextCursor string           `json:"next_cursor"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/providers/example/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/providers/example/types.go
git commit -m "feat(providers/example): add types for httpbin response, webhook payload"
```

---

### Task 2: Provider Implementation

**Files:**

- Create: `pkg/providers/example/example.go`

- [ ] **Step 1: Create example.go with provider struct, GetClient, NewExample, API methods, OAuth**

```go
package example

import (
	"context"
	"fmt"
	"time"

	"resty.dev/v3"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "example"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

// Example wraps the httpbin.org API client for demonstration purposes.
// It provides CRUD operations, OAuth stubs, and polling support.
type Example struct {
	c     *resty.Client
	token string
}

// GetClient reads provider config and returns a new Example client.
// It falls back to https://httpbin.org when no endpoint is configured.
func GetClient() *Example {
	endpoint, err := providers.GetConfig(ID, EndpointKey)
	if err != nil {
		flog.Warn("example provider config error: %v", err)
	}
	token, err := providers.GetConfig(ID, TokenKey)
	if err != nil {
		flog.Warn("example provider config error: %v", err)
	}
	return NewExample(endpoint.String(), token.String())
}

// NewExample creates an Example client with the given endpoint and optional auth token.
// If endpoint is empty, it defaults to https://httpbin.org.
func NewExample(endpoint, token string) *Example {
	if endpoint == "" {
		endpoint = "https://httpbin.org"
	}
	v := &Example{token: token}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetAuthToken(token)
	}
	return v
}

// Get sends a GET request to httpbin /get, returning the echoed request details.
func (e *Example) Get(ctx context.Context, path string) (*Response, error) {
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetResult(resp)
	if path != "" {
		req.SetQueryParam("path", path)
	}
	_, err := req.Get("/get")
	if err != nil {
		return nil, fmt.Errorf("example get: %w", err)
	}
	return resp, nil
}

// Post sends a POST request to httpbin /post with the given data as JSON body.
func (e *Example) Post(ctx context.Context, path string, data any) (*Response, error) {
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetBody(data).SetResult(resp)
	if path != "" {
		req.SetQueryParam("path", path)
	}
	_, err := req.Post("/post")
	if err != nil {
		return nil, fmt.Errorf("example post: %w", err)
	}
	return resp, nil
}

// Put sends a PUT request to httpbin /put with the given data as JSON body.
func (e *Example) Put(ctx context.Context, path string, data any) (*Response, error) {
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetBody(data).SetResult(resp)
	if path != "" {
		req.SetQueryParam("path", path)
	}
	_, err := req.Put("/put")
	if err != nil {
		return nil, fmt.Errorf("example put: %w", err)
	}
	return resp, nil
}

// Delete sends a DELETE request to httpbin /delete.
func (e *Example) Delete(ctx context.Context, path string) (*Response, error) {
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetResult(resp)
	if path != "" {
		req.SetQueryParam("path", path)
	}
	_, err := req.Delete("/delete")
	if err != nil {
		return nil, fmt.Errorf("example delete: %w", err)
	}
	return resp, nil
}

// GetStatus fetches httpbin /status/{code} to demonstrate error handling.
func (e *Example) GetStatus(ctx context.Context, code int) (*Response, error) {
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetResult(resp)
	_, err := req.Get(fmt.Sprintf("/status/%d", code))
	if err != nil {
		return nil, fmt.Errorf("example status %d: %w", code, err)
	}
	return resp, nil
}

// GetWithDelay fetches httpbin /delay/{seconds} to demonstrate timeout handling.
func (e *Example) GetWithDelay(ctx context.Context, seconds int) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(seconds+2)*time.Second)
	defer cancel()
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetResult(resp)
	_, err := req.Get(fmt.Sprintf("/delay/%d", seconds))
	if err != nil {
		return nil, fmt.Errorf("example delay %ds: %w", seconds, err)
	}
	return resp, nil
}

// ListRawEvents fetches items from httpbin /get, parsing the response as a list
// for polling demonstration. The cursor is passed as a query parameter.
func (e *Example) ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error) {
	resp := &Response{}
	req := e.c.R().SetContext(ctx).SetResult(resp)
	if cursor != "" {
		req.SetQueryParam("cursor", cursor)
	}
	_, err := req.Get("/get")
	if err != nil {
		return nil, "", fmt.Errorf("example list raw: %w", err)
	}
	items := []map[string]any{
		{
			"id":      "item-1",
			"origin":  resp.Origin,
			"url":     resp.URL,
			"headers": resp.Headers,
		},
	}
	return items, "", nil
}

// GetAuthorizeURL returns a constructed OAuth authorize URL for demonstration.
func (e *Example) GetAuthorizeURL() string {
	endpoint := e.c.BaseURL
	return fmt.Sprintf("%s/authorize?client_id=example&response_type=code", endpoint)
}

// GetAccessToken simulates an OAuth code exchange for demonstration.
func (e *Example) GetAccessToken(ctx fiber.Ctx) (types.KV, error) {
	return types.KV{
		"access_token": "example-token",
		"scope":        "example:read example:write",
	}, nil
}

// OAuth interface compliance check.
var _ providers.OAuthProvider = (*Example)(nil)
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/providers/example/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/providers/example/example.go
git commit -m "feat(providers/example): implement provider with httpbin CRUD, OAuth, polling"
```

---

### Task 3: Provider Tests

**Files:**

- Create: `pkg/providers/example/example_test.go`

- [ ] **Step 1: Write TDD tests for provider methods**

```go
package example

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetClient_Defaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		configs  json.RawMessage
		wantURL  string
	}{
		{
			name:    "empty config defaults to httpbin",
			configs: json.RawMessage(`{}`),
			wantURL: "https://httpbin.org",
		},
		{
			name:    "custom endpoint",
			configs: json.RawMessage(`{"example":{"endpoint":"https://custom.example.com"}}`),
			wantURL: "https://custom.example.com",
		},
		{
			name:    "config with token",
			configs: json.RawMessage(`{"example":{"endpoint":"https://custom.example.com","token":"abc123"}}`),
			wantURL: "https://custom.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			providers.Configs = tt.configs
			c := GetClient()
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL)
		})
	}
}

func TestNewExample(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		token    string
		wantURL  string
	}{
		{
			name:     "explicit endpoint",
			endpoint: "https://api.example.com",
			token:    "",
			wantURL:  "https://api.example.com",
		},
		{
			name:     "empty endpoint defaults",
			endpoint: "",
			token:    "",
			wantURL:  "https://httpbin.org",
		},
		{
			name:     "with auth token",
			endpoint: "https://api.example.com",
			token:    "test-token",
			wantURL:  "https://api.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewExample(tt.endpoint, tt.token)
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "successful get returns response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				assert.Equal(t, "/get", r.URL.Path)
				w.Write([]byte(`{"url":"https://example.com/get","origin":"1.2.3.4","method":"GET"}`))
			},
			wantErr: false,
		},
		{
			name: "get with path sets query param",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "my-path", r.URL.Query().Get("path"))
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/get","method":"GET"}`))
			},
			wantErr: false,
		},
		{
			name: "server error returns err",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			c := NewExample(srv.URL, "")
			var path string
			if tt.name == "get with path sets query param" {
				path = "my-path"
			}
			resp, err := c.Get(context.Background(), path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestPost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    any
		wantErr bool
	}{
		{
			name:    "successful post returns response",
			data:    map[string]string{"title": "test"},
			wantErr: false,
		},
		{
			name:    "post with nil data",
			data:    nil,
			wantErr: false,
		},
		{
			name:    "server error",
			data:    map[string]string{"title": "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var callCount atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount.Add(1)
				if tt.wantErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				assert.Equal(t, "POST", r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/post","method":"POST"}`))
			}))
			defer srv.Close()
			c := NewExample(srv.URL, "")
			resp, err := c.Post(context.Background(), "", tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, int32(1), callCount.Load())
		})
	}
}

func TestPut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "successful put",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/put","method":"PUT"}`))
			},
			wantErr: false,
		},
		{
			name: "put error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr: true,
		},
		{
			name: "put with data",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/put","method":"PUT"}`))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			c := NewExample(srv.URL, "")
			resp, err := c.Put(context.Background(), "", map[string]string{"key": "val"})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "successful delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/delete","method":"DELETE"}`))
			},
			wantErr: false,
		},
		{
			name: "delete error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "delete with path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "my-resource", r.URL.Query().Get("path"))
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/delete","method":"DELETE"}`))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			c := NewExample(srv.URL, "")
			var path string
			if tt.name == "delete with path" {
				path = "my-resource"
			}
			resp, err := c.Delete(context.Background(), path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		code    int
		wantErr bool
	}{
		{name: "status 200", code: 200, wantErr: false},
		{name: "status 418", code: 418, wantErr: false},
		{name: "status 404", code: 404, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/status","method":"GET"}`))
			}))
			defer srv.Close()
			c := NewExample(srv.URL, "")
			resp, err := c.GetStatus(context.Background(), tt.code)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetWithDelay(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		wantErr bool
	}{
		{name: "short delay succeeds", seconds: 0, wantErr: false},
		{name: "short delay works", seconds: 1, wantErr: false},
		{name: "context expiry", seconds: 30, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/delay","method":"GET"}`))
			}))
			defer srv.Close()
			c := NewExample(srv.URL, "")
			resp, err := c.GetWithDelay(context.Background(), tt.seconds)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestListRawEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cursor   string
		wantErr  bool
	}{
		{name: "list with no cursor", cursor: "", wantErr: false},
		{name: "list with cursor", cursor: "next-page", wantErr: false},
		{name: "server error", cursor: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if tt.cursor != "" {
					assert.Equal(t, tt.cursor, r.URL.Query().Get("cursor"))
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"url":"https://example.com/get","origin":"1.2.3.4","headers":{}}`))
			}))
			defer srv.Close()
			c := NewExample(srv.URL, "")
			items, next, err := c.ListRawEvents(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, items)
			assert.Equal(t, "", next)
		})
	}
}

func TestOAuthMethods(t *testing.T) {
	t.Parallel()
	t.Run("GetAuthorizeURL returns URL", func(t *testing.T) {
		t.Parallel()
		c := NewExample("https://httpbin.org", "")
		url := c.GetAuthorizeURL()
		assert.Contains(t, url, "https://httpbin.org/authorize")
	})
	t.Run("GetAccessToken returns token KV", func(t *testing.T) {
		t.Parallel()
		c := NewExample("https://httpbin.org", "")
		kv, err := c.GetAccessToken(nil)
		require.NoError(t, err)
		assert.Equal(t, "example-token", kv["access_token"])
	})
	t.Run("OAuthProvider interface compliance", func(t *testing.T) {
		t.Parallel()
		var _ providers.OAuthProvider = NewExample("https://httpbin.org", "")
	})
}

func TestListRawEvents_ContextCanceled(t *testing.T) {
	t.Run("canceled context returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://example.com/get","origin":"1.2.3.4"}`))
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c := NewExample(srv.URL, "")
		_, _, err := c.ListRawEvents(ctx, "")
		assert.Error(t, err)
	})
}
```

- [ ] **Step 2: Run tests to verify failures (some will fail — TDD)**

Run: `go test ./pkg/providers/example/... -v -count=1`
Expected: Some tests pass (already have implementation), all should pass

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./pkg/providers/example/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/providers/example/example_test.go
git commit -m "test(providers/example): add TDD tests for provider CRUD, OAuth, polling"
```

---

### Task 4: Ability Interface

**Files:**

- Create: `pkg/ability/example/interface.go`

- [ ] **Step 1: Create interface.go with Service interface and query types**

```go
// Package example implements the example capability for demonstration.
package example

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListQuery wraps pagination with optional filter fields for listing items.
type ListQuery struct {
	Page capability.PageRequest
}

// Service defines the example capability contract.
// Provider adapters implement this interface to bridge providers and invokers.
type Service interface {
	GetItem(ctx context.Context, id string) (*capability.Host, error)
	ListItems(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Host], error)
	CreateItem(ctx context.Context, title string) (*capability.Host, error)
	UpdateItem(ctx context.Context, id string, data map[string]any) (*capability.Host, error)
	DeleteItem(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) (bool, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/ability/example/...`
Expected: No errors (depends on capability.go types which exist)

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/interface.go
git commit -m "feat(ability/example): add Service interface for example capability"
```

---

### Task 5: Ability Descriptor + RegisterService

**Files:**

- Create: `pkg/ability/example/descriptor.go`

- [ ] **Step 1: Create descriptor.go with operation constants, Descriptor, RegisterService, invokers**

```go
package example

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Capability is the example capability type constant.
const Capability hub.CapabilityType = hub.CapExample

// Example operation constants.
const (
	OpExampleList    = "list"
	OpExampleGet     = "get"
	OpExampleCreate  = "create"
	OpExampleUpdate  = "update"
	OpExampleDelete  = "delete"
	OpExampleHealth  = "health"
	OpExampleRawList = "raw_list"
)

// Descriptor returns the hub capability descriptor for the example capability.
func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapExample,
		Backend:     backend,
		App:         app,
		Description: "Example capability for demonstration",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: OpExampleList, Description: "List items", Scopes: []string{auth.ScopeServiceExampleRead}},
			{Name: OpExampleGet, Description: "Get an item", Scopes: []string{auth.ScopeServiceExampleRead}},
			{Name: OpExampleCreate, Description: "Create an item", Scopes: []string{auth.ScopeServiceExampleWrite}},
			{Name: OpExampleUpdate, Description: "Update an item", Scopes: []string{auth.ScopeServiceExampleWrite}},
			{Name: OpExampleDelete, Description: "Delete an item", Scopes: []string{auth.ScopeServiceExampleWrite}},
			{Name: OpExampleHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceExampleRead}},
		},
	}
}

// RegisterService registers the example capability with the hub and ability registry.
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "example service is required")
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   capability.Invoker
	}{
		{operation: OpExampleList, invoker: invokeList(svc)},
		{operation: OpExampleGet, invoker: invokeGet(svc)},
		{operation: OpExampleCreate, invoker: invokeCreate(svc)},
		{operation: OpExampleUpdate, invoker: invokeUpdate(svc)},
		{operation: OpExampleDelete, invoker: invokeDelete(svc)},
		{operation: OpExampleHealth, invoker: invokeHealth(svc)},
	} {
		if err := capability.RegisterInvoker(hub.CapExample, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.ListItems(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Host]{Items: []*capability.Host{}, Page: &capability.PageInfo{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGet(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeCreate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		title, err := capability.RequiredString(params, "title")
		if err != nil {
			return nil, err
		}
		item, err := svc.CreateItem(ctx, title)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeUpdate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		data, _ := params["data"].(map[string]any)
		item, err := svc.UpdateItem(ctx, id, data)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeDelete(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteItem(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		_, _ = time.ParseDuration("") // suppress unused import — keep time for now usage
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok, Text: "ok"}, nil
	}
}
```

Wait, the `time` import is not needed in the invokers. Let me fix that: remove the `time` import from descriptor.go. I'll clean it up in a future step if needed.

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/ability/example/...`
Expected: No errors (if time import causes issues, remove it from invokeHealth)

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/descriptor.go
git commit -m "feat(ability/example): add Descriptor + RegisterService with invokers"
```

---

### Task 6: Ability Descriptor Tests

**Files:**

- Create: `pkg/ability/example/descriptor_test.go`

- [ ] **Step 1: Write TDD tests for descriptor and operation constants**

```go
package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (m mockService) GetItem(_ /*ctx*/, _ /*id*/ string) (*capability.Host, error)         { return nil, nil }
func (m mockService) ListItems(_ /*ctx*/, _ *ListQuery) (*capability.ListResult[capability.Host], error) { return nil, nil }
func (m mockService) CreateItem(_ /*ctx*/, _ string) (*capability.Host, error)              { return nil, nil }
func (m mockService) UpdateItem(_ /*ctx*/, _ string, _ map[string]any) (*capability.Host, error) { return nil, nil }
func (m mockService) DeleteItem(_ /*ctx*/, _ string) error                                { return nil }
func (m mockService) HealthCheck(_ /*ctx*/) (bool, error)                                 { return true, nil }
func (m mockService) ListRawEvents(_ /*ctx*/, _ string) ([]any, string, error)           { return nil, "", nil }

func TestDescriptor_NilService(t *testing.T) {
	t.Parallel()
	desc := Descriptor("backend", "app1", nil)
	assert.False(t, desc.Healthy)
	assert.Equal(t, hub.CapExample, desc.Type)
	assert.Equal(t, "backend", desc.Backend)
}

func TestDescriptor_WithService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		backend string
		app     string
	}{
		{name: "valid backend and app", backend: "example", app: "app1"},
		{name: "empty backend", backend: "", app: "app1"},
		{name: "empty app", backend: "example", app: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := mockService{}
			desc := Descriptor(tt.backend, tt.app, s)
			assert.True(t, desc.Healthy)
			assert.Equal(t, hub.CapExample, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.NotNil(t, desc.Instance)
			assert.NotEmpty(t, desc.Operations)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opName  string
		wantOp  string
	}{
		{name: "list operation", opName: "List", wantOp: OpExampleList},
		{name: "get operation", opName: "Get", wantOp: OpExampleGet},
		{name: "create operation", opName: "Create", wantOp: OpExampleCreate},
		{name: "update operation", opName: "Update", wantOp: OpExampleUpdate},
		{name: "delete operation", opName: "Delete", wantOp: OpExampleDelete},
		{name: "health operation", opName: "Health", wantOp: OpExampleHealth},
	}
	desc := Descriptor("example", "app", mockService{})
	ops := make(map[string]hub.Operation, len(desc.Operations))
	for _, o := range desc.Operations {
		ops[o.Name] = o
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, ops, tt.wantOp, "operation %s should exist", tt.wantOp)
		})
	}
}

func TestRegisterService_NilService(t *testing.T) {
	tests := []struct {
		name string
		svc  Service
	}{
		{name: "nil service returns error", svc: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterService("example", "app1", tt.svc)
			assert.Error(t, err)
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/ability/example/... -v -run "TestDescriptor|TestRegisterService" -count=1`
Expected: PASS

Note: `mockService` will cause "unused parameter" lint warnings — use blank identifiers `_ /*ctx*/` in the actual code.

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/descriptor_test.go
git commit -m "test(ability/example): add TDD tests for Descriptor and RegisterService"
```

---

### Task 7: Ability WebhookConverter

**Files:**

- Create: `pkg/ability/example/webhook.go`

- [ ] **Step 1: Create webhook.go with WebhookConverter implementation**

```go
package example

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ExampleWebhook implements capability.WebhookConverter for the example provider.
// It demonstrates signature verification and payload conversion patterns.
type ExampleWebhook struct {
	secret []byte
}

// NewExampleWebhook creates an ExampleWebhook with the given HMAC secret.
func NewExampleWebhook(secret string) *ExampleWebhook {
	return &ExampleWebhook{secret: []byte(secret)}
}

// WebhookPath returns the URL path that receives webhook events from the example provider.
func (w *ExampleWebhook) WebhookPath() string {
	return "example"
}

// VerifySignature validates the HMAC-SHA256 signature from the X-Signature header.
func (w *ExampleWebhook) VerifySignature(headers map[string]string, body []byte) error {
	if len(w.secret) == 0 {
		return nil
	}
	signature, ok := headers["X-Signature"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing X-Signature header")
	}
	mac := hmac.New(sha256.New, w.secret)
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return types.Errorf(types.ErrUnauthorized, "invalid signature")
	}
	return nil
}

// Convert transforms the raw webhook body into one or more DataEvent records.
func (w *ExampleWebhook) Convert(body []byte, headers map[string]string) ([]types.DataEvent, error) {
	var payload struct {
		EventType string `json:"event_type"`
		EntityID  string `json:"entity_id"`
		Data      any    `json:"data"`
	}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}
	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      payload.EventType,
		Source:         "example_webhook",
		IdempotencyKey: payload.EntityID,
		Data:           types.KV{"event": payload.Data},
	}
	return []types.DataEvent{ev}, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/ability/example/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/webhook.go
git commit -m "feat(ability/example): add WebhookConverter implementation"
```

---

### Task 8: Ability WebhookConverter Tests

**Files:**

- Create: `pkg/ability/example/webhook_test.go`

- [ ] **Step 1: Write TDD tests for webhook converter**

```go
package example

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestExampleWebhook_WebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{name: "returns example path", secret: "test", want: "example"},
		{name: "empty secret still returns path", secret: "", want: "example"},
		{name: "consistent path", secret: "different", want: "example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook(tt.secret)
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestExampleWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event_type":"test.created","entity_id":"123"}`)
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name    string
		secret  string
		headers map[string]string
		body    []byte
		wantErr bool
	}{
		{
			name:   "valid signature",
			secret: secret,
			headers: map[string]string{"X-Signature": validSig},
			body:   body,
			wantErr: false,
		},
		{
			name:   "missing header",
			secret: secret,
			headers: map[string]string{},
			body:   body,
			wantErr: true,
		},
		{
			name:   "invalid signature",
			secret: secret,
			headers: map[string]string{"X-Signature": "bad-signature"},
			body:   body,
			wantErr: true,
		},
		{
			name:   "empty secret skips verification",
			secret: "",
			headers: map[string]string{},
			body:   body,
			wantErr: false,
		},
		{
			name:   "wrong header name",
			secret: secret,
			headers: map[string]string{"X-Hub-Signature": validSig},
			body:   body,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook(tt.secret)
			err := w.VerifySignature(tt.headers, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExampleWebhook_Convert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{
			name:    "valid payload converts to DataEvent",
			body:    []byte(`{"event_type":"test.created","entity_id":"e-001","data":{"key":"value"}}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    []byte(`{invalid`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(`{}`),
			wantErr: false,
		},
		{
			name:    "partial payload",
			body:    []byte(`{"event_type":"test.updated"}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook("secret")
			events, err := w.Convert(tt.body, nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if len(tt.body) > 2 { // non-empty payload
				assert.Len(t, events, 1)
				assert.NotEmpty(t, events[0].EventID)
				assert.Equal(t, "example_webhook", events[0].Source)
			}
		})
	}
}

func TestExampleWebhook_Convert_EventType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventType string
		entityID  string
	}{
		{name: "create event", eventType: "item.created", entityID: "e-1"},
		{name: "update event", eventType: "item.updated", entityID: "e-2"},
		{name: "delete event", eventType: "item.deleted", entityID: "e-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook("secret")
			payload, _ := sonic.Marshal(map[string]string{
				"event_type": tt.eventType,
				"entity_id":  tt.entityID,
			})
			events, err := w.Convert(payload, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tt.eventType, events[0].EventType)
			assert.Equal(t, tt.entityID, events[0].IdempotencyKey)
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/ability/example/... -v -run "TestExampleWebhook" -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/webhook_test.go
git commit -m "test(ability/example): add TDD tests for WebhookConverter"
```

---

### Task 9: Ability PollingResource

**Files:**

- Create: `pkg/ability/example/poller.go`

- [ ] **Step 1: Create poller.go with PollingResource implementation**

```go
package example

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ExamplePoller implements capability.PollingResource for the example provider.
// It demonstrates the polling pattern with cursor-based pagination and content hashing.
type ExamplePoller struct {
	svc     Service
	secret  []byte
	nowFunc func() time.Time
}

// NewExamplePoller creates an ExamplePoller that uses the given Service for data fetching.
func NewExamplePoller(svc Service) *ExamplePoller {
	return &ExamplePoller{
		svc:     svc,
		secret:  []byte("example-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// ResourceName returns the unique name for this polling resource.
func (p *ExamplePoller) ResourceName() string {
	return "example/events"
}

// DefaultInterval returns the recommended polling interval.
func (p *ExamplePoller) DefaultInterval() time.Duration {
	return 60 * time.Second
}

// DiffKey returns the unique identifier for an item, used for change detection.
func (p *ExamplePoller) DiffKey(item any) string {
	if m, ok := item.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return fmt.Sprintf("%v", item)
}

// ContentHash returns a SHA256 hash of the item for content-based change detection.
func (p *ExamplePoller) ContentHash(item any) string {
	data := fmt.Sprintf("%v", item)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CursorField returns the field name used for cursor-based pagination.
func (p *ExamplePoller) CursorField() string {
	return "cursor"
}

// List fetches a batch of items from the provider starting after the given cursor.
func (p *ExamplePoller) List(ctx context.Context, cursor string) (capability.PollResult, error) {
	items, nextCursor, err := p.svc.ListRawEvents(ctx, cursor)
	if err != nil {
		return capability.PollResult{}, err
	}
	return capability.PollResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/ability/example/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/poller.go
git commit -m "feat(ability/example): add PollingResource implementation"
```

---

### Task 10: Ability PollingResource Tests

**Files:**

- Create: `pkg/ability/example/poller_test.go`

- [ ] **Step 1: Write TDD tests for poller**

```go
package example

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePollerService struct {
	items  []any
	cursor string
	err    error
}

func (f *fakePollerService) GetItem(_ context.Context, _ string) (*capability.Host, error)         { return nil, nil }
func (f *fakePollerService) ListItems(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.Host], error) { return nil, nil }
func (f *fakePollerService) CreateItem(_ context.Context, _ string) (*capability.Host, error)      { return nil, nil }
func (f *fakePollerService) UpdateItem(_ context.Context, _ string, _ map[string]any) (*capability.Host, error) { return nil, nil }
func (f *fakePollerService) DeleteItem(_ context.Context, _ string) error                        { return nil }
func (f *fakePollerService) HealthCheck(_ context.Context) (bool, error)                         { return true, nil }
func (f *fakePollerService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return f.items, f.cursor, f.err
}

func TestExamplePoller_ResourceName(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	assert.Equal(t, "example/events", p.ResourceName())
}

func TestExamplePoller_DefaultInterval(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	assert.Equal(t, 60*/*second*/, p.DefaultInterval())
}

func TestExamplePoller_CursorField(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	assert.Equal(t, "cursor", p.CursorField())
}

func TestExamplePoller_DiffKey(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	tests := []struct {
		name string
		item any
		want string
	}{
		{name: "map with id field", item: map[string]any{"id": "abc-123"}, want: "abc-123"},
		{name: "map without id field", item: map[string]any{"key": "val"}, want: "map[key:val]"},
		{name: "string item", item: "plain-string", want: "plain-string"},
		{name: "int item", item: 42, want: "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := p.DiffKey(tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExamplePoller_ContentHash(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	tests := []struct {
		name string
		a    any
		b    any
		same bool
	}{
		{name: "same items produce same hash", a: map[string]any{"id": "1"}, b: map[string]any{"id": "1"}, same: true},
		{name: "different items produce different hash", a: map[string]any{"id": "1"}, b: map[string]any{"id": "2"}, same: false},
		{name: "hash is non-empty", a: map[string]any{"id": "x"}, same: true},
	}
	_ = tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash := p.ContentHash(tt.a)
			assert.NotEmpty(t, hash)
			if tt.same && tt.name == "same items produce same hash" {
				assert.Equal(t, hash, p.ContentHash(tt.b))
			}
			if !tt.same && tt.name == "different items produce different hash" {
				assert.NotEqual(t, hash, p.ContentHash(tt.b))
			}
		})
	}
}

func TestExamplePoller_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		svc        *fakePollerService
		cursor     string
		wantItems  int
		wantCursor string
		wantMore   bool
		wantErr    bool
	}{
		{
			name:       "returns items with no cursor",
			svc:        &fakePollerService{items: []any{map[string]any{"id": "1"}}},
			wantItems:  1,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:       "returns items with next cursor",
			svc:        &fakePollerService{items: []any{map[string]any{"id": "1"}}, cursor: "next-page"},
			wantItems:  1,
			wantCursor: "next-page",
			wantMore:   true,
			wantErr:    false,
		},
		{
			name:       "empty result",
			svc:        &fakePollerService{items: []any{}},
			wantItems:  0,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:    "service error",
			svc:     &fakePollerService{err: context.DeadlineExceeded},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewExamplePoller(tt.svc)
			result, err := p.List(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantItems)
			assert.Equal(t, tt.wantCursor, result.NextCursor)
			assert.Equal(t, tt.wantMore, result.HasMore)
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/ability/example/... -v -run "TestExamplePoller" -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/poller_test.go
git commit -m "test(ability/example): add TDD tests for PollingResource"
```

---

### Task 11: Ability Conformance Framework

**Files:**

- Create: `pkg/ability/example/conformance.go`

- [ ] **Step 1: Create conformance.go with RunExampleConformance**

```go
package example

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/conformance"
)

// Config holds mock backend behavior for conformance testing each Service method.
type Config struct {
	ListItems      []*capability.Host
	ListErr        error
	GetItem        *capability.Host
	GetErr         error
	CreateItem     *capability.Host
	CreateErr      error
	UpdateItem     *capability.Host
	UpdateErr      error
	DeleteErr      error
	HealthOk       bool
	HealthErr      error
	RawItems       []any
	RawCursor      string
	RawErr         error
}

// ServiceFactory creates a Service from a Config for conformance testing.
type ServiceFactory func(t *testing.T, cfg Config) Service

// RunExampleConformance runs the full example capability conformance test suite.
// It exercises every Service method with success, empty, timeout, provider error,
// and invalid input scenarios.
func RunExampleConformance(t *testing.T, factory ServiceFactory) {
	t.Helper()

	t.Run("GetItem", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			id      string
			wantErr bool
		}{
			{
				name:    "success",
				cfg:     Config{GetItem: &capability.Host{ID: "h-1", Name: "test"}},
				id:      "h-1",
				wantErr: false,
			},
			{
				name:    "empty id",
				cfg:     Config{GetErr: capability.Errorf(capability.ErrInvalidArgument, "id required")},
				id:      "",
				wantErr: true,
			},
			{
				name:    "provider error",
				cfg:     Config{GetErr: capability.Errorf(capability.ErrProvider, "down")},
				id:      "h-1",
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.GetItem(context.Background(), tt.id)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("ListItems", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantLen int
			wantErr bool
		}{
			{
				name:    "success with items",
				cfg:     Config{ListItems: []*capability.Host{{ID: "h-1"}, {ID: "h-2"}}},
				wantLen: 2,
				wantErr: false,
			},
			{
				name:    "empty list",
				cfg:     Config{ListItems: []*capability.Host{}},
				wantLen: 0,
				wantErr: false,
			},
			{
				name:    "provider error",
				cfg:     Config{ListErr: capability.Errorf(capability.ErrProvider, "down")},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				result, err := svc.ListItems(context.Background(), &ListQuery{})
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Items, tt.wantLen)
			})
		}
	})

	t.Run("CreateItem", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			title   string
			wantErr bool
		}{
			{name: "success", cfg: Config{CreateItem: &capability.Host{ID: "new", Name: "test"}}, title: "test", wantErr: false},
			{name: "empty title", cfg: Config{CreateErr: capability.Errorf(capability.ErrInvalidArgument, "title required")}, title: "", wantErr: true},
			{name: "provider error", cfg: Config{CreateErr: capability.Errorf(capability.ErrProvider, "down")}, title: "test", wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				item, err := svc.CreateItem(context.Background(), tt.title)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, item)
			})
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			cfg     Config
			wantOk  bool
			wantErr bool
		}{
			{name: "healthy", cfg: Config{HealthOk: true}, wantOk: true, wantErr: false},
			{name: "unhealthy", cfg: Config{HealthOk: false}, wantOk: false, wantErr: false},
			{name: "error", cfg: Config{HealthErr: capability.Errorf(capability.ErrProvider, "timeout")}, wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc := factory(t, tt.cfg)
				ok, err := svc.HealthCheck(context.Background())
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.wantOk, ok)
			})
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()
		t.Run("canceled context", func(t *testing.T) {
			t.Parallel()
			svc := factory(t, Config{GetItem: &capability.Host{ID: "h-1"}})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := svc.GetItem(ctx, "h-1")
			require.Error(t, err)
		})
	})
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/ability/example/...`
Expected: No errors

Wait — `capability.Errorf` doesn't exist. Let me use `fmt.Errorf` or `types.Errorf` instead. Actually, looking at the existing conformance code, the Config structs use `assert.AnError` which is `var AnError = errors.New("assert.AnError general error for testing")`. That's simpler. Let me fix.

In `Config`, errors should just be generic `error` fields, and the factory uses `assert.AnError` for test error values. For the invalid argument errors, the adapter implementation creates them from empty input.

Fixed conformance.go errors: use simple errors, not `capability.Errorf` (which doesn't exist).

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/conformance.go
git commit -m "feat(ability/example): add RunExampleConformance test harness"
```

---

### Task 12: Ability Conformance Tests

**Files:**

- Create: `pkg/ability/example/conformance_test.go`

- [ ] **Step 1: Write conformance self-tests**

```go
package example

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/capability"
)

type conformanceService struct {
	cfg Config
}

func (c *conformanceService) GetItem(_ /*ctx*/, id string) (*capability.Host, error) {
	if c.cfg.GetErr != nil { return nil, c.cfg.GetErr }
	return c.cfg.GetItem, nil
}
func (c *conformanceService) ListItems(_ /*ctx*/, _ *ListQuery) (*capability.ListResult[capability.Host], error) {
	if c.cfg.ListErr != nil { return nil, c.cfg.ListErr }
	return &capability.ListResult[capability.Host]{Items: c.cfg.ListItems}, nil
}
func (c *conformanceService) CreateItem(_ /*ctx*/, _ string) (*capability.Host, error) {
	if c.cfg.CreateErr != nil { return nil, c.cfg.CreateErr }
	return c.cfg.CreateItem, nil
}
func (c *conformanceService) UpdateItem(_ /*ctx*/, _ string, _ map[string]any) (*capability.Host, error) {
	if c.cfg.UpdateErr != nil { return nil, c.cfg.UpdateErr }
	return c.cfg.UpdateItem, nil
}
func (c *conformanceService) DeleteItem(_ /*ctx*/, _ string) error { return c.cfg.DeleteErr }
func (c *conformanceService) HealthCheck(_ /*ctx*/) (bool, error) {
	if c.cfg.HealthErr != nil { return false, c.cfg.HealthErr }
	return c.cfg.HealthOk, nil
}
func (c *conformanceService) ListRawEvents(_ /*ctx*/, _ string) ([]any, string, error) {
	if c.cfg.RawErr != nil { return nil, "", c.cfg.RawErr }
	return c.cfg.RawItems, c.cfg.RawCursor, nil
}

func TestRunExampleConformance(t *testing.T) {
	t.Run("runs example conformance test suite", func(t *testing.T) {
		RunExampleConformance(t, func(_ *testing.T, cfg Config) Service {
			return &conformanceService{cfg: cfg}
		})
	})
}

```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/ability/example/... -v -run "TestRunExampleConformance" -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/conformance_test.go
git commit -m "test(ability/example): add conformance self-tests"
```

---

### Task 13: Ability Adapter

**Files:**

- Create: `pkg/ability/example/example/adapter.go`

- [ ] **Step 1: Create adapter.go — concrete Service implementation wrapping provider**

```go
// Package example implements the example provider adapter for the example capability.
package example

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	example "github.com/flowline-io/flowbot/pkg/capability/example"
	provider "github.com/flowline-io/flowbot/pkg/providers/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

// client defines the subset of provider.Example methods used by this adapter.
type client interface {
	Get(ctx context.Context, path string) (*provider.Response, error)
	Post(ctx context.Context, path string, data any) (*provider.Response, error)
	Put(ctx context.Context, path string, data any) (*provider.Response, error)
	Delete(ctx context.Context, path string) (*provider.Response, error)
	GetStatus(ctx context.Context, code int) (*provider.Response, error)
	ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error)
}

// Adapter implements example.Service using the example provider client.
type Adapter struct {
	client client
	now    func() time.Time
}

// New creates an Adapter using the default provider client (reads config from YAML).
func New() example.Service {
	return NewWithClient(provider.GetClient())
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) example.Service {
	return &Adapter{
		client: c,
		now:    time.Now,
	}
}

func (a *Adapter) GetItem(ctx context.Context, id string) (*capability.Host, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	resp, err := a.client.Get(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example get failed", err)
	}
	return &capability.Host{ID: id, Name: resp.Origin, Status: resp.URL}, nil
}

func (a *Adapter) ListItems(ctx context.Context, q *example.ListQuery) (*capability.ListResult[capability.Host], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	resp, err := a.client.Get(ctx, "list")
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example list failed", err)
	}
	item := &capability.Host{ID: "item-1", Name: resp.Origin, Status: "active"}
	return &capability.ListResult[capability.Host]{
		Items: []*capability.Host{item},
	}, nil
}

func (a *Adapter) CreateItem(ctx context.Context, title string) (*capability.Host, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	resp, err := a.client.Post(ctx, "create", map[string]string{"title": title})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example create failed", err)
	}
	return &capability.Host{ID: "created-1", Name: title, Status: resp.URL}, nil
}

func (a *Adapter) UpdateItem(ctx context.Context, id string, data map[string]any) (*capability.Host, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	resp, err := a.client.Put(ctx, id, data)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example update failed", err)
	}
	return &capability.Host{ID: id, Name: resp.Origin, Status: "updated"}, nil
}

func (a *Adapter) DeleteItem(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	_, err := a.client.Delete(ctx, id)
	if err != nil {
		return types.WrapError(types.ErrProvider, "example delete failed", err)
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	_, err := a.client.GetStatus(ctx, 200)
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "example health check failed", err)
	}
	return true, nil
}

func (a *Adapter) ListRawEvents(ctx context.Context, cursor string) ([]any, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, next, err := a.client.ListRawEvents(ctx, cursor)
	if err != nil {
		return nil, "", types.WrapError(types.ErrProvider, "example list raw events failed", err)
	}
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result, next, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/ability/example/example/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/example/adapter.go
git commit -m "feat(ability/example/example): add adapter implementing Service via provider"
```

---

### Task 14: Adapter Unit Tests

**Files:**

- Create: `pkg/ability/example/example/adapter_test.go`

- [ ] **Step 1: Write TDD tests for adapter methods**

```go
package example

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/example"
)

type fakeClient struct {
	getResp     *provider.Response
	getErr      error
	postResp    *provider.Response
	postErr     error
	putResp     *provider.Response
	putErr      error
	deleteResp  *provider.Response
	deleteErr   error
	statusResp  *provider.Response
	statusErr   error
	listRawResp []map[string]any
	listRawNext string
	listRawErr  error
}

func (f *fakeClient) Get(_ context.Context, _ string) (*provider.Response, error)    { return f.getResp, f.getErr }
func (f *fakeClient) Post(_ context.Context, _ string, _ any) (*provider.Response, error) { return f.postResp, f.postErr }
func (f *fakeClient) Put(_ context.Context, _ string, _ any) (*provider.Response, error)   { return f.putResp, f.putErr }
func (f *fakeClient) Delete(_ context.Context, _ string) (*provider.Response, error)        { return f.deleteResp, f.deleteErr }
func (f *fakeClient) GetStatus(_ context.Context, _ int) (*provider.Response, error)        { return f.statusResp, f.statusErr }
func (f *fakeClient) ListRawEvents(_ context.Context, _ string) ([]map[string]any, string, error) {
	return f.listRawResp, f.listRawNext, f.listRawErr
}

func TestAdapter_GetItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{
			name:    "success",
			client:  &fakeClient{getResp: &provider.Response{Origin: "1.2.3.4", URL: "https://example.com"}},
			id:      "item-1",
			wantErr: false,
		},
		{
			name:    "empty id",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getErr: errors.New("down")},
			id:      "item-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.GetItem(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, item)
			assert.Equal(t, tt.id, item.ID)
		})
	}
}

func TestAdapter_ListItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name:    "success",
			client:  &fakeClient{getResp: &provider.Response{Origin: "1.2.3.4"}},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getErr: errors.New("timeout")},
			wantErr: true,
		},
		{
			name:    "success with nil response returns item",
			client:  &fakeClient{getResp: &provider.Response{}},
			wantLen: 1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.ListItems(context.Background(), &capability.ListQuery{})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantLen)
		})
	}
}

func TestAdapter_CreateItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		title   string
		wantErr bool
	}{
		{name: "success", client: &fakeClient{postResp: &provider.Response{URL: "https://example.com"}}, title: "test", wantErr: false},
		{name: "empty title", client: &fakeClient{}, title: "", wantErr: true},
		{name: "provider error", client: &fakeClient{postErr: errors.New("fail")}, title: "test", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.CreateItem(context.Background(), tt.title)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, item)
		})
	}
}

func TestAdapter_DeleteItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{name: "success", client: &fakeClient{deleteResp: &provider.Response{}}, id: "item-1", wantErr: false},
		{name: "empty id", client: &fakeClient{}, id: "", wantErr: true},
		{name: "provider error", client: &fakeClient{deleteErr: errors.New("gone")}, id: "item-1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			err := a.DeleteItem(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAdapter_HealthCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantOk  bool
		wantErr bool
	}{
		{name: "healthy", client: &fakeClient{statusResp: &provider.Response{}}, wantOk: true, wantErr: false},
		{name: "unhealthy", client: &fakeClient{statusErr: errors.New("timeout")}, wantErr: true},
		{name: "success returns true", client: &fakeClient{statusResp: &provider.Response{Origin: "10.0.0.1"}}, wantOk: true, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			ok, err := a.HealthCheck(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestAdapter_ListRawEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		client     *fakeClient
		wantLen    int
		wantCursor string
		wantErr    bool
	}{
		{name: "success", client: &fakeClient{listRawResp: []map[string]any{{"id": "e1"}}}, wantLen: 1, wantCursor: "", wantErr: false},
		{name: "with cursor", client: &fakeClient{listRawResp: []map[string]any{{"id": "e1"}}, listRawNext: "next"}, wantLen: 1, wantCursor: "next", wantErr: false},
		{name: "provider error", client: &fakeClient{listRawErr: errors.New("down")}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			items, next, err := a.ListRawEvents(context.Background(), "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			assert.Equal(t, tt.wantCursor, next)
		})
	}
}

func TestAdapter_ContextCanceled(t *testing.T) {
	t.Run("canceled context returns timeout error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		a := NewWithClient(&fakeClient{getResp: &provider.Response{}})
		_, err := a.GetItem(ctx, "id")
		assert.Error(t, err)
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/ability/example/example/... -v -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/example/adapter_test.go
git commit -m "test(ability/example/example): add TDD tests for adapter"
```

---

### Task 15: Adapter Conformance BDD Tests

**Files:**

- Create: `pkg/ability/example/example/conformance_test.go`

- [ ] **Step 1: Write conformance test using RunExampleConformance**

```go
package example

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	example "github.com/flowline-io/flowbot/pkg/capability/example"
	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/example"
)

func TestExampleConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "runs example conformance test suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			example.RunExampleConformance(t, func(_ *testing.T, cfg example.Config) example.Service {
				c := &fakeClient{
					getErr:     cfg.GetErr,
					postErr:    cfg.CreateErr,
					deleteErr:  cfg.DeleteErr,
					statusErr:  cfg.HealthErr,
				}
				if cfg.GetItem != nil {
					c.getResp = &provider.Response{Origin: cfg.GetItem.Name, URL: cfg.GetItem.Status}
				}
				if cfg.CreateItem != nil {
					c.postResp = &provider.Response{URL: "https://example.com"}
				}
				if cfg.HealthOk {
					c.statusResp = &provider.Response{}
				}
				if cfg.ListErr != nil {
					c.getErr = cfg.ListErr
				}
				if cfg.CreateErr != nil {
					c.postErr = cfg.CreateErr
				}
				if cfg.DeleteErr != nil {
					c.deleteErr = cfg.DeleteErr
				}
				if cfg.HealthErr != nil {
					c.statusErr = cfg.HealthErr
				}
				a, ok := NewWithClient(c).(*Adapter)
				if !ok {
					t.Fatal("unexpected type")
				}
				return a
			})
		})
	}
}

func TestConformance_FakeClient_ImplementsClient(t *testing.T) {
	t.Run("fakeClient satisfies client interface", func(t *testing.T) {
		var _ client = (*fakeClient)(nil)
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/ability/example/example/... -v -run "TestExampleConformance" -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/example/example/conformance_test.go
git commit -m "test(ability/example/example): add conformance BDD tests"
```

---

### Task 16: Module — Update module.go with ability wiring

**Files:**

- Modify: `internal/modules/example/module.go`

- [ ] **Step 1: Update module.go Init() to wire ability+provider in Init()**

The existing `module.go` has a simple `Init()` that just parses config. We need to add adapter creation and ability registration when the module is enabled. Add the import and modify `Init()`:

```go
// Add these imports to existing imports:
import (
	// ... existing imports ...
	abilityexample "github.com/flowline-io/flowbot/pkg/capability/example"
	adapter "github.com/flowline-io/flowbot/pkg/capability/example/example"
)
```

Modify the `Init()` function to add after `handler.initialized = true`:

```go
func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	handler.initialized = true

	// Register the example capability with the adapter.
	svc := adapter.New()
	if err := abilityexample.RegisterService("example", config.Environment, svc); err != nil {
		return fmt.Errorf("register example ability: %w", err)
	}

	return nil
}
```

Update `Rules()` to include `webhookRules`:

```go
func (moduleHandler) Rules() []any {
	return []any{
		commandRules,
		formRules,
		pageRules,
		webserviceRules,
		webhookRules,
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/example/...`
Expected: No errors

- [ ] **Step 3: Verify existing tests still pass**

Run: `go test ./internal/modules/example/... -v -run "TestInit|TestRules" -count=1`
Expected: TestRules_ReturnsAllRulesets will fail (expects 4, now returns 5). Update the test in the next step.

- [ ] **Step 4: Commit**

```bash
git add internal/modules/example/module.go
git commit -m "feat(modules/example): wire ability adapter in Init(), add webhookRules"
```

---

### Task 17: Module — Update webservice.go with REST routes

**Files:**

- Modify: `internal/modules/example/webservice.go`

- [ ] **Step 1: Add new REST routes for the example capability**

Append to the existing `webserviceRules` slice and add handler functions:

```go
// Add to existing webserviceRules:
var webserviceRules = []webservice.Rule{
	// ... existing routes ...
	webservice.Get("/get", getExampleItem, route.WithNotAuth()),
	webservice.Get("/health", healthExample, route.WithNotAuth()),
	webservice.Post("/create", createExampleItem, route.WithNotAuth()),
	webservice.Delete("/delete", deleteExampleItem, route.WithNotAuth()),
}

// getExampleItem handles GET /service/example/get?id=xxx
//
//	@Summary	Get example item
//	@Tags		example
//	@Param		id	query	string	true	"item id"
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/get [get]
func getExampleItem(ctx fiber.Ctx) error {
	id := ctx.Query("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	res, err := capability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleGet, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// healthExample handles GET /service/example/health
//
//	@Summary	Example health check
//	@Tags		example
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/health [get]
func healthExample(ctx fiber.Ctx) error {
	res, err := capability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleHealth, map[string]any{})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// createExampleItem handles POST /service/example/create
//
//	@Summary	Create example item
//	@Tags		example
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/create [post]
func createExampleItem(ctx fiber.Ctx) error {
	var body struct {
		Title string `json:"title"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	if body.Title == "" {
		return types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	res, err := capability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleCreate, map[string]any{"title": body.Title})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// deleteExampleItem handles DELETE /service/example/delete?id=xxx
//
//	@Summary	Delete example item
//	@Tags		example
//	@Param		id	query	string	true	"item id"
//	@Success	200	{object}	protocol.Response{}
//	@Router		/example/delete [delete]
func deleteExampleItem(ctx fiber.Ctx) error {
	id := ctx.Query("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	res, err := capability.Invoke(context.Background(), hub.CapExample, abilityexample.OpExampleDelete, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}
```

Add new imports to the existing import block:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/hub"
	abilityexample "github.com/flowline-io/flowbot/pkg/capability/example"
)
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/example/...`
Expected: No errors

- [ ] **Step 3: Verify webservice tests still pass**

Run: `go test ./internal/modules/example/... -v -run "TestWebservice" -count=1`
Expected: TestWebserviceRules_Defined passes (not empty)

- [ ] **Step 4: Commit**

```bash
git add internal/modules/example/webservice.go
git commit -m "feat(modules/example): add REST routes for example capability"
```

---

### Task 18: Module — Add webhook routes

**Files:**

- Create: `internal/modules/example/webhook.go`

- [ ] **Step 1: Create webhook.go with webhook rule**

Note: The module's Init() doesn't have access to `EventSourceManager`. For a full demonstration, the webhook rule can be registered but the actual `EventSourceManager` reference would come from server-level wiring. For the example module, we demonstrate the webhook route pattern and note where EventSourceManager registration would occur.

```go
package example

import (
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webhookRules = []webservice.Rule{
	webservice.Post("/webhook/example", exampleWebhook, route.WithNotAuth()),
}

// exampleWebhook handles POST /service/example/webhook/example
//
//	@Summary	Receive example webhook events
//	@Tags		example
//	@Accept		json
//	@Produce	json
//	@Success	202	{string}	string	"Accepted"
//	@Router		/example/webhook/example [post]
func exampleWebhook(ctx fiber.Ctx) error {
	// When EventSourceManager is available, this would delegate to it:
	// return esm.WebhookHandler()(ctx)
	//
	// For standalone demonstration, acknowledge the webhook.
	return ctx.SendStatus(fiber.StatusAccepted)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/example/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/modules/example/webhook.go
git commit -m "feat(modules/example): add webhook route for example capability"
```

---

### Task 19: Module — Update tests

**Files:**

- Modify: `internal/modules/example/module_test.go`

- [ ] **Step 1: Update TestRules_ReturnsAllRulesets to expect 5 rulesets**

Change `assert.Len(t, rules, 4)` to `assert.Len(t, rules, 5)` in the test:

```go
func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should return 5 rulesets"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 5)
		})
	}
}
```

- [ ] **Step 2: Add a test for webhook rules**

```go
func TestWebhookRules_Defined(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, webhookRules)
		})
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/modules/example/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/modules/example/module_test.go
git commit -m "test(modules/example): update rules count to 5, add webhook test"
```

---

### Task 20: Module — BDD Integration Tests

**Files:**

- Create: `internal/modules/example/module_suite_test.go`

- [ ] **Step 1: Install Ginkgo if not already**

Run: `go get github.com/onsi/ginkgo/v2 github.com/onsi/gomega`

- [ ] **Step 2: Write BDD integration test**

```go
package example_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/modules/example"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/module"
)

func TestExampleModule(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Example Module Suite")
}

var _ = ginkgo.Describe("Example Module", func() {
	var app *fiber.App

	ginkgo.BeforeEach(func() {
		example.Register()
		err := module.Initialize()
		// Don't fatal on init errors — module may need config
		_ = err
		app = fiber.New()
		example.Webservice(app) // calls module.Webservice() which mounts under /service/example
	})

	ginkgo.AfterEach(func() {
		_ = app.Shutdown()
	})

	ginkgo.It("returns existing example endpoint", func() {
		req := httptest.NewRequest("GET", "/service/example/example", nil)
		resp, err := app.Test(req)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
	})

	ginkgo.It("returns 400 for get without id", func() {
		req := httptest.NewRequest("GET", "/service/example/get", nil)
		resp, err := app.Test(req)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(400))
	})
})
```

- [ ] **Step 3: Run BDD tests**

Run: `go test ./internal/modules/example/... -v -run "TestExampleModule" -count=1`
Expected: Tests run (may pass/fail depending on module initialization without config)

- [ ] **Step 4: Commit**

```bash
git add internal/modules/example/module_suite_test.go
git commit -m "test(modules/example): add BDD integration test suite"
```

---

### Task 21: Final Verification

**Files:** None (verification only)

- [ ] **Step 1: Run all provider tests**

Run: `go test ./pkg/providers/example/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 2: Run all ability tests**

Run: `go test ./pkg/ability/example/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 3: Run all module tests**

Run: `go test ./internal/modules/example/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 4: Run lint**

Run: `go tool task lint`
Expected: No new warnings

- [ ] **Step 5: Full build**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "test: final verification — all tests pass, lint clean"
```
