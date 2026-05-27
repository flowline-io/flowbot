package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// MemoClient provides access to the memo API.
type MemoClient struct {
	c *Client
}

// ListMemosQuery contains query parameters for listing memos.
type ListMemosQuery struct {
	Limit  int
	Cursor string
}

// MemoListResult holds the paginated list response extracted from InvokeResult.
type MemoListResult struct {
	Items []*ability.Memo `json:"data"`
	Page  MemoPage        `json:"page"`
}

// MemoPage holds pagination metadata.
type MemoPage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
}

// MemoItemResult holds a single memo extracted from InvokeResult.
type MemoItemResult struct {
	Item ability.Memo `json:"data"`
}

// MemoHealthResult holds the health check result extracted from InvokeResult.
type MemoHealthResult struct {
	Healthy bool `json:"data"`
}

// List returns a paginated list of memos.
func (m *MemoClient) List(ctx context.Context, query *ListMemosQuery) (*MemoListResult, error) {
	if query != nil {
		if err := validateListMemosQuery(query); err != nil {
			return nil, err
		}
	}
	path := "/service/memo"
	if query != nil {
		v := url.Values{}
		if query.Limit > 0 {
			v.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			v.Set("cursor", query.Cursor)
		}
		if len(v) > 0 {
			path = path + "?" + v.Encode()
		}
	}
	var result MemoListResult
	err := m.c.Get(ctx, path, &result)
	return &result, err
}

func validateListMemosQuery(query *ListMemosQuery) error {
	if query.Limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", query.Limit)
	}
	if query.Limit > 100 {
		return fmt.Errorf("limit exceeds maximum of 100")
	}
	if len(query.Cursor) > 4096 {
		return fmt.Errorf("cursor exceeds maximum length of 4096")
	}
	return nil
}

// Get returns a single memo by its resource name (e.g., "memos/123").
func (m *MemoClient) Get(ctx context.Context, name string) (*ability.Memo, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	var result MemoItemResult
	path := "/service/memo?" + url.Values{"name": {name}}.Encode()
	err := m.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// CreateMemoRequest is the request body for creating a memo.
type CreateMemoRequest struct {
	Content    string `json:"content"`
	Visibility string `json:"visibility,omitempty"`
}

// Create creates a new memo.
func (m *MemoClient) Create(ctx context.Context, content, visibility string) (*ability.Memo, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	body := CreateMemoRequest{
		Content:    content,
		Visibility: visibility,
	}
	var result MemoItemResult
	err := m.c.Post(ctx, "/service/memo", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// UpdateMemoRequest is the request body for updating a memo.
type UpdateMemoRequest struct {
	Content    string `json:"content,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	Pinned     *bool  `json:"pinned"`
}

// Update updates an existing memo.
func (m *MemoClient) Update(ctx context.Context, name string, req *UpdateMemoRequest) (*ability.Memo, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	var result MemoItemResult
	path := "/service/memo?" + url.Values{"name": {name}}.Encode()
	err := m.c.Patch(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// Delete removes a memo by its resource name.
func (m *MemoClient) Delete(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	path := "/service/memo?" + url.Values{"name": {name}}.Encode()
	err := m.c.Delete(ctx, path, nil, nil)
	return err
}

// Health checks whether the memo backend is reachable.
func (m *MemoClient) Health(ctx context.Context) (bool, error) {
	var result MemoHealthResult
	err := m.c.Get(ctx, "/service/memo/health", &result)
	if err != nil {
		return false, err
	}
	return result.Healthy, nil
}
