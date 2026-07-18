package transmission

import (
	"context"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers the transmission capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapTransmission,
		App:         app,
		Description: "Download capability for Transmission",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpAdd, Description: "Add a torrent by magnet or HTTP(S) URL", Scopes: []string{auth.ScopeServiceTransmissionWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "url", Type: "string", Required: true, Description: "Magnet link or torrent file URL"},
				},
				Handler: invokeAdd(svc, app),
			},
			{
				Name: OpList, Description: "List torrents", Scopes: []string{auth.ScopeServiceTransmissionRead},
				Handler: invokeList(svc),
			},
			{
				Name: OpStop, Description: "Stop torrents by ID", Scopes: []string{auth.ScopeServiceTransmissionWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "ids", Type: "array", Required: true, Description: "Torrent IDs to stop"},
				},
				Handler: invokeStop(svc),
			},
			{
				Name: OpRemove, Description: "Remove torrents by ID", Scopes: []string{auth.ScopeServiceTransmissionWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "ids", Type: "array", Required: true, Description: "Torrent IDs to remove"},
				},
				Handler: invokeRemove(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceTransmissionRead},
				Handler: invokeHealth(svc),
			},
		},
	})
}

func invokeAdd(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		url, err := capability.RequiredString(params, "url")
		if err != nil {
			return nil, err
		}
		item, err := svc.AddTorrent(ctx, AddTorrentInput{URL: url})
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: item,
			Resource: &capability.ResourceMeta{
				EntityID: strconv.FormatInt(item.ID, 10),
				App:      app,
			},
		}, nil
	}
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		items, err := svc.ListTorrents(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: items}, nil
	}
}

func invokeStop(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		ids, err := requiredInt64Slice(params, "ids")
		if err != nil {
			return nil, err
		}
		if err := svc.StopTorrents(ctx, StopTorrentsInput{IDs: ids}); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]any{"stopped": len(ids)}}, nil
	}
}

func invokeRemove(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		ids, err := requiredInt64Slice(params, "ids")
		if err != nil {
			return nil, err
		}
		if err := svc.RemoveTorrents(ctx, RemoveTorrentsInput{IDs: ids}); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]any{"removed": len(ids)}}, nil
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
