package url

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/utils"
	"gorm.io/gorm"
	"strings"
)

const Name = "url"

var handler bot

func init() {
	bots.Register(Name, &handler)
}

type bot struct {
	initialized bool
	bots.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func (bot) Init(jsonconf json.RawMessage) error {

	// Check if the handler is already initialized
	if handler.initialized {
		return errors.New("already initialized")
	}

	var config configType
	if err := json.Unmarshal(jsonconf, &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !config.Enabled {
		logs.Info.Printf("bot %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (bot) IsReady() bool {
	return handler.initialized
}

func (b bot) Input(_ types.Context, _ types.KV, content interface{}) (types.MsgPayload, error) {
	text := types.ExtractText(content)
	if utils.IsUrl(text) {
		url, err := store.Chatbot.UrlGetByUrl(text)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return types.TextMsg{Text: "query url error"}, nil
		}
		if url.ID > 0 {
			return types.LinkMsg{Url: fmt.Sprintf("%s/u/%s", types.AppUrl(), url.Flag)}, nil
		}
		flag := strings.ToLower(types.Id().String())
		err = store.Chatbot.UrlCreate(model.Url{
			Flag:  flag,
			URL:   text,
			State: model.UrlStateEnable,
		})
		if err != nil {
			return types.TextMsg{Text: "create error"}, nil
		}
		return types.LinkMsg{Url: fmt.Sprintf("%s/u/%s", types.AppUrl(), flag)}, nil
	} else {
		url, err := store.Chatbot.UrlGetByFlag(text)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return types.TextMsg{Text: "query url error"}, nil
		}
		if url.ID > 0 {
			return types.LinkMsg{Url: url.URL}, nil
		}
		return types.TextMsg{Text: "empty"}, nil
	}
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}
