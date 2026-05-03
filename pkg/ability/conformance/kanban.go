package conformance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	kb "github.com/flowline-io/flowbot/pkg/ability/kanban"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// KanbanConfig configures the fake backend for each kanban conformance subtest.
type KanbanConfig struct {
	Tasks        []*ability.Task
	TasksErr     error
	Task         *ability.Task
	TaskErr      error
	CreateTaskID int
	CreateTask   *ability.Task
	CreateErr    error
	UpdateTask   *ability.Task
	UpdateErr    error
	MoveTask     *ability.Task
	MoveErr      error
	DeleteErr    error
	CloseErr     error
	Columns      []map[string]any
	ColumnsErr   error
	SearchTasks  []*ability.Task
	SearchErr    error
	CheckClient  bool
}

// KanbanServiceFactory creates a fresh kanban Service wired to a fake backend
// whose behavior is determined by the config parameter.
type KanbanServiceFactory func(t *testing.T, cfg KanbanConfig) kb.Service

// RunKanbanConformance runs the standard kanban capability conformance suite.
func RunKanbanConformance(t *testing.T, factory KanbanServiceFactory) {
	t.Run("list tasks success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			Tasks: []*ability.Task{
				{ID: 1, Title: "Task 1", ProjectID: 1, ColumnID: 2},
			},
		})
		result, err := svc.ListTasks(t.Context(), &kb.TaskQuery{ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Items)
		assert.NotNil(t, result.Page)
		assert.Len(t, result.Items, 1)
	})

	t.Run("list tasks empty", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		result, err := svc.ListTasks(t.Context(), &kb.TaskQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Items)
	})

	t.Run("list tasks timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.ListTasks(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list tasks provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{TasksErr: assert.AnError})
		_, err := svc.ListTasks(t.Context(), &kb.TaskQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			Task: &ability.Task{ID: 1, Title: "Task 1", ProjectID: 1},
		})
		item, err := svc.GetTask(t.Context(), 1)
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, 1, item.ID)
	})

	t.Run("get task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.GetTask(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("get task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{TaskErr: assert.AnError})
		_, err := svc.GetTask(t.Context(), 1)
		RequireProviderError(t, err)
	})

	t.Run("create task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			CreateTask: &ability.Task{ID: 10, Title: "New Task", ProjectID: 1},
		})
		item, err := svc.CreateTask(t.Context(), kb.CreateTaskRequest{Title: "New Task", ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "New Task", item.Title)
	})

	t.Run("create task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.CreateTask(CanceledContext(), kb.CreateTaskRequest{Title: "Test"})
		RequireTimeoutError(t, err)
	})

	t.Run("create task empty title", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.CreateTask(t.Context(), kb.CreateTaskRequest{})
		RequireInvalidArgError(t, err)
	})

	t.Run("create task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{CreateErr: assert.AnError})
		_, err := svc.CreateTask(t.Context(), kb.CreateTaskRequest{Title: "Test"})
		RequireProviderError(t, err)
	})

	t.Run("update task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			UpdateTask: &ability.Task{ID: 1, Title: "Updated", ProjectID: 1},
		})
		item, err := svc.UpdateTask(t.Context(), 1, kb.UpdateTaskRequest{Title: "Updated"})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "Updated", item.Title)
	})

	t.Run("update task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.UpdateTask(CanceledContext(), 1, kb.UpdateTaskRequest{})
		RequireTimeoutError(t, err)
	})

	t.Run("update task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{UpdateErr: assert.AnError})
		_, err := svc.UpdateTask(t.Context(), 1, kb.UpdateTaskRequest{Title: "X"})
		RequireProviderError(t, err)
	})

	t.Run("delete task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		err := svc.DeleteTask(t.Context(), 1)
		require.NoError(t, err)
	})

	t.Run("delete task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		err := svc.DeleteTask(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("delete task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{DeleteErr: assert.AnError})
		err := svc.DeleteTask(t.Context(), 1)
		RequireProviderError(t, err)
	})

	t.Run("move task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			MoveTask: &ability.Task{ID: 1, Title: "Task 1", ColumnID: 3},
		})
		item, err := svc.MoveTask(t.Context(), 1, kb.MoveTaskRequest{ColumnID: 3, ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, 3, item.ColumnID)
	})

	t.Run("move task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.MoveTask(CanceledContext(), 1, kb.MoveTaskRequest{})
		RequireTimeoutError(t, err)
	})

	t.Run("move task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{MoveErr: assert.AnError})
		_, err := svc.MoveTask(t.Context(), 1, kb.MoveTaskRequest{ColumnID: 3})
		RequireProviderError(t, err)
	})

	t.Run("complete task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		err := svc.CompleteTask(t.Context(), 1)
		require.NoError(t, err)
	})

	t.Run("complete task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		err := svc.CompleteTask(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("complete task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{CloseErr: assert.AnError})
		err := svc.CompleteTask(t.Context(), 1)
		RequireProviderError(t, err)
	})

	t.Run("get columns success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			Columns: []map[string]any{{"id": 1, "title": "Backlog"}},
		})
		cols, err := svc.GetColumns(t.Context(), 1)
		require.NoError(t, err)
		assert.Len(t, cols, 1)
	})

	t.Run("get columns timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.GetColumns(CanceledContext(), 1)
		RequireTimeoutError(t, err)
	})

	t.Run("get columns provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{ColumnsErr: assert.AnError})
		_, err := svc.GetColumns(t.Context(), 1)
		RequireProviderError(t, err)
	})

	t.Run("search tasks success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			SearchTasks: []*ability.Task{{ID: 1, Title: "Match"}},
		})
		result, err := svc.SearchTasks(t.Context(), &kb.SearchQuery{Q: "test", ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})

	t.Run("search tasks timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.SearchTasks(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("search tasks provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{SearchErr: assert.AnError})
		_, err := svc.SearchTasks(t.Context(), &kb.SearchQuery{})
		RequireProviderError(t, err)
	})
}
