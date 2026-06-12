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
	"github.com/flowline-io/flowbot/pkg/cache"
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

	chatKey := cache.NewKey("chat", "session", dmCtx.uid.String())
	sessionID := loadChatSessionID(dmCtx.ctx, chatKey, dmCtx.uid)
	payload, sessionID := manageChatSession(dmCtx.ctx, chatKey, msg.AltMessage, sessionID, nil, dmCtx.uid)
	refreshChatSessionCache(dmCtx.ctx, chatKey, sessionID)

	if sessionID != "" && !persistDirectUserMessage(dmCtx, sessionID, msg) {
		return
	}

	payload = buildHelpMessage(msg.AltMessage, payload)
	dispatchDirectMessage(caller, dmCtx, msg, sessionID, payload)
}

func buildDirectMessageContext(eventCtx context.Context, eventID string, msg protocol.MessageEventData) (directMessageContext, error) {
	uid, err := registerPlatformUser(msg)
	if err != nil {
		return directMessageContext{}, err
	}

	topic, err := registerPlatformChannel(msg)
	if err != nil {
		return directMessageContext{}, err
	}

	ctx := types.Context{Id: eventID, AsUser: uid}
	ctx.SetContext(eventCtx)
	ctx.SetTimeout(10 * time.Minute)

	platform, err := store.Database.GetPlatformByName(ctx.Context(), msg.Self.Platform)
	if err != nil {
		return directMessageContext{}, err
	}

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
	findMessage, err := store.Database.GetMessageByPlatform(dmCtx.ctx.Context(), dmCtx.platformID, dmCtx.msg.MessageId)
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

func loadChatSessionID(ctx types.Context, chatKey cache.Key, uid types.Uid) string {
	sessionID, ok, err := cacheStore.Get(ctx.Context(), chatKey)
	if err != nil {
		flog.Error(err)
		return ""
	}
	if !ok {
		return ""
	}
	flog.Debug("[chat-agent] session cache hit uid=%s session=%s", uid, sessionID)
	return sessionID
}

func refreshChatSessionCache(ctx types.Context, chatKey cache.Key, sessionID string) {
	if sessionID == "" {
		return
	}
	if err := cacheStore.Set(ctx.Context(), chatKey, sessionID, cache.TTLSession); err != nil {
		flog.Error(fmt.Errorf("refresh chat session cache: %w", err))
	}
}

func persistDirectUserMessage(dmCtx directMessageContext, sessionID string, msg protocol.MessageEventData) bool {
	err := store.Database.CreateMessage(dmCtx.ctx.Context(), gen.Message{
		Flag:          types.Id(),
		PlatformID:    dmCtx.platformID,
		PlatformMsgID: msg.MessageId,
		Topic:         dmCtx.topic,
		Role:          types.User,
		Session:       sessionID,
		Content:       schema.JSON{"text": msg.AltMessage},
		State:         int(schema.MessageCreated),
	})
	if err != nil {
		flog.Error(err)
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
		go runChatAgent(caller, msg, dmCtx.uid, sessionID, dmCtx.platformID, dmCtx.topic)
		return
	}

	if sessionID == "" && payload == nil {
		payload = dispatchToModules(dmCtx.ctx, msg.AltMessage)
	}
	if payload == nil {
		return
	}

	flog.Debug("incoming send message action topic %v payload %+v", msg.MessageId, payload)
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(payload),
		},
	})
	flog.Info("[event] %+v  response: %+v", msg, resp)
}
