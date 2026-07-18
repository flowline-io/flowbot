// Package karakeep implements the bookmark management capability.
package karakeep

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the karakeep capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapKarakeep,
		App:         app,
		Description: "Bookmark capability",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: types.EventBookmarkCreated, Description: "Fires when a bookmark is created"},
			{Name: types.EventBookmarkUpdated, Description: "Fires when a bookmark is updated"},
			{Name: types.EventBookmarkArchived, Description: "Fires when a bookmark is archived"},
			{Name: types.EventBookmarkDeleted, Description: "Fires when a bookmark is deleted"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpList, Description: "List bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "archived", Type: "bool", Required: false, Description: "Filter by archive status"},
					{Name: "favourited", Type: "bool", Required: false, Description: "Filter by favourite status"},
				},
				Handler: invokeList(svc),
			},
			{
				Name: OpGet, Description: "Get a bookmark", Scopes: []string{auth.ScopeServiceBookmarkRead},
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Bookmark ID"}},
				Handler: invokeGet(svc),
			},
			{
				Name: OpCreate, Description: "Create a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "url", Type: "string", Required: true, Description: "URL to bookmark"}},
				Handler: invokeCreate(svc),
			},
			{
				Name: OpDelete, Description: "Delete a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Bookmark ID"}},
				Handler: invokeDelete(svc),
			},
			{
				Name: OpArchive, Description: "Archive a bookmark", Scopes: []string{auth.ScopeServiceBookmarkWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Bookmark ID"}},
				Handler: invokeArchive(svc),
			},
			{
				Name: OpSearch, Description: "Search bookmarks", Scopes: []string{auth.ScopeServiceBookmarkRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "q", Type: "string", Required: false, Description: "Search query"},
				},
				Handler: invokeSearch(svc),
			},
			{
				Name: OpAttachTags, Description: "Attach tags", Scopes: []string{auth.ScopeServiceBookmarkWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Bookmark ID"},
					{Name: "tags", Type: "[]string", Required: true, Description: "Tags to attach"},
				},
				Handler: invokeAttachTags(svc),
			},
			{
				Name: OpDetachTags, Description: "Detach tags", Scopes: []string{auth.ScopeServiceBookmarkWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Bookmark ID"},
					{Name: "tags", Type: "[]string", Required: true, Description: "Tags to detach"},
				},
				Handler: invokeDetachTags(svc),
			},
			{
				Name: OpCheckURL, Description: "Check whether a URL exists", Scopes: []string{auth.ScopeServiceBookmarkRead},
				Input:   []hub.ParamDef{{Name: "url", Type: "string", Required: true, Description: "URL to check"}},
				Handler: invokeCheckURL(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceBookmarkRead},
				Handler: invokeHealth(svc),
			},
		},
	})
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		if value, ok := capability.BoolParam(params, "archived"); ok {
			q.Archived = &value
		}
		if value, ok := capability.BoolParam(params, "favourited"); ok {
			q.Favourited = &value
		}
		result, err := svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult(OpList, result), nil
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

func invokeCreate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		url, err := capability.RequiredString(params, "url")
		if err != nil {
			return nil, err
		}
		item, err := svc.Create(ctx, url)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: item,
			Text: fmt.Sprintf("bookmark created: %s", item.URL),
			Events: []capability.EventRef{{
				EventType: types.EventBookmarkCreated,
				EntityID:  item.ID,
			}},
		}, nil
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
		return &capability.InvokeResult{Text: "bookmark deleted"}, nil
	}
}

func invokeArchive(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		archived, err := svc.Archive(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: map[string]bool{"archived": archived},
			Text: "bookmark archived",
			Events: []capability.EventRef{{
				EventType: types.EventBookmarkArchived,
				EntityID:  id,
			}},
		}, nil
	}
}

func invokeSearch(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &SearchQuery{Page: capability.PageRequestFromParams(params)}
		q.Q, _ = capability.StringParam(params, "q")
		result, err := svc.Search(ctx, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult(OpSearch, result), nil
	}
}

func invokeAttachTags(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, tags, err := idAndTags(params)
		if err != nil {
			return nil, err
		}
		if err := svc.AttachTags(ctx, id, tags); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "tags attached"}, nil
	}
}

func invokeDetachTags(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, tags, err := idAndTags(params)
		if err != nil {
			return nil, err
		}
		if err := svc.DetachTags(ctx, id, tags); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "tags detached"}, nil
	}
}

func invokeCheckURL(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		url, err := capability.RequiredString(params, "url")
		if err != nil {
			return nil, err
		}
		exists, id, err := svc.CheckURL(ctx, url)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]any{"exists": exists, "id": id}}, nil
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
