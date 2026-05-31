package memo

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Descriptor returns the hub capability descriptor for the memo capability.
func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapMemo,
		Backend:     backend,
		App:         app,
		Description: "Memo capability for short-form note-taking",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{
				Name:        ability.OpMemoList,
				Description: "List memos",
				Scopes:      []string{auth.ScopeServiceMemoRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
				},
			},
			{
				Name:        ability.OpMemoGet,
				Description: "Get a memo",
				Scopes:      []string{auth.ScopeServiceMemoRead},
				Input: []hub.ParamDef{
					{Name: "name", Type: "string", Required: true, Description: "Memo name"},
				},
			},
			{
				Name:        ability.OpMemoCreate,
				Description: "Create a memo",
				Scopes:      []string{auth.ScopeServiceMemoWrite},
				Input: []hub.ParamDef{
					{Name: "content", Type: "string", Required: true, Description: "Memo content"},
					{Name: "visibility", Type: "string", Required: false, Description: "Visibility setting"},
				},
			},
			{
				Name:        ability.OpMemoUpdate,
				Description: "Update a memo",
				Scopes:      []string{auth.ScopeServiceMemoWrite},
				Input: []hub.ParamDef{
					{Name: "name", Type: "string", Required: true, Description: "Memo name"},
				},
			},
			{
				Name:        ability.OpMemoDelete,
				Description: "Delete a memo",
				Scopes:      []string{auth.ScopeServiceMemoWrite},
				Input: []hub.ParamDef{
					{Name: "name", Type: "string", Required: true, Description: "Memo name"},
				},
			},
			{
				Name:        ability.OpMemoHealth,
				Description: "Health check",
				Scopes:      []string{auth.ScopeServiceMemoRead},
			},
		},
	}
}

// RegisterService registers the memo capability with the hub and ability registry.
// It returns nil and logs a warning when svc is nil (provider not configured).
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		flog.Warn("memo capability: service is nil, skipping registration for %s/%s", backend, app)
		return nil
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpMemoList, invoker: invokeList(svc)},
		{operation: ability.OpMemoGet, invoker: invokeGet(svc)},
		{operation: ability.OpMemoCreate, invoker: invokeCreate(svc, backend)},
		{operation: ability.OpMemoUpdate, invoker: invokeUpdate(svc)},
		{operation: ability.OpMemoDelete, invoker: invokeDelete(svc)},
		{operation: ability.OpMemoHealth, invoker: invokeHealth(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapMemo, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeList(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &ListQuery{Page: ability.PageRequestFromParams(params)}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Memo]{Items: []*ability.Memo{}, Page: &ability.PageInfo{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGet(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		name, err := ability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		item, err := svc.Get(ctx, name)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item}, nil
	}
}

func invokeCreate(svc Service, backend string) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		content, err := ability.RequiredString(params, "content")
		if err != nil {
			return nil, err
		}
		visibility, _ := ability.StringParam(params, "visibility")
		item, err := svc.Create(ctx, content, visibility)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{
			Data: item,
			Resource: &ability.ResourceMeta{
				EntityID: item.Name,
				App:      backend,
			},
		}, nil
	}
}

func invokeUpdate(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		name, err := ability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		item, err := svc.Update(ctx, name, params)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item}, nil
	}
}

func invokeDelete(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		name, err := ability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		if err := svc.Delete(ctx, name); err != nil {
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
		return &ability.InvokeResult{Data: ok}, nil
	}
}
