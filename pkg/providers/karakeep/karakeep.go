package karakeep

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID          = "karakeep"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Karakeep struct {
	c *resty.Client
}

func GetClient() *Karakeep {
	endpoint, err := providers.GetConfig(ID, EndpointKey)
	if err != nil {
		flog.Warn("karakeep provider config error: %v", err)
	}
	apiKey, err := providers.GetConfig(ID, ApikeyKey)
	if err != nil {
		flog.Warn("karakeep provider config error: %v", err)
	}

	return NewKarakeep(endpoint.String(), apiKey.String())
}

func NewKarakeep(endpoint string, apiKey string) *Karakeep {
	v := &Karakeep{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetAuthToken(apiKey)

	return v
}

func (i *Karakeep) GetAllBookmarks(query *BookmarksQuery) (*BookmarksResponse, error) {
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

func (i *Karakeep) GetAllTags() ([]Tag, error) {
	resp, err := i.c.R().
		SetResult(&TagsResponse{}).
		Get("/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to get all tags: %w", err)
	}

	result := resp.Result().(*TagsResponse)
	return result.Tags, nil
}

func (i *Karakeep) AttachTagsToBookmark(bookmarkId string, tags []string) ([]string, error) {
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

func (i *Karakeep) DetachTagsToBookmark(bookmarkId string, tags []string) ([]string, error) {
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

func (i *Karakeep) ArchiveBookmark(id string) (bool, error) {
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

func (i *Karakeep) CreateBookmark(url string) (*Bookmark, error) {
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

// GetBookmark retrieves a single bookmark by its ID.  This corresponds
// to GET /bookmarks/:bookmarkId in the karakeep API and returns the
// full bookmark object, including fields that are not present in the
// list response (summarizationStatus, source, userId, etc.).
//
// The caller is responsible for handling a nil result or any error
// produced by the underlying HTTP client.
func (i *Karakeep) GetBookmark(id string) (*Bookmark, error) {
	resp, err := i.c.R().
		SetResult(&Bookmark{}).
		Get(fmt.Sprintf("/bookmarks/%s", id))
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmark %s: %w", id, err)
	}

	result := resp.Result().(*Bookmark)
	return result, nil
}

func (i *Karakeep) CheckUrlExists(url string) (*string, error) {
	resp, err := i.c.R().
		SetResult(&CheckUrlResponse{}).
		SetQueryParam("url", url).
		Get("/bookmarks/check-url")
	if err != nil {
		return nil, fmt.Errorf("failed to check URL existence: %w", err)
	}

	result := resp.Result().(*CheckUrlResponse)
	return result.BookmarkId, nil
}

func (i *Karakeep) SearchBookmarks(query *SearchBookmarksQuery) (*BookmarksResponse, error) {
	request := i.c.R().SetResult(&BookmarksResponse{})

	if query == nil {
		query = &SearchBookmarksQuery{}
	}

	if query.Q != "" {
		request.SetQueryParam("q", query.Q)
	}
	if query.SortOrder != "" {
		request.SetQueryParam("sortOrder", query.SortOrder)
	}
	if query.Limit > 0 {
		request.SetQueryParam("limit", fmt.Sprintf("%d", query.Limit))
	}
	if query.Cursor != "" {
		request.SetQueryParam("cursor", query.Cursor)
	}
	if query.IncludeContent {
		request.SetQueryParam("includeContent", "true")
	}

	resp, err := request.Get("/bookmarks/search")
	if err != nil {
		return nil, fmt.Errorf("failed to search bookmarks: %w", err)
	}

	result := resp.Result().(*BookmarksResponse)
	if result == nil {
		result = &BookmarksResponse{Bookmarks: make([]Bookmark, 0)}
	}
	return result, nil
}
