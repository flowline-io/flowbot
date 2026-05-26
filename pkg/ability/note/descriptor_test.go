package note

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockNoteService struct{}

func (*mockNoteService) List(_ context.Context, _ *ListQuery) (*ability.ListResult[ability.Note], error) {
	return nil, nil
}
func (*mockNoteService) Get(_ context.Context, _ string) (*ability.Note, error) { return nil, nil }
func (*mockNoteService) Create(_ context.Context, _, _, _, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*mockNoteService) Update(_ context.Context, _, _, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*mockNoteService) Delete(_ context.Context, _ string) error               { return nil }
func (*mockNoteService) GetContent(_ context.Context, _ string) (string, error) { return "", nil }
func (*mockNoteService) SetContent(_ context.Context, _, _ string) error        { return nil }
func (*mockNoteService) Search(_ context.Context, _ string) (*ability.ListResult[ability.Note], error) {
	return nil, nil
}
func (*mockNoteService) GetAppInfo(_ context.Context) (*ability.Note, error) { return nil, nil }

func TestDescriptor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		backend     string
		app         string
		svc         Service
		wantHealthy bool
	}{
		{"nil service produces unhealthy descriptor", "trilium", "trilium", nil, false},
		{"non-nil service produces healthy descriptor", "trilium", "trilium", &mockNoteService{}, true},
		{"different backend and app names produce correct descriptor", "joplin", "joplin-instance", &mockNoteService{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor(tt.backend, tt.app, tt.svc)
			assert.Equal(t, hub.CapNote, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.Equal(t, tt.wantHealthy, desc.Healthy)
			assert.Equal(t, "Note capability for note-taking systems", desc.Description)
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
		{"has list operation", ability.OpNoteList},
		{"has get operation", ability.OpNoteGet},
		{"has create operation", ability.OpNoteCreate},
		{"has update operation", ability.OpNoteUpdate},
		{"has delete operation", ability.OpNoteDelete},
		{"has get_content operation", ability.OpNoteGetContent},
		{"has set_content operation", ability.OpNoteSetContent},
		{"has search operation", ability.OpNoteSearch},
		{"has get_app_info operation", ability.OpNoteGetAppInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor("n", "n", nil)
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
		{name: "nil service returns error", svc: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterService("trilium", "app1", tt.svc)
			assert.Error(t, err)
		})
	}
}
