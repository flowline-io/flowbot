package server

import (
	"errors"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// send message
func onSendEvent() {
	event.On(event.SendEvent, func(data types.KV) error {
		//topic, ok := data.String("topic")
		//if !ok {
		//	return errors.New("error param topic")
		//}
		//topicUid, ok := data.Int64("topic_uid")
		//if !ok {
		//	return errors.New("error param topic_uid")
		//}
		//message, ok := data.String("message")
		//if !ok {
		//	return errors.New("error param message")
		//}
		//botSend(topic, types.Uid(topicUid), types.TextMsg{Text: message})
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
		uid := types.Uid(uidStr)
		if uid.IsZero() {
			return errors.New("error param uid")
		}

		sessionStore.Range(func(sid string, s *Session) bool {
			if s.uid == uid {
				s.queueOut(&ServerComMessage{
					//Code:    http.StatusOK,
					//Message: "",
					//Data:    data,
				})
			}
			return true
		})

		return nil
	})
}

func onPlatformMetaEvent() {
	event.On("meta.*", func(data types.KV) error {
		// todo
		flog.Info("%v", data)
		return nil
	})
}

func onPlatformMessageEvent() {
	event.On("message.*", func(data types.KV) error {
		// todo
		flog.Info("%v", data)

		var pe protocol.Event
		if e, ok := data.Any("event"); ok {
			if v, ok := e.(protocol.Event); ok {
				pe = v
			}
		}

		var caller *platforms.Caller
		if e, ok := data.Any("caller"); ok {
			if v, ok := e.(*platforms.Caller); ok {
				caller = v
			}
		}

		hookIncomingMessage(caller, pe)

		return nil
	})
}

func onPlatformNoticeEvent() {
	event.On("event.*", func(data types.KV) error {
		// todo
		flog.Info("%v", data)
		return nil
	})
}
