package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// stubExampleService is a minimal fake backend for exercising RunExampleConformance.
type stubExampleService struct {
	cfg ExampleConfig
}

func (*stubExampleService) checkCtx(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	return nil
}

func (s *stubExampleService) GetItem(ctx context.Context, id string) (*capability.Host, error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.GetErr != nil {
		return nil, types.WrapError(types.ErrProvider, "get failed", s.cfg.GetErr)
	}
	if s.cfg.GetItem != nil {
		return s.cfg.GetItem, nil
	}
	return &capability.Host{Name: "default", Status: "ok"}, nil
}

func (s *stubExampleService) ListItems(ctx context.Context, _ *ExampleListQuery) (*capability.ListResult[capability.Host], error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.ListErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", s.cfg.ListErr)
	}
	if s.cfg.ListItems != nil {
		return &capability.ListResult[capability.Host]{Items: s.cfg.ListItems}, nil
	}
	return &capability.ListResult[capability.Host]{
		Items: []*capability.Host{},
		Page:  &capability.PageInfo{},
	}, nil
}

func (s *stubExampleService) CreateItem(ctx context.Context, title string, _ types.KV) (*capability.Host, error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title required")
	}
	if s.cfg.CreateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "create failed", s.cfg.CreateErr)
	}
	if s.cfg.CreateItem != nil {
		return s.cfg.CreateItem, nil
	}
	return &capability.Host{Name: title, Status: "ok"}, nil
}

func (s *stubExampleService) UpdateItem(ctx context.Context, id string, _ map[string]any) (*capability.Host, error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.UpdateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "update failed", s.cfg.UpdateErr)
	}
	if s.cfg.UpdateItem != nil {
		return s.cfg.UpdateItem, nil
	}
	return &capability.Host{Name: "updated", Status: "ok"}, nil
}

func (s *stubExampleService) DeleteItem(ctx context.Context, id string) error {
	if err := s.checkCtx(ctx); err != nil {
		return err
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.DeleteErr != nil {
		return types.WrapError(types.ErrProvider, "delete failed", s.cfg.DeleteErr)
	}
	return nil
}

func (s *stubExampleService) HealthCheck(ctx context.Context) (bool, error) {
	if err := s.checkCtx(ctx); err != nil {
		return false, err
	}
	if s.cfg.HealthErr != nil {
		return false, types.WrapError(types.ErrProvider, "health failed", s.cfg.HealthErr)
	}
	return s.cfg.HealthOk, nil
}

func (s *stubExampleService) ListRawEvents(ctx context.Context, _ string) ([]any, string, error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, "", err
	}
	if s.cfg.RawErr != nil {
		return nil, "", types.WrapError(types.ErrProvider, "raw list failed", s.cfg.RawErr)
	}
	return s.cfg.RawItems, s.cfg.RawCursor, nil
}

func stubExampleFactory(_ *testing.T, cfg ExampleConfig) ExampleService {
	return &stubExampleService{cfg: cfg}
}

func TestRunExampleConformanceWithStub(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "stub service passes example conformance suite"},
		{name: "stub factory returns configured service"},
		{name: "stub handles empty config defaults"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RunExampleConformance(t, stubExampleFactory)
		})
	}
}

func TestStubExampleServiceDirect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  ExampleConfig
		call func(t *testing.T, svc ExampleService)
	}{
		{
			name: "get item returns configured host",
			cfg:  ExampleConfig{GetItem: &capability.Host{Name: "host-1", Status: "ok"}},
			call: func(t *testing.T, svc ExampleService) {
				item, err := svc.GetItem(t.Context(), "id-1")
				require.NoError(t, err)
				assert.Equal(t, "host-1", item.Name)
			},
		},
		{
			name: "create rejects empty title",
			cfg:  ExampleConfig{},
			call: func(t *testing.T, svc ExampleService) {
				_, err := svc.CreateItem(t.Context(), "", nil)
				RequireInvalidArgError(t, err)
			},
		},
		{
			name: "health returns configured status",
			cfg:  ExampleConfig{HealthOk: true},
			call: func(t *testing.T, svc ExampleService) {
				ok, err := svc.HealthCheck(t.Context())
				require.NoError(t, err)
				assert.True(t, ok)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := stubExampleFactory(t, tt.cfg)
			tt.call(t, svc)
		})
	}
}

// stubBookmarkService is a minimal fake backend for exercising RunBookmarkConformance.
type stubBookmarkService struct {
	cfg BookmarkConfig
}

func (*stubBookmarkService) checkCtx(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	return nil
}

