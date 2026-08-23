package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ConfluenceClient provides access to the Confluence API via Flowbot server.
type ConfluenceClient struct {
	c *Client
}

// ConfluenceListResult holds a paginated list from InvokeResult.
type ConfluenceListResult[T any] struct {
	Data []T            `json:"data"`
	Page ConfluencePage `json:"page"`
}

// ConfluencePage holds pagination metadata.
type ConfluencePage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
}

// ConfluenceItemResult holds a single item from InvokeResult.
type ConfluenceItemResult[T any] struct {
	Data T `json:"data"`
}

// ListSpaces returns Confluence spaces.
func (c *ConfluenceClient) ListSpaces(ctx context.Context, limit int, cursor string) (*ConfluenceListResult[capability.ConfluenceSpace], error) {
	path := confluencePathWithPage("/service/confluence/spaces", limit, cursor)
	var result ConfluenceListResult[capability.ConfluenceSpace]
	err := c.c.Get(ctx, path, &result)
	return &result, err
}

// ListPages returns pages in a space.
func (c *ConfluenceClient) ListPages(ctx context.Context, spaceKey string, limit int, cursor string) (*ConfluenceListResult[capability.ConfluencePage], error) {
	if spaceKey == "" {
		return nil, fmt.Errorf("space_key is required")
	}
	base := fmt.Sprintf("/service/confluence/spaces/%s/pages", url.PathEscape(spaceKey))
	path := confluencePathWithPage(base, limit, cursor)
	var result ConfluenceListResult[capability.ConfluencePage]
	err := c.c.Get(ctx, path, &result)
	return &result, err
}

// GetPage returns a page by id.
func (c *ConfluenceClient) GetPage(ctx context.Context, pageID string) (*capability.ConfluencePage, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	var result ConfluenceItemResult[capability.ConfluencePage]
	path := fmt.Sprintf("/service/confluence/pages/%s", url.PathEscape(pageID))
	err := c.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetPageContent returns storage-format page content.
func (c *ConfluenceClient) GetPageContent(ctx context.Context, pageID string) (string, error) {
	if pageID == "" {
		return "", fmt.Errorf("page_id is required")
	}
	var result string
	path := fmt.Sprintf("/service/confluence/pages/%s/content", url.PathEscape(pageID))
	err := c.c.Get(ctx, path, &result)
	return result, err
}

// SearchPages searches pages with CQL.
func (c *ConfluenceClient) SearchPages(ctx context.Context, cql string, limit int, cursor string) (*ConfluenceListResult[capability.ConfluencePage], error) {
	if cql == "" {
		return nil, fmt.Errorf("cql is required")
	}
	v := url.Values{"cql": {cql}}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		v.Set("cursor", cursor)
	}
	var result ConfluenceListResult[capability.ConfluencePage]
	path := "/service/confluence/search?" + v.Encode()
	err := c.c.Get(ctx, path, &result)
	return &result, err
}

// CreatePageRequest is the body for creating a page.
type CreatePageRequest struct {
	SpaceKey string `json:"space_key,omitempty"`
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`
}

// CreatePage creates a new page.
func (c *ConfluenceClient) CreatePage(ctx context.Context, req *CreatePageRequest) (*capability.ConfluencePage, error) {
	if req == nil || req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	var result ConfluenceItemResult[capability.ConfluencePage]
	err := c.c.Post(ctx, "/service/confluence/pages", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdatePageRequest is the body for updating a page.
type UpdatePageRequest struct {
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

// UpdatePage updates a page.
func (c *ConfluenceClient) UpdatePage(ctx context.Context, pageID string, req *UpdatePageRequest) (*capability.ConfluencePage, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if req == nil {
		req = &UpdatePageRequest{}
	}
	var result ConfluenceItemResult[capability.ConfluencePage]
	path := fmt.Sprintf("/service/confluence/pages/%s", url.PathEscape(pageID))
	err := c.c.Patch(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeletePage deletes a page.
func (c *ConfluenceClient) DeletePage(ctx context.Context, pageID string) error {
	if pageID == "" {
		return fmt.Errorf("page_id is required")
	}
	path := fmt.Sprintf("/service/confluence/pages/%s", url.PathEscape(pageID))
	return c.c.Delete(ctx, path, nil, nil)
}

// Health checks Confluence connectivity.
func (c *ConfluenceClient) Health(ctx context.Context) (bool, error) {
	var result bool
	err := c.c.Get(ctx, "/service/confluence/health", &result)
	return result, err
}

func confluencePathWithPage(base string, limit int, cursor string) string {
	v := url.Values{}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		v.Set("cursor", cursor)
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
}
