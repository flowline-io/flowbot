package reader

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
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
			{Name: ability.OpReaderListFeeds, Description: "List feeds", Scopes: []string{auth.ScopeServiceReaderRead}},
			{Name: ability.OpReaderCreateFeed, Description: "Create a feed", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: ability.OpReaderListEntries, Description: "List entries", Scopes: []string{auth.ScopeServiceReaderRead}},
			{Name: ability.OpReaderMarkEntryRead, Description: "Mark entry as read", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: ability.OpReaderMarkEntryUnread, Description: "Mark entry as unread", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: ability.OpReaderStarEntry, Description: "Star an entry", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: ability.OpReaderUnstarEntry, Description: "Unstar an entry", Scopes: []string{auth.ScopeServiceReaderWrite}},
		},
	}
}

func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "reader service is required")
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
