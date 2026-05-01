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
			{Name: "list_feeds", Description: "List feeds", Scopes: []string{auth.ScopeServiceReaderRead}},
			{Name: "create_feed", Description: "Create a feed", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: "list_entries", Description: "List entries", Scopes: []string{auth.ScopeServiceReaderRead}},
			{Name: "mark_entry_read", Description: "Mark entry as read", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: "mark_entry_unread", Description: "Mark entry as unread", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: "star_entry", Description: "Star an entry", Scopes: []string{auth.ScopeServiceReaderWrite}},
			{Name: "unstar_entry", Description: "Unstar an entry", Scopes: []string{auth.ScopeServiceReaderWrite}},
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
		{operation: "list_feeds", invoker: invokeListFeeds(svc)},
		{operation: "create_feed", invoker: invokeCreateFeed(svc)},
		{operation: "list_entries", invoker: invokeListEntries(svc)},
		{operation: "mark_entry_read", invoker: invokeMarkEntryRead(svc)},
		{operation: "mark_entry_unread", invoker: invokeMarkEntryUnread(svc)},
		{operation: "star_entry", invoker: invokeStarEntry(svc)},
		{operation: "unstar_entry", invoker: invokeUnstarEntry(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapReader, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeListFeeds(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		result, err := svc.ListFeeds(ctx, &FeedQuery{})
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
		feedURL, err := requiredString(params, "feed_url")
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
		q := &EntryQuery{}
		if v, ok := stringParam(params, "status"); ok {
			q.Status = v
		}
		if v, ok := int64Param(params, "feed_id"); ok {
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
		id, err := requiredInt64(params, "id")
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
		id, err := requiredInt64(params, "id")
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
		id, err := requiredInt64(params, "id")
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
		id, err := requiredInt64(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.UnstarEntry(ctx, id); err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Text: "entry unstarred"}, nil
	}
}

func requiredString(params map[string]any, key string) (string, error) {
	value, ok := stringParam(params, key)
	if !ok || value == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func stringParam(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return "", false
	}
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func requiredInt64(params map[string]any, key string) (int64, error) {
	value, ok := int64Param(params, key)
	if !ok {
		return 0, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func int64Param(params map[string]any, key string) (int64, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	}
	return 0, false
}
