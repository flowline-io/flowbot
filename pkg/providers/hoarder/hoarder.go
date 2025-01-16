package hoarder

import (
	"context"
	"fmt"
	openapi "github.com/flowline-io/sdk-hoarder-api"
)

const (
	ID          = "hoarder"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Hoarder struct {
	ctx context.Context
	c   *openapi.APIClient
}

func NewHoarder(endpoint string, apiKey string) *Hoarder {
	v := &Hoarder{}

	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{{URL: endpoint}}
	v.c = openapi.NewAPIClient(cfg)

	ctx := context.WithValue(context.Background(), openapi.ContextServerIndex, 0)
	ctx = context.WithValue(ctx, openapi.ContextAccessToken, apiKey)
	v.ctx = ctx

	return v
}

func (i *Hoarder) GetAllBookmarks(limit int) ([]openapi.Bookmark, error) {
	resp, _, err := i.c.BookmarksAPI.BookmarksGet(i.ctx).Limit(float32(limit)).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get all bookmarks: %w", err)
	}

	return resp.Bookmarks, nil
}

func (i *Hoarder) GetAllTags() ([]openapi.Tag, error) {
	resp, _, err := i.c.TagsAPI.TagsGet(i.ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get all tags: %w", err)
	}

	return resp.Tags, nil
}

func (i *Hoarder) AttachTagsToBookmark(bookmarkId string, tags []string) ([]string, error) {
	var list []openapi.BookmarksBookmarkIdTagsPostRequestTagsInner
	for n := range tags {
		list = append(list, openapi.BookmarksBookmarkIdTagsPostRequestTagsInner{
			TagName: &tags[n],
		})
	}
	tagsReq := openapi.BookmarksBookmarkIdTagsPostRequest{}
	tagsReq.SetTags(list)
	resp, _, err := i.c.BookmarksAPI.BookmarksBookmarkIdTagsPost(i.ctx, bookmarkId).BookmarksBookmarkIdTagsPostRequest(tagsReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to bookmark: %w", err)
	}

	return resp.Attached, nil
}

func (i *Hoarder) ArchiveBookmark(id string) (bool, error) {
	archived := true
	resp, _, err := i.c.BookmarksAPI.BookmarksBookmarkIdPatch(i.ctx, id).BookmarksBookmarkIdPatchRequest(openapi.BookmarksBookmarkIdPatchRequest{
		Archived: &archived,
	}).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to get all bookmarks: %w", err)
	}

	return resp.Archived, nil
}
