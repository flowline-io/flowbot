package reader

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringParam_Matches(t *testing.T) {
	v, ok := stringParam(map[string]any{"key": "hello"}, "key")
	assert.True(t, ok)
	assert.Equal(t, "hello", v)
}

func TestStringParam_NonString(t *testing.T) {
	_, ok := stringParam(map[string]any{"key": 42}, "key")
	assert.False(t, ok)
}

func TestStringParam_Missing(t *testing.T) {
	_, ok := stringParam(map[string]any{}, "key")
	assert.False(t, ok)
}

func TestStringParam_Nil(t *testing.T) {
	_, ok := stringParam(map[string]any{"key": nil}, "key")
	assert.False(t, ok)
}

func TestInt64Param_Int64(t *testing.T) {
	v, ok := int64Param(map[string]any{"key": int64(100)}, "key")
	assert.True(t, ok)
	assert.Equal(t, int64(100), v)
}

func TestInt64Param_Int(t *testing.T) {
	v, ok := int64Param(map[string]any{"key": 42}, "key")
	assert.True(t, ok)
	assert.Equal(t, int64(42), v)
}

func TestInt64Param_Float64(t *testing.T) {
	v, ok := int64Param(map[string]any{"key": float64(99.0)}, "key")
	assert.True(t, ok)
	assert.Equal(t, int64(99), v)
}

func TestInt64Param_String(t *testing.T) {
	_, ok := int64Param(map[string]any{"key": "42"}, "key")
	assert.False(t, ok)
}

func TestInt64Param_Missing(t *testing.T) {
	_, ok := int64Param(map[string]any{}, "key")
	assert.False(t, ok)
}

func TestRequiredString_Present(t *testing.T) {
	v, err := requiredString(map[string]any{"feed_url": "https://rss.example.com"}, "feed_url")
	require.NoError(t, err)
	assert.Equal(t, "https://rss.example.com", v)
}

func TestRequiredString_Missing(t *testing.T) {
	_, err := requiredString(map[string]any{}, "feed_url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feed_url is required")
}

func TestRequiredString_Empty(t *testing.T) {
	_, err := requiredString(map[string]any{"feed_url": ""}, "feed_url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feed_url is required")
}

func TestRequiredInt64_Present(t *testing.T) {
	v, err := requiredInt64(map[string]any{"id": int64(42)}, "id")
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)
}

func TestRequiredInt64_Missing(t *testing.T) {
	_, err := requiredInt64(map[string]any{}, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestRequiredInt64_FromFloat64(t *testing.T) {
	v, err := requiredInt64(map[string]any{"id": float64(99)}, "id")
	require.NoError(t, err)
	assert.Equal(t, int64(99), v)
}

func TestDescriptor(t *testing.T) {
	desc := Descriptor("miniflux", "miniflux", nil)
	assert.Equal(t, hub.CapReader, desc.Type)
	assert.Equal(t, "miniflux", desc.Backend)
	assert.Equal(t, "miniflux", desc.App)
	assert.False(t, desc.Healthy)
	assert.Equal(t, "Reader capability", desc.Description)
	assert.Len(t, desc.Operations, 7)

	desc2 := Descriptor("miniflux", "miniflux", &mockReaderService{})
	assert.True(t, desc2.Healthy)
}

func TestDescriptor_Operations(t *testing.T) {
	desc := Descriptor("m", "m", nil)
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	assert.Contains(t, opNames, ability.OpReaderListFeeds)
	assert.Contains(t, opNames, ability.OpReaderCreateFeed)
	assert.Contains(t, opNames, ability.OpReaderListEntries)
	assert.Contains(t, opNames, ability.OpReaderMarkEntryRead)
	assert.Contains(t, opNames, ability.OpReaderStarEntry)
}

type mockReaderService struct{}

func (m *mockReaderService) ListFeeds(_ context.Context, _ *FeedQuery) (*ability.ListResult[ability.Feed], error) {
	return nil, nil
}
func (m *mockReaderService) CreateFeed(_ context.Context, _ string) (*ability.Feed, error) {
	return nil, nil
}
func (m *mockReaderService) ListEntries(_ context.Context, _ *EntryQuery) (*ability.ListResult[ability.Entry], error) {
	return nil, nil
}
func (m *mockReaderService) MarkEntryRead(_ context.Context, _ int64) error   { return nil }
func (m *mockReaderService) MarkEntryUnread(_ context.Context, _ int64) error { return nil }
func (m *mockReaderService) StarEntry(_ context.Context, _ int64) error       { return nil }
func (m *mockReaderService) UnstarEntry(_ context.Context, _ int64) error     { return nil }
