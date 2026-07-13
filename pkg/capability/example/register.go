package example

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the example capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapExample,
		App:         app,
		Description: "Example capability for demonstration",
		Instance:    svc,
		Ops: []capability.OpDef{
			{Name: OpList, Description: "List items", Scopes: []string{auth.ScopeServiceExampleRead}, Handler: invokeList(svc)},
			{Name: OpGet, Description: "Get an item", Scopes: []string{auth.ScopeServiceExampleRead}, Handler: invokeGet(svc)},
			{Name: OpCreate, Description: "Create an item", Scopes: []string{auth.ScopeServiceExampleWrite}, Mutation: true, Handler: invokeCreate(svc, app)},
			{Name: OpUpdate, Description: "Update an item", Scopes: []string{auth.ScopeServiceExampleWrite}, Mutation: true, Handler: invokeUpdate(svc)},
			{Name: OpDelete, Description: "Delete an item", Scopes: []string{auth.ScopeServiceExampleWrite}, Mutation: true, Handler: invokeDelete(svc)},
			{Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceExampleRead}, Handler: invokeHealth(svc)},
		},
	})
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.ListItems(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Host]{Items: []*capability.Host{}, Page: &capability.PageInfo{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGet(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeCreate(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		title, err := capability.RequiredString(params, "title")
		if err != nil {
			return nil, err
		}
		var tags types.KV
		if t, ok := params["tags"].(types.KV); ok {
			tags = t
		}
		item, err := svc.CreateItem(ctx, title, tags)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: item,
			Resource: &capability.ResourceMeta{
				EntityID: item.ID,
				App:      app,
			},
		}, nil
	}
}

func invokeUpdate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		data, ok := params["data"].(map[string]any)
		if !ok {
			data = nil
		}
		item, err := svc.UpdateItem(ctx, id, data)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeDelete(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteItem(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok, Text: "ok"}, nil
	}
}
