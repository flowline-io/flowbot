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

type Service interface {
	ListTasks(ctx context.Context, q *TaskQuery) (*ability.ListResult[ability.Task], error)
	GetTask(ctx context.Context, id int) (*ability.Task, error)
	CreateTask(ctx context.Context, req CreateTaskRequest) (*ability.Task, error)
	MoveTask(ctx context.Context, id int, columnID int) (*ability.Task, error)
	CompleteTask(ctx context.Context, id int) error
}
