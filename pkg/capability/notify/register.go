// Package notify provides the notification capability for the capability framework.
// It allows pipeline steps to send notifications via the notification gateway.
package notify

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// serviceMarker is a non-nil instance used for hub registration.
type serviceMarker struct{}

// Register registers the notify capability with hub and invoker registry.
func Register() error {
	return capability.Register(capability.Spec{
		Type:        hub.CapNotify,
		Description: "Send notifications through the notification gateway",
		Instance:    serviceMarker{},
		Ops: []capability.OpDef{
			{
				Name: OpSend, Description: "Send a notification using a template", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "template_id", Type: "string", Required: true, Description: "Template ID to render"},
					{Name: "channels", Type: "[]string", Required: true, Description: "Channels to send to"},
					{Name: "payload", Type: "map[string]any", Required: false, Description: "Template data payload"},
				},
				Handler: sendInvoker,
			},
			{
				Name: OpDigest, Description: "Send an aggregated digest notification", Mutation: true,
				Handler: digestInvoker,
			},
		},
	})
}

// sendInvoker is the capability.Invoker for notify.send operations.
// It delegates to the notification gateway.
func sendInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
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

	err = Send(ctx, uid, templateID, channels, payload)
	if err != nil {
		return nil, err
	}

	return &capability.InvokeResult{
		Data: map[string]any{"sent": true},
		Text: "notification sent",
	}, nil
}

func digestInvoker(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	return nil, types.Errorf(types.ErrUnavailable, "notify digest is not implemented")
}
