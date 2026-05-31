// Package kanban implements the Kanban board capability.
package kanban

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapKanban,
		Backend:     backend,
		App:         app,
		Description: "Kanban capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{
				Name:        ability.OpKanbanListTasks,
				Description: "List tasks",
				Scopes:      []string{auth.ScopeServiceKanbanRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID filter"},
					{Name: "status", Type: "string", Required: false, Description: "Task status filter"},
				},
			},
			{
				Name:        ability.OpKanbanGetTask,
				Description: "Get a task",
				Scopes:      []string{auth.ScopeServiceKanbanRead},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
				},
			},
			{
				Name:        ability.OpKanbanCreateTask,
				Description: "Create a task",
				Scopes:      []string{auth.ScopeServiceKanbanWrite},
				Input: []hub.ParamDef{
					{Name: "title", Type: "string", Required: false, Description: "Task title"},
					{Name: "description", Type: "string", Required: false, Description: "Task description"},
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID"},
					{Name: "column_id", Type: "int", Required: false, Description: "Column ID"},
					{Name: "tags", Type: "[]string", Required: false, Description: "Tags to assign"},
					{Name: "reference", Type: "string", Required: false, Description: "Reference URL or text"},
				},
			},
			{
				Name:        ability.OpKanbanUpdateTask,
				Description: "Update a task",
				Scopes:      []string{auth.ScopeServiceKanbanWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
					{Name: "title", Type: "string", Required: false, Description: "New title"},
					{Name: "description", Type: "string", Required: false, Description: "New description"},
				},
			},
			{
				Name:        ability.OpKanbanDeleteTask,
				Description: "Delete a task",
				Scopes:      []string{auth.ScopeServiceKanbanWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
				},
			},
			{
				Name:        ability.OpKanbanMoveTask,
				Description: "Move a task",
				Scopes:      []string{auth.ScopeServiceKanbanWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
					{Name: "column_id", Type: "int", Required: false, Description: "Target column ID"},
					{Name: "position", Type: "int", Required: false, Description: "Position in column"},
					{Name: "swimlane_id", Type: "int", Required: false, Description: "Target swimlane ID"},
					{Name: "project_id", Type: "int", Required: false, Description: "Target project ID"},
				},
			},
			{
				Name:        ability.OpKanbanCompleteTask,
				Description: "Complete a task",
				Scopes:      []string{auth.ScopeServiceKanbanWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
				},
			},
			{
				Name:        ability.OpKanbanGetColumns,
				Description: "Get columns",
				Scopes:      []string{auth.ScopeServiceKanbanRead},
				Input: []hub.ParamDef{
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID (defaults to 1)"},
				},
			},
			{
				Name:        ability.OpKanbanSearchTasks,
				Description: "Search tasks",
				Scopes:      []string{auth.ScopeServiceKanbanRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "q", Type: "string", Required: false, Description: "Search query"},
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID filter"},
				},
			},
		},
		Events: []hub.EventDef{
			{Name: types.EventKanbanTaskCreated, Description: "Fires when a task is created"},
			{Name: types.EventKanbanTaskUpdated, Description: "Fires when a task is updated"},
			{Name: types.EventKanbanTaskCompleted, Description: "Fires when a task is completed"},
			{Name: types.EventKanbanTaskOpened, Description: "Fires when a task is opened"},
			{Name: types.EventKanbanTaskMoved, Description: "Fires when a task is moved"},
		},
	}
}

// RegisterService registers the kanban capability with the hub and ability registry.
// It returns nil and logs a warning when svc is nil (provider not configured).
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		flog.Warn("kanban capability: service is nil, skipping registration for %s/%s", backend, app)
		return nil
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpKanbanListTasks, invoker: invokeListTasks(svc)},
		{operation: ability.OpKanbanGetTask, invoker: invokeGetTask(svc)},
		{operation: ability.OpKanbanCreateTask, invoker: invokeCreateTask(svc)},
		{operation: ability.OpKanbanUpdateTask, invoker: invokeUpdateTask(svc)},
		{operation: ability.OpKanbanDeleteTask, invoker: invokeDeleteTask(svc)},
		{operation: ability.OpKanbanMoveTask, invoker: invokeMoveTask(svc)},
		{operation: ability.OpKanbanCompleteTask, invoker: invokeCompleteTask(svc)},
		{operation: ability.OpKanbanGetColumns, invoker: invokeGetColumns(svc)},
		{operation: ability.OpKanbanSearchTasks, invoker: invokeSearchTasks(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapKanban, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeListTasks(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &TaskQuery{Page: ability.PageRequestFromParams(params)}
		if v, ok := ability.IntParam(params, "project_id"); ok {
			q.ProjectID = v
		}
		if v, ok := ability.StringParam(params, "status"); ok {
			q.Status = v
		}
		result, err := svc.ListTasks(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Task]{Items: []*ability.Task{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetTask(ctx, id)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: item.Title}, nil
	}
}

func invokeCreateTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		req := CreateTaskRequest{}
		req.Title, _ = ability.StringParam(params, "title")
		req.Description, _ = ability.StringParam(params, "description")
		if v, ok := ability.IntParam(params, "project_id"); ok {
			req.ProjectID = v
		}
		if v, ok := ability.IntParam(params, "column_id"); ok {
			req.ColumnID = v
		}
		if v, ok := stringListParam(params, "tags"); ok {
			req.Tags = v
		}
		req.Reference, _ = ability.StringParam(params, "reference")
		item, err := svc.CreateTask(ctx, req)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{
			Data: item,
			Text: "task created: " + item.Title,
			Events: []ability.EventRef{{
				EventType: types.EventKanbanTaskCreated,
				EntityID:  fmt.Sprintf("%d", item.ID),
			}},
		}, nil
	}
}

func invokeUpdateTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		req := UpdateTaskRequest{}
		req.Title, _ = ability.StringParam(params, "title")
		req.Description, _ = ability.StringParam(params, "description")
		item, err := svc.UpdateTask(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: "task updated"}, nil
	}
}

func invokeDeleteTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteTask(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "task deleted"}, nil
	}
}

func invokeMoveTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		req := MoveTaskRequest{}
		if v, ok := ability.IntParam(params, "column_id"); ok {
			req.ColumnID = v
		}
		if v, ok := ability.IntParam(params, "position"); ok {
			req.Position = v
		}
		if v, ok := ability.IntParam(params, "swimlane_id"); ok {
			req.SwimlaneID = v
		}
		if v, ok := ability.IntParam(params, "project_id"); ok {
			req.ProjectID = v
		}
		item, err := svc.MoveTask(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: "task moved"}, nil
	}
}

func invokeCompleteTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.CompleteTask(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{
			Text: "task completed",
			Events: []ability.EventRef{{
				EventType: types.EventKanbanTaskCompleted,
				EntityID:  fmt.Sprintf("%d", id),
			}},
		}, nil
	}
}

func invokeGetColumns(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		projectID := 1
		if v, ok := ability.IntParam(params, "project_id"); ok {
			projectID = v
		}
		columns, err := svc.GetColumns(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: columns}, nil
	}
}

func invokeSearchTasks(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &SearchQuery{Page: ability.PageRequestFromParams(params)}
		q.Q, _ = ability.StringParam(params, "q")
		if v, ok := ability.IntParam(params, "project_id"); ok {
			q.ProjectID = v
		}
		result, err := svc.SearchTasks(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Task]{Items: []*ability.Task{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func stringListParam(params map[string]any, key string) ([]string, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return nil, false
	}
	switch v := value.(type) {
	case []string:
		return v, true
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, true
	}
	return nil, false
}
