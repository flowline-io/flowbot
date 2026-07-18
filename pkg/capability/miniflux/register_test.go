package miniflux

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockReaderService struct{}

func (*mockReaderService) ListFeeds(_ context.Context, _ *FeedQuery) (*capability.ListResult[capability.Feed], error) {
	return nil, nil
}
func (*mockReaderService) CreateFeed(_ context.Context, _ string) (*capability.Feed, error) {
	return nil, nil
}
func (*mockReaderService) ListEntries(_ context.Context, _ *EntryQuery) (*capability.ListResult[capability.Entry], error) {
	return nil, nil
}
func (*mockReaderService) MarkEntryRead(_ context.Context, _ int64) error   { return nil }
func (*mockReaderService) MarkEntryUnread(_ context.Context, _ int64) error { return nil }
func (*mockReaderService) StarEntry(_ context.Context, _ int64) error       { return nil }
func (*mockReaderService) UnstarEntry(_ context.Context, _ int64) error     { return nil }
func (*mockReaderService) HealthCheck(_ context.Context) (bool, error)      { return true, nil }

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockReaderService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockReaderService{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.app, tt.svc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister_Operations(t *testing.T) {
	require.NoError(t, Register("miniflux", &mockReaderService{}))
	desc, ok := hub.Default.Get(hub.CapMiniflux)
	require.True(t, ok)
	assert.Equal(t, hub.CapMiniflux, desc.Type)
	assert.Equal(t, "miniflux", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 8)
	assert.Len(t, desc.Events, 4)

	tests := []struct {
		name string
		op   string
	}{
		{"has list_feeds operation", OpListFeeds},
		{"has create_feed operation", OpCreateFeed},
		{"has list_entries operation", OpListEntries},
		{"has mark_entry_read operation", OpMarkEntryRead},
		{"has star_entry operation", OpStarEntry},
		{"has health operation", OpHealth},
	}
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, opNames, tt.op)
		})
	}
}

func TestRegister_Events(t *testing.T) {
	require.NoError(t, Register("miniflux", &mockReaderService{}))
	desc, ok := hub.Default.Get(hub.CapMiniflux)
	require.True(t, ok)

	tests := []struct {
		name  string
		event string
	}{
		{"has reader.entry.new event", "reader.entry.new"},
		{"has reader.entry.saved event", "reader.entry.saved"},
		{"has reader.entry.starred event", "reader.entry.starred"},
		{"has reader.entry.read event", "reader.entry.read"},
	}
	eventNames := make([]string, len(desc.Events))
	for i, ev := range desc.Events {
		eventNames[i] = ev.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, eventNames, tt.event)
		})
	}
}
