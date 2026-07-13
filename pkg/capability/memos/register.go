package memos

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers the memos capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapMemos,
		App:         app,
		Description: "Memo capability for short-form note-taking",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpList, Description: "List memos", Scopes: []string{auth.ScopeServiceMemoRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
				},
				Handler: invokeList(svc),
			},
			{
				Name: OpGet, Description: "Get a memo", Scopes: []string{auth.ScopeServiceMemoRead},
				Input:   []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Memo name"}},
				Handler: invokeGet(svc),
			},
			{
				Name: OpCreate, Description: "Create a memo", Scopes: []string{auth.ScopeServiceMemoWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "content", Type: "string", Required: true, Description: "Memo content"},
					{Name: "visibility", Type: "string", Required: false, Description: "Visibility setting"},
				},
				Handler: invokeCreate(svc, app),
			},
			{
				Name: OpUpdate, Description: "Update a memo", Scopes: []string{auth.ScopeServiceMemoWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Memo name"}},
				Handler: invokeUpdate(svc),
			},
			{
				Name: OpDelete, Description: "Delete a memo", Scopes: []string{auth.ScopeServiceMemoWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Memo name"}},
				Handler: invokeDelete(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceMemoRead},
				Handler: invokeHealth(svc),
			},
		},
	})
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Memo]{Items: []*capability.Memo{}, Page: &capability.PageInfo{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGet(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		name, err := capability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		item, err := svc.Get(ctx, name)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeCreate(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		content, err := capability.RequiredString(params, "content")
		if err != nil {
			return nil, err
		}
		visibility, _ := capability.StringParam(params, "visibility")
		item, err := svc.Create(ctx, content, visibility)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: item,
			Resource: &capability.ResourceMeta{
				EntityID: item.Name,
				App:      app,
			},
		}, nil
	}
}

func invokeUpdate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		name, err := capability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		item, err := svc.Update(ctx, name, params)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeDelete(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		name, err := capability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		if err := svc.Delete(ctx, name); err != nil {
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
		return &capability.InvokeResult{Data: ok}, nil
	}
}
