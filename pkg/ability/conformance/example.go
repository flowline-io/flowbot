package conformance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	exsvc "github.com/flowline-io/flowbot/pkg/ability/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ExampleConfig configures the fake backend for each example conformance subtest.
type ExampleConfig struct {
	GetItem    *ability.Host
	GetErr     error
	ListItems  []*ability.Host
	ListErr    error
	CreateItem *ability.Host
	CreateErr  error
	UpdateItem *ability.Host
	UpdateErr  error
	DeleteErr  error
	HealthOk   bool
	HealthErr  error
	RawItems   []any
	RawCursor  string
	RawErr     error
}

// ExampleServiceFactory creates a fresh example Service wired to a fake backend.
type ExampleServiceFactory func(t *testing.T, cfg ExampleConfig) exsvc.Service

// RunExampleConformance runs the standard example capability conformance suite.
func RunExampleConformance(t *testing.T, factory ExampleServiceFactory) {
	t.Run("get item success", func(t *testing.T) {
		svc := factory(t, ExampleConfig{
			GetItem: &ability.Host{Name: "test-item", Status: "ok"},
		})
		item, err := svc.GetItem(t.Context(), "item-1")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "test-item", item.Name)
	})

	t.Run("get item timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.GetItem(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("get item empty id", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.GetItem(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get item provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{GetErr: assert.AnError})
		_, err := svc.GetItem(t.Context(), "item-1")
		RequireProviderError(t, err)
	})

	t.Run("list items success", func(t *testing.T) {
		svc := factory(t, ExampleConfig{
			ListItems: []*ability.Host{
				{Name: "item-1", Status: "ok"},
				{Name: "item-2", Status: "ok"},
			},
		})
		result, err := svc.ListItems(t.Context(), &exsvc.ListQuery{Page: ability.PageRequest{Limit: 20}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list items empty", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		result, err := svc.ListItems(t.Context(), &exsvc.ListQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list items timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.ListItems(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list items provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{ListErr: assert.AnError})
		_, err := svc.ListItems(t.Context(), &exsvc.ListQuery{})
		RequireProviderError(t, err)
	})

	t.Run("create item success", func(t *testing.T) {
		svc := factory(t, ExampleConfig{
			CreateItem: &ability.Host{Name: "new-item", Status: "ok"},
		})
		item, err := svc.CreateItem(t.Context(), "new-item", types.KV{"key": "value"})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "new-item", item.Name)
	})

	t.Run("create item timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.CreateItem(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("create item empty title", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.CreateItem(t.Context(), "", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("create item provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{CreateErr: assert.AnError})
		_, err := svc.CreateItem(t.Context(), "new-item", nil)
		RequireProviderError(t, err)
	})

	t.Run("delete item timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		err := svc.DeleteItem(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("delete item empty id", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		err := svc.DeleteItem(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("delete item provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{DeleteErr: assert.AnError})
		err := svc.DeleteItem(t.Context(), "item-1")
		RequireProviderError(t, err)
	})

	t.Run("update item success", func(t *testing.T) {
		svc := factory(t, ExampleConfig{
			UpdateItem: &ability.Host{Name: "updated-item", Status: "active"},
		})
		item, err := svc.UpdateItem(t.Context(), "item-1", map[string]any{"title": "updated"})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "updated-item", item.Name)
	})

	t.Run("update item timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.UpdateItem(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("update item empty id", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.UpdateItem(t.Context(), "", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("update item provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{UpdateErr: assert.AnError})
		_, err := svc.UpdateItem(t.Context(), "item-1", nil)
		RequireProviderError(t, err)
	})

	t.Run("list raw events success", func(t *testing.T) {
		svc := factory(t, ExampleConfig{
			RawItems:  []any{map[string]any{"id": "e1"}, map[string]any{"id": "e2"}},
			RawCursor: "next-cursor",
		})
		items, cursor, err := svc.ListRawEvents(t.Context(), "")
		require.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, "next-cursor", cursor)
	})

	t.Run("list raw events with cursor", func(t *testing.T) {
		svc := factory(t, ExampleConfig{
			RawItems:  []any{map[string]any{"id": "e3"}},
			RawCursor: "",
		})
		items, cursor, err := svc.ListRawEvents(t.Context(), "prev-cursor")
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Empty(t, cursor)
	})

	t.Run("list raw events timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, _, err := svc.ListRawEvents(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("list raw events provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{RawErr: assert.AnError})
		_, _, err := svc.ListRawEvents(t.Context(), "")
		RequireProviderError(t, err)
	})

	t.Run("health check ok", func(t *testing.T) {
		svc := factory(t, ExampleConfig{HealthOk: true})
		ok, err := svc.HealthCheck(t.Context())
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("health check not ok", func(t *testing.T) {
		svc := factory(t, ExampleConfig{HealthOk: false})
		ok, err := svc.HealthCheck(t.Context())
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("health check timeout", func(t *testing.T) {
		svc := factory(t, ExampleConfig{})
		_, err := svc.HealthCheck(CanceledContext())
		RequireTimeoutError(t, err)
	})

	t.Run("health check provider error", func(t *testing.T) {
		svc := factory(t, ExampleConfig{HealthErr: assert.AnError})
		_, err := svc.HealthCheck(t.Context())
		RequireProviderError(t, err)
	})
}
