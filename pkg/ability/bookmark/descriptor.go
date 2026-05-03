package bookmark

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
		Type:        hub.CapBookmark,
		Backend:     backend,
		App:         app,
		Description: "Bookmark capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: ability.OpBookmarkList, Description: "List bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead}},
			{Name: ability.OpBookmarkGet, Description: "Get a bookmark", Scopes: []string{auth.ScopeServiceBookmarkRead}},
			{Name: ability.OpBookmarkCreate, Description: "Create a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: ability.OpBookmarkDelete, Description: "Delete a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: ability.OpBookmarkArchive, Description: "Archive a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: ability.OpBookmarkSearch, Description: "Search bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead}},
			{Name: ability.OpBookmarkAttachTags, Description: "Attach tags", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: ability.OpBookmarkDetachTags, Description: "Detach tags", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: ability.OpBookmarkCheckURL, Description: "Check whether a URL exists", Scopes: []string{auth.ScopeServiceBookmarkRead}},
		},
	}
}

func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "bookmark service is required")
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpBookmarkList, invoker: invokeList(svc)},
		{operation: ability.OpBookmarkGet, invoker: invokeGet(svc)},
		{operation: ability.OpBookmarkCreate, invoker: invokeCreate(svc)},
		{operation: ability.OpBookmarkDelete, invoker: invokeDelete(svc)},
		{operation: ability.OpBookmarkArchive, invoker: invokeArchive(svc)},
		{operation: ability.OpBookmarkSearch, invoker: invokeSearch(svc)},
		{operation: ability.OpBookmarkAttachTags, invoker: invokeAttachTags(svc)},
		{operation: ability.OpBookmarkDetachTags, invoker: invokeDetachTags(svc)},
		{operation: ability.OpBookmarkCheckURL, invoker: invokeCheckURL(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapBookmark, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeList(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &ListQuery{Page: ability.PageRequestFromParams(params)}
		if value, ok := ability.BoolParam(params, "archived"); ok {
			q.Archived = &value
		}
		if value, ok := ability.BoolParam(params, "favourited"); ok {
			q.Favourited = &value
		}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult(ability.OpBookmarkList, result), nil
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

func invokeCreate(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		url, err := ability.RequiredString(params, "url")
		if err != nil {
			return nil, err
		}
		item, err := svc.Create(ctx, url)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{
			Data: item,
			Text: fmt.Sprintf("bookmark created: %s", item.URL),
			Events: []ability.EventRef{{
				EventType: types.EventBookmarkCreated,
				EntityID:  item.ID,
			}},
		}, nil
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
		return &ability.InvokeResult{Text: "bookmark deleted"}, nil
	}
}

func invokeArchive(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		archived, err := svc.Archive(ctx, id)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{
			Data: map[string]bool{"archived": archived},
			Text: "bookmark archived",
			Events: []ability.EventRef{{
				EventType: types.EventBookmarkArchived,
				EntityID:  id,
			}},
		}, nil
	}
}

func invokeSearch(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &SearchQuery{Page: ability.PageRequestFromParams(params)}
		q.Q, _ = ability.StringParam(params, "q")
		result, err := svc.Search(ctx, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult(ability.OpBookmarkSearch, result), nil
	}
}

func invokeAttachTags(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, tags, err := idAndTags(params)
		if err != nil {
			return nil, err
		}
		if err := svc.AttachTags(ctx, id, tags); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "tags attached"}, nil
	}
}

func invokeDetachTags(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, tags, err := idAndTags(params)
		if err != nil {
			return nil, err
		}
		if err := svc.DetachTags(ctx, id, tags); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "tags detached"}, nil
	}
}

func invokeCheckURL(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		url, err := ability.RequiredString(params, "url")
		if err != nil {
			return nil, err
		}
		exists, id, err := svc.CheckURL(ctx, url)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: map[string]any{"exists": exists, "id": id}}, nil
	}
}
