package rules

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
)

func NewConfig() (types.Config, error) {
	conf := rulego.NewConfig()

	// debug
	conf.OnDebug = func(chainId, flowType string, nodeId string, msg types.RuleMsg, relationType string, err error) {
		log := ""
		switch flowType {
		case types.In:
			log = fmt.Sprintf("[rule] %s: ----> (node: %s), message: %+v error: %v", chainId, nodeId, msg, err)
		case types.Out:
			log = fmt.Sprintf("[rule] %s: (node: %s) --%s-->, message: %+v error: %v", chainId, nodeId, relationType, msg, err)
		default:
			log = fmt.Sprintf("[rule] %s %s %s %s, message: %+v error: %v", chainId, flowType, nodeId, relationType, msg, err)
		}

		if err != nil {
			flog.Warn(log)
		} else {
			flog.Info(log)
		}
	}

	// parser
	conf.Parser = &DslParser{}

	// logger
	conf.Logger = flog.RulegoLogger

	// global properties
	metadata := types.NewProperties()
	metadata.PutValue("app", "flowbot")
	metadata.PutValue("version", version.Buildtags)
	metadata.PutValue("build", version.Buildstamp)
	conf.Properties = metadata

	return conf, nil
}