func (s *stubBookmarkService) List(ctx context.Context, q *capability.BookmarkListQuery) (*capability.ListResult[capability.Bookmark], error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.ListErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", s.cfg.ListErr)
	}
	limit := 20
	if q != nil && q.Page.Limit > 0 {
		limit = q.Page.Limit
	}
	items := s.cfg.ListItems
	if items == nil {
		items = []*capability.Bookmark{}
	}
	hasMore := s.cfg.ListNextCursor != ""
	return &capability.ListResult[capability.Bookmark]{
		Items: items,
		Page:  &capability.PageInfo{Limit: limit, HasMore: hasMore, NextCursor: s.cfg.ListNextCursor},
	}, nil
}

func (s *stubBookmarkService) Get(ctx context.Context, id string) (*capability.Bookmark, error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.GetErr != nil {
		return nil, types.WrapError(types.ErrProvider, "get failed", s.cfg.GetErr)
	}
	if s.cfg.GetItem != nil {
		return s.cfg.GetItem, nil
	}
	return &capability.Bookmark{ID: id}, nil
}

func (s *stubBookmarkService) Create(ctx context.Context, url string) (*capability.Bookmark, error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if url == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "url required")
	}
	if s.cfg.CreateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "create failed", s.cfg.CreateErr)
	}
	if s.cfg.CreateItem != nil {
		return s.cfg.CreateItem, nil
	}
	return &capability.Bookmark{URL: url}, nil
}

func (s *stubBookmarkService) Delete(ctx context.Context, id string) error {
	if err := s.checkCtx(ctx); err != nil {
		return err
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.DeleteErr != nil {
		return types.WrapError(types.ErrProvider, "delete failed", s.cfg.DeleteErr)
	}
	return nil
}

func (s *stubBookmarkService) Archive(ctx context.Context, id string) (bool, error) {
	if err := s.checkCtx(ctx); err != nil {
		return false, err
	}
	if id == "" {
		return false, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.ArchiveErr != nil {
		return false, types.WrapError(types.ErrProvider, "archive failed", s.cfg.ArchiveErr)
	}
	if s.cfg.ArchiveResult != nil {
		return *s.cfg.ArchiveResult, nil
	}
	return true, nil
}

func (s *stubBookmarkService) Search(ctx context.Context, _ *capability.BookmarkSearchQuery) (*capability.ListResult[capability.Bookmark], error) {
	if err := s.checkCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.SearchErr != nil {
		return nil, types.WrapError(types.ErrProvider, "search failed", s.cfg.SearchErr)
	}
	items := s.cfg.SearchItems
	if items == nil {
		items = []*capability.Bookmark{}
	}
	return &capability.ListResult[capability.Bookmark]{
		Items: items,
		Page:  &capability.PageInfo{},
	}, nil
}

func (s *stubBookmarkService) AttachTags(ctx context.Context, id string, tags []string) error {
	if err := s.checkCtx(ctx); err != nil {
		return err
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if len(tags) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "tags required")
	}
	if s.cfg.AttachTagsErr != nil {
		return types.WrapError(types.ErrProvider, "attach tags failed", s.cfg.AttachTagsErr)
	}
	return nil
}

func (s *stubBookmarkService) DetachTags(ctx context.Context, id string, tags []string) error {
	if err := s.checkCtx(ctx); err != nil {
		return err
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if len(tags) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "tags required")
	}
	if s.cfg.DetachTagsErr != nil {
		return types.WrapError(types.ErrProvider, "detach tags failed", s.cfg.DetachTagsErr)
	}
	return nil
}

func (s *stubBookmarkService) CheckURL(ctx context.Context, url string) (bool, string, error) {
	if err := s.checkCtx(ctx); err != nil {
		return false, "", err
	}
	if url == "" {
		return false, "", types.Errorf(types.ErrInvalidArgument, "url required")
	}
	if s.cfg.CheckURLErr != nil {
		return false, "", types.WrapError(types.ErrProvider, "check url failed", s.cfg.CheckURLErr)
	}
	if s.cfg.CheckURLExists {
		return true, s.cfg.CheckURLID, nil
	}
	return false, "", nil
}

func stubBookmarkFactory(_ *testing.T, cfg BookmarkConfig) BookmarkService {
	return &stubBookmarkService{cfg: cfg}
}

func TestRunBookmarkConformanceWithStub(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "stub service passes bookmark conformance suite"},
		{name: "stub bookmark factory wires config"},
		{name: "stub bookmark handles pagination cursor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RunBookmarkConformance(t, stubBookmarkFactory)
		})
	}
}
