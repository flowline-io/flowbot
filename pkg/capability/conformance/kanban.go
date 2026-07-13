package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// KanbanTaskQuery wraps pagination and filters for listing tasks.
type KanbanTaskQuery = capability.KanbanTaskQuery

// KanbanCreateTaskRequest holds fields for creating a task.
type KanbanCreateTaskRequest = capability.KanbanCreateTaskRequest

// KanbanUpdateTaskRequest holds fields for updating a task.
type KanbanUpdateTaskRequest = capability.KanbanUpdateTaskRequest

// KanbanMoveTaskRequest holds fields for moving a task.
type KanbanMoveTaskRequest = capability.KanbanMoveTaskRequest

// KanbanSearchQuery wraps pagination for searching tasks.
type KanbanSearchQuery = capability.KanbanSearchQuery

// KanbanService is the kanban capability contract used by conformance tests.
type KanbanService interface {
	ListTasks(ctx context.Context, q *KanbanTaskQuery) (*capability.ListResult[capability.Task], error)
	GetTask(ctx context.Context, id int) (*capability.Task, error)
	CreateTask(ctx context.Context, req KanbanCreateTaskRequest) (*capability.Task, error)
	UpdateTask(ctx context.Context, id int, req KanbanUpdateTaskRequest) (*capability.Task, error)
	DeleteTask(ctx context.Context, id int) error
	MoveTask(ctx context.Context, id int, req KanbanMoveTaskRequest) (*capability.Task, error)
	CompleteTask(ctx context.Context, id int) error
	GetColumns(ctx context.Context, projectID int) ([]map[string]any, error)
	SearchTasks(ctx context.Context, q *KanbanSearchQuery) (*capability.ListResult[capability.Task], error)
}

// KanbanConfig configures the fake backend for each kanban conformance subtest.
type KanbanConfig struct {
	Tasks        []*capability.Task
	TasksErr     error
	Task         *capability.Task
	TaskErr      error
	CreateTaskID int
	CreateTask   *capability.Task
	CreateErr    error
	UpdateTask   *capability.Task
	UpdateErr    error
	MoveTask     *capability.Task
	MoveErr      error
	DeleteErr    error
	CloseErr     error
	Columns      []map[string]any
	ColumnsErr   error
	SearchTasks  []*capability.Task
	SearchErr    error
	CheckClient  bool
}

// KanbanServiceFactory creates a fresh kanban Service wired to a fake backend
// whose behavior is determined by the config parameter.
type KanbanServiceFactory func(t *testing.T, cfg KanbanConfig) KanbanService

// RunKanbanConformance runs the standard kanban capability conformance suite.
func RunKanbanConformance(t *testing.T, factory KanbanServiceFactory) {
	t.Run("list tasks success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			Tasks: []*capability.Task{
				{ID: 1, Title: "Task 1", ProjectID: 1, ColumnID: 2},
			},
		})
		result, err := svc.ListTasks(t.Context(), &capability.KanbanTaskQuery{ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Items)
		assert.NotNil(t, result.Page)
		assert.Len(t, result.Items, 1)
	})

	t.Run("list tasks empty", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		result, err := svc.ListTasks(t.Context(), &capability.KanbanTaskQuery{})
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
		_, err := svc.ListTasks(t.Context(), &capability.KanbanTaskQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			Task: &capability.Task{ID: 1, Title: "Task 1", ProjectID: 1},
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
			CreateTask: &capability.Task{ID: 10, Title: "New Task", ProjectID: 1},
		})
		item, err := svc.CreateTask(t.Context(), capability.KanbanCreateTaskRequest{Title: "New Task", ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "New Task", item.Title)
	})

	t.Run("create task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.CreateTask(CanceledContext(), capability.KanbanCreateTaskRequest{Title: "Test"})
		RequireTimeoutError(t, err)
	})

	t.Run("create task empty title", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.CreateTask(t.Context(), capability.KanbanCreateTaskRequest{})
		RequireInvalidArgError(t, err)
	})

	t.Run("create task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{CreateErr: assert.AnError})
		_, err := svc.CreateTask(t.Context(), capability.KanbanCreateTaskRequest{Title: "Test"})
		RequireProviderError(t, err)
	})

	t.Run("update task success", func(t *testing.T) {
		svc := factory(t, KanbanConfig{
			UpdateTask: &capability.Task{ID: 1, Title: "Updated", ProjectID: 1},
		})
		item, err := svc.UpdateTask(t.Context(), 1, capability.KanbanUpdateTaskRequest{Title: "Updated"})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "Updated", item.Title)
	})

	t.Run("update task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.UpdateTask(CanceledContext(), 1, capability.KanbanUpdateTaskRequest{})
		RequireTimeoutError(t, err)
	})

	t.Run("update task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{UpdateErr: assert.AnError})
		_, err := svc.UpdateTask(t.Context(), 1, capability.KanbanUpdateTaskRequest{Title: "X"})
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
			MoveTask: &capability.Task{ID: 1, Title: "Task 1", ColumnID: 3},
		})
		item, err := svc.MoveTask(t.Context(), 1, capability.KanbanMoveTaskRequest{ColumnID: 3, ProjectID: 1})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, 3, item.ColumnID)
	})

	t.Run("move task timeout", func(t *testing.T) {
		svc := factory(t, KanbanConfig{})
		_, err := svc.MoveTask(CanceledContext(), 1, capability.KanbanMoveTaskRequest{})
		RequireTimeoutError(t, err)
	})

	t.Run("move task provider error", func(t *testing.T) {
		svc := factory(t, KanbanConfig{MoveErr: assert.AnError})
		_, err := svc.MoveTask(t.Context(), 1, capability.KanbanMoveTaskRequest{ColumnID: 3})
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
			SearchTasks: []*capability.Task{{ID: 1, Title: "Match"}},
		})
		result, err := svc.SearchTasks(t.Context(), &capability.KanbanSearchQuery{Q: "test", ProjectID: 1})
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
		_, err := svc.SearchTasks(t.Context(), &capability.KanbanSearchQuery{})
		RequireProviderError(t, err)
	})
}
