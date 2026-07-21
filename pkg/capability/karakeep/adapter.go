// Package karakeep implements the Karakeep adapter for the bookmark capability.
package karakeep

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"

	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types"
)

var defaultCursorSecret = []byte("flowbot-ability-bookmark-karakeep-cursor-v1")

type client interface {
	GetAllBookmarks(ctx context.Context, query *provider.BookmarksQuery) (*provider.BookmarksResponse, error)
	GetBookmark(ctx context.Context, id string) (*provider.Bookmark, error)
	CreateBookmark(ctx context.Context, url string) (*provider.Bookmark, error)
	ArchiveBookmark(ctx context.Context, id string) (bool, error)
	SearchBookmarks(ctx context.Context, query *provider.SearchBookmarksQuery) (*provider.BookmarksResponse, error)
	AttachTagsToBookmark(ctx context.Context, bookmarkID string, tags []string) ([]string, error)
	DetachTagsToBookmark(ctx context.Context, bookmarkID string, tags []string) ([]string, error)
	CheckUrlExists(ctx context.Context, url string) (*string, error)
}

type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

func New() Service {
	return NewWithClient(provider.GetClient())
}

func (a *Adapter) SetCursorSecret(secret []byte) {
	a.cursorSecret = secret
}

func NewWithClient(client client) Service {
	return &Adapter{
		client:       client,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

func (a *Adapter) List(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Bookmark], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark list canceled", err)
	}
	if q == nil {
		q = &ListQuery{}
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
	resp, err := a.client.GetAllBookmarks(ctx, query)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "karakeep list bookmarks", err)
	}
	return a.listResult(resp, query.Limit)
}

func (a *Adapter) Get(ctx context.Context, id string) (*capability.Bookmark, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark get canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	item, err := a.client.GetBookmark(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "karakeep get bookmark", err)
	}
	if item == nil {
		return nil, types.Errorf(types.ErrNotFound, "bookmark %s not found", id)
	}
	return toBookmark(item), nil
}

func (a *Adapter) Create(ctx context.Context, url string) (*capability.Bookmark, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark create canceled", err)
	}
	if url == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	item, err := a.client.CreateBookmark(ctx, url)
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
	// Karakeep hard-delete is not exposed; archive matches CLI "delete" semantics.
	_, err := a.client.ArchiveBookmark(ctx, id)
	if err != nil {
		return types.WrapError(types.ErrProvider, "karakeep delete bookmark", err)
	}
	return nil
}

func (a *Adapter) Archive(ctx context.Context, id string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "bookmark archive canceled", err)
	}
	if id == "" {
		return false, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	archived, err := a.client.ArchiveBookmark(ctx, id)
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "karakeep archive bookmark", err)
	}
	return archived, nil
}

func (a *Adapter) Search(ctx context.Context, q *SearchQuery) (*capability.ListResult[capability.Bookmark], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "bookmark search canceled", err)
	}
	if q == nil {
		q = &SearchQuery{}
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
	resp, err := a.client.SearchBookmarks(ctx, query)
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
	if _, err := a.client.AttachTagsToBookmark(ctx, id, tags); err != nil {
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
	if _, err := a.client.DetachTagsToBookmark(ctx, id, tags); err != nil {
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
	id, err := a.client.CheckUrlExists(ctx, url)
	if err != nil {
		return false, "", types.WrapError(types.ErrProvider, "karakeep check url", err)
	}
	if id == nil || *id == "" {
		return false, "", nil
	}
	return true, *id, nil
}

// HealthCheck reports whether the Karakeep backend is reachable by listing a single bookmark page.
func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	_, err := a.client.GetAllBookmarks(ctx, &provider.BookmarksQuery{Limit: 1})
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "karakeep health check failed", err)
	}
	return true, nil
}

func (a *Adapter) providerCursor(cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	payload, err := capability.DecodeCursor(a.cursorSecret, cursor, a.now())
	if err != nil {
		return "", err
	}
	return payload.ProviderCursor, nil
}

func (a *Adapter) listResult(resp *provider.BookmarksResponse, limit int) (*capability.ListResult[capability.Bookmark], error) {
	if resp == nil {
		resp = &provider.BookmarksResponse{}
	}
	items := make([]*capability.Bookmark, 0, len(resp.Bookmarks))
	for i := range resp.Bookmarks {
		items = append(items, toBookmark(&resp.Bookmarks[i]))
	}
	page := &capability.PageInfo{
		Limit:   limit,
		HasMore: resp.NextCursor != "",
	}
	if resp.NextCursor != "" {
		cursor, err := capability.EncodeCursor(a.cursorSecret, capability.CursorPayload{
			Capability:     "karakeep",
			Strategy:       "cursor",
			ProviderCursor: resp.NextCursor,
			Limit:          limit,
		})
		if err != nil {
			return nil, err
		}
		page.NextCursor = cursor
	}
	return &capability.ListResult[capability.Bookmark]{Items: items, Page: page}, nil
}

func normalizedLimit(limit int) int {
	if limit <= 0 || limit > provider.MaxPageSize {
		return provider.MaxPageSize
	}
	return limit
}

func toBookmark(item *provider.Bookmark) *capability.Bookmark {
	if item == nil {
		return nil
	}
	var createdAt time.Time
	if item.CreatedAt != "" {
		parsed, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err != nil {
			flog.Warn("karakeep adapter: parse bookmark created_at: %v", err)
		} else {
			createdAt = parsed
		}
	}
	tags := make([]string, 0, len(item.Tags))
	for _, tag := range item.Tags {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	return &capability.Bookmark{
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
