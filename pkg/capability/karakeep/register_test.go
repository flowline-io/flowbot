package karakeep

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockBookmarkService struct{}

func (*mockBookmarkService) List(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.Bookmark], error) {
	return nil, nil
}
func (*mockBookmarkService) Get(_ context.Context, _ string) (*capability.Bookmark, error) {
	return nil, nil
}
func (*mockBookmarkService) Create(_ context.Context, _ string) (*capability.Bookmark, error) {
	return nil, nil
}
func (*mockBookmarkService) Delete(_ context.Context, _ string) error { return nil }
func (*mockBookmarkService) Archive(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (*mockBookmarkService) Search(_ context.Context, _ *SearchQuery) (*capability.ListResult[capability.Bookmark], error) {
	return nil, nil
}
func (*mockBookmarkService) AttachTags(_ context.Context, _ string, _ []string) error { return nil }
func (*mockBookmarkService) DetachTags(_ context.Context, _ string, _ []string) error { return nil }
func (*mockBookmarkService) CheckURL(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockBookmarkService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockBookmarkService{}, wantErr: false},
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
	require.NoError(t, Register("karakeep", &mockBookmarkService{}))
	desc, ok := hub.Default.Get(hub.CapKarakeep)
	require.True(t, ok)
	assert.Equal(t, hub.CapKarakeep, desc.Type)
	assert.Equal(t, "karakeep", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 9)
	assert.Len(t, desc.Events, 4)

	tests := []struct {
		name string
		op   string
	}{
		{"has list operation", OpList},
		{"has get operation", OpGet},
		{"has create operation", OpCreate},
		{"has delete operation", OpDelete},
		{"has archive operation", OpArchive},
		{"has search operation", OpSearch},
		{"has attach_tags operation", OpAttachTags},
		{"has detach_tags operation", OpDetachTags},
		{"has check_url operation", OpCheckURL},
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
	require.NoError(t, Register("karakeep", &mockBookmarkService{}))
	desc, ok := hub.Default.Get(hub.CapKarakeep)
	require.True(t, ok)

	tests := []struct {
		name  string
		event string
	}{
		{"has bookmark.created event", "bookmark.created"},
		{"has bookmark.updated event", "bookmark.updated"},
		{"has bookmark.archived event", "bookmark.archived"},
		{"has bookmark.deleted event", "bookmark.deleted"},
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
