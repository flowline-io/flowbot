// Package memos implements the Memos provider for note-taking and knowledge management.
// It wraps the Memos REST API (https://usememos.com) using personal access tokens.
package memos

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	// ID is the provider identifier used in configuration and registration.
	ID = "memos"
	// EndpointKey is the config key for the Memos instance base URL.
	EndpointKey = "endpoint"
	// TokenKey is the config key for the personal access token or access token.
	TokenKey = "token"
	// WebhookTokenKey is the config key for the inbound webhook Bearer token.
	WebhookTokenKey = "webhook_token"
)

// GetWebhookToken reads the webhook Bearer token from the memos provider config.
func GetWebhookToken() string {
	tok, err := providers.GetConfig(ID, WebhookTokenKey)
	if err != nil {
		return ""
	}
	return tok.String()
}

// Memos wraps the Memos REST API client.
type Memos struct {
	c *resty.Client
}

// GetClient reads provider config and returns a new Memos client.
// It returns nil when the endpoint is not configured.
func GetClient() *Memos {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewMemos(endpoint.String(), token.String())
}

// NewMemos creates a Memos client with the given endpoint and access token.
// If endpoint is empty, it returns nil.
func NewMemos(endpoint, token string) *Memos {
	if endpoint == "" {
		return nil
	}
	v := &Memos{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetAuthToken(token)
	}
	return v
}

// CreateMemo creates a new memo via POST /api/v1/memos.
func (v *Memos) CreateMemo(ctx context.Context, content, visibility string) (*Memo, error) {
	if visibility == "" {
		visibility = "PRIVATE"
	}
	body := CreateMemoRequest{
		Memo: Memo{
			Content:    content,
			Visibility: visibility,
		},
	}
	resp := &Memo{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(resp).
		Post("/api/v1/memos")
	if err != nil {
		return nil, fmt.Errorf("memos create memo: %w", err)
	}
	if httpResp.StatusCode() >= 200 && httpResp.StatusCode() < 300 {
		return resp, nil
	}
	return nil, fmt.Errorf("memos create memo: unexpected status %d: %s", httpResp.StatusCode(), httpResp.String())
}

// GetMemo retrieves a memo by its resource name (e.g., "memos/123") via GET /api/v1/{name}.
func (v *Memos) GetMemo(ctx context.Context, name string) (*Memo, error) {
	resp := &Memo{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetResult(resp).
		SetPathParam("name", name).
		Get("/api/v1/{name}")
	if err != nil {
		return nil, fmt.Errorf("memos get memo %s: %w", name, err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, fmt.Errorf("memos get memo %s: unexpected status %d: %s", name, httpResp.StatusCode(), httpResp.String())
}

// ListMemos lists memos with pagination and filters via GET /api/v1/memos.
func (v *Memos) ListMemos(ctx context.Context, params ListMemosParams) (*ListMemosResponse, error) {
	resp := &ListMemosResponse{}
	req := v.c.R().SetContext(ctx).SetResult(resp)
	if params.PageSize > 0 {
		req.SetQueryParam("pageSize", strconv.Itoa(int(params.PageSize)))
	}
	if params.PageToken != "" {
		req.SetQueryParam("pageToken", params.PageToken)
	}
	if params.State != "" {
		req.SetQueryParam("state", params.State)
	}
	if params.OrderBy != "" {
		req.SetQueryParam("orderBy", params.OrderBy)
	}
	if params.Filter != "" {
		req.SetQueryParam("filter", params.Filter)
	}
	httpResp, err := req.Get("/api/v1/memos")
	if err != nil {
		return nil, fmt.Errorf("memos list memos: %w", err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, fmt.Errorf("memos list memos: unexpected status %d: %s", httpResp.StatusCode(), httpResp.String())
}

// UpdateMemo updates a memo via PATCH /api/v1/{memo.name}.
// name is the resource name (e.g., "memos/123").
// fields lists the field mask paths to update (e.g., ["content", "visibility"]).
func (v *Memos) UpdateMemo(ctx context.Context, name, content, visibility string, pinned *bool, fields []string) (*Memo, error) {
	memo := Memo{
		Name: name,
	}
	if content != "" {
		memo.Content = content
	}
	if visibility != "" {
		memo.Visibility = visibility
	}
	if pinned != nil {
		memo.Pinned = *pinned
	}
	body := UpdateMemoRequest{
		Memo:       memo,
		UpdateMask: fields,
	}
	resp := &Memo{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(resp).
		SetPathParam("name", name).
		Patch("/api/v1/{name}")
	if err != nil {
		return nil, fmt.Errorf("memos update memo %s: %w", name, err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, fmt.Errorf("memos update memo %s: unexpected status %d: %s", name, httpResp.StatusCode(), httpResp.String())
}

// DeleteMemo deletes a memo via DELETE /api/v1/{name}.
// name is the resource name (e.g., "memos/123").
func (v *Memos) DeleteMemo(ctx context.Context, name string) error {
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetPathParam("name", name).
		Delete("/api/v1/{name}")
	if err != nil {
		return fmt.Errorf("memos delete memo %s: %w", name, err)
	}
	if httpResp.StatusCode() == http.StatusOK || httpResp.StatusCode() == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("memos delete memo %s: unexpected status %d: %s", name, httpResp.StatusCode(), httpResp.String())
}

// GetCurrentUser retrieves the currently authenticated user via GET /api/v1/auth/me.
func (v *Memos) GetCurrentUser(ctx context.Context) (*User, error) {
	resp := &struct {
		User User `json:"user"`
	}{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetResult(resp).
		Get("/api/v1/auth/me")
	if err != nil {
		return nil, fmt.Errorf("memos get current user: %w", err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return &resp.User, nil
	}
	return nil, fmt.Errorf("memos get current user: unexpected status %d: %s", httpResp.StatusCode(), httpResp.String())
}

// ListRawEvents lists memos as raw events for polling support.
// The cursor is used as the page token for pagination.
func (v *Memos) ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error) {
	resp := &ListMemosResponse{}
	req := v.c.R().SetContext(ctx).SetResult(resp)
	if cursor != "" {
		req.SetQueryParam("pageToken", cursor)
	}
	req.SetQueryParam("pageSize", strconv.Itoa(MaxPageSize))
	httpResp, err := req.Get("/api/v1/memos")
	if err != nil {
		return nil, "", fmt.Errorf("memos list raw events: %w", err)
	}
	if httpResp.StatusCode() != http.StatusOK {
		return nil, "", fmt.Errorf("memos list raw events: unexpected status %d: %s", httpResp.StatusCode(), httpResp.String())
	}
	items := make([]map[string]any, len(resp.Memos))
	for i, m := range resp.Memos {
		items[i] = map[string]any{
			"name":       m.Name,
			"content":    m.Content,
			"visibility": m.Visibility,
			"creator":    m.Creator,
			"pinned":     m.Pinned,
			"tags":       m.Tags,
			"snippet":    m.Snippet,
		}
		if m.CreateTime != nil {
			items[i]["createTime"] = m.CreateTime.Format("2006-01-02T15:04:05Z07:00")
		}
		if m.UpdateTime != nil {
			items[i]["updateTime"] = m.UpdateTime.Format("2006-01-02T15:04:05Z07:00")
		}
	}
	return items, resp.NextPageToken, nil
}
