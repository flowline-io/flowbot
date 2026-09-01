// Package confluence implements the Atlassian Confluence Cloud API provider.
package confluence

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID                 = "confluence"
	SiteURLKey         = "site_url"
	EmailKey           = "email"
	APITokenKey        = "api_token"
	WebhookTokenKey    = "webhook_token"
	DefaultSpaceKeyKey = "default_space_key"
	apiPathPrefix      = "/wiki/rest/api"
)

// Confluence is an HTTP client for the Confluence Cloud REST API.
type Confluence struct {
	c *resty.Client
}

// GetWebhookToken reads the inbound webhook query token from config.
func GetWebhookToken() string {
	tok, err := providers.GetConfig(ID, WebhookTokenKey)
	if err != nil {
		return ""
	}
	return tok.String()
}

// GetDefaultSpaceKey reads the optional default space key from config.
func GetDefaultSpaceKey() string {
	key, err := providers.GetConfig(ID, DefaultSpaceKeyKey)
	if err != nil {
		return ""
	}
	return key.String()
}

// GetClient builds a Confluence client from vendors.confluence config.
// Returns nil when site_url, email, or api_token is not configured.
func GetClient() *Confluence {
	siteURL, _ := providers.GetConfig(ID, SiteURLKey)
	email, _ := providers.GetConfig(ID, EmailKey)
	apiToken, _ := providers.GetConfig(ID, APITokenKey)
	if siteURL.String() == "" || email.String() == "" || apiToken.String() == "" {
		return nil
	}
	return NewConfluence(siteURL.String(), email.String(), apiToken.String())
}

// NewConfluence creates a Confluence API client.
func NewConfluence(siteURL, email, apiToken string) *Confluence {
	if siteURL == "" {
		return nil
	}
	base := strings.TrimRight(siteURL, "/") + apiPathPrefix
	c := utils.DefaultRestyClient()
	c.SetBaseURL(base)
	if email != "" && apiToken != "" {
		c.SetBasicAuth(email, apiToken)
	}
	return &Confluence{c: c}
}

func (c *Confluence) req(ctx context.Context) *resty.Request {
	return c.c.R().SetContext(ctx)
}

func checkStatus(resp *resty.Response, op string) error {
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return nil
	}
	if resp.StatusCode() == http.StatusNotFound {
		return types.Errorf(types.ErrNotFound, "confluence %s: not found", op)
	}
	return fmt.Errorf("confluence %s: unexpected status %d: %s", op, resp.StatusCode(), resp.String())
}

// ListSpaces returns spaces with pagination.
func (c *Confluence) ListSpaces(ctx context.Context, start, limit int) (*SpaceListResponse, error) {
	if limit <= 0 {
		limit = 25
	}
	resp := &SpaceListResponse{}
	r, err := c.req(ctx).
		SetResult(resp).
		SetQueryParam("start", fmt.Sprintf("%d", start)).
		SetQueryParam("limit", fmt.Sprintf("%d", limit)).
		Get("/space")
	if err != nil {
		return nil, fmt.Errorf("confluence list spaces: %w", err)
	}
	if err := checkStatus(r, "list spaces"); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListPages returns pages in a space.
func (c *Confluence) ListPages(ctx context.Context, spaceKey string, start, limit int) (*PageListResponse, error) {
	if spaceKey == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "space_key is required")
	}
	if limit <= 0 {
		limit = 25
	}
	resp := &PageListResponse{}
	r, err := c.req(ctx).
		SetResult(resp).
		SetQueryParam("spaceKey", spaceKey).
		SetQueryParam("type", "page").
		SetQueryParam("expand", "space").
		SetQueryParam("start", fmt.Sprintf("%d", start)).
		SetQueryParam("limit", fmt.Sprintf("%d", limit)).
		Get("/content")
	if err != nil {
		return nil, fmt.Errorf("confluence list pages: %w", err)
	}
	if err := checkStatus(r, "list pages"); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPage returns a page by id with optional body expansion.
func (c *Confluence) GetPage(ctx context.Context, pageID string, expandBody bool) (*Page, error) {
	if pageID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "page_id is required")
	}
	page := &Page{}
	req := c.req(ctx).SetResult(page).SetPathParam("pageID", pageID)
	if expandBody {
		req.SetQueryParam("expand", "body.storage,space,version")
	} else {
		req.SetQueryParam("expand", "space,version")
	}
	r, err := req.Get("/content/{pageID}")
	if err != nil {
		return nil, fmt.Errorf("confluence get page: %w", err)
	}
	if err := checkStatus(r, "get page"); err != nil {
		return nil, err
	}
	return page, nil
}

