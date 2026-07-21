package karakeep

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
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

func (f *fakeClient) GetAllBookmarks(_ context.Context, query *provider.BookmarksQuery) (*provider.BookmarksResponse, error) {
	f.listQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp == nil {
		f.listResp = &provider.BookmarksResponse{}
	}
	return f.listResp, nil
}

func (f *fakeClient) GetBookmark(_ context.Context, id string) (*provider.Bookmark, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getItem != nil {
		return f.getItem, nil
	}
	return testBookmark(id, "https://example.com"), nil
}

func (f *fakeClient) CreateBookmark(_ context.Context, url string) (*provider.Bookmark, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.created != nil {
		return f.created, nil
	}
	return testBookmark("created", url), nil
}

func (f *fakeClient) ArchiveBookmark(_ context.Context, _ string) (bool, error) {
	if f.archiveErr != nil {
		return false, f.archiveErr
	}
	return f.archiveResp, nil
}

func (f *fakeClient) SearchBookmarks(_ context.Context, query *provider.SearchBookmarksQuery) (*provider.BookmarksResponse, error) {
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

func (f *fakeClient) AttachTagsToBookmark(_ context.Context, _ string, tags []string) ([]string, error) {
	if f.attachTagsErr != nil {
		return nil, f.attachTagsErr
	}
	return tags, nil
}

func (f *fakeClient) DetachTagsToBookmark(_ context.Context, _ string, tags []string) ([]string, error) {
	if f.detachTagsErr != nil {
		return nil, f.detachTagsErr
	}
	return tags, nil
}

func (f *fakeClient) CheckUrlExists(_ context.Context, _ string) (*string, error) {
	if f.checkURLErr != nil {
		return nil, f.checkURLErr
	}
	if f.checkURLResp != nil {
		return f.checkURLResp, nil
	}
	id := "bookmark-id"
	return &id, nil
}

func testBookmark(id, url string) *provider.Bookmark {
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

func decodeTestCursor(t *testing.T, adapter *Adapter, cursor string) capability.CursorPayload {
	t.Helper()
	payload, err := capability.DecodeCursor(adapter.cursorSecret, cursor, adapter.now())
	require.NoError(t, err)
	return payload
}

func TestAdapter_HealthCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantOK  bool
		wantErr bool
	}{
		{name: "healthy", client: &fakeClient{}, wantOK: true},
		{name: "provider error", client: &fakeClient{listErr: assert.AnError}, wantErr: true},
		{name: "canceled context", client: &fakeClient{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			ctx := t.Context()
			if tt.name == "canceled context" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			ok, err := a.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				assert.False(t, ok)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestToBookmark_CreatedAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		createdAt string
		wantZero  bool
	}{
		{name: "valid RFC3339", createdAt: "2024-01-01T00:00:00Z", wantZero: false},
		{name: "empty createdAt skips parse", createdAt: "", wantZero: true},
		{name: "invalid createdAt yields zero time", createdAt: "not-a-time", wantZero: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toBookmark(&provider.Bookmark{
				Id:        "b-1",
				CreatedAt: tt.createdAt,
				Content:   provider.BookmarkContent{Url: "https://example.com"},
			})
			require.NotNil(t, got)
			if tt.wantZero {
				assert.True(t, got.CreatedAt.IsZero())
			} else {
				assert.False(t, got.CreatedAt.IsZero())
			}
		})
	}
}
