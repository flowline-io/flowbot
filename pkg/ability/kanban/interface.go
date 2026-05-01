package kanban

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type TaskQuery struct {
	Page      ability.PageRequest
	ProjectID int
	ColumnID  int
	Status    string
}

type CreateTaskRequest struct {
	Title       string
	Description string
	ProjectID   int
	ColumnID    int
	Tags        []string
	Reference   string
}

type UpdateTaskRequest struct {
	Title       string
	Description string
}

type MoveTaskRequest struct {
	ColumnID   int
	Position   int
	SwimlaneID int
	ProjectID  int
}

type SearchQuery struct {
	Page      ability.PageRequest
	ProjectID int
	Q         string
}

type Service interface {
	ListTasks(ctx context.Context, q *TaskQuery) (*ability.ListResult[ability.Task], error)
	GetTask(ctx context.Context, id int) (*ability.Task, error)
	CreateTask(ctx context.Context, req CreateTaskRequest) (*ability.Task, error)
	UpdateTask(ctx context.Context, id int, req UpdateTaskRequest) (*ability.Task, error)
	DeleteTask(ctx context.Context, id int) error
	MoveTask(ctx context.Context, id int, req MoveTaskRequest) (*ability.Task, error)
	CompleteTask(ctx context.Context, id int) error
	GetColumns(ctx context.Context, projectID int) ([]map[string]any, error)
	SearchTasks(ctx context.Context, q *SearchQuery) (*ability.ListResult[ability.Task], error)
}
