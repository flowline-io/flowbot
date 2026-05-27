package memo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockMemoService struct{}

func (*mockMemoService) List(_ context.Context, _ *ListQuery) (*ability.ListResult[ability.Memo], error) {
	return nil, nil
}
func (*mockMemoService) Get(_ context.Context, _ string) (*ability.Memo, error) { return nil, nil }
func (*mockMemoService) Create(_ context.Context, _, _ string) (*ability.Memo, error) {
	return nil, nil
}
func (*mockMemoService) Update(_ context.Context, _ string, _ map[string]any) (*ability.Memo, error) {
	return nil, nil
}
func (*mockMemoService) Delete(_ context.Context, _ string) error              { return nil }
func (*mockMemoService) HealthCheck(_ context.Context) (bool, error)           { return true, nil }
func (*mockMemoService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return nil, "", nil
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
		{"nil service produces unhealthy descriptor", "memos", "memos", nil, false},
		{"non-nil service produces healthy descriptor", "memos", "memos", &mockMemoService{}, true},
		{"different backend and app names produce correct descriptor", "my-memos", "my-memos-instance", &mockMemoService{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor(tt.backend, tt.app, tt.svc)
			assert.Equal(t, hub.CapMemo, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.Equal(t, tt.wantHealthy, desc.Healthy)
			assert.Equal(t, "Memo capability for short-form note-taking", desc.Description)
			assert.Len(t, desc.Operations, 6)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"has list operation", ability.OpMemoList},
		{"has get operation", ability.OpMemoGet},
		{"has create operation", ability.OpMemoCreate},
		{"has update operation", ability.OpMemoUpdate},
		{"has delete operation", ability.OpMemoDelete},
		{"has health operation", ability.OpMemoHealth},
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
			err := RegisterService("memos", "app1", tt.svc)
			assert.Error(t, err)
		})
	}
}
