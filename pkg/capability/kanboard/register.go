// Package kanboard implements the Kanban board capability.
package kanboard

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the kanboard capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapKanboard,
		App:         app,
		Description: "Kanban capability",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: types.EventKanbanTaskCreated, Description: "Fires when a task is created"},
			{Name: types.EventKanbanTaskUpdated, Description: "Fires when a task is updated"},
			{Name: types.EventKanbanTaskCompleted, Description: "Fires when a task is completed"},
			{Name: types.EventKanbanTaskOpened, Description: "Fires when a task is opened"},
			{Name: types.EventKanbanTaskMoved, Description: "Fires when a task is moved"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpListTasks, Description: "List tasks", Scopes: []string{auth.ScopeServiceKanbanRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID filter"},
					{Name: "status", Type: "string", Required: false, Description: "Task status filter"},
				},
				Handler: invokeListTasks(svc),
			},
			{
				Name: OpGetTask, Description: "Get a task", Scopes: []string{auth.ScopeServiceKanbanRead},
				Input:   []hub.ParamDef{{Name: "id", Type: "int", Required: true, Description: "Task ID"}},
				Handler: invokeGetTask(svc),
			},
			{
				Name: OpCreateTask, Description: "Create a task", Scopes: []string{auth.ScopeServiceKanbanWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "title", Type: "string", Required: false, Description: "Task title"},
					{Name: "description", Type: "string", Required: false, Description: "Task description"},
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID"},
					{Name: "column_id", Type: "int", Required: false, Description: "Column ID"},
					{Name: "tags", Type: "[]string", Required: false, Description: "Tags to assign"},
					{Name: "reference", Type: "string", Required: false, Description: "Reference URL or text"},
				},
				Handler: invokeCreateTask(svc),
			},
			{
				Name: OpUpdateTask, Description: "Update a task", Scopes: []string{auth.ScopeServiceKanbanWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
					{Name: "title", Type: "string", Required: false, Description: "New title"},
					{Name: "description", Type: "string", Required: false, Description: "New description"},
				},
				Handler: invokeUpdateTask(svc),
			},
			{
				Name: OpDeleteTask, Description: "Delete a task", Scopes: []string{auth.ScopeServiceKanbanWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "int", Required: true, Description: "Task ID"}},
				Handler: invokeDeleteTask(svc),
			},
			{
				Name: OpMoveTask, Description: "Move a task", Scopes: []string{auth.ScopeServiceKanbanWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "int", Required: true, Description: "Task ID"},
					{Name: "column_id", Type: "int", Required: false, Description: "Target column ID"},
					{Name: "position", Type: "int", Required: false, Description: "Position in column"},
					{Name: "swimlane_id", Type: "int", Required: false, Description: "Target swimlane ID"},
					{Name: "project_id", Type: "int", Required: false, Description: "Target project ID"},
				},
				Handler: invokeMoveTask(svc),
			},
			{
				Name: OpCompleteTask, Description: "Complete a task", Scopes: []string{auth.ScopeServiceKanbanWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "int", Required: true, Description: "Task ID"}},
				Handler: invokeCompleteTask(svc),
			},
			{
				Name: OpGetColumns, Description: "Get columns", Scopes: []string{auth.ScopeServiceKanbanRead},
				Input:   []hub.ParamDef{{Name: "project_id", Type: "int", Required: false, Description: "Project ID (defaults to 1)"}},
				Handler: invokeGetColumns(svc),
			},
			{
				Name: OpSearchTasks, Description: "Search tasks", Scopes: []string{auth.ScopeServiceKanbanRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "q", Type: "string", Required: false, Description: "Search query"},
					{Name: "project_id", Type: "int", Required: false, Description: "Project ID filter"},
				},
				Handler: invokeSearchTasks(svc),
			},
		},
	})
}

func invokeListTasks(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &TaskQuery{Page: capability.PageRequestFromParams(params)}
		if v, ok := capability.IntParam(params, "project_id"); ok {
			q.ProjectID = v
		}
		if v, ok := capability.StringParam(params, "status"); ok {
			q.Status = v
		}
		result, err := svc.ListTasks(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Task]{Items: []*capability.Task{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetTask(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetTask(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: item.Title}, nil
	}
}

func invokeCreateTask(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		req := CreateTaskRequest{}
		req.Title, _ = capability.StringParam(params, "title")
		req.Description, _ = capability.StringParam(params, "description")
		if v, ok := capability.IntParam(params, "project_id"); ok {
			req.ProjectID = v
		}
		if v, ok := capability.IntParam(params, "column_id"); ok {
			req.ColumnID = v
		}
		if v, ok := stringListParam(params, "tags"); ok {
			req.Tags = v
		}
		req.Reference, _ = capability.StringParam(params, "reference")
		item, err := svc.CreateTask(ctx, req)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: item,
			Text: "task created: " + item.Title,
			Events: []capability.EventRef{{
				EventType: types.EventKanbanTaskCreated,
				EntityID:  fmt.Sprintf("%d", item.ID),
			}},
		}, nil
	}
}

func invokeUpdateTask(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		req := UpdateTaskRequest{}
		req.Title, _ = capability.StringParam(params, "title")
		req.Description, _ = capability.StringParam(params, "description")
		item, err := svc.UpdateTask(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: "task updated"}, nil
	}
}

func invokeDeleteTask(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteTask(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "task deleted"}, nil
	}
}

func invokeMoveTask(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		req := MoveTaskRequest{}
		if v, ok := capability.IntParam(params, "column_id"); ok {
			req.ColumnID = v
		}
		if v, ok := capability.IntParam(params, "position"); ok {
			req.Position = v
		}
		if v, ok := capability.IntParam(params, "swimlane_id"); ok {
			req.SwimlaneID = v
		}
		if v, ok := capability.IntParam(params, "project_id"); ok {
			req.ProjectID = v
		}
		item, err := svc.MoveTask(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: "task moved"}, nil
	}
}

func invokeCompleteTask(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.CompleteTask(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Text: "task completed",
			Events: []capability.EventRef{{
				EventType: types.EventKanbanTaskCompleted,
				EntityID:  fmt.Sprintf("%d", id),
			}},
		}, nil
	}
}

func invokeGetColumns(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		projectID := 1
		if v, ok := capability.IntParam(params, "project_id"); ok {
			projectID = v
		}
		columns, err := svc.GetColumns(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: columns}, nil
	}
}

func invokeSearchTasks(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &SearchQuery{Page: capability.PageRequestFromParams(params)}
		q.Q, _ = capability.StringParam(params, "q")
		if v, ok := capability.IntParam(params, "project_id"); ok {
			q.ProjectID = v
		}
		result, err := svc.SearchTasks(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Task]{Items: []*capability.Task{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
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
