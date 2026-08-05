package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type directMessageContext struct {
	ctx        types.Context
	msg        protocol.MessageEventData
	uid        types.Uid
	topic      string
	platformID int64
}

// directIncomingMessage handles incoming message events for direct channels.
//
// It will register the user and channel if they don't already exist, then
// dispatch the message to the appropriate handler based on the content.
//
// eventCtx carries trace context from the consuming Watermill router middleware.
func directIncomingMessage(eventCtx context.Context, caller *platforms.Caller, e protocol.Event) {
	msg, ok := e.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	dmCtx, err := buildDirectMessageContext(eventCtx, e.Id, msg)
	if err != nil {
		flog.Error(err)
		return
	}
	if isDuplicateDirectMessage(dmCtx) {
		return
	}

	module.Behavior(dmCtx.uid, module.MessageBotIncomingBehavior, 1)

	scope := chatScope(msg)
	sessionID := loadThreadSessionID(dmCtx.ctx, dmCtx.uid, scope)
	payload, sessionID := manageChatSession(dmCtx.ctx, dmCtx.uid, scope, msg.AltMessage, sessionID, nil)

	// Module commands (version, etc.) win over the agent even when chat is enabled,
	// and always reply in the channel (no thread).
	if payload == nil && !chatagent.IsChatControlCommand(msg.AltMessage) {
		if modPayload := dispatchToModules(dmCtx.ctx, msg.AltMessage); modPayload != nil {
			sendDirectPlatformReply(caller, msg, modPayload, "")
			return
		}
	}

	sessionID = ensureThreadSession(dmCtx.ctx, dmCtx.uid, scope, sessionID, msg.AltMessage)
	refreshThreadSessionCache(dmCtx.ctx, dmCtx.uid, scope, sessionID)

	if sessionID != "" && !persistDirectUserMessage(dmCtx, sessionID, msg) {
		return
	}

	payload = buildHelpMessage(msg.AltMessage, payload)
	dispatchDirectMessage(caller, dmCtx, msg, sessionID, payload)
}

func buildDirectMessageContext(eventCtx context.Context, eventID string, msg protocol.MessageEventData) (directMessageContext, error) {
	platform, err := store.PlatformStoreFromDB().GetPlatformByName(eventCtx, msg.Self.Platform)
	if err != nil {
		return directMessageContext{}, err
	}

	uid, err := registerPlatformUser(msg, platform)
	if err != nil {
		return directMessageContext{}, err
	}

	topic, err := registerPlatformChannel(msg, platform)
	if err != nil {
		return directMessageContext{}, err
	}

	ctx := types.Context{
		Id:       eventID,
		AsUser:   uid,
		Topic:    topic,
		Platform: msg.Self.Platform,
	}
	ctx.SetContext(eventCtx)
	ctx.SetTimeout(10 * time.Minute)

	return directMessageContext{
		ctx:        ctx,
		msg:        msg,
		uid:        uid,
		topic:      topic,
		platformID: platform.ID,
	}, nil
}

func isDuplicateDirectMessage(dmCtx directMessageContext) bool {
	if dmCtx.msg.MessageId == "" {
		return false
	}
	findMessage, err := store.MessageStoreFromDB().GetMessageByPlatform(dmCtx.ctx.Context(), dmCtx.platformID, dmCtx.msg.MessageId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		flog.Error(err)
		return true
	}
	if findMessage != nil {
		flog.Info("message %s %s already exists", dmCtx.msg.Self.Platform, dmCtx.msg.MessageId)
		return true
	}
	return false
}

func persistDirectUserMessage(dmCtx directMessageContext, sessionID string, msg protocol.MessageEventData) bool {
	content := schema.JSON{"text": msg.AltMessage}
	if msg.ThreadId != "" {
		content["thread_id"] = msg.ThreadId
	}
	err := store.MessageStoreFromDB().CreateMessage(dmCtx.ctx.Context(), gen.Message{
		Flag:          types.Id(),
		PlatformID:    dmCtx.platformID,
		PlatformMsgID: msg.MessageId,
		Topic:         dmCtx.topic,
		Role:          types.User,
		Session:       sessionID,
		Content:       content,
		State:         int(schema.MessageCreated),
	})
	if err != nil {
		flog.Error(fmt.Errorf("persist direct user message: %w", err))
		return false
	}
	return true
}

func dispatchDirectMessage(
	caller *platforms.Caller,
	dmCtx directMessageContext,
	msg protocol.MessageEventData,
	sessionID string,
	payload types.MsgPayload,
) {
	if sessionID != "" && !chatagent.IsChatControlCommand(msg.AltMessage) {
		flog.Info("[chat-agent] dispatch agent run uid=%s session=%s platform=%s msg_id=%s text_len=%d",
			dmCtx.uid, sessionID, msg.Self.Platform, msg.MessageId, len(msg.AltMessage))
		go runChatAgent(dmCtx.ctx.Context(), caller, msg, dmCtx.uid, sessionID, dmCtx.platformID, dmCtx.topic)
		return
	}

	if payload == nil {
		return
	}
	sendDirectPlatformReply(caller, msg, payload, chatCommandThreadID(msg, msg.AltMessage))
}

func sendDirectPlatformReply(caller *platforms.Caller, msg protocol.MessageEventData, payload types.MsgPayload, threadID string) {
	flog.Debug("incoming send message action topic %v payload %+v", msg.MessageId, payload)
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: platformSendParams(msg.TopicId, threadID, caller.Adapter.MessageConvert(payload)),
	})
	flog.Info("[event] %+v  response: %+v", msg, resp)
}

func platformSendParams(topic, threadID string, message protocol.Message) types.KV {
	params := types.KV{
		"topic":   topic,
		"message": message,
	}
	if threadID != "" {
		params["thread_id"] = threadID
	}
	return params
}
