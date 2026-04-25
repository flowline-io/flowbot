package client

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/validate"
)

// KanbanClient provides access to the kanban API.
type KanbanClient struct {
	c *Client
}

// List returns all kanban tasks for the given project and status.
// Use kanboard.Active (1) for active tasks or kanboard.Inactive (0) for closed tasks.
func (k *KanbanClient) List(ctx context.Context, projectID int, status kanboard.StatusId) ([]kanboard.Task, error) {
	if projectID <= 0 {
		return nil, fmt.Errorf("project_id must be positive, got %d", projectID)
	}

	var result []kanboard.Task
	path := fmt.Sprintf("/service/kanban?project_id=%d&status_id=%d", projectID, status)
	err := k.c.Get(ctx, path, &result)
	return result, err
}

// ListAll returns all kanban tasks for the given project regardless of status.
func (k *KanbanClient) ListAll(ctx context.Context, projectID int) ([]kanboard.Task, error) {
	if projectID <= 0 {
		return nil, fmt.Errorf("project_id must be positive, got %d", projectID)
	}

	var result []kanboard.Task
	path := fmt.Sprintf("/service/kanban?project_id=%d", projectID)
	err := k.c.Get(ctx, path, &result)
	return result, err
}

// Get returns a single kanban task by ID.
func (k *KanbanClient) Get(ctx context.Context, id int) (*kanboard.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be positive, got %d", id)
	}

	var result kanboard.Task
	path := fmt.Sprintf("/service/kanban/%d", id)
	err := k.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRequest contains the data needed to create a new kanban task.
type KanbanCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ProjectID   int    `json:"project_id,omitempty"`
	ColumnID    int    `json:"column_id,omitempty"`
}

// CreateResult contains the result of creating a kanban task.
type KanbanCreateResult struct {
	ID int64 `json:"id"`
}

// Create creates a new kanban task.
func (k *KanbanClient) Create(ctx context.Context, req KanbanCreateRequest) (*KanbanCreateResult, error) {
	if err := validateCreateRequest(&req); err != nil {
		return nil, err
	}

	var result KanbanCreateResult
	err := k.c.Post(ctx, "/service/kanban", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func validateCreateRequest(req *KanbanCreateRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) > validate.TitleMaxLen {
		return fmt.Errorf("title exceeds maximum length of %d", validate.TitleMaxLen)
	}
	if len(req.Description) > validate.DescMaxLen {
		return fmt.Errorf("description exceeds maximum length of %d", validate.DescMaxLen)
	}
	return nil
}

// UpdateRequest contains the data for updating a kanban task.
type KanbanUpdateRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// UpdateResult contains the result of updating a kanban task.
type KanbanUpdateResult struct {
	Success bool `json:"success"`
}

// Update updates an existing kanban task.
func (k *KanbanClient) Update(ctx context.Context, id int, req KanbanUpdateRequest) (*KanbanUpdateResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be positive, got %d", id)
	}
	if err := validateUpdateRequest(&req); err != nil {
		return nil, err
	}

	var result KanbanUpdateResult
	path := fmt.Sprintf("/service/kanban/%d", id)
	err := k.c.Patch(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func validateUpdateRequest(req *KanbanUpdateRequest) error {
	if req.Title != "" && len(req.Title) > validate.TitleMaxLen {
		return fmt.Errorf("title exceeds maximum length of %d", validate.TitleMaxLen)
	}
	if len(req.Description) > validate.DescMaxLen {
		return fmt.Errorf("description exceeds maximum length of %d", validate.DescMaxLen)
	}
	return nil
}

// Close closes (deletes) a kanban task.
func (k *KanbanClient) Close(ctx context.Context, id int) (*KanbanUpdateResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be positive, got %d", id)
	}

	var result KanbanUpdateResult
	path := fmt.Sprintf("/service/kanban/%d", id)
	err := k.c.Delete(ctx, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// MoveRequest contains the parameters for moving a kanban task.
type KanbanMoveRequest struct {
	ColumnID   int `json:"column_id"`
	Position   int `json:"position,omitempty"`
	SwimlaneID int `json:"swimlane_id,omitempty"`
	ProjectID  int `json:"project_id,omitempty"`
}

// MoveResult contains the result of moving a kanban task.
type KanbanMoveResult struct {
	Success bool `json:"success"`
}

// Move moves a kanban task to a different column and/or position.
func (k *KanbanClient) Move(ctx context.Context, id int, req KanbanMoveRequest) (*KanbanMoveResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be positive, got %d", id)
	}
	if err := validateMoveRequest(&req); err != nil {
		return nil, err
	}

	var result KanbanMoveResult
	path := fmt.Sprintf("/service/kanban/%d/move", id)
	err := k.c.Post(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func validateMoveRequest(req *KanbanMoveRequest) error {
	if req.ColumnID <= 0 {
		return fmt.Errorf("column_id must be positive, got %d", req.ColumnID)
	}
	if req.Position < 0 {
		return fmt.Errorf("position must be non-negative, got %d", req.Position)
	}
	if req.SwimlaneID < 0 {
		return fmt.Errorf("swimlane_id must be non-negative, got %d", req.SwimlaneID)
	}
	if req.ProjectID < 0 {
		return fmt.Errorf("project_id must be non-negative, got %d", req.ProjectID)
	}
	return nil
}

// Column represents a kanban column.
type KanbanColumn struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// ListColumns returns all columns for the given project.
func (k *KanbanClient) ListColumns(ctx context.Context, projectID int) ([]KanbanColumn, error) {
	if projectID <= 0 {
		return nil, fmt.Errorf("project_id must be positive, got %d", projectID)
	}

	var result []KanbanColumn
	path := fmt.Sprintf("/service/kanban/columns?project_id=%d", projectID)
	err := k.c.Get(ctx, path, &result)
	return result, err
}
