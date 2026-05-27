package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
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

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := router.Run(context.Background()); err != nil {
					flog.Error(err)
				}
			}()

			return nil
		},
		OnStop: func(_ context.Context) error {
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

func onPlatformMessageEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	var raw struct {
		Id         string             `json:"id"`
		Time       int64              `json:"time"`
		Type       protocol.EventType `json:"type"`
		DetailType string             `json:"detail_type"`
		Data       json.RawMessage    `json:"data"`
	}
	err := sonic.Unmarshal(msg.Payload, &raw)
	if err != nil {
		return fmt.Errorf("failed to unmarshal platform message event: %w", err)
	}

	var v protocol.MessageEventData
	err = sonic.Unmarshal(raw.Data, &v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal platform message event data: %w", err)
	}

	pe := protocol.Event{
		Id:         raw.Id,
		Time:       raw.Time,
		Type:       raw.Type,
		DetailType: raw.DetailType,
		Data:       v,
	}

	caller, err := platforms.GetCaller(v.Self.Platform)
	if err != nil {
		return fmt.Errorf("failed to get caller: %w", err)
	}

	// update online status
	onlineStatus(pe)
	// check grp or p2p
	if strings.HasSuffix(pe.DetailType, ".direct") {
		directIncomingMessage(msg.Context(), caller, pe)
	}
	if strings.HasSuffix(pe.DetailType, ".group") {
		groupIncomingMessage(msg.Context(), caller, pe)
	}

	return nil
}
