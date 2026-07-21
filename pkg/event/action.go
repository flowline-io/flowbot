// Package event provides event system definition and data event types.
package event

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
)

// SendMessage delivers a message payload to the user's channels, scoped to a specific topic
// when ctx.Topic is set, or broadcast to all channels the user is a member of.
func SendMessage(ctx types.Context, msg types.MsgPayload, pub message.Publisher) error {
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
	user, err := store.Database.GetUserByFlag(ctx.Context(), ctx.AsUser.String())
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	platformUsers, err := store.Database.GetPlatformUsersByUserId(ctx.Context(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get platform users: %w", err)
	}
	platformSet := sets.NewInt()
	for _, item := range platformUsers {
		platformSet.Insert(int(item.PlatformID))
	}

	if ctx.Topic != "" {
		return sendToTopic(ctx, payload, platformSet, pub)
	}

	return sendToAll(ctx, payload, platformUsers, pub)
}

func sendToTopic(ctx types.Context, payload types.EventPayload, platformSet sets.Int, pub message.Publisher) error {
	platformChannel, err := store.Database.GetPlatformChannelByFlag(ctx.Context(), ctx.Topic)
	if err != nil {
		return fmt.Errorf("failed to get platform channel: %w", err)
	}
	if !platformSet.Has(int(platformChannel.PlatformID)) {
		return fmt.Errorf("topic %s does not belong to any of the user's platforms (got platform %d)", ctx.Topic, platformChannel.PlatformID)
	}
	platform, err := store.Database.GetPlatform(ctx.Context(), platformChannel.PlatformID)
	if err != nil {
		return fmt.Errorf("failed to get platform: %w", err)
	}

	if platform.Name == "" {
		return fmt.Errorf("empty platform user %s topic %s", ctx.AsUser, ctx.Topic)
	}

	return publishWith(ctx.Context(), pub, types.MessageSendEvent, types.Message{
		Platform: platform.Name,
		Topic:    ctx.Topic,
		Payload:  payload,
	})
}

func sendToAll(ctx types.Context, payload types.EventPayload, platformUsers []*gen.PlatformUser, pub message.Publisher) error {
	platforms, err := store.Database.GetPlatforms(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to get platforms: %w", err)
	}
	platformName := make(map[int64]string, len(platforms))
	for _, item := range platforms {
		platformName[item.ID] = item.Name
	}

	userFlags := make([]string, 0, len(platformUsers))
	for _, item := range platformUsers {
		userFlags = append(userFlags, item.Flag)
	}
	channelUsersByFlag, err := store.Database.GetPlatformChannelUsersByUserFlags(ctx.Context(), userFlags)
	if err != nil {
		return fmt.Errorf("failed to get platform channel users: %w", err)
	}
	channelUserMap := make(map[string][]*gen.PlatformChannelUser, len(userFlags))
	for _, cu := range channelUsersByFlag {
		channelUserMap[cu.UserFlag] = append(channelUserMap[cu.UserFlag], cu)
	}

	for _, item := range platformUsers {
		if platformName[item.PlatformID] == "" {
			continue
		}

		for _, channelUser := range channelUserMap[item.Flag] {
			err = publishWith(ctx.Context(), pub, types.MessageSendEvent, types.Message{
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

// BotEventFire publishes a bot-run event to the event bus, triggering any subscribed pipeline handlers.
func BotEventFire(ctx types.Context, eventName string, param types.KV, pub message.Publisher) error {
	return publishWith(ctx.Context(), pub, types.BotRunEvent, types.BotEvent{
		EventName: eventName,
		Uid:       ctx.AsUser.String(),
		Topic:     ctx.Topic,
		Param:     param,
	})
}

// publishWith publishes a message to the given topic using the provided publisher, with OpenTelemetry tracing.
func publishWith(ctx context.Context, pub message.Publisher, topic string, payload any) error {
	msg, err := NewMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to new message: %w", err)
	}

	ctx, publishSpan := trace.StartSpan(ctx, "event.publish "+topic,
		attribute.String("messaging.destination", topic),
		attribute.String("messaging.message.id", msg.UUID),
	)
	defer publishSpan.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		msg.Metadata.Set(k, v)
	}
	msg.Metadata.Set("x-otel-topic", topic)

	err = pub.Publish(topic, msg)
	if err != nil {
		publishSpan.RecordError(err)
		publishSpan.SetStatus(codes.Error, err.Error())
	}
	return err
}
