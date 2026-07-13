package notify

import (
	"context"

	pkgnotify "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Send dispatches a notification through the notification gateway.
// It renders the template for the given templateID and sends via each channel.
func Send(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error {
	return pkgnotify.GatewaySend(ctx, uid, templateID, channels, payload)
}
