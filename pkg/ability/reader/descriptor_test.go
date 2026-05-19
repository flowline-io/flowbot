package reader

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestDescriptor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		svc     Service
		healthy bool
	}{
		{"nil service produces unhealthy descriptor", nil, false},
		{"non-nil service produces healthy descriptor", &mockReaderService{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor("miniflux", "miniflux", tt.svc)
			assert.Equal(t, hub.CapReader, desc.Type)
			assert.Equal(t, "miniflux", desc.Backend)
			assert.Equal(t, "miniflux", desc.App)
			assert.Equal(t, tt.healthy, desc.Healthy)
			assert.Equal(t, "Reader capability", desc.Description)
			assert.Len(t, desc.Operations, 7)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"has list_feeds operation", ability.OpReaderListFeeds},
		{"has create_feed operation", ability.OpReaderCreateFeed},
		{"has list_entries operation", ability.OpReaderListEntries},
		{"has mark_entry_read operation", ability.OpReaderMarkEntryRead},
		{"has star_entry operation", ability.OpReaderStarEntry},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor("m", "m", nil)
			opNames := make([]string, len(desc.Operations))
			for i, op := range desc.Operations {
				opNames[i] = op.Name
			}
			assert.Contains(t, opNames, tt.op)
		})
	}
}

type mockReaderService struct{}

func (*mockReaderService) ListFeeds(_ context.Context, _ *FeedQuery) (*ability.ListResult[ability.Feed], error) {
	return nil, nil
}
func (*mockReaderService) CreateFeed(_ context.Context, _ string) (*ability.Feed, error) {
	return nil, nil
}
func (*mockReaderService) ListEntries(_ context.Context, _ *EntryQuery) (*ability.ListResult[ability.Entry], error) {
	return nil, nil
}
func (*mockReaderService) MarkEntryRead(_ context.Context, _ int64) error   { return nil }
func (*mockReaderService) MarkEntryUnread(_ context.Context, _ int64) error { return nil }
func (*mockReaderService) StarEntry(_ context.Context, _ int64) error       { return nil }
func (*mockReaderService) UnstarEntry(_ context.Context, _ int64) error     { return nil }
