package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// NoteListQuery wraps pagination for listing notes.
type NoteListQuery = capability.NoteListQuery

// NoteService is the note capability contract used by conformance tests.
type NoteService interface {
	List(ctx context.Context, q *NoteListQuery) (*capability.ListResult[capability.Note], error)
	Get(ctx context.Context, id string) (*capability.Note, error)
	Create(ctx context.Context, title, content, typ, parentNoteID string) (*capability.Note, error)
	Update(ctx context.Context, id, title, content string) (*capability.Note, error)
	Delete(ctx context.Context, id string) error
	GetContent(ctx context.Context, id string) (string, error)
	SetContent(ctx context.Context, id, content string) error
	Search(ctx context.Context, query string) (*capability.ListResult[capability.Note], error)
	GetAppInfo(ctx context.Context) (*capability.Note, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}

// NoteConfig configures the fake backend for each note conformance subtest.
type NoteConfig struct {
	ListItems     []*capability.Note
	ListErr       error
	GetItem       *capability.Note
	GetErr        error
	CreateItem    *capability.Note
	CreateErr     error
	UpdateItem    *capability.Note
	UpdateErr     error
	DeleteErr     error
	Content       string
	ContentErr    error
	SetContentErr error
	SearchItems   []*capability.Note
	SearchErr     error
	AppInfo       *capability.Note
	AppInfoErr    error
	RawItems      []any
	RawCursor     string
	RawErr        error
}

// NoteServiceFactory creates a fresh note Service wired to a fake backend
// whose behavior is determined by the config parameter.
type NoteServiceFactory func(t *testing.T, cfg NoteConfig) NoteService

// RunNoteConformance runs the standard note capability conformance suite.
func RunNoteConformance(t *testing.T, factory NoteServiceFactory) {
	t.Run("list success", func(t *testing.T) {
		svc := factory(t, NoteConfig{
			ListItems: []*capability.Note{
				{ID: "n-1"},
				{ID: "n-2"},
			},
		})
		result, err := svc.List(t.Context(), &capability.NoteListQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list empty", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		result, err := svc.List(t.Context(), &capability.NoteListQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Items)
	})

	t.Run("list timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.List(CanceledContext(), &capability.NoteListQuery{})
		RequireTimeoutError(t, err)
	})

	t.Run("list provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{ListErr: assert.AnError})
		_, err := svc.List(t.Context(), &capability.NoteListQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get success", func(t *testing.T) {
		svc := factory(t, NoteConfig{GetItem: &capability.Note{ID: "n-1", Title: "test"}})
		item, err := svc.Get(t.Context(), "n-1")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "test", item.Title)
	})

	t.Run("get empty id", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Get(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Get(CanceledContext(), "n-1")
		RequireTimeoutError(t, err)
	})

	t.Run("get provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{GetErr: assert.AnError})
		_, err := svc.Get(t.Context(), "n-1")
		RequireProviderError(t, err)
	})

	t.Run("create success", func(t *testing.T) {
		svc := factory(t, NoteConfig{CreateItem: &capability.Note{ID: "new", Title: "test"}})
		item, err := svc.Create(t.Context(), "test", "", "text", "")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "test", item.Title)
	})

	t.Run("create empty title", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Create(t.Context(), "", "", "text", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("create timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Create(CanceledContext(), "test", "", "text", "")
		RequireTimeoutError(t, err)
	})

	t.Run("create provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{CreateErr: assert.AnError})
		_, err := svc.Create(t.Context(), "test", "", "text", "")
		RequireProviderError(t, err)
	})

	t.Run("update success", func(t *testing.T) {
		svc := factory(t, NoteConfig{UpdateItem: &capability.Note{ID: "n-1", Title: "updated"}})
		item, err := svc.Update(t.Context(), "n-1", "new title", "")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "updated", item.Title)
	})

	t.Run("update empty id", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Update(t.Context(), "", "new title", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("update timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Update(CanceledContext(), "n-1", "new title", "")
		RequireTimeoutError(t, err)
	})

	t.Run("update provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{UpdateErr: assert.AnError})
		_, err := svc.Update(t.Context(), "n-1", "new title", "")
		RequireProviderError(t, err)
	})

	t.Run("delete success", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		err := svc.Delete(t.Context(), "n-1")
		require.NoError(t, err)
	})

	t.Run("delete empty id", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		err := svc.Delete(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("delete timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		err := svc.Delete(CanceledContext(), "n-1")
		RequireTimeoutError(t, err)
	})

	t.Run("delete provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{DeleteErr: assert.AnError})
		err := svc.Delete(t.Context(), "n-1")
		RequireProviderError(t, err)
	})

	t.Run("get content success", func(t *testing.T) {
		svc := factory(t, NoteConfig{Content: "hello world"})
		content, err := svc.GetContent(t.Context(), "n-1")
		require.NoError(t, err)
		assert.Equal(t, "hello world", content)
	})

	t.Run("get content empty id", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.GetContent(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get content timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.GetContent(CanceledContext(), "n-1")
		RequireTimeoutError(t, err)
	})

	t.Run("get content provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{ContentErr: assert.AnError})
		_, err := svc.GetContent(t.Context(), "n-1")
		RequireProviderError(t, err)
	})

	t.Run("set content success", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		err := svc.SetContent(t.Context(), "n-1", "new content")
		require.NoError(t, err)
	})

	t.Run("set content empty id", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		err := svc.SetContent(t.Context(), "", "new content")
		RequireInvalidArgError(t, err)
	})

	t.Run("set content timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		err := svc.SetContent(CanceledContext(), "n-1", "new content")
		RequireTimeoutError(t, err)
	})

	t.Run("set content provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{SetContentErr: assert.AnError})
		err := svc.SetContent(t.Context(), "n-1", "new content")
		RequireProviderError(t, err)
	})

	t.Run("search success", func(t *testing.T) {
		svc := factory(t, NoteConfig{
			SearchItems: []*capability.Note{{ID: "n-1", Title: "Match"}},
		})
		result, err := svc.Search(t.Context(), "test")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})

	t.Run("search empty", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		result, err := svc.Search(t.Context(), "nothing")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Items)
	})

	t.Run("search timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.Search(CanceledContext(), "test")
		RequireTimeoutError(t, err)
	})

	t.Run("search provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{SearchErr: assert.AnError})
		_, err := svc.Search(t.Context(), "test")
		RequireProviderError(t, err)
	})

	t.Run("list raw events success", func(t *testing.T) {
		svc := factory(t, NoteConfig{
			RawItems:  []any{map[string]any{"noteId": "n-1"}, map[string]any{"noteId": "n-2"}},
			RawCursor: "next-cursor",
		})
		items, cursor, err := svc.ListRawEvents(t.Context(), "")
		require.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, "next-cursor", cursor)
	})

	t.Run("list raw events empty", func(t *testing.T) {
		svc := factory(t, NoteConfig{
			RawItems:  []any{},
			RawCursor: "",
		})
		items, cursor, err := svc.ListRawEvents(t.Context(), "prev")
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.Empty(t, cursor)
	})

	t.Run("list raw events timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, _, err := svc.ListRawEvents(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("list raw events provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{RawErr: assert.AnError})
		_, _, err := svc.ListRawEvents(t.Context(), "")
		RequireProviderError(t, err)
	})

	t.Run("get app info success", func(t *testing.T) {
		svc := factory(t, NoteConfig{AppInfo: &capability.Note{ID: "instance", Title: "Trilium"}})
		info, err := svc.GetAppInfo(t.Context())
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.NotEmpty(t, info.Title)
	})

	t.Run("get app info timeout", func(t *testing.T) {
		svc := factory(t, NoteConfig{})
		_, err := svc.GetAppInfo(CanceledContext())
		RequireTimeoutError(t, err)
	})

	t.Run("get app info provider error", func(t *testing.T) {
		svc := factory(t, NoteConfig{AppInfoErr: assert.AnError})
		_, err := svc.GetAppInfo(t.Context())
		RequireProviderError(t, err)
	})
}
