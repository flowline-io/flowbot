package example

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (mockService) GetItem(_ context.Context, _ string) (*ability.Host, error) { return nil, nil }
func (mockService) ListItems(_ context.Context, _ *ListQuery) (*ability.ListResult[ability.Host], error) {
	return nil, nil
}
func (mockService) CreateItem(_ context.Context, _ string) (*ability.Host, error) { return nil, nil }
func (mockService) UpdateItem(_ context.Context, _ string, _ map[string]any) (*ability.Host, error) {
	return nil, nil
}
func (mockService) DeleteItem(_ context.Context, _ string) error { return nil }
func (mockService) HealthCheck(_ context.Context) (bool, error)  { return true, nil }
func (mockService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return nil, "", nil
}

func TestDescriptor_NilService(t *testing.T) {
	t.Parallel()
	desc := Descriptor("backend", "app1", nil)
	assert.False(t, desc.Healthy)
	assert.Equal(t, hub.CapExample, desc.Type)
	assert.Equal(t, "backend", desc.Backend)
}

func TestDescriptor_WithService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		backend string
		app     string
	}{
		{name: "valid backend and app", backend: "example", app: "app1"},
		{name: "empty backend", backend: "", app: "app1"},
		{name: "empty app", backend: "example", app: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := mockService{}
			desc := Descriptor(tt.backend, tt.app, s)
			assert.True(t, desc.Healthy)
			assert.Equal(t, hub.CapExample, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.NotNil(t, desc.Instance)
			assert.NotEmpty(t, desc.Operations)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		opName string
		wantOp string
	}{
		{name: "list operation", opName: "List", wantOp: OpExampleList},
		{name: "get operation", opName: "Get", wantOp: OpExampleGet},
		{name: "create operation", opName: "Create", wantOp: OpExampleCreate},
		{name: "update operation", opName: "Update", wantOp: OpExampleUpdate},
		{name: "delete operation", opName: "Delete", wantOp: OpExampleDelete},
		{name: "health operation", opName: "Health", wantOp: OpExampleHealth},
	}
	desc := Descriptor("example", "app", mockService{})
	ops := make(map[string]hub.Operation, len(desc.Operations))
	for _, o := range desc.Operations {
		ops[o.Name] = o
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, ops, tt.wantOp, "operation %s should exist", tt.wantOp)
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
			err := RegisterService("example", "app1", tt.svc)
			assert.Error(t, err)
		})
	}
}
