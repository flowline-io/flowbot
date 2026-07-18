package trilium

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers the trilium capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapTrilium,
		App:         app,
		Description: "Note capability for note-taking systems",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpList, Description: "List notes", Scopes: []string{auth.ScopeServiceNoteRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "query", Type: "string", Required: false, Description: "Search query filter"},
				},
				Handler: invokeList(svc),
			},
			{
				Name: OpGet, Description: "Get a note", Scopes: []string{auth.ScopeServiceNoteRead},
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Note ID"}},
				Handler: invokeGet(svc),
			},
			{
				Name: OpCreate, Description: "Create a note", Scopes: []string{auth.ScopeServiceNoteWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "title", Type: "string", Required: true, Description: "Note title"},
					{Name: "content", Type: "string", Required: false, Description: "Note content"},
					{Name: "type", Type: "string", Required: false, Description: "Note type"},
					{Name: "parent_note_id", Type: "string", Required: false, Description: "Parent note ID"},
				},
				Handler: invokeCreate(svc, app),
			},
			{
				Name: OpUpdate, Description: "Update a note", Scopes: []string{auth.ScopeServiceNoteWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Note ID"},
					{Name: "title", Type: "string", Required: false, Description: "New title"},
					{Name: "content", Type: "string", Required: false, Description: "New content"},
				},
				Handler: invokeUpdate(svc),
			},
			{
				Name: OpDelete, Description: "Delete a note", Scopes: []string{auth.ScopeServiceNoteWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Note ID"}},
				Handler: invokeDelete(svc),
			},
			{
				Name: OpGetContent, Description: "Get note content", Scopes: []string{auth.ScopeServiceNoteRead},
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Note ID"}},
				Handler: invokeGetContent(svc),
			},
			{
				Name: OpSetContent, Description: "Set note content", Scopes: []string{auth.ScopeServiceNoteWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Note ID"},
					{Name: "content", Type: "string", Required: true, Description: "Note content"},
				},
				Handler: invokeSetContent(svc),
			},
			{
				Name: OpSearch, Description: "Search notes", Scopes: []string{auth.ScopeServiceNoteRead},
				Input:   []hub.ParamDef{{Name: "query", Type: "string", Required: true, Description: "Search query string"}},
				Handler: invokeSearch(svc),
			},
			{
				Name: OpGetAppInfo, Description: "Get note server info", Scopes: []string{auth.ScopeServiceNoteRead},
				Handler: invokeGetAppInfo(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceNoteRead},
				Handler: invokeHealth(svc),
			},
		},
	})
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		if query, ok := capability.StringParam(params, "query"); ok {
			q.Query = query
		}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Note]{Items: []*capability.Note{}, Page: &capability.PageInfo{}}
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
		item, err := svc.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: item.Title}, nil
	}
}

func invokeCreate(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		title, err := capability.RequiredString(params, "title")
		if err != nil {
			return nil, err
		}
		content, _ := capability.StringParam(params, "content")
		typ, _ := capability.StringParam(params, "type")
		parentNoteID, _ := capability.StringParam(params, "parent_note_id")
		item, err := svc.Create(ctx, title, content, typ, parentNoteID)
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
		title, _ := capability.StringParam(params, "title")
		content, _ := capability.StringParam(params, "content")
		item, err := svc.Update(ctx, id, title, content)
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
		if err := svc.Delete(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{}, nil
	}
}

func invokeGetContent(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		content, err := svc.GetContent(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: content}, nil
	}
}

func invokeSetContent(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		content, err := capability.RequiredString(params, "content")
		if err != nil {
			return nil, err
		}
		if err := svc.SetContent(ctx, id, content); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{}, nil
	}
}

func invokeSearch(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		query, err := capability.RequiredString(params, "query")
		if err != nil {
			return nil, err
		}
		result, err := svc.Search(ctx, query)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Note]{Items: []*capability.Note{}, Page: &capability.PageInfo{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetAppInfo(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		info, err := svc.GetAppInfo(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: info}, nil
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
