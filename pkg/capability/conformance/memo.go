package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// MemoListQuery wraps pagination for listing memos.
type MemoListQuery = capability.MemoListQuery

// MemoService is the memo capability contract used by conformance tests.
type MemoService interface {
	List(ctx context.Context, q *MemoListQuery) (*capability.ListResult[capability.Memo], error)
	Get(ctx context.Context, name string) (*capability.Memo, error)
	Create(ctx context.Context, content, visibility string) (*capability.Memo, error)
	Update(ctx context.Context, name string, data map[string]any) (*capability.Memo, error)
	Delete(ctx context.Context, name string) error
	HealthCheck(ctx context.Context) (bool, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}

// MemoConfig configures the fake backend for each memo conformance subtest.
// Fields set to non-nil/non-zero values become the fake responses; zero values
// produce empty/default success responses.
type MemoConfig struct {
	ListItems      []*capability.Memo
	ListNextCursor string
	ListErr        error
	GetItem        *capability.Memo
	GetErr         error
	CreateItem     *capability.Memo
	CreateErr      error
	UpdateItem     *capability.Memo
	UpdateErr      error
	DeleteErr      error
	HealthOk       bool
	HealthErr      error
	RawItems       []any
	RawCursor      string
	RawErr         error
}

// MemoServiceFactory creates a fresh memo Service wired to a fake backend
// whose behavior is determined by the config parameter.
type MemoServiceFactory func(t *testing.T, cfg MemoConfig) MemoService

// RunMemoConformance runs the standard memo capability conformance suite.
// The factory must wire cfg into a fresh adapter and fake client for each subtest.
func RunMemoConformance(t *testing.T, factory MemoServiceFactory) {
	t.Run("list success", func(t *testing.T) {
		svc := factory(t, MemoConfig{
			ListItems: []*capability.Memo{
				{Name: "memos/1", Content: "First"},
				{Name: "memos/2", Content: "Second"},
			},
			ListNextCursor: "next-token",
		})
		result, err := svc.List(t.Context(), &capability.MemoListQuery{Page: capability.PageRequest{Limit: 20}})
		require.NoError(t, err)
		RequireListResult(t, result, 20, true)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, "next-token", result.Page.NextCursor)
	})

	t.Run("list empty", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		result, err := svc.List(t.Context(), &capability.MemoListQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Empty(t, result.Items)
		assert.False(t, result.Page.HasMore)
	})

	t.Run("list nil query", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		result, err := svc.List(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.List(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{ListErr: assert.AnError})
		_, err := svc.List(t.Context(), &capability.MemoListQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get success", func(t *testing.T) {
		svc := factory(t, MemoConfig{
			GetItem: &capability.Memo{Name: "memos/1", Content: "Hello"},
		})
		item, err := svc.Get(t.Context(), "memos/1")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "memos/1", item.Name)
		assert.Equal(t, "Hello", item.Content)
	})

	t.Run("get timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.Get(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("get empty name", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.Get(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{GetErr: assert.AnError})
		_, err := svc.Get(t.Context(), "memos/1")
		RequireProviderError(t, err)
	})

	t.Run("create success", func(t *testing.T) {
		svc := factory(t, MemoConfig{
			CreateItem: &capability.Memo{Name: "memos/1", Content: "New Memo", Visibility: "PRIVATE"},
		})
		item, err := svc.Create(t.Context(), "New Memo", "PRIVATE")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "memos/1", item.Name)
		assert.Equal(t, "New Memo", item.Content)
	})

	t.Run("create timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.Create(CanceledContext(), "", "")
		RequireTimeoutError(t, err)
	})

	t.Run("create empty content", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.Create(t.Context(), "", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("create provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{CreateErr: assert.AnError})
		_, err := svc.Create(t.Context(), "New Memo", "")
		RequireProviderError(t, err)
	})

	t.Run("update success", func(t *testing.T) {
		svc := factory(t, MemoConfig{
			UpdateItem: &capability.Memo{Name: "memos/1", Content: "Updated", Pinned: true},
		})
		item, err := svc.Update(t.Context(), "memos/1", map[string]any{"content": "Updated", "pinned": true})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "memos/1", item.Name)
		assert.True(t, item.Pinned)
	})

	t.Run("update timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.Update(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("update empty name", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.Update(t.Context(), "", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("update provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{UpdateErr: assert.AnError})
		_, err := svc.Update(t.Context(), "memos/1", nil)
		RequireProviderError(t, err)
	})

	t.Run("delete success", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		err := svc.Delete(t.Context(), "memos/1")
		require.NoError(t, err)
	})

	t.Run("delete timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		err := svc.Delete(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("delete empty name", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		err := svc.Delete(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("delete provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{DeleteErr: assert.AnError})
		err := svc.Delete(t.Context(), "memos/1")
		RequireProviderError(t, err)
	})

	t.Run("health check ok", func(t *testing.T) {
		svc := factory(t, MemoConfig{HealthOk: true})
		ok, err := svc.HealthCheck(t.Context())
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("health check not ok", func(t *testing.T) {
		svc := factory(t, MemoConfig{HealthOk: false})
		ok, err := svc.HealthCheck(t.Context())
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("health check timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, err := svc.HealthCheck(CanceledContext())
		RequireTimeoutError(t, err)
	})

	t.Run("health check provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{HealthErr: assert.AnError})
		_, err := svc.HealthCheck(t.Context())
		RequireProviderError(t, err)
	})

	t.Run("list raw events success", func(t *testing.T) {
		svc := factory(t, MemoConfig{
			RawItems:  []any{map[string]any{"name": "memos/1"}, map[string]any{"name": "memos/2"}},
			RawCursor: "next-cursor",
		})
		items, cursor, err := svc.ListRawEvents(t.Context(), "")
		require.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, "next-cursor", cursor)
	})

	t.Run("list raw events empty", func(t *testing.T) {
		svc := factory(t, MemoConfig{
			RawItems:  []any{},
			RawCursor: "",
		})
		items, cursor, err := svc.ListRawEvents(t.Context(), "prev")
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.Empty(t, cursor)
	})

	t.Run("list raw events timeout", func(t *testing.T) {
		svc := factory(t, MemoConfig{})
		_, _, err := svc.ListRawEvents(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("list raw events provider error", func(t *testing.T) {
		svc := factory(t, MemoConfig{RawErr: assert.AnError})
		_, _, err := svc.ListRawEvents(t.Context(), "")
		RequireProviderError(t, err)
	})
}
