// Package platforms provides multi-platform integration for chat and messaging.
package platforms

import (
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type Caller struct {
	Action  protocol.Action
	Adapter protocol.Adapter
}

func (c *Caller) Do(req protocol.Request) protocol.Response {
	switch req.Action {
	case protocol.SendMessageAction:
		return c.Action.SendMessage(req)
	case protocol.UpdateMessageAction:
		return c.Action.UpdateMessage(req)
	case protocol.DeleteMessageAction:
		return c.Action.DeleteMessage(req)
	}
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("error action"))
}
