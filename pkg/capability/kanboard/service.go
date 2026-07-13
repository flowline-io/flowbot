package kanboard

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// TaskQuery wraps pagination and filters for listing tasks.
type TaskQuery = capability.KanbanTaskQuery

// CreateTaskRequest holds fields for creating a task.
type CreateTaskRequest = capability.KanbanCreateTaskRequest

// UpdateTaskRequest holds fields for updating a task.
type UpdateTaskRequest = capability.KanbanUpdateTaskRequest

// MoveTaskRequest holds fields for moving a task.
type MoveTaskRequest = capability.KanbanMoveTaskRequest

// SearchQuery wraps pagination for searching tasks.
type SearchQuery = capability.KanbanSearchQuery

// Service defines the kanban capability contract.
type Service interface {
	ListTasks(ctx context.Context, q *TaskQuery) (*capability.ListResult[capability.Task], error)
	GetTask(ctx context.Context, id int) (*capability.Task, error)
	CreateTask(ctx context.Context, req CreateTaskRequest) (*capability.Task, error)
	UpdateTask(ctx context.Context, id int, req UpdateTaskRequest) (*capability.Task, error)
	DeleteTask(ctx context.Context, id int) error
	MoveTask(ctx context.Context, id int, req MoveTaskRequest) (*capability.Task, error)
	CompleteTask(ctx context.Context, id int) error
	GetColumns(ctx context.Context, projectID int) ([]map[string]any, error)
	SearchTasks(ctx context.Context, q *SearchQuery) (*capability.ListResult[capability.Task], error)
}
