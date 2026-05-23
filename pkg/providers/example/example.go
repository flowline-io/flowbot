// Package example implements the example provider using httpbin.org for demonstration.
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
func (e *Example) GetAccessToken(_ fiber.Ctx) (types.KV, error) {
	return types.KV{
		"access_token": "example-token",
		"scope":        "example:read example:write",
	}, nil
}

// OAuth interface compliance check.
var _ providers.OAuthProvider = (*Example)(nil)
