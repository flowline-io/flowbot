package reader

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
)

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
