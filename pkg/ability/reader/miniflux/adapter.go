package miniflux

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/reader"
	provider "github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	rssClient "miniflux.app/v2/client"
)

type client interface {
	GetFeeds() (rssClient.Feeds, error)
	CreateFeed(req *rssClient.FeedCreationRequest) (int64, error)
	GetEntries(filter *rssClient.Filter) (*rssClient.EntryResultSet, error)
	UpdateEntries(entryIDs []int64, status string) error
}

type Adapter struct {
	client client
}

func New() reader.Service {
	return NewWithClient(provider.GetClient())
}

func NewWithClient(client client) reader.Service {
	return &Adapter{client: client}
}

func (a *Adapter) ListFeeds(ctx context.Context, q *reader.FeedQuery) (*ability.ListResult[ability.Feed], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "reader list feeds canceled", err)
	}
	feeds, err := a.client.GetFeeds()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "miniflux list feeds", err)
	}
	items := make([]*ability.Feed, 0, len(feeds))
	for _, f := range feeds {
		category := ""
		if f.Category != nil {
			category = f.Category.Title
		}
		items = append(items, &ability.Feed{
			ID:       f.ID,
			Title:    f.Title,
			FeedURL:  f.FeedURL,
			SiteURL:  f.SiteURL,
			Category: category,
		})
	}
	return &ability.ListResult[ability.Feed]{Items: items, Page: &ability.PageInfo{Limit: len(items)}}, nil
}

func (a *Adapter) CreateFeed(ctx context.Context, feedURL string) (*ability.Feed, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "reader create feed canceled", err)
	}
	if feedURL == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "feed_url is required")
	}
	feedID, err := a.client.CreateFeed(&rssClient.FeedCreationRequest{FeedURL: feedURL})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "miniflux create feed", err)
	}
	return &ability.Feed{
		ID:      feedID,
		FeedURL: feedURL,
	}, nil
}

func (a *Adapter) ListEntries(ctx context.Context, q *reader.EntryQuery) (*ability.ListResult[ability.Entry], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "reader list entries canceled", err)
	}
	filter := &rssClient.Filter{}
	if q.Status != "" {
		filter.Status = q.Status
	}
	if q.FeedID > 0 {
		filter.FeedID = q.FeedID
	}
	result, err := a.client.GetEntries(filter)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "miniflux list entries", err)
	}
	items := make([]*ability.Entry, 0, len(result.Entries))
	for _, e := range result.Entries {
		feedTitle := ""
		if e.Feed != nil {
			feedTitle = e.Feed.Title
		}
		items = append(items, &ability.Entry{
			ID:          e.ID,
			Title:       e.Title,
			URL:         e.URL,
			Content:     e.Content,
			Status:      e.Status,
			Starred:     e.Starred,
			PublishedAt: e.Date,
			FeedTitle:   feedTitle,
		})
	}
	total := int64(result.Total)
	return &ability.ListResult[ability.Entry]{
		Items: items,
		Page:  &ability.PageInfo{Limit: len(items), Total: &total},
	}, nil
}

func (a *Adapter) MarkEntryRead(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "reader mark read canceled", err)
	}
	if err := a.client.UpdateEntries([]int64{id}, rssClient.EntryStatusRead); err != nil {
		return types.WrapError(types.ErrProvider, "miniflux mark entry read", err)
	}
	return nil
}

func (a *Adapter) MarkEntryUnread(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "reader mark unread canceled", err)
	}
	if err := a.client.UpdateEntries([]int64{id}, rssClient.EntryStatusUnread); err != nil {
		return types.WrapError(types.ErrProvider, "miniflux mark entry unread", err)
	}
	return nil
}

func (a *Adapter) StarEntry(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "reader star entry canceled", err)
	}
	return types.Errorf(types.ErrNotImplemented, "miniflux star entry is not implemented via this adapter")
}

func (a *Adapter) UnstarEntry(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "reader unstar entry canceled", err)
	}
	return types.Errorf(types.ErrNotImplemented, "miniflux unstar entry is not implemented via this adapter")
}
