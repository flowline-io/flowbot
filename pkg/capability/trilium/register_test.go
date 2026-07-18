package trilium

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockNoteService struct{}

func (*mockNoteService) List(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.Note], error) {
	return nil, nil
}
func (*mockNoteService) Get(_ context.Context, _ string) (*capability.Note, error) { return nil, nil }
func (*mockNoteService) Create(_ context.Context, _, _, _, _ string) (*capability.Note, error) {
	return nil, nil
}
func (*mockNoteService) Update(_ context.Context, _, _, _ string) (*capability.Note, error) {
	return nil, nil
}
func (*mockNoteService) Delete(_ context.Context, _ string) error               { return nil }
func (*mockNoteService) GetContent(_ context.Context, _ string) (string, error) { return "", nil }
func (*mockNoteService) SetContent(_ context.Context, _, _ string) error        { return nil }
func (*mockNoteService) Search(_ context.Context, _ string) (*capability.ListResult[capability.Note], error) {
	return nil, nil
}
func (*mockNoteService) GetAppInfo(_ context.Context) (*capability.Note, error) { return nil, nil }
func (*mockNoteService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return nil, "", nil
}
func (*mockNoteService) HealthCheck(_ context.Context) (bool, error) { return true, nil }

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockNoteService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockNoteService{}, wantErr: false},
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
	require.NoError(t, Register("trilium", &mockNoteService{}))
	desc, ok := hub.Default.Get(hub.CapTrilium)
	require.True(t, ok)
	assert.Equal(t, hub.CapTrilium, desc.Type)
	assert.Equal(t, "trilium", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 10)

	tests := []struct {
		name string
		op   string
	}{
		{"has list operation", OpList},
		{"has get operation", OpGet},
		{"has create operation", OpCreate},
		{"has update operation", OpUpdate},
		{"has delete operation", OpDelete},
		{"has get_content operation", OpGetContent},
		{"has set_content operation", OpSetContent},
		{"has search operation", OpSearch},
		{"has get_app_info operation", OpGetAppInfo},
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
