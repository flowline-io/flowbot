package kanban

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
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
			{Name: ability.OpKanbanListTasks, Description: "List tasks", Scopes: []string{auth.ScopeServiceKanbanRead}},
			{Name: ability.OpKanbanGetTask, Description: "Get a task", Scopes: []string{auth.ScopeServiceKanbanRead}},
			{Name: ability.OpKanbanCreateTask, Description: "Create a task", Scopes: []string{auth.ScopeServiceKanbanWrite}},
			{Name: ability.OpKanbanUpdateTask, Description: "Update a task", Scopes: []string{auth.ScopeServiceKanbanWrite}},
			{Name: ability.OpKanbanDeleteTask, Description: "Delete a task", Scopes: []string{auth.ScopeServiceKanbanWrite}},
			{Name: ability.OpKanbanMoveTask, Description: "Move a task", Scopes: []string{auth.ScopeServiceKanbanWrite}},
			{Name: ability.OpKanbanCompleteTask, Description: "Complete a task", Scopes: []string{auth.ScopeServiceKanbanWrite}},
			{Name: ability.OpKanbanGetColumns, Description: "Get columns", Scopes: []string{auth.ScopeServiceKanbanRead}},
			{Name: ability.OpKanbanSearchTasks, Description: "Search tasks", Scopes: []string{auth.ScopeServiceKanbanRead}},
		},
	}
}

func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "kanban service is required")
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
		q := &TaskQuery{}
		if v, ok := intParam(params, "project_id"); ok {
			q.ProjectID = v
		}
		if v, ok := stringParam(params, "status"); ok {
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
		id, err := requiredInt(params, "id")
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
		req.Title, _ = stringParam(params, "title")
		req.Description, _ = stringParam(params, "description")
		if v, ok := intParam(params, "project_id"); ok {
			req.ProjectID = v
		}
		if v, ok := intParam(params, "column_id"); ok {
			req.ColumnID = v
		}
		if v, ok := stringListParam(params, "tags"); ok {
			req.Tags = v
		}
		req.Reference, _ = stringParam(params, "reference")
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
		id, err := requiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		req := UpdateTaskRequest{}
		req.Title, _ = stringParam(params, "title")
		req.Description, _ = stringParam(params, "description")
		item, err := svc.UpdateTask(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: "task updated"}, nil
	}
}

func invokeDeleteTask(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := requiredInt(params, "id")
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
		id, err := requiredInt(params, "id")
		if err != nil {
			return nil, err
		}
		req := MoveTaskRequest{}
		if v, ok := intParam(params, "column_id"); ok {
			req.ColumnID = v
		}
		if v, ok := intParam(params, "position"); ok {
			req.Position = v
		}
		if v, ok := intParam(params, "swimlane_id"); ok {
			req.SwimlaneID = v
		}
		if v, ok := intParam(params, "project_id"); ok {
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
		id, err := requiredInt(params, "id")
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
		if v, ok := intParam(params, "project_id"); ok {
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
		q := &SearchQuery{}
		q.Q, _ = stringParam(params, "q")
		if v, ok := intParam(params, "project_id"); ok {
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

func requiredInt(params map[string]any, key string) (int, error) {
	value, ok := intParam(params, key)
	if !ok {
		return 0, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func stringParam(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return "", false
	}
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func intParam(params map[string]any, key string) (int, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
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
