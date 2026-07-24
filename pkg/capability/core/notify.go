package core

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func notifySendInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	templateID, err := capability.RequiredString(params, "template_id")
	if err != nil {
		return nil, err
	}

	var channels []string
	if ch, ok := params["channels"]; ok {
		switch v := ch.(type) {
		case []string:
			channels = v
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					channels = append(channels, s)
				}
			}
		case string:
			channels = []string{v}
		}
	}
	if len(channels) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "channels is required")
	}

	var payload map[string]any
	if p, ok := params["payload"]; ok {
		if m, ok := p.(map[string]any); ok {
			payload = m
		} else {
			payload = map[string]any{"data": p}
		}
	} else {
		payload = params
	}

	var uid types.Uid
	if u, ok := params["uid"]; ok {
		if s, ok := u.(string); ok {
			uid = types.Uid(s)
		}
	}

	n := getNotifier()
	if n == nil {
		return nil, types.Errorf(types.ErrUnavailable, "notify gateway is not configured")
	}
	if err := n.Send(ctx, uid, templateID, channels, payload); err != nil {
		return nil, err
	}

	return &capability.InvokeResult{
		Data: map[string]any{"sent": true},
		Text: "notification sent",
	}, nil
}

func notifyHealthInvoker(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	return &capability.InvokeResult{Data: true}, nil
}
