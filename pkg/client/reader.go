package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/validate"
	rssClient "miniflux.app/v2/client"
)

// ReaderClient provides access to the reader API.
type ReaderClient struct {
	c *Client
}

// ListFeedsQuery contains query parameters for listing feeds.
type ListFeedsQuery struct{}

// ListFeeds returns all feeds.
func (r *ReaderClient) ListFeeds(ctx context.Context) (rssClient.Feeds, error) {
	var result rssClient.Feeds
	err := r.c.Get(ctx, "/service/reader", &result)
	return result, err
}

// GetFeed returns a single feed by ID.
func (r *ReaderClient) GetFeed(ctx context.Context, id int64) (*rssClient.Feed, error) {
	var result rssClient.Feed
	path := fmt.Sprintf("/service/reader/%d", id)
	err := r.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateFeedRequest contains parameters for creating a feed.
type CreateFeedRequest struct {
	FeedURL    string `json:"feed_url"`
	CategoryID int64  `json:"category_id"`
}

// CreateFeedResult contains the result of creating a feed.
type CreateFeedResult struct {
	ID int64 `json:"id"`
}

// CreateFeed creates a new feed.
func (r *ReaderClient) CreateFeed(ctx context.Context, req *CreateFeedRequest) (*CreateFeedResult, error) {
	if req.FeedURL == "" {
		return nil, fmt.Errorf("feed_url is required")
	}
	if _, err := validate.ValidateVar(req.FeedURL, validate.TagURL); err != nil {
		return nil, fmt.Errorf("invalid feed_url: %w", err)
	}

	var result CreateFeedResult
	err := r.c.Post(ctx, "/service/reader", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateFeedRequest contains parameters for updating a feed.
type UpdateFeedRequest struct {
	Title                       string `json:"title,omitempty"`
	FeedURL                     string `json:"feed_url,omitempty"`
	SiteURL                     string `json:"site_url,omitempty"`
	ScraperRules                string `json:"scraper_rules,omitempty"`
	RewriteRules                string `json:"rewrite_rules,omitempty"`
	UrlRewriteRules             string `json:"urlrewrite_rules,omitempty"`
	BlocklistRules              string `json:"blocklist_rules,omitempty"`
	KeeplistRules               string `json:"keeplist_rules,omitempty"`
	BlockFilterEntryRules       string `json:"block_filter_entry_rules,omitempty"`
	KeepFilterEntryRules        string `json:"keep_filter_entry_rules,omitempty"`
	UserAgent                   string `json:"user_agent,omitempty"`
	Cookie                      string `json:"cookie,omitempty"`
	Username                    string `json:"username,omitempty"`
	Password                    string `json:"password,omitempty"`
	Crawler                     *bool  `json:"crawler,omitempty"`
	IgnoreHTTPCache             *bool  `json:"ignore_http_cache,omitempty"`
	AllowSelfSignedCertificates *bool  `json:"allow_self_signed_certificates,omitempty"`
	FetchViaProxy               *bool  `json:"fetch_via_proxy,omitempty"`
	IgnoreEntryUpdates          *bool  `json:"ignore_entry_updates,omitempty"`
	DisableHTTP2                *bool  `json:"disable_http2,omitempty"`
	HideGlobally                *bool  `json:"hide_globally,omitempty"`
	Disabled                    *bool  `json:"disabled,omitempty"`
}

// UpdateFeed updates an existing feed.
func (r *ReaderClient) UpdateFeed(ctx context.Context, id int64, req *UpdateFeedRequest) (*rssClient.Feed, error) {
	var result rssClient.Feed
	path := fmt.Sprintf("/service/reader/%d", id)
	err := r.c.Patch(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RefreshFeedResult contains the result of refreshing a feed.
type RefreshFeedResult struct {
	Success bool `json:"success"`
}

// RefreshFeed triggers a refresh of a feed.
func (r *ReaderClient) RefreshFeed(ctx context.Context, id int64) (*RefreshFeedResult, error) {
	var result RefreshFeedResult
	path := fmt.Sprintf("/service/reader/%d/refresh", id)
	err := r.c.Post(ctx, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListEntriesQuery contains query parameters for listing entries.
type ListEntriesQuery struct {
	Status     string
	Limit      int
	Offset     int
	Order      string
	Direction  string
	Starred    bool
	FeedID     int64
	CategoryID int64
}

// ListEntries returns entries with optional filtering.
func (r *ReaderClient) ListEntries(ctx context.Context, query *ListEntriesQuery) (*rssClient.EntryResultSet, error) {
	path := "/service/reader/entries"
	params := url.Values{}

	if query != nil {
		if query.Status != "" {
			params.Set("status", query.Status)
		}
		if query.Limit > 0 {
			params.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Offset > 0 {
			params.Set("offset", strconv.Itoa(query.Offset))
		}
		if query.Order != "" {
			params.Set("order", query.Order)
		}
		if query.Direction != "" {
			params.Set("direction", query.Direction)
		}
		if query.Starred {
			params.Set("starred", "true")
		}
		if query.FeedID > 0 {
			params.Set("feed_id", strconv.FormatInt(query.FeedID, 10))
		}
		if query.CategoryID > 0 {
			params.Set("category_id", strconv.FormatInt(query.CategoryID, 10))
		}
	}

	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	var result rssClient.EntryResultSet
	err := r.c.Get(ctx, path, &result)
	return &result, err
}

// UpdateEntriesRequest contains parameters for updating entries status.
type UpdateEntriesRequest struct {
	EntryIDs []int64 `json:"entry_ids"`
	Status   string  `json:"status"`
}

// UpdateEntriesResult contains the result of updating entries.
type UpdateEntriesResult struct {
	Success bool `json:"success"`
}

// UpdateEntriesStatus updates the status of multiple entries.
func (r *ReaderClient) UpdateEntriesStatus(ctx context.Context, req *UpdateEntriesRequest) (*UpdateEntriesResult, error) {
	if len(req.EntryIDs) == 0 {
		return nil, fmt.Errorf("entry_ids is required")
	}
	if req.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	var result UpdateEntriesResult
	err := r.c.Patch(ctx, "/service/reader/entries", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetFeedEntriesQuery contains query parameters for getting feed entries.
type GetFeedEntriesQuery struct {
	Status    string
	Limit     int
	Offset    int
	Order     string
	Direction string
	Starred   bool
}

// GetFeedEntries returns entries for a specific feed.
func (r *ReaderClient) GetFeedEntries(ctx context.Context, feedID int64, query *GetFeedEntriesQuery) (*rssClient.EntryResultSet, error) {
	path := fmt.Sprintf("/service/reader/%d/entries", feedID)
	params := url.Values{}

	if query != nil {
		if query.Status != "" {
			params.Set("status", query.Status)
		}
		if query.Limit > 0 {
			params.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Offset > 0 {
			params.Set("offset", strconv.Itoa(query.Offset))
		}
		if query.Order != "" {
			params.Set("order", query.Order)
		}
		if query.Direction != "" {
			params.Set("direction", query.Direction)
		}
		if query.Starred {
			params.Set("starred", "true")
		}
	}

	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	var result rssClient.EntryResultSet
	err := r.c.Get(ctx, path, &result)
	return &result, err
}
