package platforms

import (
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"gorm.io/gorm"
)

var callers = make(map[string]*Caller)

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
		return protocol.Message{
			protocol.Text("error message payload"),
		}
	}
	switch v := d.(type) {
	case types.TextMsg:
		return protocol.Message{
			protocol.Text(v.Text),
		}
	case types.InfoMsg:
		_, info := v.Convert()
		txt := ""
		if kv, ok := info.(map[string]any); ok {
			txt, _ = types.KV(kv).String("txt")
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
	default:
		return protocol.Message{
			protocol.Text(fmt.Sprintf("error message type %+v", d)),
		}
	}
}

func PlatformRegister(name string, caller *Caller) error {
	_, err := store.Database.GetPlatformByName(name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_, err = store.Database.CreatePlatform(&model.Platform{
			Name: name,
		})
		if err != nil {
			return err
		}
	}
	callers[name] = caller
	return nil
}

func GetCaller(name string) (*Caller, error) {
	if c, ok := callers[name]; ok {
		return c, nil
	}
	return nil, errors.New("caller not found")
}
