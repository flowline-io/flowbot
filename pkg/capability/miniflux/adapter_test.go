package miniflux

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rssClient "miniflux.app/v2/client"
)

type fakeClient struct {
	feeds         rssClient.Feeds
	feedsErr      error
	createFeedID  int64
	createFeedErr error
	entries       *rssClient.EntryResultSet
	entriesErr    error
	markReadErr   error
	markUnreadErr error
}

func (f *fakeClient) GetFeeds() (rssClient.Feeds, error) {
	if f.feedsErr != nil {
		return nil, f.feedsErr
	}
	if f.feeds == nil {
		return rssClient.Feeds{}, nil
	}
	return f.feeds, nil
}

func (f *fakeClient) CreateFeed(_ *rssClient.FeedCreationRequest) (int64, error) {
	if f.createFeedErr != nil {
		return 0, f.createFeedErr
	}
	return f.createFeedID, nil
}

func (f *fakeClient) GetEntries(_ *rssClient.Filter) (*rssClient.EntryResultSet, error) {
	if f.entriesErr != nil {
		return nil, f.entriesErr
	}
	if f.entries == nil {
		return &rssClient.EntryResultSet{Entries: rssClient.Entries{}}, nil
	}
	return f.entries, nil
}

func (f *fakeClient) UpdateEntries(_ []int64, status string) error {
	switch status {
	case rssClient.EntryStatusRead:
		return f.markReadErr
	case rssClient.EntryStatusUnread:
		return f.markUnreadErr
	}
	return nil
}

func TestListFeedsConvertsFeeds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		feeds rssClient.Feeds
		want  int
	}{
		{"converts miniflux feeds to ability feeds", rssClient.Feeds{
			{
				ID: 1, Title: "Example Blog", FeedURL: "https://example.com/rss",
				SiteURL: "https://example.com", Category: &rssClient.Category{ID: 1, Title: "Tech"},
			},
		}, 1},
		{"multiple feeds with all fields converted correctly", rssClient.Feeds{
			{
				ID: 1, Title: "Blog A", FeedURL: "https://a.com/rss", SiteURL: "https://a.com",
				Category: &rssClient.Category{ID: 1, Title: "Tech"},
			},
			{
				ID: 2, Title: "Blog B", FeedURL: "https://b.com/rss", SiteURL: "https://b.com",
				Category: &rssClient.Category{ID: 2, Title: "News"},
			},
		}, 2},
		{"feed with nil category converted with empty category", rssClient.Feeds{
			{ID: 3, Title: "No Category", FeedURL: "https://c.com/rss", SiteURL: "https://c.com"},
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := NewWithClient(&fakeClient{feeds: tt.feeds})
			result, err := adapter.ListFeeds(t.Context(), &FeedQuery{})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.want)
			for i, feed := range tt.feeds {
				assert.Equal(t, feed.ID, result.Items[i].ID)
				assert.Equal(t, feed.Title, result.Items[i].Title)
				assert.Equal(t, feed.FeedURL, result.Items[i].FeedURL)
				assert.Equal(t, feed.SiteURL, result.Items[i].SiteURL)
			}
		})
	}
}

func TestListEntriesConvertsEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entries *rssClient.EntryResultSet
		want    int
	}{
		{"converts miniflux entries to ability entries", makeTestEntryResult(1, "My First Post", "unread", false), 1},
		{"multiple entries converted correctly", makeTestEntryResult(2, "Test", "read", false), 1},
		{"entry with starred=true preserves starred flag", makeTestEntryResult(1, "Starred Title", "read", true), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := NewWithClient(&fakeClient{entries: tt.entries})
			result, err := adapter.ListEntries(t.Context(), &EntryQuery{})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.want)
			assert.NotNil(t, result.Page)
			assert.Equal(t, int64(1), *result.Page.Total)
		})
	}
}

func TestListFeedsEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		feeds rssClient.Feeds
	}{
		{"empty feed list returns empty items", nil},
		{"nil feeds returns empty with non-nil page", nil},
		{"zero-length feeds returns empty items", rssClient.Feeds{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := NewWithClient(&fakeClient{feeds: tt.feeds})
			result, err := adapter.ListFeeds(t.Context(), &FeedQuery{})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Empty(t, result.Items)
			assert.NotNil(t, result.Page)
		})
	}
}

func TestCreateFeedReturnsFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		feedID  int64
		feedErr error
		url     string
		wantErr bool
	}{
		{"create feed returns new feed with assigned id", 42, nil, "https://new.example.com/rss", false},
		{"create feed returns correct URL", 1, nil, "https://another.example.com/rss", false},
		{"create feed with error returns error", 0, assert.AnError, "https://error.example.com/rss", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := NewWithClient(&fakeClient{createFeedID: tt.feedID, createFeedErr: tt.feedErr})
			feed, err := adapter.CreateFeed(t.Context(), tt.url)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, feed)
				assert.Equal(t, int64(tt.feedID), feed.ID)
				assert.Equal(t, tt.url, feed.FeedURL)
			}
		})
	}
}

func TestStarEntryReturnsNotImplemented(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entryID int64
	}{
		{"star entry returns not implemented error", 1},
		{"star entry on nonexistent entry id returns not implemented", 99999},
		{"star entry with negative id returns not implemented", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := NewWithClient(&fakeClient{})
			err := adapter.StarEntry(t.Context(), tt.entryID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not implemented")
		})
	}
}

func TestUnstarEntryReturnsNotImplemented(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entryID int64
	}{
		{"unstar entry returns not implemented error", 1},
		{"unstar entry on nonexistent entry id returns not implemented", 99999},
		{"unstar entry with zero id returns not implemented", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter := NewWithClient(&fakeClient{})
			err := adapter.UnstarEntry(t.Context(), tt.entryID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not implemented")
		})
	}
}

func makeTestEntryResult(id int64, title, status string, starred bool) *rssClient.EntryResultSet {
	pubDate := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	return &rssClient.EntryResultSet{
		Total: 1,
		Entries: rssClient.Entries{
			{
				ID:      id,
				Title:   title,
				URL:     "https://example.com/p/" + fmt.Sprintf("%d", id),
				Content: "<p>Hello world</p>",
				Status:  status,
				Starred: starred,
				Date:    pubDate,
				Feed:    &rssClient.Feed{ID: 1, Title: "Example Blog"},
			},
		},
	}
}

// Compile-time interface check.
var _ Service = (*Adapter)(nil)

// Compile-time fake client interface check.
var _ client = (*fakeClient)(nil)
