package platforms

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
)

type Caller struct {
	Action  protocol.Action
	Adapter protocol.Adapter
}

func (c *Caller) Do(req protocol.Request) protocol.Response {
	switch req.Action {
	case protocol.SendMessageAction:
		return c.Action.SendMessage(req)
	}
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func MessageConvert(data any) protocol.Message {
	d, ok := data.(types.MsgPayload)
	if !ok {
		return nil
	}
	switch v := d.(type) {
	case types.TextMsg:
		return protocol.Message{
			protocol.Text(v.Text),
		}
	case types.InfoMsg:
		_, model := v.Convert()
		txt := ""
		if v, ok := model.(map[string]any); ok {
			txt, _ = types.KV(v).String("txt")
		}
		return protocol.Message{
			protocol.Text(v.Title),
			protocol.Text(txt),
		}
	case types.LinkMsg:
		return protocol.Message{
			protocol.Text(v.Title),
			protocol.Url(v.Url),
		}
	}
	return nil
}
