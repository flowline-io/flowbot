package bookmark

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockBookmarkService struct{}

func (*mockBookmarkService) List(_ context.Context, _ *ListQuery) (*ability.ListResult[ability.Bookmark], error) {
	return nil, nil
}
func (*mockBookmarkService) Get(_ context.Context, _ string) (*ability.Bookmark, error) {
	return nil, nil
}
func (*mockBookmarkService) Create(_ context.Context, _ string) (*ability.Bookmark, error) {
	return nil, nil
}
func (*mockBookmarkService) Delete(_ context.Context, _ string) error { return nil }
func (*mockBookmarkService) Archive(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (*mockBookmarkService) Search(_ context.Context, _ *SearchQuery) (*ability.ListResult[ability.Bookmark], error) {
	return nil, nil
}
func (*mockBookmarkService) AttachTags(_ context.Context, _ string, _ []string) error { return nil }
func (*mockBookmarkService) DetachTags(_ context.Context, _ string, _ []string) error { return nil }
func (*mockBookmarkService) CheckURL(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

func TestDescriptor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		backend     string
		app         string
		svc         Service
		wantHealthy bool
	}{
		{"nil service produces unhealthy descriptor", "karakeep", "karakeep", nil, false},
		{"non-nil service produces healthy descriptor", "karakeep", "karakeep", &mockBookmarkService{}, true},
		{"different backend and app names produce correct descriptor", "linkding", "linkding-instance", &mockBookmarkService{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor(tt.backend, tt.app, tt.svc)
			assert.Equal(t, hub.CapBookmark, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.Equal(t, tt.wantHealthy, desc.Healthy)
			assert.Equal(t, "Bookmark capability", desc.Description)
			assert.Len(t, desc.Operations, 9)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"has list operation", ability.OpBookmarkList},
		{"has get operation", ability.OpBookmarkGet},
		{"has create operation", ability.OpBookmarkCreate},
		{"has delete operation", ability.OpBookmarkDelete},
		{"has archive operation", ability.OpBookmarkArchive},
		{"has search operation", ability.OpBookmarkSearch},
		{"has attach_tags operation", ability.OpBookmarkAttachTags},
		{"has detach_tags operation", ability.OpBookmarkDetachTags},
		{"has check_url operation", ability.OpBookmarkCheckURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor("k", "k", nil)
			opNames := make([]string, len(desc.Operations))
			for i, op := range desc.Operations {
				opNames[i] = op.Name
			}
			assert.Contains(t, opNames, tt.op)
		})
	}
}

func TestRegisterService_NilService(t *testing.T) {
	tests := []struct {
		name string
		svc  Service
	}{
		{name: "nil service returns nil", svc: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterService("karakeep", "app1", tt.svc)
			assert.NoError(t, err)
		})
	}
}
