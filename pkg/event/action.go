package event

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	jsoniter "github.com/json-iterator/go"
)

func SendMessage(ctx context.Context, uid, topic string, msg types.MsgPayload) error {
	// payload
	src, err := jsoniter.Marshal(msg)
	if err != nil {
		return err
	}
	payload := types.EventPayload{
		Typ: types.Tye(msg),
		Src: src,
	}

	// get user
	user, err := store.Database.GetUserByFlag(uid)
	if err != nil {
		return err
	}
	platformUsers, err := store.Database.GetPlatformUsersByUserId(user.ID)
	if err != nil {
		return err
	}
	platformSet := sets.NewInt()
	for _, item := range platformUsers {
		platformSet.Insert(int(item.PlatformID))
	}

	// send topic
	if topic != "" {
		platformChannel, err := store.Database.GetPlatformChannelByFlag(topic)
		if err != nil {
			return err
		}
		if !platformSet.Has(int(platformChannel.PlatformID)) {
			return fmt.Errorf("topic %s not platform %d", topic, platformChannel.PlatformID)
		}
		platform, err := store.Database.GetPlatform(platformChannel.PlatformID)
		if err != nil {
			return err
		}

		if platform.Name == "" {
			return fmt.Errorf("empty platform user %s topic %s", uid, topic)
		}

		return PublishMessage(ctx, types.MessageSendEvent, types.Message{
			Platform: platform.Name,
			Topic:    topic,
			Payload:  payload,
		})
	}

	// send all
	platforms, err := store.Database.GetPlatforms()
	if err != nil {
		return err
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
			err = PublishMessage(ctx, types.MessageSendEvent, types.Message{
				Platform: platformName[item.PlatformID],
				Topic:    channelUser.ChannelFlag,
				Payload:  payload,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
