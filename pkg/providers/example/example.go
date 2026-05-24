// Package example implements the example provider using jsonplaceholder.typicode.com for demonstration.
package example

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"resty.dev/v3"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID               = "example"
	EndpointKey      = "endpoint"
	TokenKey         = "token"
	WebhookSecretKey = "webhook_secret"
)

// Example wraps the jsonplaceholder.typicode.com API client for demonstration purposes.
// It provides CRUD operations, OAuth stubs, and polling support.
type Example struct {
	c     *resty.Client
	token string
}

// GetClient reads provider config and returns a new Example client.
// It falls back to https://jsonplaceholder.typicode.com when no endpoint is configured.
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
// If endpoint is empty, it defaults to https://jsonplaceholder.typicode.com.
func NewExample(endpoint, token string) *Example {
	if endpoint == "" {
		endpoint = "https://jsonplaceholder.typicode.com"
	}
	v := &Example{token: token}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetAuthToken(token)
	}
	return v
}

// Get fetches a jsonplaceholder post by ID via GET /posts/{path}.
// If path is empty it defaults to "1".
func (e *Example) Get(ctx context.Context, path string) (*Response, error) {
	resp := &Response{}
	if path == "" {
		path = "1"
	}
	_, err := e.c.R().SetContext(ctx).SetResult(resp).Get("/posts/" + path)
	if err != nil {
		return nil, fmt.Errorf("example get: %w", err)
	}
	return resp, nil
}

// Post creates a new jsonplaceholder post via POST /posts with the given data as JSON body.
func (e *Example) Post(ctx context.Context, _ string, data any) (*Response, error) {
	resp := &Response{}
	_, err := e.c.R().SetContext(ctx).SetBody(data).SetResult(resp).Post("/posts")
	if err != nil {
		return nil, fmt.Errorf("example post: %w", err)
	}
	return resp, nil
}

// Put updates a jsonplaceholder post via PUT /posts/{path} with the given data as JSON body.
// If path is empty it defaults to "1".
func (e *Example) Put(ctx context.Context, path string, data any) (*Response, error) {
	resp := &Response{}
	if path == "" {
		path = "1"
	}
	_, err := e.c.R().SetContext(ctx).SetBody(data).SetResult(resp).Put("/posts/" + path)
	if err != nil {
		return nil, fmt.Errorf("example put: %w", err)
	}
	return resp, nil
}

// Delete deletes a jsonplaceholder post via DELETE /posts/{path}.
// If path is empty it defaults to "1".
func (e *Example) Delete(ctx context.Context, path string) (*Response, error) {
	resp := &Response{}
	if path == "" {
		path = "1"
	}
	_, err := e.c.R().SetContext(ctx).SetResult(resp).Delete("/posts/" + path)
	if err != nil {
		return nil, fmt.Errorf("example delete: %w", err)
	}
	return resp, nil
}

// GetStatus fetches a jsonplaceholder post by ID to demonstrate HTTP status handling.
// Non-existent IDs return 404 errors.
func (e *Example) GetStatus(ctx context.Context, code int) (*Response, error) {
	resp := &Response{}
	_, err := e.c.R().SetContext(ctx).SetResult(resp).Get(fmt.Sprintf("/posts/%d", code))
	if err != nil {
		return nil, fmt.Errorf("example status %d: %w", code, err)
	}
	return resp, nil
}

// GetWithDelay fetches a jsonplaceholder post with a timeout context to demonstrate
// timeout handling patterns.
func (e *Example) GetWithDelay(ctx context.Context, seconds int) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(seconds+2)*time.Second)
	defer cancel()
	resp := &Response{}
	_, err := e.c.R().SetContext(ctx).SetResult(resp).Get("/posts/1")
	if err != nil {
		return nil, fmt.Errorf("example delay %ds: %w", seconds, err)
	}
	return resp, nil
}

// ListRawEvents fetches jsonplaceholder posts as a list for polling demonstration.
// The cursor is used as a page number for pagination.
func (e *Example) ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error) {
	var raw []map[string]any
	req := e.c.R().SetContext(ctx).SetResult(&raw)
	if cursor != "" {
		req.SetQueryParam("_page", cursor)
	}
	_, err := req.Get("/posts")
	if err != nil {
		return nil, "", fmt.Errorf("example list raw: %w", err)
	}
	nextCursor := ""
	if len(raw) > 0 && cursor != "" {
		page, parseErr := strconv.Atoi(cursor)
		if parseErr == nil {
			nextCursor = strconv.Itoa(page + 1)
		}
	}
	return raw, nextCursor, nil
}

// GetAuthorizeURL returns a constructed OAuth authorize URL for demonstration.
func (e *Example) GetAuthorizeURL(state string) string {
	endpoint := e.c.BaseURL()
	redirectURI := providers.RedirectURI(ID, state)
	return fmt.Sprintf("%s/authorize?client_id=example&response_type=code&redirect_uri=%s&state=%s", endpoint, redirectURI, state)
}

// GetAccessToken simulates an OAuth code exchange for demonstration.
func (*Example) GetAccessToken(_ fiber.Ctx) (*providers.OAuthToken, error) {
	return &providers.OAuthToken{
		Name:        ID,
		Type:        ID,
		AccessToken: "example-token",
		TokenType:   "bearer",
		Scope:       "example:read example:write",
	}, nil
}

// GetWebhookSecret reads the webhook HMAC secret from the example provider config.
func GetWebhookSecret() string {
	sec, err := providers.GetConfig(ID, WebhookSecretKey)
	if err != nil {
		return ""
	}
	return sec.String()
}

// OAuth interface compliance check.
var _ providers.OAuthProvider = (*Example)(nil)
