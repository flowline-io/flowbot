package kanboard

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (*mockService) ListTasks(_ context.Context, _ *TaskQuery) (*capability.ListResult[capability.Task], error) {
	return nil, nil
}
func (*mockService) GetTask(_ context.Context, _ int) (*capability.Task, error) { return nil, nil }
func (*mockService) CreateTask(_ context.Context, _ CreateTaskRequest) (*capability.Task, error) {
	return nil, nil
}
func (*mockService) UpdateTask(_ context.Context, _ int, _ UpdateTaskRequest) (*capability.Task, error) {
	return nil, nil
}
func (*mockService) DeleteTask(_ context.Context, _ int) error { return nil }
func (*mockService) MoveTask(_ context.Context, _ int, _ MoveTaskRequest) (*capability.Task, error) {
	return nil, nil
}
func (*mockService) CompleteTask(_ context.Context, _ int) error { return nil }
func (*mockService) GetColumns(_ context.Context, _ int) ([]map[string]any, error) {
	return nil, nil
}
func (*mockService) SearchTasks(_ context.Context, _ *SearchQuery) (*capability.ListResult[capability.Task], error) {
	return nil, nil
}
func (*mockService) HealthCheck(_ context.Context) (bool, error) { return true, nil }

func TestStringListParam_StringSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
		want []string
	}{
		{"string slice tags are returned as-is", []string{"a", "b"}, []string{"a", "b"}},
		{"single element string slice returns correctly", []string{"only"}, []string{"only"}},
		{"multi-element with empty strings returns correctly", []string{"a", "", "c"}, []string{"a", "", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, ok := stringListParam(map[string]any{"tags": tt.tags}, "tags")
			assert.True(t, ok)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestStringListParam_AnySlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
		want []string
	}{
		{"any slice tags are converted to strings", []any{"x", "y"}, []string{"x", "y"}},
		{"single element any slice converted correctly", []any{"single"}, []string{"single"}},
		{"any slice with empty string preserved", []any{"a", "", "c"}, []string{"a", "", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, ok := stringListParam(map[string]any{"tags": tt.tags}, "tags")
			assert.True(t, ok)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestStringListParam_AnySliceMixedTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
		want []string
	}{
		{"mixed type any slice with non-string values skipped", []any{"a", 42}, []string{"a"}},
		{"mixed types with float and string returns only strings", []any{"a", 3.14, "b"}, []string{"a", "b"}},
		{"mixed types all non-string returns empty", []any{1, true, 3.5}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, ok := stringListParam(map[string]any{"tags": tt.tags}, "tags")
			assert.True(t, ok)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestStringListParam_Missing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing key returns false", map[string]any{}},
		{"missing key with other keys present returns false", map[string]any{"other": "val"}},
		{"empty params map returns false", map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ok := stringListParam(tt.params, "tags")
			assert.False(t, ok)
		})
	}
}

func TestStringListParam_Nil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]any
	}{
		{"nil value returns false", map[string]any{"tags": nil}},
		{"nil map returns false for any key", nil},
		{"nil value in populated map returns false", map[string]any{"tags": nil, "other": "val"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ok := stringListParam(tt.params, "tags")
			assert.False(t, ok)
		})
	}
}

func TestStringListParam_NonSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		val  any
	}{
		{"non-slice value returns false", "string"},
		{"integer value returns false", 42},
		{"boolean value returns false", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ok := stringListParam(map[string]any{"tags": tt.val}, "tags")
			assert.False(t, ok)
		})
	}
}

func TestStringListParam_EmptyAnySlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
	}{
		{"empty any slice returns true with empty result", []any{}},
		{"empty string slice returns true with empty result", []string{}},
		{"empty any slice with other keys returns true with empty result", []any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := map[string]any{"tags": tt.tags}
			if tt.name == "empty any slice with other keys returns true with empty result" {
				params["other"] = "val"
			}
			v, ok := stringListParam(params, "tags")
			assert.True(t, ok)
			assert.Empty(t, v)
		})
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockService{}, wantErr: false},
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
	require.NoError(t, Register("kanboard", &mockService{}))
	desc, ok := hub.Default.Get(hub.CapKanboard)
	require.True(t, ok)
	assert.Equal(t, hub.CapKanboard, desc.Type)
	assert.Equal(t, "kanboard", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 10)
	assert.Len(t, desc.Events, 5)

	tests := []struct {
		name string
		op   string
	}{
		{"has list_tasks operation", OpListTasks},
		{"has get_task operation", OpGetTask},
		{"has create_task operation", OpCreateTask},
		{"has move_task operation", OpMoveTask},
		{"has complete_task operation", OpCompleteTask},
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
	require.NoError(t, Register("kanboard", &mockService{}))
	desc, ok := hub.Default.Get(hub.CapKanboard)
	require.True(t, ok)

	tests := []struct {
		name  string
		event string
	}{
		{"has kanban.task.created event", "kanban.task.created"},
		{"has kanban.task.updated event", "kanban.task.updated"},
		{"has kanban.task.completed event", "kanban.task.completed"},
		{"has kanban.task.opened event", "kanban.task.opened"},
		{"has kanban.task.moved event", "kanban.task.moved"},
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