// GetPageContent returns the storage-format body of a page.
func (c *Confluence) GetPageContent(ctx context.Context, pageID string) (string, error) {
	page, err := c.GetPage(ctx, pageID, true)
	if err != nil {
		return "", err
	}
	if page.Body == nil || page.Body.Storage == nil {
		return "", nil
	}
	return page.Body.Storage.Value, nil
}

// SearchPages runs a CQL search for pages.
func (c *Confluence) SearchPages(ctx context.Context, cql string, start, limit int) (*SearchResponse, error) {
	if strings.TrimSpace(cql) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "cql is required")
	}
	if limit <= 0 {
		limit = 25
	}
	resp := &SearchResponse{}
	r, err := c.req(ctx).
		SetResult(resp).
		SetQueryParam("cql", cql).
		SetQueryParam("start", fmt.Sprintf("%d", start)).
		SetQueryParam("limit", fmt.Sprintf("%d", limit)).
		Get("/content/search")
	if err != nil {
		return nil, fmt.Errorf("confluence search pages: %w", err)
	}
	if err := checkStatus(r, "search pages"); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreatePage creates a page in a space with storage-format content.
func (c *Confluence) CreatePage(ctx context.Context, spaceKey, title, content string) (*Page, error) {
	if spaceKey == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "space_key is required")
	}
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	body := CreatePageRequest{
		Type:  "page",
		Title: title,
		Space: map[string]string{"key": spaceKey},
	}
	if content != "" {
		body.Body = map[string]StorageBody{
			"storage": {Value: content, Representation: "storage"},
		}
	}
	page := &Page{}
	r, err := c.req(ctx).SetResult(page).SetBody(body).Post("/content")
	if err != nil {
		return nil, fmt.Errorf("confluence create page: %w", err)
	}
	if err := checkStatus(r, "create page"); err != nil {
		return nil, err
	}
	return page, nil
}

// UpdatePage updates page title and/or storage-format content.
func (c *Confluence) UpdatePage(ctx context.Context, pageID, title, content string) (*Page, error) {
	if pageID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "page_id is required")
	}
	current, err := c.GetPage(ctx, pageID, true)
	if err != nil {
		return nil, err
	}
	version := 1
	if current.Version != nil && current.Version.Number > 0 {
		version = current.Version.Number + 1
	}
	updateTitle := title
	if updateTitle == "" {
		updateTitle = current.Title
	}
	reqBody := UpdatePageRequest{
		ID:      pageID,
		Type:    "page",
		Title:   updateTitle,
		Version: Version{Number: version},
	}
	if content != "" {
		reqBody.Body = map[string]StorageBody{
			"storage": {Value: content, Representation: "storage"},
		}
	}
	page := &Page{}
	r, err := c.req(ctx).SetResult(page).SetPathParam("pageID", pageID).SetBody(reqBody).Put("/content/{pageID}")
	if err != nil {
		return nil, fmt.Errorf("confluence update page: %w", err)
	}
	if err := checkStatus(r, "update page"); err != nil {
		return nil, err
	}
	return page, nil
}

// DeletePage deletes a page by id.
func (c *Confluence) DeletePage(ctx context.Context, pageID string) error {
	if pageID == "" {
		return types.Errorf(types.ErrInvalidArgument, "page_id is required")
	}
	r, err := c.req(ctx).SetPathParam("pageID", pageID).Delete("/content/{pageID}")
	if err != nil {
		return fmt.Errorf("confluence delete page: %w", err)
	}
	return checkStatus(r, "delete page")
}

// GetCurrentUser returns the authenticated user (health check).
func (c *Confluence) GetCurrentUser(ctx context.Context) (map[string]any, error) {
	result := map[string]any{}
	r, err := c.req(ctx).SetResult(&result).Get("/user/current")
	if err != nil {
		return nil, fmt.Errorf("confluence get current user: %w", err)
	}
	if err := checkStatus(r, "get current user"); err != nil {
		return nil, err
	}
	return result, nil
}
