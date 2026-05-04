// Package notify provides the notification capability for the ability framework.
// It allows pipeline steps to send notifications via the notification gateway.
package notify

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Descriptor returns the hub capability descriptor for notify.
func Descriptor() hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapNotify,
		Description: "Send notifications through the notification gateway",
		Healthy:     true,
		Operations: []hub.Operation{
			{
				Name:        ability.OpNotifySend,
				Description: "Send a notification using a template",
				Input: []hub.ParamDef{
					{Name: "template_id", Type: "string", Required: true, Description: "Template ID to render"},
					{Name: "channels", Type: "[]string", Required: true, Description: "Channels to send to"},
					{Name: "payload", Type: "map[string]any", Required: false, Description: "Template data payload"},
				},
			},
			{
				Name:        ability.OpNotifyDigest,
				Description: "Send an aggregated digest notification",
			},
		},
	}
}

// Register registers the notify capability invokers with the ability registry.
func Register() error {
	if err := hub.Default.Register(Descriptor()); err != nil {
		return err
	}

	return ability.RegisterInvoker(hub.CapNotify, ability.OpNotifySend, sendInvoker)
}

// sendInvoker is the ability.Invoker for notify.send operations.
// It delegates to the notification gateway.
func sendInvoker(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
	templateID, err := ability.RequiredString(params, "template_id")
	if err != nil {
		return nil, err
	}

	// extract channels
	var channels []string
	if ch, ok := params["channels"]; ok {
		switch v := ch.(type) {
		case []string:
			channels = v
		case []interface{}:
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

	// extract payload
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

	// extract UID from context if available
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

	return &ability.InvokeResult{
		Data: map[string]any{"sent": true},
		Text: "notification sent",
	}, nil
}
