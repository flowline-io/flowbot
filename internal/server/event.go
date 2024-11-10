package server

import (
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
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
		return err
	}

	// ignore send
	if pe.Platform == "" || pe.Topic == "" {
		return nil
	}

	msgPayload := types.ToPayload(pe.Payload.Typ, pe.Payload.Src)

	caller, err := platforms.GetCaller(pe.Platform)
	if err != nil {
		return err
	}

	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   pe.Topic,
			"message": caller.Adapter.MessageConvert(msgPayload),
		},
	})

	if resp.Status != protocol.Success {
		return errors.New(resp.Message)
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

	var pe protocol.Event
	err := jsoniter.Unmarshal(msg.Payload, &pe)
	if err != nil {
		return err
	}

	data, err := jsoniter.Marshal(pe.Data)
	if err != nil {
		return err
	}
	var v protocol.MessageEventData
	err = jsoniter.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	pe.Data = v

	caller, err := platforms.GetCaller(v.Self.Platform)
	if err != nil {
		return err
	}

	hookIncomingMessage(caller, pe)

	return nil
}
