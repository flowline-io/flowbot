package kanboard

import (
	"context"
	"testing"

	"errors"

	"github.com/flowline-io/flowbot/pkg/ability"
	kb "github.com/flowline-io/flowbot/pkg/ability/kanban"
	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (f *fakeClient) GetAllTasks(ctx context.Context, projectID int, status provider.StatusId) ([]*provider.Task, error) {
	if f.tasksErr != nil {
		return nil, f.tasksErr
	}
	if f.tasks == nil {
		return []*provider.Task{}, nil
	}
	return f.tasks, nil
}

func (f *fakeClient) GetTask(ctx context.Context, taskID int) (*provider.Task, error) {
	if f.taskErr != nil {
		return nil, f.taskErr
	}
	if f.task != nil {
		return f.task, nil
	}
	return &provider.Task{ID: taskID, Title: "Default", ProjectID: 1}, nil
}

func (f *fakeClient) CreateTask(ctx context.Context, task *provider.Task) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	if f.createTaskID > 0 {
		return f.createTaskID, nil
	}
	return 99, nil
}

func (f *fakeClient) UpdateTask(ctx context.Context, taskID int, task *provider.Task) (bool, error) {
	if f.updateErr != nil {
		return false, f.updateErr
	}
	return f.updateResult, nil
}

func (f *fakeClient) CloseTask(ctx context.Context, taskID int) (bool, error) {
	if f.closeErr != nil {
		return false, f.closeErr
	}
	return f.closeResult, nil
}

func (f *fakeClient) RemoveTask(ctx context.Context, taskID int) (bool, error) {
	if f.deleteErr != nil {
		return false, f.deleteErr
	}
	return f.deleteResult, nil
}

func (f *fakeClient) MoveTaskPosition(ctx context.Context, projectID, taskID, columnID, position, swimlaneID int) (bool, error) {
	if f.moveErr != nil {
		return false, f.moveErr
	}
	return f.moveResult, nil
}

func (f *fakeClient) GetColumns(ctx context.Context, projectID int) ([]types.KV, error) {
	if f.columnsErr != nil {
		return nil, f.columnsErr
	}
	if f.columns != nil {
		cols := f.columns()
		result := make([]types.KV, 0, len(cols))
		for _, c := range cols {
			kv := make(types.KV)
			for k, v := range c {
				kv[k] = v
			}
			result = append(result, kv)
		}
		return result, nil
	}
	return []types.KV{}, nil
}

func (f *fakeClient) SearchTasks(ctx context.Context, projectID int, query string) ([]*provider.Task, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchTasks == nil {
		return []*provider.Task{}, nil
	}
	return f.searchTasks, nil
}

func TestListTasksConvertsProviderTasks(t *testing.T) {
	adapter := NewWithClient(&fakeClient{
		tasks: []*provider.Task{
			{ID: 1, Title: "Task 1", ProjectID: 1, ColumnID: 2, Tags: []any{"go", "api"}},
		},
	}).(*Adapter)

	result, err := adapter.ListTasks(t.Context(), &kb.TaskQuery{ProjectID: 1})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 1)
	assert.Equal(t, 1, result.Items[0].ID)
	assert.Equal(t, "Task 1", result.Items[0].Title)
	assert.Equal(t, 1, result.Items[0].ProjectID)
	assert.Equal(t, 2, result.Items[0].ColumnID)
}

func TestListTasksEmpty(t *testing.T) {
	adapter := NewWithClient(&fakeClient{}).(*Adapter)
	result, err := adapter.ListTasks(t.Context(), &kb.TaskQuery{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.NotNil(t, result.Page)
}

func TestGetTaskReturnsConvertedTask(t *testing.T) {
	adapter := NewWithClient(&fakeClient{
		task: &provider.Task{ID: 5, Title: "Test", ProjectID: 1},
	}).(*Adapter)

	task, err := adapter.GetTask(t.Context(), 5)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, 5, task.ID)
	assert.Equal(t, "Test", task.Title)
}

func TestCreateTaskReturnsAbilityTask(t *testing.T) {
	adapter := NewWithClient(&fakeClient{
		createTaskID: 10,
	}).(*Adapter)

	task, err := adapter.CreateTask(t.Context(), kb.CreateTaskRequest{
		Title:     "New Task",
		ProjectID: 1,
		Tags:      []string{"go"},
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, 10, task.ID)
	assert.Equal(t, "New Task", task.Title)
}

func TestToAbilityTaskNil(t *testing.T) {
	result := toAbilityTask(nil)
	assert.Nil(t, result)
}

func TestTagsToAnyAndBack(t *testing.T) {
	input := []string{"a", "b", "c"}
	anyTags := tagsToAny(input)
	assert.Len(t, anyTags, 3)
	result := anyToStringSlice(anyTags)
	assert.Equal(t, input, result)
}

func TestCheckClientWithNilClient(t *testing.T) {
	adapter := &Adapter{client: nil}
	err := adapter.checkClient()
	require.Error(t, err)
	assert.True(t, errors.Is(err, types.ErrUnavailable))
}

// Compile-time interface check.
var _ kb.Service = (*Adapter)(nil)

// Capture unused var to avoid compiler warnings.
var _ = &ability.ListResult[ability.Task]{}
