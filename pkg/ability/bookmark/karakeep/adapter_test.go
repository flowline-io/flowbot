package karakeep

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	listQuery     *provider.BookmarksQuery
	listResp      *provider.BookmarksResponse
	listErr       error
	getItem       *provider.Bookmark
	getErr        error
	created       *provider.Bookmark
	createErr     error
	archiveResp   bool
	archiveErr    error
	searchQuery   *provider.SearchBookmarksQuery
	searchResp    *provider.BookmarksResponse
	searchErr     error
	attachTagsErr error
	detachTagsErr error
	checkURLResp  *string
	checkURLErr   error
	deleteErr     error
}

func (f *fakeClient) GetAllBookmarks(query *provider.BookmarksQuery) (*provider.BookmarksResponse, error) {
	f.listQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp == nil {
		f.listResp = &provider.BookmarksResponse{}
	}
	return f.listResp, nil
}

func (f *fakeClient) GetBookmark(id string) (*provider.Bookmark, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getItem != nil {
		return f.getItem, nil
	}
	return testBookmark(id, "https://example.com"), nil
}

func (f *fakeClient) CreateBookmark(url string) (*provider.Bookmark, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.created != nil {
		return f.created, nil
	}
	return testBookmark("created", url), nil
}

func (f *fakeClient) ArchiveBookmark(id string) (bool, error) {
	if f.archiveErr != nil {
		return false, f.archiveErr
	}
	return f.archiveResp, nil
}

func (f *fakeClient) SearchBookmarks(query *provider.SearchBookmarksQuery) (*provider.BookmarksResponse, error) {
	f.searchQuery = query
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchResp != nil {
		return f.searchResp, nil
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &provider.BookmarksResponse{}, nil
}

func (f *fakeClient) AttachTagsToBookmark(bookmarkID string, tags []string) ([]string, error) {
	if f.attachTagsErr != nil {
		return nil, f.attachTagsErr
	}
	return tags, nil
}

func (f *fakeClient) DetachTagsToBookmark(bookmarkID string, tags []string) ([]string, error) {
	if f.detachTagsErr != nil {
		return nil, f.detachTagsErr
	}
	return tags, nil
}

func (f *fakeClient) CheckUrlExists(url string) (*string, error) {
	if f.checkURLErr != nil {
		return nil, f.checkURLErr
	}
	if f.checkURLResp != nil {
		return f.checkURLResp, nil
	}
	id := "bookmark-id"
	return &id, nil
}

func testBookmark(id string, url string) *provider.Bookmark {
	title := "Example"
	summary := "Summary"
	return &provider.Bookmark{
		Id:        id,
		CreatedAt: "2024-01-01T00:00:00Z",
		Title:     &title,
		Summary:   &summary,
		Content: provider.BookmarkContent{
			Url:   url,
			Title: &title,
		},
		Tags: []provider.BookmarkTagsInner{{Name: "go"}},
	}
}

func decodeTestCursor(t *testing.T, adapter *Adapter, cursor string) ability.CursorPayload {
	t.Helper()
	payload, err := ability.DecodeCursor(adapter.cursorSecret, cursor, adapter.now())
	require.NoError(t, err)
	return payload
}
