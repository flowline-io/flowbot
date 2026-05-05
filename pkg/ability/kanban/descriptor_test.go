package kanban

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
)

func TestStringListParam_StringSlice(t *testing.T) {
	v, ok := stringListParam(map[string]any{"tags": []string{"a", "b"}}, "tags")
	assert.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, v)
}

func TestStringListParam_AnySlice(t *testing.T) {
	v, ok := stringListParam(map[string]any{"tags": []any{"x", "y"}}, "tags")
	assert.True(t, ok)
	assert.Equal(t, []string{"x", "y"}, v)
}

func TestStringListParam_AnySliceMixedTypes(t *testing.T) {
	v, ok := stringListParam(map[string]any{"tags": []any{"a", 42}}, "tags")
	assert.True(t, ok)
	assert.Equal(t, []string{"a"}, v)
}

func TestStringListParam_Missing(t *testing.T) {
	_, ok := stringListParam(map[string]any{}, "tags")
	assert.False(t, ok)
}

func TestStringListParam_Nil(t *testing.T) {
	_, ok := stringListParam(map[string]any{"tags": nil}, "tags")
	assert.False(t, ok)
}

func TestStringListParam_NonSlice(t *testing.T) {
	_, ok := stringListParam(map[string]any{"tags": "string"}, "tags")
	assert.False(t, ok)
}

func TestStringListParam_EmptyAnySlice(t *testing.T) {
	v, ok := stringListParam(map[string]any{"tags": []any{}}, "tags")
	assert.True(t, ok)
	assert.Empty(t, v)
}

func TestDescriptor(t *testing.T) {
	desc := Descriptor("kanboard", "kanboard", nil)
	assert.Equal(t, hub.CapKanban, desc.Type)
	assert.Equal(t, "kanboard", desc.Backend)
	assert.Equal(t, "kanboard", desc.App)
	assert.False(t, desc.Healthy)
	assert.Equal(t, "Kanban capability", desc.Description)
	assert.Len(t, desc.Operations, 9)

	desc2 := Descriptor("kanboard", "kanboard", &mockService{})
	assert.True(t, desc2.Healthy)
}

func TestDescriptor_Operations(t *testing.T) {
	desc := Descriptor("k", "k", nil)
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	assert.Contains(t, opNames, ability.OpKanbanListTasks)
	assert.Contains(t, opNames, ability.OpKanbanGetTask)
	assert.Contains(t, opNames, ability.OpKanbanCreateTask)
	assert.Contains(t, opNames, ability.OpKanbanMoveTask)
	assert.Contains(t, opNames, ability.OpKanbanCompleteTask)
}

type mockService struct{}

func (m *mockService) ListTasks(_ context.Context, _ *TaskQuery) (*ability.ListResult[ability.Task], error) {
	return nil, nil
}
func (m *mockService) GetTask(_ context.Context, _ int) (*ability.Task, error) { return nil, nil }
func (m *mockService) CreateTask(_ context.Context, _ CreateTaskRequest) (*ability.Task, error) {
	return nil, nil
}
func (m *mockService) UpdateTask(_ context.Context, _ int, _ UpdateTaskRequest) (*ability.Task, error) {
	return nil, nil
}
func (m *mockService) DeleteTask(_ context.Context, _ int) error { return nil }
func (m *mockService) MoveTask(_ context.Context, _ int, _ MoveTaskRequest) (*ability.Task, error) {
	return nil, nil
}
func (m *mockService) CompleteTask(_ context.Context, _ int) error { return nil }
func (m *mockService) GetColumns(_ context.Context, _ int) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockService) SearchTasks(_ context.Context, _ *SearchQuery) (*ability.ListResult[ability.Task], error) {
	return nil, nil
}
