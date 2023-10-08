package pocket

import (
	"encoding/json"
	"errors"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"github.com/flowline-io/flowbot/pkg/utils"
	"gorm.io/gorm"
)

const Name = "pocket"

var handler bot
var Config configType

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

	if err := json.Unmarshal(jsonconf, &Config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !Config.Enabled {
		flog.Info("bot %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (bot) IsReady() bool {
	return handler.initialized
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Cron(send types.SendFunc) (*cron.Ruleset, error) {
	return bots.RunCron(cronRules, Name, send)
}

func (b bot) Input(ctx types.Context, _ types.KV, content interface{}) (types.MsgPayload, error) {
	//text, err := drafty.PlainText(content)
	//if err != nil {
	//	return nil, err
	//}
	text := content.(string) // fixme to plain text

	if utils.IsUrl(text) {
		url := text
		oauth, err := store.Chatbot.OAuthGet(ctx.AsUser, ctx.Original, Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			flog.Error(err)
		}
		if oauth.Token == "" {
			return types.TextMsg{Text: "App is unauthorized"}, nil
		}

		key, _ := providers.GetConfig(pocket.ID, pocket.ClientIdKey)
		provider := pocket.NewPocket(key.String(), "", "", oauth.Token)
		_, err = provider.Add(url)
		if err != nil {
			flog.Error(err)
			return types.TextMsg{Text: "Add error"}, nil
		}

		return types.TextMsg{Text: "ok"}, nil
	}

	return nil, nil
}
