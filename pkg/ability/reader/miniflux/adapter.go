package miniflux

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/reader"
	provider "github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	rssClient "miniflux.app/v2/client"
)

var defaultCursorSecret = []byte("flowbot-ability-reader-miniflux-cursor-v1")

type client interface {
	GetFeeds() (rssClient.Feeds, error)
	CreateFeed(req *rssClient.FeedCreationRequest) (int64, error)
	GetEntries(filter *rssClient.Filter) (*rssClient.EntryResultSet, error)
	UpdateEntries(entryIDs []int64, status string) error
}

type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

func New() reader.Service {
	return NewWithClient(provider.GetClient())
}

func NewWithClient(client client) reader.Service {
	return &Adapter{
		client:       client,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

func (a *Adapter) ListFeeds(ctx context.Context, q *reader.FeedQuery) (*ability.ListResult[ability.Feed], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "reader list feeds canceled", err)
	}
	if q == nil {
		q = &reader.FeedQuery{}
	}
	feeds, err := a.client.GetFeeds()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "miniflux list feeds", err)
	}
	limit := normalizedLimit(q.Page.Limit)
	total := len(feeds)
	items := make([]*ability.Feed, 0, limit)
	for i, f := range feeds {
		if i >= limit {
			break
		}
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
	return &ability.ListResult[ability.Feed]{
		Items: items,
		Page:  &ability.PageInfo{Limit: limit, HasMore: limit < total},
	}, nil
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
	if q == nil {
		q = &reader.EntryQuery{}
	}
	offset, limit, err := a.decodeCursor(q.Page)
	if err != nil {
		return nil, err
	}
	filter := &rssClient.Filter{
		Offset: offset,
		Limit:  limit,
	}
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
	hasMore := offset+len(items) < int(total)
	page := &ability.PageInfo{Limit: limit, Total: &total, HasMore: hasMore}
	if hasMore {
		nextCursor, err := ability.EncodeCursor(a.cursorSecret, ability.CursorPayload{
			Capability: "reader",
			Backend:    "miniflux",
			Strategy:   "offset",
			Offset:     offset + len(items),
			Limit:      limit,
		})
		if err != nil {
			return nil, err
		}
		page.NextCursor = nextCursor
	}
	return &ability.ListResult[ability.Entry]{Items: items, Page: page}, nil
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

func (a *Adapter) decodeCursor(page ability.PageRequest) (int, int, error) {
	limit := normalizedLimit(page.Limit)
	if page.Cursor == "" {
		return 0, limit, nil
	}
	payload, err := ability.DecodeCursor(a.cursorSecret, page.Cursor, a.now())
	if err != nil {
		return 0, 0, err
	}
	if payload.Limit > 0 {
		limit = payload.Limit
	}
	return payload.Offset, limit, nil
}

func normalizedLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 100
	}
	return limit
}
