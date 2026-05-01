package karakeep

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	bm "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types"
)

var defaultCursorSecret = []byte("flowbot-ability-bookmark-karakeep-cursor-v1")

type client interface {
	GetAllBookmarks(query *provider.BookmarksQuery) (*provider.BookmarksResponse, error)
	GetBookmark(id string) (*provider.Bookmark, error)
	CreateBookmark(url string) (*provider.Bookmark, error)
	ArchiveBookmark(id string) (bool, error)
	SearchBookmarks(query *provider.SearchBookmarksQuery) (*provider.BookmarksResponse, error)
	AttachTagsToBookmark(bookmarkID string, tags []string) ([]string, error)
	DetachTagsToBookmark(bookmarkID string, tags []string) ([]string, error)
	CheckUrlExists(url string) (*string, error)
}

type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

func New() bm.Service {
	return NewWithClient(provider.GetClient())
}

func (a *Adapter) SetCursorSecret(secret []byte) {
	a.cursorSecret = secret
}

func NewWithClient(client client) bm.Service {
	return &Adapter{
		client:       client,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

func (a *Adapter) List(ctx context.Context, q *bm.ListQuery) (*ability.ListResult[ability.Bookmark], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark list canceled", err)
	}
	if q == nil {
		q = &bm.ListQuery{}
	}
	providerCursor, err := a.providerCursor(q.Page.Cursor)
	if err != nil {
		return nil, err
	}
	query := &provider.BookmarksQuery{
		Limit:  normalizedLimit(q.Page.Limit),
		Cursor: providerCursor,
	}
	if q.Archived != nil {
		query.Archived = *q.Archived
	}
	if q.Favourited != nil {
		query.Favourited = *q.Favourited
	}
	resp, err := a.client.GetAllBookmarks(query)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "karakeep list bookmarks", err)
	}
	return a.listResult(resp, query.Limit)
}

func (a *Adapter) Get(ctx context.Context, id string) (*ability.Bookmark, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark get canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	item, err := a.client.GetBookmark(id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "karakeep get bookmark", err)
	}
	if item == nil {
		return nil, types.Errorf(types.ErrNotFound, "bookmark %s not found", id)
	}
	return toBookmark(item), nil
}

func (a *Adapter) Create(ctx context.Context, url string) (*ability.Bookmark, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark create canceled", err)
	}
	if url == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	item, err := a.client.CreateBookmark(url)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "karakeep create bookmark", err)
	}
	if item == nil {
		return nil, types.Errorf(types.ErrProvider, "karakeep create bookmark returned empty response")
	}
	return toBookmark(item), nil
}

func (a *Adapter) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "bookmark delete canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return types.Errorf(types.ErrNotImplemented, "karakeep bookmark delete is not implemented")
}

func (a *Adapter) Archive(ctx context.Context, id string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "bookmark archive canceled", err)
	}
	if id == "" {
		return false, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	archived, err := a.client.ArchiveBookmark(id)
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "karakeep archive bookmark", err)
	}
	return archived, nil
}

func (a *Adapter) Search(ctx context.Context, q *bm.SearchQuery) (*ability.ListResult[ability.Bookmark], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark search canceled", err)
	}
	if q == nil {
		q = &bm.SearchQuery{}
	}
	providerCursor, err := a.providerCursor(q.Page.Cursor)
	if err != nil {
		return nil, err
	}
	query := &provider.SearchBookmarksQuery{
		Q:         q.Q,
		Limit:     normalizedLimit(q.Page.Limit),
		Cursor:    providerCursor,
		SortOrder: q.Page.SortOrder,
	}
	resp, err := a.client.SearchBookmarks(query)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "karakeep search bookmarks", err)
	}
	return a.listResult(resp, query.Limit)
}

func (a *Adapter) AttachTags(ctx context.Context, id string, tags []string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "bookmark attach tags canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if len(tags) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "tags are required")
	}
	if _, err := a.client.AttachTagsToBookmark(id, tags); err != nil {
		return types.WrapError(types.ErrProvider, "karakeep attach tags", err)
	}
	return nil
}

func (a *Adapter) DetachTags(ctx context.Context, id string, tags []string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "bookmark detach tags canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if len(tags) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "tags are required")
	}
	if _, err := a.client.DetachTagsToBookmark(id, tags); err != nil {
		return types.WrapError(types.ErrProvider, "karakeep detach tags", err)
	}
	return nil
}

func (a *Adapter) CheckURL(ctx context.Context, url string) (bool, string, error) {
	if err := ctx.Err(); err != nil {
		return false, "", types.WrapError(types.ErrTimeout, "bookmark check url canceled", err)
	}
	if url == "" {
		return false, "", types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	id, err := a.client.CheckUrlExists(url)
	if err != nil {
		return false, "", types.WrapError(types.ErrProvider, "karakeep check url", err)
	}
	if id == nil || *id == "" {
		return false, "", nil
	}
	return true, *id, nil
}

func (a *Adapter) providerCursor(cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	payload, err := ability.DecodeCursor(a.cursorSecret, cursor, a.now())
	if err != nil {
		return "", err
	}
	return payload.ProviderCursor, nil
}

func (a *Adapter) listResult(resp *provider.BookmarksResponse, limit int) (*ability.ListResult[ability.Bookmark], error) {
	if resp == nil {
		resp = &provider.BookmarksResponse{}
	}
	items := make([]*ability.Bookmark, 0, len(resp.Bookmarks))
	for i := range resp.Bookmarks {
		items = append(items, toBookmark(&resp.Bookmarks[i]))
	}
	page := &ability.PageInfo{
		Limit:   limit,
		HasMore: resp.NextCursor != "",
	}
	if resp.NextCursor != "" {
		cursor, err := ability.EncodeCursor(a.cursorSecret, ability.CursorPayload{
			Capability:     "bookmark",
			Backend:        provider.ID,
			Strategy:       "cursor",
			ProviderCursor: resp.NextCursor,
			Limit:          limit,
		})
		if err != nil {
			return nil, err
		}
		page.NextCursor = cursor
	}
	return &ability.ListResult[ability.Bookmark]{Items: items, Page: page}, nil
}

func normalizedLimit(limit int) int {
	if limit <= 0 || limit > provider.MaxPageSize {
		return provider.MaxPageSize
	}
	return limit
}

func toBookmark(item *provider.Bookmark) *ability.Bookmark {
	if item == nil {
		return nil
	}
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		flog.Warn("karakeep adapter: parse bookmark created_at: %v", err)
	}
	tags := make([]string, 0, len(item.Tags))
	for _, tag := range item.Tags {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	return &ability.Bookmark{
		ID:         item.Id,
		URL:        item.Content.Url,
		Title:      firstNonEmpty(item.GetTitle(), stringValue(item.Content.Title)),
		Summary:    item.GetSummary(),
		Tags:       tags,
		Archived:   item.Archived,
		Favourited: item.Favourited,
		CreatedAt:  createdAt,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
