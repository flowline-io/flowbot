package okr

import (
	"encoding/json"
	"errors"
	"github.com/emicklei/go-restful/v3"
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"net/http"
)

const Name = "okr"

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

func (b bot) Rules() []interface{} {
	return []interface{}{
		commandRules,
		formRules,
		pageRules,
	}
}

func (bot) Webapp() func(rw http.ResponseWriter, req *http.Request) {
	return webapp
}

func (bot) Webservice() *restful.WebService {
	return bots.Webservice(Name, serviceVersion, webserviceRules)
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Form(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return bots.RunForm(formRules, ctx, values)
}

func (b bot) Page(ctx types.Context, flag string) (string, error) {
	return bots.RunPage(pageRules, ctx, flag)
}
