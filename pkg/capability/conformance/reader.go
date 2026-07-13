package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ReaderService is the reader capability contract used by conformance tests.
type ReaderService interface {
	ListFeeds(ctx context.Context, q *capability.ReaderFeedQuery) (*capability.ListResult[capability.Feed], error)
	CreateFeed(ctx context.Context, feedURL string) (*capability.Feed, error)
	ListEntries(ctx context.Context, q *capability.ReaderEntryQuery) (*capability.ListResult[capability.Entry], error)
	MarkEntryRead(ctx context.Context, id int64) error
	MarkEntryUnread(ctx context.Context, id int64) error
	StarEntry(ctx context.Context, id int64) error
	UnstarEntry(ctx context.Context, id int64) error
}

// ReaderConfig configures the fake backend for each reader conformance subtest.
type ReaderConfig struct {
	Feeds         []*capability.Feed
	FeedsErr      error
	CreateFeedID  int64
	CreateFeedErr error
	Entries       []*capability.Entry
	EntriesTotal  int64
	EntriesErr    error
	MarkReadErr   error
	MarkUnreadErr error
	StarErr       error
	UnstarErr     error
}

// ReaderServiceFactory creates a fresh reader Service wired to a fake backend
// whose behavior is determined by the config parameter.
type ReaderServiceFactory func(t *testing.T, cfg ReaderConfig) ReaderService

// RunReaderConformance runs the standard reader capability conformance suite.
func RunReaderConformance(t *testing.T, factory ReaderServiceFactory) {
	t.Run("list feeds success", func(t *testing.T) {
		svc := factory(t, ReaderConfig{
			Feeds: []*capability.Feed{{ID: 1, Title: "Blog", FeedURL: "https://blog.example.com/rss", SiteURL: "https://blog.example.com"}},
		})
		result, err := svc.ListFeeds(t.Context(), &capability.ReaderFeedQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Items)
		assert.NotNil(t, result.Page)
		assert.Len(t, result.Items, 1)
	})

	t.Run("list feeds empty", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		result, err := svc.ListFeeds(t.Context(), &capability.ReaderFeedQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list feeds timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		_, err := svc.ListFeeds(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list feeds provider error", func(t *testing.T) {
		svc := factory(t, ReaderConfig{FeedsErr: assert.AnError})
		_, err := svc.ListFeeds(t.Context(), &capability.ReaderFeedQuery{})
		RequireProviderError(t, err)
	})

	t.Run("create feed success", func(t *testing.T) {
		svc := factory(t, ReaderConfig{CreateFeedID: 42})
		feed, err := svc.CreateFeed(t.Context(), "https://new.example.com/rss")
		require.NoError(t, err)
		require.NotNil(t, feed)
		assert.Equal(t, int64(42), feed.ID)
	})

	t.Run("create feed timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		_, err := svc.CreateFeed(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("create feed empty url", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		_, err := svc.CreateFeed(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("create feed provider error", func(t *testing.T) {
		svc := factory(t, ReaderConfig{CreateFeedErr: assert.AnError})
		_, err := svc.CreateFeed(t.Context(), "https://example.com/rss")
		RequireProviderError(t, err)
	})

	t.Run("list entries success", func(t *testing.T) {
		svc := factory(t, ReaderConfig{
			Entries:      []*capability.Entry{{ID: 1, Title: "Post", URL: "https://blog.example.com/p/1", Status: "unread"}},
			EntriesTotal: 100,
		})
		result, err := svc.ListEntries(t.Context(), &capability.ReaderEntryQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Items)
		assert.Len(t, result.Items, 1)
		assert.NotNil(t, result.Page)
	})

	t.Run("list entries with filter", func(t *testing.T) {
		svc := factory(t, ReaderConfig{
			Entries: []*capability.Entry{},
		})
		result, err := svc.ListEntries(t.Context(), &capability.ReaderEntryQuery{Status: "read", FeedID: 5})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Items)
	})

	t.Run("list entries timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		_, err := svc.ListEntries(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list entries provider error", func(t *testing.T) {
		svc := factory(t, ReaderConfig{EntriesErr: assert.AnError})
		_, err := svc.ListEntries(t.Context(), &capability.ReaderEntryQuery{})
		RequireProviderError(t, err)
	})

	t.Run("mark entry read success", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		err := svc.MarkEntryRead(t.Context(), 1)
		require.NoError(t, err)
	})

	t.Run("mark entry read timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		err := svc.MarkEntryRead(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("mark entry read provider error", func(t *testing.T) {
		svc := factory(t, ReaderConfig{MarkReadErr: assert.AnError})
		err := svc.MarkEntryRead(t.Context(), 1)
		RequireProviderError(t, err)
	})

	t.Run("mark entry unread success", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		err := svc.MarkEntryUnread(t.Context(), 1)
		require.NoError(t, err)
	})

	t.Run("mark entry unread timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		err := svc.MarkEntryUnread(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("mark entry unread provider error", func(t *testing.T) {
		svc := factory(t, ReaderConfig{MarkUnreadErr: assert.AnError})
		err := svc.MarkEntryUnread(t.Context(), 1)
		RequireProviderError(t, err)
	})

	t.Run("star entry timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		err := svc.StarEntry(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("unstar entry timeout", func(t *testing.T) {
		svc := factory(t, ReaderConfig{})
		err := svc.UnstarEntry(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})
}
