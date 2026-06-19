package chatagent

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// ResolveDeliveryContext loads platform delivery info from the latest session message.
func ResolveDeliveryContext(ctx context.Context, sessionID string) ScheduledDelivery {
	if store.Database == nil || sessionID == "" {
		return ScheduledDelivery{}
	}
	messages, err := store.Database.GetMessagesBySession(ctx, sessionID)
	if err != nil {
		flog.Debug("[chat-agent] delivery context session=%s: %v", sessionID, err)
		return ScheduledDelivery{}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Topic == "" {
			continue
		}
		delivery := ScheduledDelivery{
			Topic:      msg.Topic,
			PlatformID: msg.PlatformID,
		}
		if msg.PlatformID > 0 {
			platformRow, perr := store.Database.GetPlatform(ctx, msg.PlatformID)
			if perr == nil && platformRow != nil {
				delivery.Platform = platformRow.Name
			}
		}
		return delivery
	}
	return ScheduledDelivery{}
}

func deliveryFromTask(task *gen.ChatScheduledTask) ScheduledDelivery {
	if task == nil || task.Delivery == nil {
		return ScheduledDelivery{}
	}
	d := ScheduledDelivery{}
	if v, ok := task.Delivery["platform"].(string); ok {
		d.Platform = v
	}
	if v, ok := task.Delivery["topic"].(string); ok {
		d.Topic = v
	}
	switch v := task.Delivery["platform_id"].(type) {
	case float64:
		d.PlatformID = int64(v)
	case int64:
		d.PlatformID = v
	case int:
		d.PlatformID = int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			d.PlatformID = parsed
		}
	}
	return d
}

func deliveryToMap(d ScheduledDelivery) map[string]any {
	if d.Platform == "" && d.Topic == "" && d.PlatformID == 0 {
		return nil
	}
	return map[string]any{
		"platform":    d.Platform,
		"topic":       d.Topic,
		"platform_id": d.PlatformID,
	}
}

// deliverScheduledReply pushes the agent reply to the user's chat or notify channels.
func deliverScheduledReply(ctx context.Context, task *gen.ChatScheduledTask, text string) {
	if task == nil || text == "" {
		return
	}
	header := fmt.Sprintf("[Scheduled: %s]\n", task.Name)
	body := header + text

	delivery := deliveryFromTask(task)
	if delivery.Platform != "" && delivery.Topic != "" {
		sendErr := sendPlatformScheduledReply(delivery, body)
		if sendErr == nil {
			return
		}
		flog.Warn("[chat-agent] scheduled delivery platform task=%s: %v", task.Flag, sendErr)
	}

	uid := types.Uid(task.UID)
	err := notify.GatewaySend(ctx, uid, "agent.status", []string{"slack", "ntfy"}, map[string]any{
		notify.PayloadKeySummary: body,
		"task_name":              task.Name,
		"task_id":                task.Flag,
	})
	if err != nil {
		flog.Warn("[chat-agent] scheduled delivery notify task=%s uid=%s: %v", task.Flag, uid, err)
	}
}

func sendPlatformScheduledReply(delivery ScheduledDelivery, text string) error {
	caller, err := platforms.GetCaller(delivery.Platform)
	if err != nil {
		return err
	}
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   delivery.Topic,
			"message": caller.Adapter.MessageConvert(types.TextMsg{Text: text}),
		},
	})
	if resp.Status != protocol.Success {
		return fmt.Errorf("platform send failed: %+v", resp)
	}
	return nil
}
