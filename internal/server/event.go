package server

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	json "github.com/json-iterator/go"
)

// send message
func onSendEvent() {
	// todo SendEvent

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
}

// push instruct
func onPushInstruct() {
	// todo InstructEvent

	//uidStr, ok := data.String("uid")
	//if !ok {
	//	return errors.New("error param uid")
	//}
	//uid := types.Uid(uidStr)
	//if uid.IsZero() {
	//	return errors.New("error param uid")
	//}
	//
	//sessionStore.Range(func(sid string, s *Session) bool {
	//	if s.uid == uid {
	//		// todo send message
	//		//s.queueOut(&ServerComMessage{
	//		//	//Code:    http.StatusOK,
	//		//	//Message: "",
	//		//	//Data:    data,
	//		//})
	//	}
	//	return true
	//})
}

func onPlatformMetaEvent() {
	// todo "meta.*"
}

func onPlatformNoticeEvent() {
	// todo "event.*"
}

func onPlatformMessageEventHandler(msg *message.Message) error {
	flog.Debug("on message event %+v", msg)

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

	var caller *platforms.Caller
	// todo make caller

	hookIncomingMessage(caller, pe)

	return nil
}
