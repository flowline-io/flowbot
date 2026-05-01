package archive

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
		Type:        hub.CapArchive,
		Backend:     backend,
		App:         app,
		Description: "Archive capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: "add", Description: "Add URL to archive", Scopes: []string{auth.ScopeServiceArchiveWrite}},
			{Name: "search", Description: "Search archive", Scopes: []string{auth.ScopeServiceArchiveRead}},
			{Name: "get", Description: "Get archived item", Scopes: []string{auth.ScopeServiceArchiveRead}},
		},
	}
}

func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "archive service is required")
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: "add", invoker: invokeAdd(svc)},
		{operation: "search", invoker: invokeSearch(svc)},
		{operation: "get", invoker: invokeGet(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapArchive, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeAdd(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		url, err := requiredString(params, "url")
		if err != nil {
			return nil, err
		}
		item, err := svc.Add(ctx, AddRequest{URL: url})
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{
			Data: item,
			Text: fmt.Sprintf("archive added: %s", item.URL),
			Events: []ability.EventRef{{
				EventType: types.EventArchiveItemCreated,
				EntityID:  item.ID,
			}},
		}, nil
	}
}

func invokeSearch(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q, _ := stringParam(params, "q")
		result, err := svc.Search(ctx, &SearchQuery{Q: q})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.ArchiveItem]{Items: []*ability.ArchiveItem{}}
		}
		return &ability.InvokeResult{Data: result.Items, Page: result.Page}, nil
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
	return fmt.Sprintf("%v", value), true
}
