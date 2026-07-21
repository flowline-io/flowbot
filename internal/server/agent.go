package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

// agentAction handle agent message
//
// When action is types.Collect, it will call bot.Collect
// When action is types.Pull, it will return list of instruct
// When action is types.Ack, it will update instruct state to done
// When action is types.Online, it will register agent
// When action is types.Offline, it will update agent online duration
//
// eventCtx carries trace context from the HTTP request or event pipeline.
func agentAction(eventCtx context.Context, uid types.Uid, data types.AgentData) (any, error) {
	ctx := types.Context{
		AsUser: uid,
	}
	ctx.SetContext(eventCtx)
	switch data.Action {
	case types.PullAction:
		return handlePullAction(ctx, uid)
	case types.AckAction:
		return nil, handleAckAction(ctx.Context(), data)
	case types.OnlineAction:
		return nil, handleOnlineAction(ctx, uid, data)
	case types.OfflineAction:
		return nil, handleOfflineAction(ctx, uid, data)
	case types.MessageAction:
		return nil, handleMessageAction(ctx, uid, data)
	}
	return nil, nil
}

func handlePullAction(ctx types.Context, uid types.Uid) (any, error) {
	list, err := store.Database.ListInstruct(ctx.Context(), uid, false, 10)
	if err != nil {
		return nil, err
	}
	var instruct []types.KV
	instruct = []types.KV{}
	for _, item := range list {
		instruct = append(instruct, types.KV{
			"no":        item.No,
			"bot":       item.Bot,
			"flag":      item.Flag,
			"content":   item.Content,
			"expire_at": item.ExpireAt,
		})
	}
	return instruct, nil
}

func handleAckAction(ctx context.Context, data types.AgentData) error {
	no, ok := data.Content.String("no")
	if !ok {
		return errors.New("error instruct no")
	}

	err := store.Database.UpdateInstruct(ctx, &gen.Instruct{
		No:    no,
		State: int(schema.InstructDone),
	})
	if err != nil {
		return err
	}
	return nil
}

func handleOnlineAction(ctx types.Context, uid types.Uid, data types.AgentData) error {
	hostid, ok := data.Content.String("hostid")
	if !ok {
		return errors.New("error hostid")
	}
	hostname, _ := data.Content.String("hostname")

	err := registerAgent(uid, "", hostid, hostname)
	if err != nil {
		flog.Error(err)
		return errors.New("register agent error")
	}

	key := cache.NewKey("online", "agent", hostid)
	_, ok, _ = cacheStore.Get(ctx.Context(), key)
	if !ok {
		err = notify.GatewaySendDefaultChannel(ctx.Context(), uid, "agent.status", map[string]any{
			"hostid":   hostid,
			"hostname": hostname,
			"status":   "online",
		})
		if err != nil && !notify.WarnSkipNoDefault(err, "agent online") {
			flog.Error(fmt.Errorf("send message error %w", err))
		}
	}

	err = cacheStore.Set(ctx.Context(), key, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLShort)
	if err != nil {
		flog.Error(err)
		return errors.New("set agent online error")
	}
	return nil
}

func handleOfflineAction(ctx types.Context, uid types.Uid, data types.AgentData) error {
	hostid, ok := data.Content.String("hostid")
	if !ok {
		return errors.New("error hostid")
	}
	hostname, _ := data.Content.String("hostname")

	err := store.Database.UpdateAgentOnlineDuration(ctx.Context(), uid, "", hostid, time.Now())
	if err != nil {
		flog.Error(fmt.Errorf("update online duration error %w", err))
	}

	err = cacheStore.Del(ctx.Context(), cache.NewKey("online", "agent", hostid))
	if err != nil {
		flog.Error(fmt.Errorf("del agent online stats error %w", err))
	}

	err = notify.GatewaySendDefaultChannel(ctx.Context(), uid, "agent.status", map[string]any{
		"hostid":   hostid,
		"hostname": hostname,
		"status":   "offline",
	})
	if err != nil && !notify.WarnSkipNoDefault(err, "agent offline") {
		flog.Error(fmt.Errorf("send message error %w", err))
	}
	return nil
}

func handleMessageAction(ctx types.Context, uid types.Uid, data types.AgentData) error {
	message, ok := data.Content.String("message")
	if !ok {
		return errors.New("empty message")
	}

	err := notify.GatewaySendDefaultChannel(ctx.Context(), uid, "agent.status", map[string]any{
		"message": message,
	})
	if err != nil && !notify.WarnSkipNoDefault(err, "agent message") {
		flog.Error(fmt.Errorf("send message error %w", err))
	}
	return nil
}
