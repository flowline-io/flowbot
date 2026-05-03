package karakeep

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	bm "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/stretchr/testify/assert"
)

func TestKarakeepConformance(t *testing.T) {
	conformance.RunBookmarkConformance(t, func(t *testing.T, cfg conformance.BookmarkConfig) bm.Service {
		c := &fakeClient{
			listResp:  cfgToListResponse(cfg),
			listErr:   cfg.ListErr,
			getItem:   cfgToProviderBookmark(cfg.GetItem),
			getErr:    cfg.GetErr,
			created:   cfgToProviderBookmark(cfg.CreateItem),
			createErr: cfg.CreateErr,
			deleteErr: cfg.DeleteErr,
			archiveResp: func() bool {
				if cfg.ArchiveResult != nil {
					return *cfg.ArchiveResult
				}
				return true
			}(),
			archiveErr:    cfg.ArchiveErr,
			searchResp:    cfgToSearchResponse(cfg),
			searchErr:     cfg.SearchErr,
			attachTagsErr: cfg.AttachTagsErr,
			detachTagsErr: cfg.DetachTagsErr,
			checkURLResp:  cfgToCheckURLResp(cfg),
			checkURLErr:   cfg.CheckURLErr,
		}
		a := NewWithClient(c).(*Adapter)
		a.cursorSecret = conformance.CursorSecret
		a.now = conformance.TestTime
		return a
	})
}

func cfgToListResponse(cfg conformance.BookmarkConfig) *provider.BookmarksResponse {
	if cfg.ListErr != nil {
		return nil
	}
	bookmarks := make([]provider.Bookmark, 0, len(cfg.ListItems))
	for _, item := range cfg.ListItems {
		bookmarks = append(bookmarks, abilityBookmarkToProvider(item))
	}
	return &provider.BookmarksResponse{
		Bookmarks:  bookmarks,
		NextCursor: cfg.ListNextCursor,
	}
}

func cfgToSearchResponse(cfg conformance.BookmarkConfig) *provider.BookmarksResponse {
	if cfg.SearchErr != nil {
		return nil
	}
	bookmarks := make([]provider.Bookmark, 0, len(cfg.SearchItems))
	for _, item := range cfg.SearchItems {
		bookmarks = append(bookmarks, abilityBookmarkToProvider(item))
	}
	return &provider.BookmarksResponse{
		Bookmarks:  bookmarks,
		NextCursor: cfg.SearchNextCursor,
	}
}

func cfgToProviderBookmark(item *ability.Bookmark) *provider.Bookmark {
	if item == nil {
		return nil
	}
	title := item.Title
	return &provider.Bookmark{
		Id:        item.ID,
		CreatedAt: "2024-01-01T00:00:00Z",
		Title:     &title,
		Content: provider.BookmarkContent{
			Url:   item.URL,
			Title: &title,
		},
	}
}

func abilityBookmarkToProvider(item *ability.Bookmark) provider.Bookmark {
	title := item.Title
	return provider.Bookmark{
		Id:        item.ID,
		CreatedAt: "2024-01-01T00:00:00Z",
		Title:     &title,
		Content: provider.BookmarkContent{
			Url:   item.URL,
			Title: &title,
		},
	}
}

func cfgToCheckURLResp(cfg conformance.BookmarkConfig) *string {
	if cfg.CheckURLErr != nil || !cfg.CheckURLExists {
		return nil
	}
	id := cfg.CheckURLID
	return &id
}

func TestConformanceFakeClientInterfaces(t *testing.T) {
	var _ client = (*fakeClient)(nil)
}

func TestTestBookmarkHelper(t *testing.T) {
	b := testBookmark("1", "https://example.com")
	assert.Equal(t, "1", b.Id)
	assert.Equal(t, "https://example.com", b.Content.Url)
}
