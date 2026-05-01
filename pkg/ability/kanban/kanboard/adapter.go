package kanboard

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/kanban"
	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
)

type client interface {
	GetAllTasks(ctx context.Context, projectID int, status provider.StatusId) ([]*provider.Task, error)
	GetTask(ctx context.Context, taskID int) (*provider.Task, error)
	CreateTask(ctx context.Context, task *provider.Task) (int64, error)
	UpdateTask(ctx context.Context, taskID int, task *provider.Task) (bool, error)
	CloseTask(ctx context.Context, taskID int) (bool, error)
	RemoveTask(ctx context.Context, taskID int) (bool, error)
	MoveTaskPosition(ctx context.Context, projectID, taskID, columnID, position, swimlaneID int) (bool, error)
	GetColumns(ctx context.Context, projectID int) ([]types.KV, error)
	SearchTasks(ctx context.Context, projectID int, query string) ([]*provider.Task, error)
}

type Adapter struct {
	client client
}

func New() kanban.Service {
	client, err := provider.GetClient()
	if err != nil {
		return &Adapter{client: nil}
	}
	return NewWithClient(client)
}

func NewWithClient(client client) kanban.Service {
	return &Adapter{client: client}
}

func (a *Adapter) checkClient() error {
	if a.client == nil {
		return types.Errorf(types.ErrUnavailable, "kanboard client not available")
	}
	return nil
}

func (a *Adapter) ListTasks(ctx context.Context, q *kanban.TaskQuery) (*ability.ListResult[ability.Task], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban list tasks canceled", err)
	}
	projectID := q.ProjectID
	if projectID == 0 {
		projectID = 1
	}
	statusID := provider.Active
	if q.Status == "inactive" {
		statusID = provider.Inactive
	}
	tasks, err := a.client.GetAllTasks(ctx, projectID, statusID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard list tasks", err)
	}
	items := make([]*ability.Task, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, toAbilityTask(t))
	}
	return &ability.ListResult[ability.Task]{Items: items, Page: &ability.PageInfo{Limit: len(items)}}, nil
}

func (a *Adapter) GetTask(ctx context.Context, id int) (*ability.Task, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban get task canceled", err)
	}
	task, err := a.client.GetTask(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard get task", err)
	}
	result := toAbilityTask(task)
	return result, nil
}

func (a *Adapter) CreateTask(ctx context.Context, req kanban.CreateTaskRequest) (*ability.Task, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban create task canceled", err)
	}
	if req.Title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	projectID := req.ProjectID
	if projectID == 0 {
		projectID = 1
	}
	taskID, err := a.client.CreateTask(ctx, &provider.Task{
		Title:       req.Title,
		Description: req.Description,
		ProjectID:   projectID,
		ColumnID:    req.ColumnID,
		Tags:        tagsToAny(req.Tags),
		Reference:   req.Reference,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard create task", err)
	}
	return &ability.Task{
		ID:          int(taskID),
		Title:       req.Title,
		Description: req.Description,
		ProjectID:   projectID,
		ColumnID:    req.ColumnID,
		Tags:        req.Tags,
		Reference:   req.Reference,
	}, nil
}

func (a *Adapter) UpdateTask(ctx context.Context, id int, req kanban.UpdateTaskRequest) (*ability.Task, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban update task canceled", err)
	}
	_, err := a.client.UpdateTask(ctx, id, &provider.Task{
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard update task", err)
	}
	task, err := a.client.GetTask(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard get task after update", err)
	}
	return toAbilityTask(task), nil
}

func (a *Adapter) DeleteTask(ctx context.Context, id int) error {
	if err := a.checkClient(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "kanban delete task canceled", err)
	}
	_, err := a.client.RemoveTask(ctx, id)
	if err != nil {
		return types.WrapError(types.ErrProvider, "kanboard delete task", err)
	}
	return nil
}

func (a *Adapter) MoveTask(ctx context.Context, id int, req kanban.MoveTaskRequest) (*ability.Task, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban move task canceled", err)
	}
	projectID := req.ProjectID
	if projectID == 0 {
		projectID = 1
	}
	_, err := a.client.MoveTaskPosition(ctx, projectID, id, req.ColumnID, req.Position, req.SwimlaneID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard move task", err)
	}
	task, err := a.client.GetTask(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard get task after move", err)
	}
	return toAbilityTask(task), nil
}

func (a *Adapter) CompleteTask(ctx context.Context, id int) error {
	if err := a.checkClient(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "kanban complete task canceled", err)
	}
	_, err := a.client.CloseTask(ctx, id)
	if err != nil {
		return types.WrapError(types.ErrProvider, "kanboard close task", err)
	}
	return nil
}

func (a *Adapter) GetColumns(ctx context.Context, projectID int) ([]map[string]any, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban get columns canceled", err)
	}
	columns, err := a.client.GetColumns(ctx, projectID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard get columns", err)
	}
	result := make([]map[string]any, 0, len(columns))
	for _, col := range columns {
		resultMap := make(map[string]any)
		for k, v := range col {
			resultMap[k] = v
		}
		result = append(result, resultMap)
	}
	return result, nil
}

func (a *Adapter) SearchTasks(ctx context.Context, q *kanban.SearchQuery) (*ability.ListResult[ability.Task], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "kanban search tasks canceled", err)
	}
	projectID := q.ProjectID
	if projectID == 0 {
		projectID = 1
	}
	tasks, err := a.client.SearchTasks(ctx, projectID, q.Q)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "kanboard search tasks", err)
	}
	items := make([]*ability.Task, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, toAbilityTask(t))
	}
	return &ability.ListResult[ability.Task]{Items: items, Page: &ability.PageInfo{Limit: len(items)}}, nil
}

func toAbilityTask(t *provider.Task) *ability.Task {
	if t == nil {
		return nil
	}
	return &ability.Task{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		ProjectID:   t.ProjectID,
		ColumnID:    t.ColumnID,
		Tags:        anyToStringSlice(t.Tags),
		Reference:   t.Reference,
	}
}

func tagsToAny(tags []string) []any {
	result := make([]any, len(tags))
	for i, tag := range tags {
		result[i] = tag
	}
	return result
}

func anyToStringSlice(items []any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

var _ = fmt.Sprintf
