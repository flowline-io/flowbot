// Package confluence implements the Confluence Cloud capability.
package confluence

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the confluence capability with hub and invoker registry.
func Register(app string, svc Service) error {
	return capability.Register(buildSpec(app, svc))
}

// CatalogSpec returns capability metadata for documentation.
func CatalogSpec() capability.Spec {
	return buildSpec("", nil)
}

func buildSpec(app string, svc Service) capability.Spec {
	return capability.Spec{
		Type:        hub.CapConfluence,
		App:         app,
		Description: "Confluence Cloud spaces and pages",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: types.EventConfluencePageCreated, Description: "Fires when a page is created"},
			{Name: types.EventConfluencePageUpdated, Description: "Fires when a page is updated"},
			{Name: types.EventConfluencePageDeleted, Description: "Fires when a page is deleted"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpListSpaces, Description: "List spaces", Scopes: []string{auth.ScopeServiceConfluenceRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
				},
				Handler: invokeListSpaces(svc),
			},
			{
				Name: OpListPages, Description: "List pages in a space", Scopes: []string{auth.ScopeServiceConfluenceRead},
				Input: []hub.ParamDef{
					{Name: "space_key", Type: "string", Required: false, Description: "Space key"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
				},
				Handler: invokeListPages(svc),
			},
			{
				Name: OpGetPage, Description: "Get a page", Scopes: []string{auth.ScopeServiceConfluenceRead},
				Input:   []hub.ParamDef{{Name: "page_id", Type: "string", Required: true, Description: "Page ID"}},
				Handler: invokeGetPage(svc),
			},
			{
				Name: OpGetPageContent, Description: "Get page storage content", Scopes: []string{auth.ScopeServiceConfluenceRead},
				Input:   []hub.ParamDef{{Name: "page_id", Type: "string", Required: true, Description: "Page ID"}},
				Handler: invokeGetPageContent(svc),
			},
			{
				Name: OpSearchPages, Description: "Search pages with CQL", Scopes: []string{auth.ScopeServiceConfluenceRead},
				Input: []hub.ParamDef{
					{Name: "cql", Type: "string", Required: true, Description: "CQL query"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
				},
				Handler: invokeSearchPages(svc),
			},
			{
				Name: OpCreatePage, Description: "Create a page", Scopes: []string{auth.ScopeServiceConfluenceWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "space_key", Type: "string", Required: false, Description: "Space key"},
					{Name: "title", Type: "string", Required: true, Description: "Page title"},
					{Name: "content", Type: "string", Required: false, Description: "Storage-format XHTML content"},
				},
				Handler: invokeCreatePage(svc, app),
			},
			{
				Name: OpUpdatePage, Description: "Update a page", Scopes: []string{auth.ScopeServiceConfluenceWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "page_id", Type: "string", Required: true, Description: "Page ID"},
					{Name: "title", Type: "string", Required: false, Description: "New title"},
					{Name: "content", Type: "string", Required: false, Description: "Storage-format XHTML content"},
				},
				Handler: invokeUpdatePage(svc, app),
			},
			{
				Name: OpDeletePage, Description: "Delete a page", Scopes: []string{auth.ScopeServiceConfluenceWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "page_id", Type: "string", Required: true, Description: "Page ID"}},
				Handler: invokeDeletePage(svc, app),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceConfluenceRead},
				Handler: invokeHealth(svc),
			},
		},
	}
}

func invokeListSpaces(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.ListSpaces(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.ConfluenceSpace]{Items: []*capability.ConfluenceSpace{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeListPages(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		spaceKey, _ := capability.StringParam(params, "space_key")
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.ListPages(ctx, spaceKey, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.ConfluencePage]{Items: []*capability.ConfluencePage{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetPage(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		pageID, err := capability.RequiredString(params, "page_id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetPage(ctx, pageID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: item.Title}, nil
	}
}

func invokeGetPageContent(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		pageID, err := capability.RequiredString(params, "page_id")
		if err != nil {
			return nil, err
		}
		content, err := svc.GetPageContent(ctx, pageID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: content, Text: content}, nil
	}
}

func invokeSearchPages(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		cql, err := capability.RequiredString(params, "cql")
		if err != nil {
			return nil, err
		}
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.SearchPages(ctx, cql, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.ConfluencePage]{Items: []*capability.ConfluencePage{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeCreatePage(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		spaceKey, _ := capability.StringParam(params, "space_key")
		title, err := capability.RequiredString(params, "title")
		if err != nil {
			return nil, err
		}
		content, _ := capability.StringParam(params, "content")
		item, err := svc.CreatePage(ctx, spaceKey, title, content)
		if err != nil {
			return nil, err
		}
		return mutationResult(item, types.EventConfluencePageCreated, app, "page created: "+item.Title), nil
	}
}

func invokeUpdatePage(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		pageID, err := capability.RequiredString(params, "page_id")
		if err != nil {
			return nil, err
		}
		title, _ := capability.StringParam(params, "title")
		content, _ := capability.StringParam(params, "content")
		item, err := svc.UpdatePage(ctx, pageID, title, content)
		if err != nil {
			return nil, err
		}
		return mutationResult(item, types.EventConfluencePageUpdated, app, "page updated"), nil
	}
}

func invokeDeletePage(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		pageID, err := capability.RequiredString(params, "page_id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeletePage(ctx, pageID); err != nil {
			return nil, err
		}
		return deletePageResult(pageID, app), nil
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
