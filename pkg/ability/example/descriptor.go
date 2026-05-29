package example

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Descriptor returns the hub capability descriptor for the example capability.
func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapExample,
		Backend:     backend,
		App:         app,
		Description: "Example capability for demonstration",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: ability.OpExampleList, Description: "List items", Scopes: []string{auth.ScopeServiceExampleRead}},
			{Name: ability.OpExampleGet, Description: "Get an item", Scopes: []string{auth.ScopeServiceExampleRead}},
			{Name: ability.OpExampleCreate, Description: "Create an item", Scopes: []string{auth.ScopeServiceExampleWrite}},
			{Name: ability.OpExampleUpdate, Description: "Update an item", Scopes: []string{auth.ScopeServiceExampleWrite}},
			{Name: ability.OpExampleDelete, Description: "Delete an item", Scopes: []string{auth.ScopeServiceExampleWrite}},
			{Name: ability.OpExampleHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceExampleRead}},
		},
	}
}

// RegisterService registers the example capability with the hub and ability registry.
// It returns nil and logs a warning when svc is nil (provider not configured).
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		flog.Warn("example capability: service is nil, skipping registration for %s/%s", backend, app)
		return nil
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpExampleList, invoker: invokeList(svc)},
		{operation: ability.OpExampleGet, invoker: invokeGet(svc)},
		{operation: ability.OpExampleCreate, invoker: invokeCreate(svc, backend)},
		{operation: ability.OpExampleUpdate, invoker: invokeUpdate(svc)},
		{operation: ability.OpExampleDelete, invoker: invokeDelete(svc)},
		{operation: ability.OpExampleHealth, invoker: invokeHealth(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapExample, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeList(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &ListQuery{Page: ability.PageRequestFromParams(params)}
		result, err := svc.ListItems(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Host]{Items: []*ability.Host{}, Page: &ability.PageInfo{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGet(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item}, nil
	}
}

func invokeCreate(svc Service, backend string) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		title, err := ability.RequiredString(params, "title")
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
		return &ability.InvokeResult{
			Data: item,
			Resource: &ability.ResourceMeta{
				EntityID: item.ID,
				App:      backend,
			},
		}, nil
	}
}

func invokeUpdate(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredString(params, "id")
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
		return &ability.InvokeResult{Data: item}, nil
	}
}

func invokeDelete(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteItem(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{}, nil
	}
}

func invokeHealth(svc Service) ability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*ability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: ok, Text: "ok"}, nil
	}
}
