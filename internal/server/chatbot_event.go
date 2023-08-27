package server

import (
	"errors"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/event"
	"net/http"
)

// send message
func onSendEvent() {
	event.On(event.SendEvent, func(data types.KV) error {
		topic, ok := data.String("topic")
		if !ok {
			return errors.New("error param topic")
		}
		topicUid, ok := data.Int64("topic_uid")
		if !ok {
			return errors.New("error param topic_uid")
		}
		message, ok := data.String("message")
		if !ok {
			return errors.New("error param message")
		}
		botSend(topic, types.Uid(topicUid), types.TextMsg{Text: message})
		return nil
	})
}

// push instruct
func onPushInstruct() {
	event.On(event.InstructEvent, func(data types.KV) error {
		uidStr, ok := data.String("uid")
		if !ok {
			return errors.New("error param uid")
		}
		uid := types.ParseUserId(uidStr)
		if uid.IsZero() {
			return errors.New("error param uid")
		}

		sessionStore.Range(func(sid string, s *Session) bool {
			if s.uid == uid {
				s.queueOutExtra(&types.ServerComMessage{
					Code:    http.StatusOK,
					Message: "",
					Data:    data,
				})
			}
			return true
		})

		return nil
	})
}
