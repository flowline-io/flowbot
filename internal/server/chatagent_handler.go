package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

var chatAgentService *chatagent.Service

func runChatAgent(
	parentCtx context.Context,
	caller *platforms.Caller,
	msg protocol.MessageEventData,
	uid types.Uid,
	sessionID string,
	platformID int64,
	topic string,
) {
	start := time.Now()
	runTimeout := chatagent.RunTimeout()
	flog.Info("[chat-agent] run start uid=%s session=%s platform=%s topic=%s text_len=%d timeout=%s",
		uid, sessionID, msg.Self.Platform, topic, len(msg.AltMessage), runTimeout)

	// Detach from Watermill message context (canceled when the handler returns)
	// while keeping the OTel SpanContext so the run stays on the same trace.
	ctx, cancel := fbtrace.DetachWithTimeout(parentCtx, runTimeout)
	defer cancel()
	svc := chatAgentService
	if svc == nil {
		svc = ChatAgentService()
	}
	svc.BindRunCancel(sessionID, cancel)
	defer svc.UnbindRunCancel(sessionID)

	sink, platformMsgID, streaming := startChatStream(caller, msg)

	reply, err := svc.Run(ctx, chatagent.RunRequest{
		SessionID: sessionID,
		Text:      msg.AltMessage,
	}, sink)
	if err != nil {
		handleChatRunError(ctx, caller, msg, sink, streaming, err, uid, sessionID, runTimeout, start)
		return
	}

	now := time.Now().UTC()
	err = store.Database.CreateMessage(ctx, gen.Message{
		Flag:          types.Id(),
		PlatformID:    platformID,
		PlatformMsgID: platformMsgID,
		Topic:         topic,
		Role:          types.Assistant,
		Session:       sessionID,
		Content:       schema.JSON{"text": reply},
		State:         int(schema.MessageCreated),
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] persist assistant message uid=%s session=%s: %w", uid, sessionID, err))
		if streaming {
			flushStreamSink(ctx, sink, "Failed to save reply. Please try again.")
			return
		}
		sendChatReply(caller, msg, types.TextMsg{Text: "Failed to save reply. Please try again."})
		return
	}

	flog.Info("[chat-agent] run complete uid=%s session=%s reply_len=%d duration=%s streaming=%t",
		uid, sessionID, len(reply), time.Since(start).Round(time.Millisecond), streaming)
	if !streaming {
		sendChatReply(caller, msg, types.TextMsg{Text: reply})
	}
}

func startChatStream(caller *platforms.Caller, msg protocol.MessageEventData) (chatagent.StreamSink, string, bool) {
	if msg.Self.Platform != slack.ID {
		return nil, "", false
	}

	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(types.TextMsg{Text: "Thinking..."}),
		},
	})
	messageID := platformMessageID(resp)
	if messageID == "" {
		flog.Warn("[chat-agent] stream placeholder failed topic=%s resp=%+v", msg.TopicId, resp)
		return nil, "", false
	}

	return chatagent.NewSlackStreamSink(caller, msg.TopicId, messageID), messageID, true
}

func handleChatRunError(
	ctx context.Context,
	caller *platforms.Caller,
	msg protocol.MessageEventData,
	sink chatagent.StreamSink,
	streaming bool,
	err error,
	uid types.Uid,
	sessionID string,
	runTimeout time.Duration,
	start time.Time,
) {
	if errors.Is(err, context.DeadlineExceeded) {
		flog.Warn("[chat-agent] run timed out uid=%s session=%s timeout=%s duration=%s",
			uid, sessionID, runTimeout, time.Since(start).Round(time.Millisecond))
		if streaming {
			flushStreamSink(ctx, sink, "Chat agent timed out. Please try again.")
			return
		}
		sendChatReply(caller, msg, types.TextMsg{Text: "Chat agent timed out. Please try again."})
		return
	}
	if errors.Is(err, context.Canceled) {
		flog.Info("[chat-agent] run cancelled uid=%s session=%s duration=%s",
			uid, sessionID, time.Since(start).Round(time.Millisecond))
		if streaming {
			flushStreamSink(ctx, sink, "Chat session ended.")
			return
		}
		return
	}

	flog.Error(fmt.Errorf("[chat-agent] run failed uid=%s session=%s duration=%s: %w",
		uid, sessionID, time.Since(start).Round(time.Millisecond), err))
	if streaming {
		flushStreamSink(ctx, sink, "Chat agent is unavailable. Please try again later.")
		return
	}
	sendChatReply(caller, msg, types.TextMsg{Text: "Chat agent is unavailable. Please try again later."})
}

func flushStreamSink(ctx context.Context, sink chatagent.StreamSink, text string) {
	if sink == nil {
		return
	}
	if err := sink.Flush(ctx, text); err != nil {
		flog.Warn("[chat-agent] stream cleanup flush: %v", err)
	}
}

func platformMessageID(resp protocol.Response) string {
	if resp.Status != protocol.Success || resp.Data == nil {
		return ""
	}
	switch data := resp.Data.(type) {
	case map[string]string:
		return data["message_id"]
	case map[string]any:
		if id, ok := data["message_id"].(string); ok {
			return id
		}
	}
	return ""
}

func sendChatReply(caller *platforms.Caller, msg protocol.MessageEventData, payload types.MsgPayload) {
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(payload),
		},
	})
	flog.Debug("[chat-agent] platform reply topic=%s resp=%+v", msg.TopicId, resp)
}
