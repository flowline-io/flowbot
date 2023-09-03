package platforms

import "github.com/flowline-io/flowbot/internal/types/protocol"

type Caller struct {
	Action protocol.Action
}

func (c *Caller) Do(req protocol.Request) protocol.Response {
	switch req.Action {
	case protocol.SendMessageAction:
		return c.Action.SendMessage(req)
	}
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}
