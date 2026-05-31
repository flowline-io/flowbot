// Package reader implements the RSS/feed reading capability.
package reader

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapReader,
		Backend:     backend,
		App:         app,
		Description: "Reader capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{
				Name:        ability.OpReaderListFeeds,
				Description: "List feeds",
				Scopes:      []string{auth.ScopeServiceReaderRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
				},
			},
			{
				Name:        ability.OpReaderCreateFeed,
				Description: "Create a feed",
				Scopes:      []string{auth.ScopeServiceReaderWrite},
				Input: []hub.ParamDef{
					{Name: "feed_url", Type: "string", Required: true, Description: "Feed URL to subscribe to"},
				},
			},
			{
				Name:        ability.OpReaderListEntries,
				Description: "List entries",
				Scopes:      []string{auth.ScopeServiceReaderRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "status", Type: "string", Required: false, Description: "Entry status filter"},
					{Name: "feed_id", Type: "int64", Required: false, Description: "Feed ID filter"},
				},
			},
			{
				Name:        ability.OpReaderMarkEntryRead,
				Description: "Mark entry as read",
				Scopes:      []string{auth.ScopeServiceReaderWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int64", Required: true, Description: "Entry ID"},
				},
			},
			{
				Name:        ability.OpReaderMarkEntryUnread,
				Description: "Mark entry as unread",
				Scopes:      []string{auth.ScopeServiceReaderWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int64", Required: true, Description: "Entry ID"},
				},
			},
			{
				Name:        ability.OpReaderStarEntry,
				Description: "Star an entry",
				Scopes:      []string{auth.ScopeServiceReaderWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int64", Required: true, Description: "Entry ID"},
				},
			},
			{
				Name:        ability.OpReaderUnstarEntry,
				Description: "Unstar an entry",
				Scopes:      []string{auth.ScopeServiceReaderWrite},
				Input: []hub.ParamDef{
					{Name: "id", Type: "int64", Required: true, Description: "Entry ID"},
				},
			},
		},
		Events: []hub.EventDef{
			{Name: types.EventReaderEntryNew, Description: "Fires when a new feed entry is received"},
			{Name: types.EventReaderEntrySaved, Description: "Fires when a feed entry is saved"},
			{Name: types.EventReaderEntryStarred, Description: "Fires when a feed entry is starred"},
			{Name: types.EventReaderEntryRead, Description: "Fires when a feed entry is marked as read"},
		},
	}
}

// RegisterService registers the reader capability with the hub and ability registry.
// It returns nil and logs a warning when svc is nil (provider not configured).
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		flog.Warn("reader capability: service is nil, skipping registration for %s/%s", backend, app)
		return nil
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpReaderListFeeds, invoker: invokeListFeeds(svc)},
		{operation: ability.OpReaderCreateFeed, invoker: invokeCreateFeed(svc)},
		{operation: ability.OpReaderListEntries, invoker: invokeListEntries(svc)},
		{operation: ability.OpReaderMarkEntryRead, invoker: invokeMarkEntryRead(svc)},
		{operation: ability.OpReaderMarkEntryUnread, invoker: invokeMarkEntryUnread(svc)},
		{operation: ability.OpReaderStarEntry, invoker: invokeStarEntry(svc)},
		{operation: ability.OpReaderUnstarEntry, invoker: invokeUnstarEntry(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapReader, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeListFeeds(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		result, err := svc.ListFeeds(ctx, &FeedQuery{Page: ability.PageRequestFromParams(params)})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Feed]{Items: []*ability.Feed{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeCreateFeed(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		feedURL, err := ability.RequiredString(params, "feed_url")
		if err != nil {
			return nil, err
		}
		feed, err := svc.CreateFeed(ctx, feedURL)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: feed, Text: feed.Title}, nil
	}
}

func invokeListEntries(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &EntryQuery{Page: ability.PageRequestFromParams(params)}
		if v, ok := ability.StringParam(params, "status"); ok {
			q.Status = v
		}
		if v, ok := ability.Int64Param(params, "feed_id"); ok {
			q.FeedID = v
		}
		result, err := svc.ListEntries(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Entry]{Items: []*ability.Entry{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeMarkEntryRead(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.MarkEntryRead(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "entry marked read"}, nil
	}
}

func invokeMarkEntryUnread(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.MarkEntryUnread(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "entry marked unread"}, nil
	}
}

func invokeStarEntry(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.StarEntry(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "entry starred"}, nil
	}
}

func invokeUnstarEntry(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		id, err := ability.RequiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.UnstarEntry(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "entry unstarred"}, nil
	}
}
