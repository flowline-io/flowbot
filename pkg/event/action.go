package event

import "github.com/flowline-io/flowbot/internal/types"

func SendMessage(uid, topic string, msg types.MsgPayload) error {
	return PublishMessage(types.MessageSendEvent, types.Message{
		Platform: "", // todo
		Topic:    topic,
		Payload:  msg,
	})
}
