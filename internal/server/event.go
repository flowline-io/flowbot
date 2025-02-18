package server

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	jsoniter "github.com/json-iterator/go"
)

// send message
func onMessageSendEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	var pe types.Message
	err := jsoniter.Unmarshal(msg.Payload, &pe)
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
	err := jsoniter.Unmarshal(msg.Payload, &be)
	if err != nil {
		return fmt.Errorf("failed to unmarshal bot event: %w", err)
	}

	ctx := types.Context{
		AsUser:  types.Uid(be.Uid),
		Topic:   be.Topic,
		EventId: be.EventName,
	}
	ctx.SetTimeout(10 * time.Minute)

	for name, handle := range bots.List() {
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
	err := jsoniter.Unmarshal(msg.Payload, &pe)
	if err != nil {
		return fmt.Errorf("failed to unmarshal platform message event: %w", err)
	}

	data, err := jsoniter.Marshal(pe.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal platform message event: %w", err)
	}
	var v protocol.MessageEventData
	err = jsoniter.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal platform message event: %w", err)
	}
	pe.Data = v

	caller, err := platforms.GetCaller(v.Self.Platform)
	if err != nil {
		return fmt.Errorf("failed to get caller: %w", err)
	}

	hookIncomingMessage(caller, pe)

	return nil
}
