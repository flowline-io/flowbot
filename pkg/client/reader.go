package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/validate"
)

// ReaderClient provides access to the reader API.
type ReaderClient struct {
	c *Client
}

// ListFeeds returns all feeds.
func (r *ReaderClient) ListFeeds(ctx context.Context) ([]*capability.Feed, error) {
	var result []*capability.Feed
	err := r.c.Get(ctx, "/service/miniflux", &result)
	return result, err
}

// GetFeed returns a single feed by ID by listing feeds and filtering.
// The server does not expose a dedicated get-feed endpoint.
func (r *ReaderClient) GetFeed(ctx context.Context, id int64) (*capability.Feed, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be positive, got %d", id)
	}
	feeds, err := r.ListFeeds(ctx)
	if err != nil {
		return nil, err
	}
	for _, feed := range feeds {
		if feed != nil && feed.ID == id {
			return feed, nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: "feed not found"}
}

// CreateFeedRequest contains parameters for creating a feed.
type CreateFeedRequest struct {
	FeedURL    string `json:"feed_url"`
	CategoryID int64  `json:"category_id"`
}

// CreateFeed creates a new feed.
func (r *ReaderClient) CreateFeed(ctx context.Context, req *CreateFeedRequest) (*capability.Feed, error) {
	if req == nil {
		return nil, fmt.Errorf("feed_url is required")
	}
	if req.FeedURL == "" {
		return nil, fmt.Errorf("feed_url is required")
	}
	if err := validate.ValidateVar(req.FeedURL, validate.TagURL); err != nil {
		return nil, fmt.Errorf("invalid feed_url: %w", err)
	}

	var result capability.Feed
	err := r.c.Post(ctx, "/service/miniflux", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListEntriesQuery contains query parameters for listing entries.
type ListEntriesQuery struct {
	Status string
	Limit  int
	Cursor string
	FeedID int64
}

// ListEntries returns entries with optional filtering.
func (r *ReaderClient) ListEntries(ctx context.Context, query *ListEntriesQuery) ([]*capability.Entry, error) {
	path := "/service/miniflux/entries"
	params := url.Values{}

	if query != nil {
		if query.Status != "" {
			params.Set("status", query.Status)
		}
		if query.Limit > 0 {
			params.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			params.Set("cursor", query.Cursor)
		}
		if query.FeedID > 0 {
			params.Set("feed_id", strconv.FormatInt(query.FeedID, 10))
		}
	}

	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	var result []*capability.Entry
	err := r.c.Get(ctx, path, &result)
	return result, err
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
	if req == nil || len(req.EntryIDs) == 0 {
		return nil, fmt.Errorf("entry_ids is required")
	}
	if req.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	var result UpdateEntriesResult
	err := r.c.Patch(ctx, "/service/miniflux/entries", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetFeedEntriesQuery contains query parameters for getting feed entries.
type GetFeedEntriesQuery struct {
	Status string
	Limit  int
	Cursor string
}

// GetFeedEntries returns entries for a specific feed via the shared entries endpoint.
func (r *ReaderClient) GetFeedEntries(ctx context.Context, feedID int64, query *GetFeedEntriesQuery) ([]*capability.Entry, error) {
	if feedID <= 0 {
		return nil, fmt.Errorf("feed_id must be positive, got %d", feedID)
	}
	listQuery := &ListEntriesQuery{FeedID: feedID}
	if query != nil {
		listQuery.Status = query.Status
		listQuery.Limit = query.Limit
		listQuery.Cursor = query.Cursor
	}
	return r.ListEntries(ctx, listQuery)
}
