package server

import (
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	json "github.com/json-iterator/go"
)

// send message
func onMessageSendEventHandler(msg *message.Message) error {
	flog.Debug("[event] on event %+v %+v", msg.UUID, msg.Metadata)

	var pe types.Message
	err := json.Unmarshal(msg.Payload, &pe)
	if err != nil {
		return err
	}

	if pe.Platform == "" {
		return errors.New("error param platform")
	}
	if pe.Topic == "" {
		return errors.New("error param topic")
	}

	caller, err := platforms.GetCaller(pe.Platform)
	if err != nil {
		return err
	}

	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   pe.Topic,
			"message": caller.Adapter.MessageConvert(pe.Payload),
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
	err := json.Unmarshal(msg.Payload, &pe)
	if err != nil {
		return err
	}

	data, err := json.Marshal(pe.Data)
	if err != nil {
		return err
	}
	var v protocol.MessageEventData
	err = json.Unmarshal(data, &v)
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
