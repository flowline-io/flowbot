package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/chatbot"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"go.uber.org/fx"
)

// init event
func handleEvents(lc fx.Lifecycle, router *message.Router, subscriber message.Subscriber) error {
	router.AddNoPublisherHandler(
		"onMessageChannelEvent",
		protocol.MessageChannelEvent,
		subscriber,
		onPlatformMessageEventHandler,
	)
	router.AddNoPublisherHandler(
		"onMessageDirectEvent",
		protocol.MessageDirectEvent,
		subscriber,
		onPlatformMessageEventHandler,
	)
	router.AddNoPublisherHandler(
		"onMessageSendEventHandler",
		types.MessageSendEvent,
		subscriber,
		onMessageSendEventHandler,
	)
	router.AddNoPublisherHandler(
		"onInstructPushEventHandler",
		types.InstructPushEvent,
		subscriber,
		onInstructPushEventHandler,
	)
	router.AddNoPublisherHandler(
		"onBotRunEventHandler",
		types.BotRunEvent,
		subscriber,
		onBotRunEventHandler,
	)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := router.Run(context.Background()); err != nil {
					flog.Error(err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return router.Close()
		},
	})

	return nil
}

// send message
func onMessageSendEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	var pe types.Message
	err := sonic.Unmarshal(msg.Payload, &pe)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// ignore send
	if pe.Platform == "" || pe.Topic == "" {
		return nil
	}

	msgPayload := types.ToPayload(pe.Payload.Typ, pe.Payload.Src)

	caller, err := platforms.GetCaller(pe.Platform)
	if err != nil {
		return fmt.Errorf("failed to get caller: %w", err)
	}

	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   pe.Topic,
			"message": caller.Adapter.MessageConvert(msgPayload),
		},
	})

	if resp.Status != protocol.Success {
		return fmt.Errorf("failed to send message: %s", resp.Message)
	}

	return nil
}

// push instruct
func onInstructPushEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	return nil
}

// run bot event
func onBotRunEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	var be types.BotEvent
	err := sonic.Unmarshal(msg.Payload, &be)
	if err != nil {
		return fmt.Errorf("failed to unmarshal bot event: %w", err)
	}

	ctx := types.Context{
		AsUser:      types.Uid(be.Uid),
		Topic:       be.Topic,
		EventRuleId: be.EventName,
	}
	ctx.SetTimeout(10 * time.Minute)

	for name, handle := range chatbot.List() {
		err = handle.Event(ctx, be.Param)
		if err != nil {
			return fmt.Errorf("bot %s event %s error %w", name, be.EventName, err)
		}
	}

	return nil
}

func onPlatformMessageEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	var pe protocol.Event
	err := sonic.Unmarshal(msg.Payload, &pe)
	if err != nil {
		return fmt.Errorf("failed to unmarshal platform message event: %w", err)
	}

	data, err := sonic.Marshal(pe.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal platform message event: %w", err)
	}
	var v protocol.MessageEventData
	err = sonic.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal platform message event: %w", err)
	}
	pe.Data = v

	caller, err := platforms.GetCaller(v.Self.Platform)
	if err != nil {
		return fmt.Errorf("failed to get caller: %w", err)
	}

	// update online status
	onlineStatus(pe)
	// check grp or p2p
	if strings.HasSuffix(pe.DetailType, ".direct") {
		directIncomingMessage(caller, pe)
	}
	if strings.HasSuffix(pe.DetailType, ".group") {
		groupIncomingMessage(caller, pe)
	}

	return nil
}
