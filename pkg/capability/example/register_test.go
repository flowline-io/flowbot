package example

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type mockService struct{}

func (mockService) GetItem(_ context.Context, _ string) (*capability.Host, error) { return nil, nil }
func (mockService) ListItems(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.Host], error) {
	return nil, nil
}
func (mockService) CreateItem(_ context.Context, _ string, _ types.KV) (*capability.Host, error) {
	return nil, nil
}
func (mockService) UpdateItem(_ context.Context, _ string, _ map[string]any) (*capability.Host, error) {
	return nil, nil
}
func (mockService) DeleteItem(_ context.Context, _ string) error { return nil }
func (mockService) HealthCheck(_ context.Context) (bool, error)  { return true, nil }
func (mockService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return nil, "", nil
}

func TestRegister(t *testing.T) {
	s := mockService{}
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: s, wantErr: false},
		{name: "empty app with valid service", app: "", svc: s, wantErr: false},
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
	require.NoError(t, Register("example", mockService{}))
	desc, ok := hub.Default.Get(hub.CapExample)
	require.True(t, ok)
	assert.Equal(t, hub.CapExample, desc.Type)
	assert.True(t, desc.Healthy)
	assert.NotNil(t, desc.Instance)
	assert.NotEmpty(t, desc.Operations)

	tests := []struct {
		name   string
		wantOp string
	}{
		{name: "list operation", wantOp: OpList},
		{name: "get operation", wantOp: OpGet},
		{name: "create operation", wantOp: OpCreate},
		{name: "update operation", wantOp: OpUpdate},
		{name: "delete operation", wantOp: OpDelete},
		{name: "health operation", wantOp: OpHealth},
	}
	ops := make(map[string]hub.Operation, len(desc.Operations))
	for _, o := range desc.Operations {
		ops[o.Name] = o
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, ops, tt.wantOp, "operation %s should exist", tt.wantOp)
		})
	}
}
