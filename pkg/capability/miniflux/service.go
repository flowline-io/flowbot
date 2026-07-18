package miniflux

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// FeedQuery wraps pagination for listing feeds.
type FeedQuery = capability.ReaderFeedQuery

// EntryQuery wraps pagination and filters for listing entries.
type EntryQuery = capability.ReaderEntryQuery

// Service defines the reader capability contract.
type Service interface {
	ListFeeds(ctx context.Context, q *FeedQuery) (*capability.ListResult[capability.Feed], error)
	CreateFeed(ctx context.Context, feedURL string) (*capability.Feed, error)
	ListEntries(ctx context.Context, q *EntryQuery) (*capability.ListResult[capability.Entry], error)
	MarkEntryRead(ctx context.Context, id int64) error
	MarkEntryUnread(ctx context.Context, id int64) error
	StarEntry(ctx context.Context, id int64) error
	UnstarEntry(ctx context.Context, id int64) error
	HealthCheck(ctx context.Context) (bool, error)
}
