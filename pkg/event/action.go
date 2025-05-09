package event

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
)

func SendMessage(ctx types.Context, msg types.MsgPayload) error {
	// payload
	src, err := sonic.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	payload := types.EventPayload{
		Typ: types.TypeOf(msg),
		Src: src,
	}

	// get user
	user, err := store.Database.GetUserByFlag(ctx.AsUser.String())
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	platformUsers, err := store.Database.GetPlatformUsersByUserId(user.ID)
	if err != nil {
		return fmt.Errorf("failed to get platform users: %w", err)
	}
	platformSet := sets.NewInt()
	for _, item := range platformUsers {
		platformSet.Insert(int(item.PlatformID))
	}

	// send topic
	if ctx.Topic != "" {
		platformChannel, err := store.Database.GetPlatformChannelByFlag(ctx.Topic)
		if err != nil {
			return fmt.Errorf("failed to get platform channel: %w", err)
		}
		if !platformSet.Has(int(platformChannel.PlatformID)) {
			return fmt.Errorf("topic %s not platform %d", ctx.Topic, platformChannel.PlatformID)
		}
		platform, err := store.Database.GetPlatform(platformChannel.PlatformID)
		if err != nil {
			return fmt.Errorf("failed to get platform: %w", err)
		}

		if platform.Name == "" {
			return fmt.Errorf("empty platform user %s topic %s", ctx.AsUser, ctx.Topic)
		}

		return PublishMessage(ctx.Context(), types.MessageSendEvent, types.Message{
			Platform: platform.Name,
			Topic:    ctx.Topic,
			Payload:  payload,
		})
	}

	// send all
	platforms, err := store.Database.GetPlatforms()
	if err != nil {
		return fmt.Errorf("failed to get platforms: %w", err)
	}
	platformName := make(map[int64]string, len(platforms))
	for _, item := range platforms {
		platformName[item.ID] = item.Name
	}

	for _, item := range platformUsers {
		channelUsers, err := store.Database.GetPlatformChannelUsersByUserFlag(item.Flag)
		if err != nil {
			flog.Error(err)
			continue
		}

		if platformName[item.PlatformID] == "" {
			continue
		}

		for _, channelUser := range channelUsers {
			err = PublishMessage(ctx.Context(), types.MessageSendEvent, types.Message{
				Platform: platformName[item.PlatformID],
				Topic:    channelUser.ChannelFlag,
				Payload:  payload,
			})
			if err != nil {
				return fmt.Errorf("failed to publish message: %w", err)
			}
		}
	}

	return nil
}

func BotEventFire(ctx types.Context, eventName string, param types.KV) error {
	return PublishMessage(ctx.Context(), types.BotRunEvent, types.BotEvent{
		EventName: eventName,
		Uid:       ctx.AsUser.String(),
		Topic:     ctx.Topic,
		Param:     param,
	})
}
