// Package miniflux implements the RSS/feed reading capability.
package miniflux

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the miniflux capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapMiniflux,
		App:         app,
		Description: "Reader capability",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: types.EventReaderEntryNew, Description: "Fires when a new feed entry is received"},
			{Name: types.EventReaderEntrySaved, Description: "Fires when a feed entry is saved"},
			{Name: types.EventReaderEntryStarred, Description: "Fires when a feed entry is starred"},
			{Name: types.EventReaderEntryRead, Description: "Fires when a feed entry is marked as read"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpListFeeds, Description: "List feeds", Scopes: []string{auth.ScopeServiceReaderRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
				},
				Handler: invokeListFeeds(svc),
			},
			{
				Name: OpCreateFeed, Description: "Create a feed", Scopes: []string{auth.ScopeServiceReaderWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "feed_url", Type: "string", Required: true, Description: "Feed URL to subscribe to"}},
				Handler: invokeCreateFeed(svc),
			},
			{
				Name: OpListEntries, Description: "List entries", Scopes: []string{auth.ScopeServiceReaderRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "status", Type: "string", Required: false, Description: "Entry status filter"},
					{Name: "feed_id", Type: "int64", Required: false, Description: "Feed ID filter"},
				},
				Handler: invokeListEntries(svc),
			},
			{
				Name: OpMarkEntryRead, Description: "Mark entry as read", Scopes: []string{auth.ScopeServiceReaderWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "int64", Required: true, Description: "Entry ID"}},
				Handler: invokeMarkEntryRead(svc),
			},
			{
				Name: OpMarkEntryUnread, Description: "Mark entry as unread", Scopes: []string{auth.ScopeServiceReaderWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "int64", Required: true, Description: "Entry ID"}},
				Handler: invokeMarkEntryUnread(svc),
			},
			{
				Name: OpStarEntry, Description: "Star an entry", Scopes: []string{auth.ScopeServiceReaderWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "int64", Required: true, Description: "Entry ID"}},
				Handler: invokeStarEntry(svc),
			},
			{
				Name: OpUnstarEntry, Description: "Unstar an entry", Scopes: []string{auth.ScopeServiceReaderWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "int64", Required: true, Description: "Entry ID"}},
				Handler: invokeUnstarEntry(svc),
			},
		},
	})
}

func invokeListFeeds(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.ListFeeds(ctx, &FeedQuery{Page: capability.PageRequestFromParams(params)})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Feed]{Items: []*capability.Feed{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeCreateFeed(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		feedURL, err := capability.RequiredString(params, "feed_url")
		if err != nil {
			return nil, err
		}
		feed, err := svc.CreateFeed(ctx, feedURL)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: feed, Text: feed.Title}, nil
	}
}

func invokeListEntries(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &EntryQuery{Page: capability.PageRequestFromParams(params)}
		if v, ok := capability.StringParam(params, "status"); ok {
			q.Status = v
		}
		if v, ok := capability.Int64Param(params, "feed_id"); ok {
			q.FeedID = v
		}
		result, err := svc.ListEntries(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.Entry]{Items: []*capability.Entry{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeMarkEntryRead(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.MarkEntryRead(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "entry marked read"}, nil
	}
}

func invokeMarkEntryUnread(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.MarkEntryUnread(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "entry marked unread"}, nil
	}
}

func invokeStarEntry(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.StarEntry(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "entry starred"}, nil
	}
}

func invokeUnstarEntry(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.UnstarEntry(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "entry unstarred"}, nil
	}
}
