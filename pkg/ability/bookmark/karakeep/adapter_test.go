package karakeep

import (
	"errors"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	bm "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	listQuery *provider.BookmarksQuery
	listResp  *provider.BookmarksResponse
	listErr   error
	created   *provider.Bookmark
	createErr error
}

func (f *fakeClient) GetAllBookmarks(query *provider.BookmarksQuery) (*provider.BookmarksResponse, error) {
	f.listQuery = query
	return f.listResp, f.listErr
}

func (f *fakeClient) GetBookmark(id string) (*provider.Bookmark, error) {
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
	return true, nil
}

func (f *fakeClient) SearchBookmarks(query *provider.SearchBookmarksQuery) (*provider.BookmarksResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeClient) AttachTagsToBookmark(bookmarkID string, tags []string) ([]string, error) {
	return tags, nil
}

func (f *fakeClient) DetachTagsToBookmark(bookmarkID string, tags []string) ([]string, error) {
	return tags, nil
}

func (f *fakeClient) CheckUrlExists(url string) (*string, error) {
	id := "bookmark-id"
	return &id, nil
}

func TestListConvertsBookmarksAndCursor(t *testing.T) {
	client := &fakeClient{listResp: &provider.BookmarksResponse{
		Bookmarks:  []provider.Bookmark{*testBookmark("1", "https://example.com")},
		NextCursor: "provider-next",
	}}
	adapter := NewWithClient(client).(*Adapter)
	adapter.now = func() time.Time { return time.Unix(1700000000, 0) }

	result, err := 	adapter.List(t.Context(), &bm.ListQuery{Page: ability.PageRequest{Limit: 20}})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "1", result.Items[0].ID)
	require.Equal(t, "provider-next", decodeTestCursor(t, adapter, result.Page.NextCursor).ProviderCursor)
	require.Equal(t, 20, client.listQuery.Limit)
}

func TestListDecodesOpaqueCursor(t *testing.T) {
	client := &fakeClient{listResp: &provider.BookmarksResponse{}}
	adapter := NewWithClient(client).(*Adapter)
	adapter.now = func() time.Time { return time.Unix(1700000000, 0) }
	cursor, err := ability.EncodeCursor(adapter.cursorSecret, ability.CursorPayload{ProviderCursor: "provider-current"})
	require.NoError(t, err)

	_, err = 	adapter.List(t.Context(), &bm.ListQuery{Page: ability.PageRequest{Cursor: cursor}})
	require.NoError(t, err)
	require.Equal(t, "provider-current", client.listQuery.Cursor)
}

func TestCreateWrapsProviderError(t *testing.T) {
	adapter := NewWithClient(&fakeClient{createErr: errors.New("boom")})

	_, err := 	adapter.Create(t.Context(), "https://example.com")
	require.Error(t, err)
	require.True(t, errors.Is(err, types.ErrProvider))
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
