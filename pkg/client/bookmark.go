package client

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers/karakeep"
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

// List returns all bookmarks with optional filtering.
func (b *BookmarkClient) List(ctx context.Context, query *ListBookmarksQuery) (*karakeep.BookmarksResponse, error) {
	if query != nil {
		if err := validateListBookmarksQuery(query); err != nil {
			return nil, err
		}
	}

	path := "/service/bookmark"
	if query != nil {
		path = fmt.Sprintf("/service/bookmark?limit=%d", query.Limit)
		if query.Cursor != "" {
			path += fmt.Sprintf("&cursor=%s", query.Cursor)
		}
		if query.Archived {
			path += "&archived=true"
		}
		if query.Favourited {
			path += "&favourited=true"
		}
	}

	var result karakeep.BookmarksResponse
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
func (b *BookmarkClient) Get(ctx context.Context, id string) (*karakeep.Bookmark, error) {
	var result karakeep.Bookmark
	path := fmt.Sprintf("/service/bookmark/%s", id)
	err := b.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new bookmark from a URL.
func (b *BookmarkClient) Create(ctx context.Context, url string) (*karakeep.Bookmark, error) {
	if _, err := validate.ValidateVar(url, validate.TagURL); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	var result karakeep.Bookmark
	body := map[string]string{"url": url}
	err := b.c.Post(ctx, "/service/bookmark", body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ArchiveResult contains the result of archiving a bookmark.
type ArchiveResult struct {
	Archived bool `json:"archived"`
}

// Archive archives (or unarchives) a bookmark.
func (b *BookmarkClient) Archive(ctx context.Context, id string) (*ArchiveResult, error) {
	var result ArchiveResult
	path := fmt.Sprintf("/service/bookmark/%s", id)
	err := b.c.Patch(ctx, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AttachTagsResult contains the result of attaching tags to a bookmark.
type AttachTagsResult struct {
	Attached []string `json:"attached"`
}

// AttachTags attaches tags to a bookmark.
func (b *BookmarkClient) AttachTags(ctx context.Context, id string, tags []string) (*AttachTagsResult, error) {
	if err := validateTags(tags); err != nil {
		return nil, err
	}

	var result AttachTagsResult
	path := fmt.Sprintf("/service/bookmark/%s/tags", id)
	err := b.c.Post(ctx, path, tags, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
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

// DetachTagsResult contains the result of detaching tags from a bookmark.
type DetachTagsResult struct {
	Detached []string `json:"detached"`
}

// DetachTags detaches tags from a bookmark.
func (b *BookmarkClient) DetachTags(ctx context.Context, id string, tags []string) (*DetachTagsResult, error) {
	if err := validateTags(tags); err != nil {
		return nil, err
	}

	var result DetachTagsResult
	path := fmt.Sprintf("/service/bookmark/%s/tags", id)
	err := b.c.Delete(ctx, path, tags, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
