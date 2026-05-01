package reader

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type FeedQuery struct {
	Page ability.PageRequest
}

type EntryQuery struct {
	Page   ability.PageRequest
	Status string
	FeedID int64
}

type Service interface {
	ListFeeds(ctx context.Context, q *FeedQuery) (*ability.ListResult[ability.Feed], error)
	CreateFeed(ctx context.Context, feedURL string) (*ability.Feed, error)
	ListEntries(ctx context.Context, q *EntryQuery) (*ability.ListResult[ability.Entry], error)
	MarkEntryRead(ctx context.Context, id int64) error
	MarkEntryUnread(ctx context.Context, id int64) error
	StarEntry(ctx context.Context, id int64) error
	UnstarEntry(ctx context.Context, id int64) error
}
