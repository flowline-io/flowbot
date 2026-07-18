package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/validate"
)

// BookmarkClient provides access to the bookmark API.
type BookmarkClient struct {
	c *Client
}

// ListBookmarksQuery contains query parameters for listing bookmarks.
type ListBookmarksQuery struct {
	Limit      int
	Cursor     string
	Archived   bool
	Favourited bool
}

// BookmarkListResult holds the paginated list response extracted from InvokeResult.
type BookmarkListResult struct {
	Items []*capability.Bookmark `json:"data"`
	Page  BookmarkPage           `json:"page"`
}

// BookmarkPage holds pagination metadata.
type BookmarkPage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
}

// BookmarkItemResult holds a single bookmark extracted from InvokeResult.
type BookmarkItemResult struct {
	Item capability.Bookmark `json:"data"`
}

// List returns all bookmarks with optional filtering.
func (b *BookmarkClient) List(ctx context.Context, query *ListBookmarksQuery) (*BookmarkListResult, error) {
	if query != nil {
		if err := validateListBookmarksQuery(query); err != nil {
			return nil, err
		}
	}

	path := "/service/karakeep"
	if query != nil {
		v := url.Values{}
		if query.Limit > 0 {
			v.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			v.Set("cursor", query.Cursor)
		}
		if query.Archived {
			v.Set("archived", "true")
		}
		if query.Favourited {
			v.Set("favourited", "true")
		}
		if len(v) > 0 {
			path = path + "?" + v.Encode()
		}
	}

	var result BookmarkListResult
	err := b.c.Get(ctx, path, &result)
	return &result, err
}

func validateListBookmarksQuery(query *ListBookmarksQuery) error {
	if query.Limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", query.Limit)
	}
	if query.Limit > validate.MaxSearchLimit {
		return fmt.Errorf("limit exceeds maximum of %d", validate.MaxSearchLimit)
	}
	if len(query.Cursor) > validate.QueryMaxLen {
		return fmt.Errorf("cursor exceeds maximum length of %d", validate.QueryMaxLen)
	}
	return nil
}

// Get returns a single bookmark by ID.
func (b *BookmarkClient) Get(ctx context.Context, id string) (*capability.Bookmark, error) {
	var result BookmarkItemResult
	path := fmt.Sprintf("/service/karakeep/%s", id)
	err := b.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// Create creates a new bookmark from a URL.
func (b *BookmarkClient) Create(ctx context.Context, bookmarkURL string) (*capability.Bookmark, error) {
	if err := validate.ValidateVar(bookmarkURL, validate.TagURL); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	var result BookmarkItemResult
	body := map[string]string{"url": bookmarkURL}
	err := b.c.Post(ctx, "/service/karakeep", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// ArchiveResult contains the result of archiving a bookmark.
type ArchiveResult struct {
	Archived bool `json:"archived"`
}

type bookmarkArchiveInvokeResult struct {
	Data ArchiveResult `json:"data"`
}

// Archive archives (or unarchives) a bookmark.
func (b *BookmarkClient) Archive(ctx context.Context, id string) (*ArchiveResult, error) {
	var result bookmarkArchiveInvokeResult
	path := fmt.Sprintf("/service/karakeep/%s", id)
	err := b.c.Patch(ctx, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// AttachTags attaches tags to a bookmark.
func (b *BookmarkClient) AttachTags(ctx context.Context, id string, tags []string) error {
	if err := validateTags(tags); err != nil {
		return err
	}

	path := fmt.Sprintf("/service/karakeep/%s/tags", id)
	body := map[string]any{"tags": tags}
	return b.c.Post(ctx, path, body, nil)
}

func validateTags(tags []string) error {
	if len(tags) > validate.MaxTagsCount {
		return fmt.Errorf("tag count exceeds maximum of %d", validate.MaxTagsCount)
	}
	for _, tag := range tags {
		if len(tag) < validate.MinTagLen {
			return fmt.Errorf("tag cannot be empty")
		}
		if len(tag) > validate.TagMaxLen {
			return fmt.Errorf("tag length exceeds maximum of %d characters", validate.TagMaxLen)
		}
	}
	return nil
}

// DetachTags detaches tags from a bookmark.
func (b *BookmarkClient) DetachTags(ctx context.Context, id string, tags []string) error {
	if err := validateTags(tags); err != nil {
		return err
	}

	path := fmt.Sprintf("/service/karakeep/%s/tags", id)
	body := map[string]any{"tags": tags}
	return b.c.Delete(ctx, path, body, nil)
}

// CheckUrlResult contains the result of checking if a URL exists.
type CheckUrlResult struct {
	Exists bool   `json:"exists"`
	ID     string `json:"id"`
}

type bookmarkCheckURLInvokeResult struct {
	Data CheckUrlResult `json:"data"`
}

// CheckUrl checks if a URL is already bookmarked.
func (b *BookmarkClient) CheckUrl(ctx context.Context, bookmarkURL string) (*CheckUrlResult, error) {
	if err := validate.ValidateVar(bookmarkURL, validate.TagURL); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	var result bookmarkCheckURLInvokeResult
	path := "/service/karakeep/check-url?" + url.Values{"url": {bookmarkURL}}.Encode()
	err := b.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// SearchBookmarksQuery contains query parameters for searching bookmarks.
type SearchBookmarksQuery struct {
	Q              string
	SortOrder      string
	Limit          int
	Cursor         string
	IncludeContent bool
}

// Search searches bookmarks with the given query.
func (b *BookmarkClient) Search(ctx context.Context, query *SearchBookmarksQuery) (*BookmarkListResult, error) {
	if query != nil {
		if err := validateSearchBookmarksQuery(query); err != nil {
			return nil, err
		}
	}

	path := "/service/karakeep/search"
	if query != nil {
		v := url.Values{}
		if query.Q != "" {
			v.Set("q", query.Q)
		}
		if query.SortOrder != "" {
			v.Set("sort_order", query.SortOrder)
		}
		if query.Limit > 0 {
			v.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			v.Set("cursor", query.Cursor)
		}
		if query.IncludeContent {
			v.Set("includeContent", "true")
		}
		if len(v) > 0 {
			path = path + "?" + v.Encode()
		}
	}

	var result BookmarkListResult
	err := b.c.Get(ctx, path, &result)
	return &result, err
}

func validateSearchBookmarksQuery(query *SearchBookmarksQuery) error {
	if query.Q == "" {
		return fmt.Errorf("search query is required")
	}
	if len(query.Q) > validate.QueryMaxLen {
		return fmt.Errorf("query exceeds maximum length of %d", validate.QueryMaxLen)
	}
	if query.Limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", query.Limit)
	}
	if query.Limit > validate.MaxSearchLimit {
		return fmt.Errorf("limit exceeds maximum of %d", validate.MaxSearchLimit)
	}
	if len(query.Cursor) > validate.QueryMaxLen {
		return fmt.Errorf("cursor exceeds maximum length of %d", validate.QueryMaxLen)
	}
	return nil
}
