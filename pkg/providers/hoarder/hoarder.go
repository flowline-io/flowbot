package hoarder

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"strconv"
	"time"
)

const (
	ID          = "hoarder"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Hoarder struct {
	c      *resty.Client
	apiKey string
}

func NewHoarder(endpoint string, apiKey string) *Hoarder {
	v := &Hoarder{apiKey: apiKey}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}

func (i *Hoarder) GetAllBookmarks(limit int) (*BookmarksResponse, error) {
	resp, err := i.c.R().SetAuthToken(i.apiKey).
		SetResult(&BookmarksResponse{}).
		SetQueryParam("limit", strconv.Itoa(limit)).
		Get("/api/v1/bookmarks")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get all bookmarks: %s", resp.String())
	}

	return resp.Result().(*BookmarksResponse), nil
}

func (i *Hoarder) GetAllTags() (*TagsResponse, error) {
	resp, err := i.c.R().SetAuthToken(i.apiKey).
		SetResult(&TagsResponse{}).
		Get("/api/v1/tags")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get all tags: %s", resp.String())
	}

	return resp.Result().(*TagsResponse), nil
}

func (i *Hoarder) AttachTagsToBookmark(bookmarkId string, tags []string) (*AttachedResponse, error) {
	var list []map[string]any
	for _, v := range tags {
		list = append(list, map[string]any{"tagName": v})
	}
	resp, err := i.c.R().SetAuthToken(i.apiKey).
		SetResult(&AttachedResponse{}).
		SetBody(map[string]any{
			"tags": list,
		}).
		Post(fmt.Sprintf("/api/v1/bookmarks/%s/tags", bookmarkId))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to attach tags to bookmark %s tags %v: %s", bookmarkId, tags, resp.String())
	}

	return resp.Result().(*AttachedResponse), nil
}
