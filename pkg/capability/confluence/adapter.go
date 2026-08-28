// Package confluence implements the Confluence Cloud capability adapter.
package confluence

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/confluence"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

var defaultCursorSecret = []byte("flowbot-capability-confluence-cursor-v1")

type client interface {
	ListSpaces(ctx context.Context, start, limit int) (*provider.SpaceListResponse, error)
	ListPages(ctx context.Context, spaceKey string, start, limit int) (*provider.PageListResponse, error)
	GetPage(ctx context.Context, pageID string, expandBody bool) (*provider.Page, error)
	GetPageContent(ctx context.Context, pageID string) (string, error)
	SearchPages(ctx context.Context, cql string, start, limit int) (*provider.SearchResponse, error)
	CreatePage(ctx context.Context, spaceKey, title, content string) (*provider.Page, error)
	UpdatePage(ctx context.Context, pageID, title, content string) (*provider.Page, error)
	DeletePage(ctx context.Context, pageID string) error
	GetCurrentUser(ctx context.Context) (map[string]any, error)
}

// Adapter implements Service using the Confluence provider client.
type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

// New creates an Adapter from provider config. Returns nil when not configured.
func New() Service {
	if c := provider.GetClient(); c != nil {
		return NewWithClient(c)
	}
	return nil
}

// NewWithClient creates an Adapter with the given client.
func NewWithClient(c client) Service {
	return &Adapter{
		client:       c,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

func (a *Adapter) checkClient() error {
	if a.client == nil {
		return types.Errorf(types.ErrUnavailable, "confluence client not available")
	}
	return nil
}

func resolveSpaceKey(spaceKey string) (string, error) {
	if spaceKey != "" {
		return spaceKey, nil
	}
	if defaultKey := provider.GetDefaultSpaceKey(); defaultKey != "" {
		return defaultKey, nil
	}
	return "", types.Errorf(types.ErrInvalidArgument, "space_key is required")
}

func normalizedLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (a *Adapter) ListSpaces(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.ConfluenceSpace], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	limit := 25
	start := 0
	if q != nil {
		limit = normalizedLimit(q.Page.Limit)
		start, limit = capability.OffsetPageFromCursor(q.Page.Cursor, limit, a.cursorSecret, a.now(), string(hub.CapConfluence))
	}
	resp, err := a.client.ListSpaces(ctx, start, limit)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "confluence list spaces failed", err)
	}
	return &capability.ListResult[capability.ConfluenceSpace]{
		Items: toSpaces(resp.Results),
		Page:  capability.OffsetPageInfo(resp.Start, resp.Limit, resp.Size, len(resp.Results), a.cursorSecret, string(hub.CapConfluence)),
	}, nil
}

func (a *Adapter) ListPages(ctx context.Context, spaceKey string, q *ListQuery) (*capability.ListResult[capability.ConfluencePage], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	resolved, err := resolveSpaceKey(spaceKey)
	if err != nil {
		return nil, err
	}
	limit := 25
	start := 0
	if q != nil {
		limit = normalizedLimit(q.Page.Limit)
		start, limit = capability.OffsetPageFromCursor(q.Page.Cursor, limit, a.cursorSecret, a.now(), string(hub.CapConfluence))
	}
	resp, err := a.client.ListPages(ctx, resolved, start, limit)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "confluence list pages failed", err)
	}
	return &capability.ListResult[capability.ConfluencePage]{
		Items: toPages(resp.Results),
		Page:  capability.OffsetPageInfo(resp.Start, resp.Limit, resp.Size, len(resp.Results), a.cursorSecret, string(hub.CapConfluence)),
	}, nil
}

func (a *Adapter) GetPage(ctx context.Context, pageID string) (*capability.ConfluencePage, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	page, err := a.client.GetPage(ctx, pageID, false)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "confluence get page failed", err)
	}
	return toPage(page, false), nil
}

func (a *Adapter) GetPageContent(ctx context.Context, pageID string) (string, error) {
	if err := a.checkClient(); err != nil {
		return "", err
	}
	content, err := a.client.GetPageContent(ctx, pageID)
	if err != nil {
		return "", types.WrapError(types.ErrProvider, "confluence get page content failed", err)
	}
	return content, nil
}

func (a *Adapter) SearchPages(ctx context.Context, cql string, q *ListQuery) (*capability.ListResult[capability.ConfluencePage], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	limit := 25
	start := 0
	if q != nil {
		limit = normalizedLimit(q.Page.Limit)
		start, limit = capability.OffsetPageFromCursor(q.Page.Cursor, limit, a.cursorSecret, a.now(), string(hub.CapConfluence))
	}
	resp, err := a.client.SearchPages(ctx, cql, start, limit)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "confluence search pages failed", err)
	}
	items := make([]*capability.ConfluencePage, 0, len(resp.Results))
	for _, hit := range resp.Results {
		items = append(items, toPage(&hit.Content, false))
	}
	return &capability.ListResult[capability.ConfluencePage]{
		Items: items,
		Page:  capability.OffsetPageInfo(resp.Start, resp.Limit, resp.Size, len(resp.Results), a.cursorSecret, string(hub.CapConfluence)),
	}, nil
}

func (a *Adapter) CreatePage(ctx context.Context, spaceKey, title, content string) (*capability.ConfluencePage, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	resolved, err := resolveSpaceKey(spaceKey)
	if err != nil {
		return nil, err
	}
	page, err := a.client.CreatePage(ctx, resolved, title, content)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "confluence create page failed", err)
	}
	return toPage(page, false), nil
}

func (a *Adapter) UpdatePage(ctx context.Context, pageID, title, content string) (*capability.ConfluencePage, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	page, err := a.client.UpdatePage(ctx, pageID, title, content)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "confluence update page failed", err)
	}
	return toPage(page, false), nil
}

func (a *Adapter) DeletePage(ctx context.Context, pageID string) error {
	if err := a.checkClient(); err != nil {
		return err
	}
	if err := a.client.DeletePage(ctx, pageID); err != nil {
		return types.WrapError(types.ErrProvider, "confluence delete page failed", err)
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := a.checkClient(); err != nil {
		return false, err
	}
	_, err := a.client.GetCurrentUser(ctx)
	if err != nil {
		flog.Warn("confluence health check failed: %v", err)
		return false, nil
	}
	return true, nil
}

func pageEntityID(page *capability.ConfluencePage) string {
	if page == nil {
		return ""
	}
	return page.ID
}

func eventRef(eventType, entityID string) capability.EventRef {
	return capability.EventRef{
		EventID:   types.Id(),
		EventType: eventType,
		EntityID:  entityID,
	}
}

func mutationResult(page *capability.ConfluencePage, eventType, app, text string) *capability.InvokeResult {
	entityID := pageEntityID(page)
	ev := eventRef(eventType, entityID)
	return &capability.InvokeResult{
		Data: page,
		Text: text,
		Resource: &capability.ResourceMeta{
			EventID:  ev.EventID,
			EntityID: entityID,
			App:      app,
		},
		Events: []capability.EventRef{ev},
	}
}

func deletePageResult(pageID, app string) *capability.InvokeResult {
	ev := eventRef(types.EventConfluencePageDeleted, pageID)
	return &capability.InvokeResult{
		Text: fmt.Sprintf("page deleted: %s", pageID),
		Resource: &capability.ResourceMeta{
			EventID:  ev.EventID,
			EntityID: pageID,
			App:      app,
		},
		Events: []capability.EventRef{ev},
	}
}
