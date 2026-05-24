package note

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Capability is the note capability type constant.
const Capability hub.CapabilityType = hub.CapNote

// Descriptor returns the hub capability descriptor for the note capability.
func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapNote,
		Backend:     backend,
		App:         app,
		Description: "Note capability for note-taking systems",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: ability.OpNoteList, Description: "List notes", Scopes: []string{auth.ScopeServiceNoteRead}},
			{Name: ability.OpNoteGet, Description: "Get a note", Scopes: []string{auth.ScopeServiceNoteRead}},
			{Name: ability.OpNoteCreate, Description: "Create a note", Scopes: []string{auth.ScopeServiceNoteWrite}},
			{Name: ability.OpNoteUpdate, Description: "Update a note", Scopes: []string{auth.ScopeServiceNoteWrite}},
			{Name: ability.OpNoteDelete, Description: "Delete a note", Scopes: []string{auth.ScopeServiceNoteWrite}},
			{Name: ability.OpNoteGetContent, Description: "Get note content", Scopes: []string{auth.ScopeServiceNoteRead}},
			{Name: ability.OpNoteSetContent, Description: "Set note content", Scopes: []string{auth.ScopeServiceNoteWrite}},
			{Name: ability.OpNoteSearch, Description: "Search notes", Scopes: []string{auth.ScopeServiceNoteRead}},
			{Name: ability.OpNoteGetAppInfo, Description: "Get note server info", Scopes: []string{auth.ScopeServiceNoteRead}},
		},
	}
}

// RegisterService registers the note capability with the hub and ability registry.
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "note service is required")
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpNoteList, invoker: invokeList(svc)},
		{operation: ability.OpNoteGet, invoker: invokeGet(svc)},
		{operation: ability.OpNoteCreate, invoker: invokeCreate(svc, backend)},
		{operation: ability.OpNoteUpdate, invoker: invokeUpdate(svc)},
		{operation: ability.OpNoteDelete, invoker: invokeDelete(svc)},
		{operation: ability.OpNoteGetContent, invoker: invokeGetContent(svc)},
		{operation: ability.OpNoteSetContent, invoker: invokeSetContent(svc)},
		{operation: ability.OpNoteSearch, invoker: invokeSearch(svc)},
		{operation: ability.OpNoteGetAppInfo, invoker: invokeGetAppInfo(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapNote, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeList(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &ListQuery{Page: ability.PageRequestFromParams(params)}
		if query, ok := ability.StringParam(params, "query"); ok {
			q.Query = query
		}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Note]{Items: []*ability.Note{}, Page: &ability.PageInfo{}}
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
		item, err := svc.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: item.Title}, nil
	}
}

func invokeCreate(svc Service, backend string) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		title, err := ability.RequiredString(params, "title")
		if err != nil {
			return nil, err
		}
		content, _ := ability.StringParam(params, "content")
		typ, _ := ability.StringParam(params, "type")
		parentNoteID, _ := ability.StringParam(params, "parent_note_id")
		item, err := svc.Create(ctx, title, content, typ, parentNoteID)
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
		title, _ := ability.StringParam(params, "title")
		content, _ := ability.StringParam(params, "content")
		item, err := svc.Update(ctx, id, title, content)
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
		if err := svc.Delete(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{}, nil
	}
}

func invokeGetContent(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		content, err := svc.GetContent(ctx, id)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: content}, nil
	}
}

func invokeSetContent(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		content, err := ability.RequiredString(params, "content")
		if err != nil {
			return nil, err
		}
		if err := svc.SetContent(ctx, id, content); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{}, nil
	}
}

func invokeSearch(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		query, err := ability.RequiredString(params, "query")
		if err != nil {
			return nil, err
		}
		result, err := svc.Search(ctx, query)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Note]{Items: []*ability.Note{}, Page: &ability.PageInfo{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetAppInfo(svc Service) ability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*ability.InvokeResult, error) {
		info, err := svc.GetAppInfo(ctx)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: info}, nil
	}
}
