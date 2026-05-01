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
			{Name: "list", Description: "List bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead}},
			{Name: "get", Description: "Get a bookmark", Scopes: []string{auth.ScopeServiceBookmarkRead}},
			{Name: "create", Description: "Create a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: "delete", Description: "Delete a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: "archive", Description: "Archive a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: "search", Description: "Search bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead}},
			{Name: "attach_tags", Description: "Attach tags", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: "detach_tags", Description: "Detach tags", Scopes: []string{auth.ScopeServiceBookmarkWrite}},
			{Name: "check_url", Description: "Check whether a URL exists", Scopes: []string{auth.ScopeServiceBookmarkRead}},
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
		{operation: "list", invoker: invokeList(svc)},
		{operation: "get", invoker: invokeGet(svc)},
		{operation: "create", invoker: invokeCreate(svc)},
		{operation: "delete", invoker: invokeDelete(svc)},
		{operation: "archive", invoker: invokeArchive(svc)},
		{operation: "search", invoker: invokeSearch(svc)},
		{operation: "attach_tags", invoker: invokeAttachTags(svc)},
		{operation: "detach_tags", invoker: invokeDetachTags(svc)},
		{operation: "check_url", invoker: invokeCheckURL(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapBookmark, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeList(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &ListQuery{Page: pageRequestFromParams(params)}
		if value, ok := boolParam(params, "archived"); ok {
			q.Archived = &value
		}
		if value, ok := boolParam(params, "favourited"); ok {
			q.Favourited = &value
		}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult("list", result), nil
	}
}

func invokeGet(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := requiredString(params, "id")
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
		url, err := requiredString(params, "url")
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
		id, err := requiredString(params, "id")
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
		id, err := requiredString(params, "id")
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
		q := &SearchQuery{Page: pageRequestFromParams(params)}
		q.Q, _ = stringParam(params, "q")
		result, err := svc.Search(ctx, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult("search", result), nil
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
		url, err := requiredString(params, "url")
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

func listInvokeResult(operation string, result *ability.ListResult[ability.Bookmark]) *ability.InvokeResult {
	if result == nil {
		result = &ability.ListResult[ability.Bookmark]{Items: []*ability.Bookmark{}, Page: &ability.PageInfo{}}
	}
	return &ability.InvokeResult{Operation: operation, Data: result.Items, Page: result.Page}
}
