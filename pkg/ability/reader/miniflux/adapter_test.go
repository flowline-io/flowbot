package miniflux

import (
	"testing"
	"time"

	rdr "github.com/flowline-io/flowbot/pkg/ability/reader"
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

func (f *fakeClient) CreateFeed(req *rssClient.FeedCreationRequest) (int64, error) {
	if f.createFeedErr != nil {
		return 0, f.createFeedErr
	}
	return f.createFeedID, nil
}

func (f *fakeClient) GetEntries(filter *rssClient.Filter) (*rssClient.EntryResultSet, error) {
	if f.entriesErr != nil {
		return nil, f.entriesErr
	}
	if f.entries == nil {
		return &rssClient.EntryResultSet{Entries: rssClient.Entries{}}, nil
	}
	return f.entries, nil
}

func (f *fakeClient) UpdateEntries(entryIDs []int64, status string) error {
	switch status {
	case rssClient.EntryStatusRead:
		return f.markReadErr
	case rssClient.EntryStatusUnread:
		return f.markUnreadErr
	}
	return nil
}

func TestListFeedsConvertsFeeds(t *testing.T) {
	adapter := NewWithClient(&fakeClient{
		feeds: rssClient.Feeds{
			{
				ID:      1,
				Title:   "Example Blog",
				FeedURL: "https://example.com/rss",
				SiteURL: "https://example.com",
				Category: &rssClient.Category{ID: 1, Title: "Tech"},
			},
		},
	})

	result, err := adapter.ListFeeds(t.Context(), &rdr.FeedQuery{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, int64(1), result.Items[0].ID)
	assert.Equal(t, "Example Blog", result.Items[0].Title)
	assert.Equal(t, "https://example.com/rss", result.Items[0].FeedURL)
	assert.Equal(t, "https://example.com", result.Items[0].SiteURL)
	assert.Equal(t, "Tech", result.Items[0].Category)
}

func TestListEntriesConvertsEntries(t *testing.T) {
	pubDate := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	adapter := NewWithClient(&fakeClient{
		entries: &rssClient.EntryResultSet{
			Total: 1,
			Entries: rssClient.Entries{
				{
					ID:      101,
					Title:   "My First Post",
					URL:     "https://example.com/p/1",
					Content: "<p>Hello world</p>",
					Status:  "unread",
					Starred: false,
					Date:    pubDate,
					Feed:    &rssClient.Feed{ID: 1, Title: "Example Blog"},
				},
			},
		},
	})

	result, err := adapter.ListEntries(t.Context(), &rdr.EntryQuery{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Items, 1)
	entry := result.Items[0]
	assert.Equal(t, int64(101), entry.ID)
	assert.Equal(t, "My First Post", entry.Title)
	assert.Equal(t, "https://example.com/p/1", entry.URL)
	assert.Equal(t, "<p>Hello world</p>", entry.Content)
	assert.Equal(t, "unread", entry.Status)
	assert.False(t, entry.Starred)
	assert.Equal(t, pubDate, entry.PublishedAt)
	assert.Equal(t, "Example Blog", entry.FeedTitle)
	assert.NotNil(t, result.Page)
	assert.Equal(t, int64(1), *result.Page.Total)
}

func TestListFeedsEmpty(t *testing.T) {
	adapter := NewWithClient(&fakeClient{})
	result, err := adapter.ListFeeds(t.Context(), &rdr.FeedQuery{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.NotNil(t, result.Page)
}

func TestCreateFeedReturnsFeed(t *testing.T) {
	adapter := NewWithClient(&fakeClient{createFeedID: 42})
	feed, err := adapter.CreateFeed(t.Context(), "https://new.example.com/rss")
	require.NoError(t, err)
	require.NotNil(t, feed)
	assert.Equal(t, int64(42), feed.ID)
	assert.Equal(t, "https://new.example.com/rss", feed.FeedURL)
}

func TestStarEntryReturnsNotImplemented(t *testing.T) {
	adapter := NewWithClient(&fakeClient{})
	err := adapter.StarEntry(t.Context(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestUnstarEntryReturnsNotImplemented(t *testing.T) {
	adapter := NewWithClient(&fakeClient{})
	err := adapter.UnstarEntry(t.Context(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

// Compile-time interface check.
var _ rdr.Service = (*Adapter)(nil)

// Compile-time fake client interface check.
var _ client = (*fakeClient)(nil)
