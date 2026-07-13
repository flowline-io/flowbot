package kanboard

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
)

type fakeClient struct {
	tasks        []*provider.Task
	tasksErr     error
	task         *provider.Task
	taskErr      error
	createTaskID int64
	createErr    error
	updateResult bool
	updateErr    error
	closeResult  bool
	closeErr     error
	moveResult   bool
	moveErr      error
	deleteResult bool
	deleteErr    error
	columns      func() []map[string]any
	columnsErr   error
	searchTasks  []*provider.Task
	searchErr    error
}

func (f *fakeClient) GetAllTasks(_ context.Context, _ int, _ provider.StatusId) ([]*provider.Task, error) {
	if f.tasksErr != nil {
		return nil, f.tasksErr
	}
	if f.tasks == nil {
		return []*provider.Task{}, nil
	}
	return f.tasks, nil
}

func (f *fakeClient) GetTask(_ context.Context, taskID int) (*provider.Task, error) {
	if f.taskErr != nil {
		return nil, f.taskErr
	}
	if f.task != nil {
		return f.task, nil
	}
	return &provider.Task{ID: taskID, Title: "Default", ProjectID: 1}, nil
}

func (f *fakeClient) CreateTask(_ context.Context, _ *provider.Task) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	if f.createTaskID > 0 {
		return f.createTaskID, nil
	}
	return 99, nil
}

func (f *fakeClient) UpdateTask(_ context.Context, _ int, _ *provider.Task) (bool, error) {
	if f.updateErr != nil {
		return false, f.updateErr
	}
	return f.updateResult, nil
}

func (f *fakeClient) CloseTask(_ context.Context, _ int) (bool, error) {
	if f.closeErr != nil {
		return false, f.closeErr
	}
	return f.closeResult, nil
}

func (f *fakeClient) RemoveTask(_ context.Context, _ int) (bool, error) {
	if f.deleteErr != nil {
		return false, f.deleteErr
	}
	return f.deleteResult, nil
}

func (f *fakeClient) MoveTaskPosition(_ context.Context, _, _, _, _, _ int) (bool, error) {
	if f.moveErr != nil {
		return false, f.moveErr
	}
	return f.moveResult, nil
}

func (f *fakeClient) GetColumns(_ context.Context, _ int) ([]types.KV, error) {
	if f.columnsErr != nil {
		return nil, f.columnsErr
	}
	if f.columns != nil {
		cols := f.columns()
		result := make([]types.KV, 0, len(cols))
		for _, c := range cols {
			kv := make(types.KV)
			maps.Copy(kv, c)
			result = append(result, kv)
		}
		return result, nil
	}
	return []types.KV{}, nil
}

func (f *fakeClient) SearchTasks(_ context.Context, _ int, _ string) ([]*provider.Task, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchTasks == nil {
		return []*provider.Task{}, nil
	}
	return f.searchTasks, nil
}

func TestListTasksConvertsProviderTasks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		tasks []*provider.Task
	}{
		{"converts kanboard provider tasks to ability tasks", []*provider.Task{
			{ID: 1, Title: "Task 1", ProjectID: 1, ColumnID: 2, Tags: []any{"go", "api"}},
		}},
		{"multiple tasks with different fields converted correctly", []*provider.Task{
			{ID: 1, Title: "First", ProjectID: 1, ColumnID: 1},
			{ID: 2, Title: "Second", ProjectID: 2, ColumnID: 3, Tags: []any{"urgent"}},
		}},
		{"tasks with nil tags converted with empty tags", []*provider.Task{
			{ID: 3, Title: "No Tags", ProjectID: 1, ColumnID: 2},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter, ok := NewWithClient(&fakeClient{tasks: tt.tasks}).(*Adapter)
			require.True(t, ok)
			result, err := adapter.ListTasks(t.Context(), &TaskQuery{ProjectID: 1})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Items, len(tt.tasks))
			for i, task := range tt.tasks {
				assert.Equal(t, task.ID, result.Items[i].ID)
				assert.Equal(t, task.Title, result.Items[i].Title)
				assert.Equal(t, task.ProjectID, result.Items[i].ProjectID)
				assert.Equal(t, task.ColumnID, result.Items[i].ColumnID)
			}
		})
	}
}

func TestListTasksEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fakeSetup func() *fakeClient
	}{
		{"empty task list returns empty items", func() *fakeClient { return &fakeClient{} }},
		{"nil tasks field returns empty items", func() *fakeClient { return &fakeClient{tasks: nil} }},
		{"zero-length tasks slice returns empty items", func() *fakeClient { return &fakeClient{tasks: []*provider.Task{}} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter, ok := NewWithClient(tt.fakeSetup()).(*Adapter)
			require.True(t, ok)
			result, err := adapter.ListTasks(t.Context(), &TaskQuery{})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Empty(t, result.Items)
			assert.NotNil(t, result.Page)
		})
	}
}

func TestGetTaskReturnsConvertedTask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		task *provider.Task
	}{
		{"get task returns converted ability task", &provider.Task{ID: 5, Title: "Test", ProjectID: 1}},
		{"get task with tags conversion preserves tags", &provider.Task{ID: 10, Title: "Tagged", ProjectID: 2, Tags: []any{"go", "api"}}},
		{"get task with all fields populated returns correctly", &provider.Task{ID: 15, Title: "Full", ProjectID: 3, ColumnID: 4, Tags: []any{"kanban"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter, ok := NewWithClient(&fakeClient{task: tt.task}).(*Adapter)
			require.True(t, ok)
			task, err := adapter.GetTask(t.Context(), tt.task.ID)
			require.NoError(t, err)
			require.NotNil(t, task)
			assert.Equal(t, tt.task.ID, task.ID)
			assert.Equal(t, tt.task.Title, task.Title)
		})
	}
}

func TestCreateTaskReturnsAbilityTask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		request CreateTaskRequest
		fakeID  int64
	}{
		{"create task returns ability task with assigned id", CreateTaskRequest{
			Title: "New Task", ProjectID: 1, Tags: []string{"go"},
		}, 10},
		{"create task with no tags creates task correctly", CreateTaskRequest{
			Title: "Untagged", ProjectID: 2,
		}, 20},
		{"create task with multiple tags converted correctly", CreateTaskRequest{
			Title: "Multi", ProjectID: 1, Tags: []string{"go", "api", "kanban"},
		}, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			adapter, ok := NewWithClient(&fakeClient{createTaskID: tt.fakeID}).(*Adapter)
			require.True(t, ok)
			task, err := adapter.CreateTask(t.Context(), tt.request)
			require.NoError(t, err)
			require.NotNil(t, task)
			assert.Equal(t, int(tt.fakeID), task.ID)
			assert.Equal(t, tt.request.Title, task.Title)
		})
	}
}

func TestToAbilityTaskNil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		task *provider.Task
		want func(*testing.T, *capability.Task)
	}{
		{"nil provider task returns nil", nil, func(t *testing.T, task *capability.Task) {
			assert.Nil(t, task)
		}},
		{"non-nil empty provider task returns ability task with zero values", &provider.Task{}, func(t *testing.T, task *capability.Task) {
			require.NotNil(t, task)
			assert.Zero(t, task.ID)
			assert.Empty(t, task.Title)
		}},
		{"provider task with zero id returns zero id", &provider.Task{ID: 0, Title: "Zero"}, func(t *testing.T, task *capability.Task) {
			require.NotNil(t, task)
			assert.Zero(t, task.ID)
			assert.Equal(t, "Zero", task.Title)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := toAbilityTask(tt.task)
			tt.want(t, result)
		})
	}
}

func TestTagsToAnyAndBack(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []string
	}{
		{"round trip string tags to any and back", []string{"a", "b", "c"}},
		{"empty input slice returns empty any slice", []string{}},
		{"single element round trip works", []string{"only"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			anyTags := tagsToAny(tt.input)
			assert.Len(t, anyTags, len(tt.input))
			result := anyToStringSlice(anyTags)
			assert.Equal(t, tt.input, result)
		})
	}
}

func TestCheckClientWithNilClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		adapter *Adapter
		wantErr bool
	}{
		{"nil client returns unavailable error", &Adapter{client: nil}, true},
		{"non-nil client returns nil error", &Adapter{client: &fakeClient{}}, false},
		{"client is set via NewWithClient returns nil error", NewWithClient(&fakeClient{}).(*Adapter), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.adapter.checkClient()
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrUnavailable)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Compile-time interface check.
var _ Service = (*Adapter)(nil)

// Capture unused var to avoid compiler warnings.
var _ = &capability.ListResult[capability.Task]{}
