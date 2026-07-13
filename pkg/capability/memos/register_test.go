package memos

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockMemoService struct{}

func (*mockMemoService) List(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.Memo], error) {
	return nil, nil
}
func (*mockMemoService) Get(_ context.Context, _ string) (*capability.Memo, error) { return nil, nil }
func (*mockMemoService) Create(_ context.Context, _, _ string) (*capability.Memo, error) {
	return nil, nil
}
func (*mockMemoService) Update(_ context.Context, _ string, _ map[string]any) (*capability.Memo, error) {
	return nil, nil
}
func (*mockMemoService) Delete(_ context.Context, _ string) error    { return nil }
func (*mockMemoService) HealthCheck(_ context.Context) (bool, error) { return true, nil }
func (*mockMemoService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return nil, "", nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockMemoService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockMemoService{}, wantErr: false},
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
	require.NoError(t, Register("memos", &mockMemoService{}))
	desc, ok := hub.Default.Get(hub.CapMemos)
	require.True(t, ok)
	assert.Equal(t, hub.CapMemos, desc.Type)
	assert.Equal(t, "memos", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 6)

	tests := []struct {
		name string
		op   string
	}{
		{"has list operation", OpList},
		{"has get operation", OpGet},
		{"has create operation", OpCreate},
		{"has update operation", OpUpdate},
		{"has delete operation", OpDelete},
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
