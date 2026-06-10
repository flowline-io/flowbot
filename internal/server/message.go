package server

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// groupIncomingMessage processes incoming message events for group channels.
//
// It will register the user and channel if they don't already exist, then
// iterate over all available bots and run their command handlers if they
// have any.
//
// If any bot returns a non-nil payload, it will be sent back to the group
// channel as a message.
//
// If no bot returns a payload, a default message will be sent.
//
// eventCtx carries trace context from the consuming Watermill router middleware.
func groupIncomingMessage(eventCtx context.Context, caller *platforms.Caller, e protocol.Event) {
	msg, ok := e.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	uid, err := registerPlatformUser(msg)
	if err != nil {
		flog.Error(err)
		return
	}

	topic, err := registerPlatformChannel(msg)
	if err != nil {
		flog.Error(err)
		return
	}

	ctx := types.Context{
		Id:     e.Id,
		Topic:  topic,
		AsUser: uid,
	}
	ctx.SetContext(eventCtx)
	flog.Debug("context: %+v", ctx)

	// behavior
	module.Behavior(uid, module.MessageGroupIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	for name, handle := range module.List() {
		if !handle.IsReady() {
			flog.Info("module %s unavailable", name)
			continue
		}
	}

	flog.Debug("incoming send message action topic %v payload %+v", msg.MessageId, payload)
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(payload),
		},
	})
	flog.Info("event: %+v  response: %+v", msg, resp)
}

// notifyAll send message to all users
//
// This function will send a message to all users in the database.
// If an error occurs, it will be logged and the function will continue to the next user.
func notifyAll(message string) {
	// send message
	users, err := store.Database.GetUsers(context.Background())
	if err != nil {
		flog.Error(fmt.Errorf("notify error %w", err))
		return
	}

	for _, item := range users {
		ctx := types.Context{
			AsUser: types.Uid(item.Flag),
		}
		err = notify.GatewaySend(ctx.Context(), types.Uid(item.Flag), "agent.status", []string{"slack", "ntfy"}, map[string]any{
			"message": message,
		})
		if err != nil {
			flog.Error(fmt.Errorf("notify error %w", err))
			continue
		}
	}
}

// onlineStatus handles MessageEvent protocol event and updates user online status.
// It will set user online status to 30 minutes if user is not online,
// or reset user online status to 30 minutes if user is already online.
func onlineStatus(msg protocol.Event) {
	med, ok := msg.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	ctx := context.Background()
	key := cache.NewKey("online", "user", med.UserId)
	_, ok, _ = cacheStore.Get(ctx, key)
	if !ok {
		cacheStore.Set(ctx, key, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLSession)
	}
}
