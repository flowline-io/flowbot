package hoarder

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/go-resty/resty/v2"
)

const (
	ID          = "hoarder"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Hoarder struct {
	c *resty.Client
}

func GetClient() *Hoarder {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, ApikeyKey)

	return NewHoarder(endpoint.String(), apiKey.String())
}

func NewHoarder(endpoint string, apiKey string) *Hoarder {
	v := &Hoarder{}

	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)
	v.c.SetAuthToken(apiKey)

	return v
}

func (i *Hoarder) GetAllBookmarks(query *BookmarksQuery) (*BookmarksResponse, error) {
	request := i.c.R().SetResult(&BookmarksResponse{})

	if query == nil {
		query = &BookmarksQuery{Limit: MaxPageSize}
	}

	if query.Limit > 0 {
		request.SetQueryParam("limit", fmt.Sprintf("%d", query.Limit))
	}
	if query.Archived {
		request.SetQueryParam("archived", fmt.Sprintf("%t", query.Archived))
	}
	if query.Favourited {
		request.SetQueryParam("favourited", fmt.Sprintf("%t", query.Favourited))
	}
	if query.Cursor != "" {
		request.SetQueryParam("cursor", query.Cursor)
	}

	resp, err := request.Get("/bookmarks")
	if err != nil {
		return nil, fmt.Errorf("failed to get all bookmarks: %w", err)
	}

	result := resp.Result().(*BookmarksResponse)
	if result == nil {
		result = &BookmarksResponse{Bookmarks: make([]Bookmark, 0)}
	}
	return result, nil
}

func (i *Hoarder) GetAllTags() ([]Tag, error) {
	resp, err := i.c.R().
		SetResult(&TagsResponse{}).
		Get("/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to get all tags: %w", err)
	}

	result := resp.Result().(*TagsResponse)
	return result.Tags, nil
}

func (i *Hoarder) AttachTagsToBookmark(bookmarkId string, tags []string) ([]string, error) {
	var list []BookmarkTagRequest
	for _, tag := range tags {
		list = append(list, BookmarkTagRequest{
			TagName: tag,
		})
	}

	resp, err := i.c.R().
		SetResult(&AttachTagsResponse{}).
		SetBody(list).
		Post(fmt.Sprintf("/bookmarks/%s/tags", bookmarkId))
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to bookmark: %w", err)
	}

	result := resp.Result().(*AttachTagsResponse)
	return result.Attached, nil
}

func (i *Hoarder) DetachTagsToBookmark(bookmarkId string, tags []string) ([]string, error) {
	var list []BookmarkTagRequest
	for _, tag := range tags {
		list = append(list, BookmarkTagRequest{
			TagName: tag,
		})
	}

	resp, err := i.c.R().
		SetResult(&DetachTagsResponse{}).
		SetBody(list).
		Delete(fmt.Sprintf("/bookmarks/%s/tags", bookmarkId))
	if err != nil {
		return nil, fmt.Errorf("failed to detach tags to bookmark: %w", err)
	}

	result := resp.Result().(*DetachTagsResponse)
	return result.Detached, nil
}

func (i *Hoarder) ArchiveBookmark(id string) (bool, error) {
	resp, err := i.c.R().
		SetResult(&ArchiveResponse{}).
		SetBody(map[string]bool{
			"archived": true,
		}).
		Patch(fmt.Sprintf("/bookmarks/%s", id))
	if err != nil {
		return false, fmt.Errorf("failed to archive bookmark: %w", err)
	}

	result := resp.Result().(*ArchiveResponse)
	return result.Archived, nil
}

func (i *Hoarder) CreateBookmark(url string) (*Bookmark, error) {
	resp, err := i.c.R().
		SetResult(&Bookmark{}).
		SetBody(map[string]string{
			"type": "link",
			"url":  url,
		}).
		Post("/bookmarks")
	if err != nil {
		return nil, fmt.Errorf("failed to create bookmark: %w", err)
	}

	return resp.Result().(*Bookmark), nil
}
