package workflow

import (
	"encoding/json"
	"errors"
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"net/http"
)

const Name = "workflow"

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
		flog.Info("bot %s disabled", Name)
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
		pipelineRules,
	}
}

func (bot) Webapp() func(rw http.ResponseWriter, req *http.Request) {
	return webapp
}

func (b bot) Command(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	return bots.RunCommand(commandRules, ctx, content)
}

func (b bot) Pipeline(ctx types.Context, head types.KV, content interface{}, operate types.PipelineOperate) (types.MsgPayload, string, int, error) {
	return bots.RunPipeline(pipelineRules, ctx, head, content, operate)
}

func (bot) Webservice() *restful.WebService {
	return bots.Webservice(Name, serviceVersion, webserviceRules)
}
