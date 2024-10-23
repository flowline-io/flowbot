package platforms

import (
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
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
	case types.LinkMsg:
		return protocol.Message{
			protocol.Text(v.Title),
			protocol.Url(v.Url),
		}
	default:
		s, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			flog.Error(err)
			return nil
		}

		return protocol.Message{
			protocol.Text(utils.BytesToString(s)),
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
			return fmt.Errorf("failed to create platform %s, %w", name, err)
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
