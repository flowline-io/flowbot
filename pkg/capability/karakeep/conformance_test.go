package karakeep

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/capability"

	"github.com/flowline-io/flowbot/pkg/capability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
)

func TestKarakeepConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"runs bookmark conformance test suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conformance.RunBookmarkConformance(t, func(_ *testing.T, cfg conformance.BookmarkConfig) conformance.BookmarkService {
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
				a, ok := NewWithClient(c).(*Adapter)
				if !ok {
					t.Fatal("unexpected type")
				}
				a.cursorSecret = conformance.CursorSecret
				a.now = conformance.TestTime
				return a
			})
		})
	}
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

func cfgToProviderBookmark(item *capability.Bookmark) *provider.Bookmark {
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

func abilityBookmarkToProvider(item *capability.Bookmark) provider.Bookmark {
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
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"fake client satisfies client interface"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ client = (*fakeClient)(nil)
		})
	}
}

func TestTestBookmarkHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"testBookmark helper returns correct id and url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := testBookmark("1", "https://example.com")
			assert.Equal(t, "1", b.Id)
			assert.Equal(t, "https://example.com", b.Content.Url)
		})
	}
}
