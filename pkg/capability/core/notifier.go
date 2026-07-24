package core

import (
	"context"
	"sync"

	"github.com/flowline-io/flowbot/pkg/types"
)

// Notifier dispatches template notifications (wired to pkg/notify.GatewaySend in server).
type Notifier interface {
	Send(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error
}

var (
	notifierMu sync.RWMutex
	notifier   Notifier
)

// SetNotifier wires the notification gateway used by notify_send.
func SetNotifier(n Notifier) {
	notifierMu.Lock()
	defer notifierMu.Unlock()
	notifier = n
}

func getNotifier() Notifier {
	notifierMu.RLock()
	defer notifierMu.RUnlock()
	return notifier
}

// gatewayNotifier adapts a function to Notifier.
type gatewayNotifier struct {
	fn func(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error
}

func (g gatewayNotifier) Send(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error {
	return g.fn(ctx, uid, templateID, channels, payload)
}

// SetNotifierFunc wires a send function (tests and server).
func SetNotifierFunc(fn func(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error) {
	if fn == nil {
		SetNotifier(nil)
		return
	}
	SetNotifier(gatewayNotifier{fn: fn})
}
