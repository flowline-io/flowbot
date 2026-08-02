package notify

import (
	"context"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// ChannelInapp is the seeded in-app inbox channel name.
	ChannelInapp = "inapp"
	// PayloadKeyURL is an optional deep-link URL in the gateway payload.
	PayloadKeyURL = "url"
	// PayloadKeyTitle is an optional title override in the gateway payload.
	PayloadKeyTitle = "title"
	// PayloadKeyCorrelationID links inapp and deferred external records.
	PayloadKeyCorrelationID = "correlation_id"
	// PayloadKeyEscalateAfter overrides escalate delay (Go duration string, e.g. "5m").
	PayloadKeyEscalateAfter = "escalate_after"
)

var (
	escalationMu      sync.Mutex
	escalationStop    chan struct{}
	escalationTicker  *time.Ticker
	escalationRunning bool
)

// StartEscalationWorker starts a background ticker that flushes due deferred notifications.
func StartEscalationWorker() {
	escalationMu.Lock()
	defer escalationMu.Unlock()
	if escalationRunning {
		return
	}
	escalationStop = make(chan struct{})
	escalationTicker = time.NewTicker(5 * time.Second)
	escalationRunning = true
	go runEscalationLoop(escalationStop, escalationTicker)
}

// StopEscalationWorker stops the escalation worker.
func StopEscalationWorker() {
	escalationMu.Lock()
	defer escalationMu.Unlock()
	if !escalationRunning {
		return
	}
	close(escalationStop)
	if escalationTicker != nil {
		escalationTicker.Stop()
	}
	escalationRunning = false
	escalationStop = nil
	escalationTicker = nil
}

// StopEscalationWorkerForTest stops the escalation worker (tests only).
func StopEscalationWorkerForTest() {
	StopEscalationWorker()
}

func runEscalationLoop(stop <-chan struct{}, ticker *time.Ticker) {
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			FlushDueDeferred(context.Background())
		}
	}
}

// FlushDueDeferred processes deferred records that are due for external dispatch.
func FlushDueDeferred(ctx context.Context) {
	ns := GetNotifyStore()
	if ns == nil {
		return
	}
	records, err := ns.ListDueDeferred(ctx, time.Now(), 50)
	if err != nil {
		flog.Warn("[notify] list due deferred: %v", err)
		return
	}
	for _, rec := range records {
		flushDeferredRecord(ctx, ns, rec.UID, rec.ID, rec.Channel, rec.TemplateID, rec.CorrelationID, rec.PayloadSnapshot)
	}
}

func flushDeferredRecord(ctx context.Context, ns NotifyRecords, uid string, id int64, channel, templateID, correlationID string, payload map[string]any) {
	if ns == nil {
		return
	}
	if correlationID != "" {
		unread, err := ns.HasUnreadSuccessByCorrelation(ctx, uid, correlationID)
		if err != nil {
			flog.Warn("[notify] check unread for deferred %d: %v", id, err)
			return
		}
		if !unread {
			updateDeferredStatus(ctx, ns, id, "cancelled", "")
			return
		}
	}

	eval, err := evaluateAndRenderNotification(ctx, templateID, channel, payload)
	if err != nil {
		updateDeferredStatus(ctx, ns, id, "failed", err.Error())
		return
	}
	if eval != nil && eval.action != "" {
		updateDeferredStatus(ctx, ns, id, eval.action, "")
		return
	}
	if eval == nil || eval.renderResult == nil {
		updateDeferredStatus(ctx, ns, id, "cancelled", "")
		return
	}

	msg := buildNotifyMessage(eval.renderResult, payload)
	if err := dispatchChannel(ctx, types.Uid(uid), channel, msg); err != nil {
		updateDeferredStatus(ctx, ns, id, "failed", err.Error())
		flog.Warn("[notify] deferred flush failed id=%d channel=%s: %v", id, channel, err)
		return
	}
	updateDeferredStatus(ctx, ns, id, "success", "")
}

func updateDeferredStatus(ctx context.Context, ns NotifyRecords, id int64, status, errMsg string) {
	if err := ns.UpdateRecordStatus(ctx, id, status, errMsg); err != nil {
		flog.Warn("[notify] update deferred %d to %s: %v", id, status, err)
	}
}
