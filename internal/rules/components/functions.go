package components

import (
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
	"strings"
	"sync"
)

var Functions = &FunctionsRegistry{}

type FunctionsRegistry struct {
	functions map[string]func(ctx types.RuleContext, msg types.RuleMsg)
	sync.RWMutex
}

func (x *FunctionsRegistry) Register(functionName string, f func(ctx types.RuleContext, msg types.RuleMsg)) {
	x.Lock()
	defer x.Unlock()
	if x.functions == nil {
		x.functions = make(map[string]func(ctx types.RuleContext, msg types.RuleMsg))
	}
	x.functions[functionName] = f
}

func (x *FunctionsRegistry) UnRegister(functionName string) {
	x.Lock()
	defer x.Unlock()
	if x.functions != nil {
		delete(x.functions, functionName)
	}
}

func (x *FunctionsRegistry) Get(functionName string) (func(ctx types.RuleContext, msg types.RuleMsg), bool) {
	x.RLock()
	defer x.RUnlock()
	if x.functions == nil {
		return nil, false
	}
	f, ok := x.functions[functionName]
	return f, ok
}

func (x *FunctionsRegistry) Names() []string {
	x.RLock()
	defer x.RUnlock()
	var keys = make([]string, 0, len(x.functions))
	for k := range x.functions {
		keys = append(keys, k)
	}
	return keys
}

type FunctionsNodeConfiguration struct {
	FunctionName string
	Params       map[string]interface{}
}

type FunctionsNode struct {
	Config  FunctionsNodeConfiguration
	HasVars bool
}

func (x *FunctionsNode) Type() string {
	return "flowbot/functions"
}

func (x *FunctionsNode) New() types.Node {
	return &FunctionsNode{Config: FunctionsNodeConfiguration{
		FunctionName: "test",
	}}
}

func (x *FunctionsNode) Init(_ types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)

	if strings.Contains(x.Config.FunctionName, "${") {
		x.HasVars = true
	}
	return err
}

func (x *FunctionsNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	funcName := x.getFunctionName(ctx, msg)
	if f, ok := Functions.Get(funcName); ok {
		// merge params to msg.data
		if len(x.Config.Params) > 0 {
			if msg.DataType == types.JSON {
				var data map[string]interface{}
				if msg.Data == "" {
					data = make(map[string]interface{})
				} else {
					err := sonic.Unmarshal(utils.StringToBytes(msg.Data), &data)
					if err != nil {
						ctx.TellFailure(msg, err)
						return
					}
				}
				for k, v := range x.Config.Params {
					data[k] = v
				}
				b, err := sonic.Marshal(data)
				if err != nil {
					ctx.TellFailure(msg, err)
					return
				}
				msg.Data = utils.BytesToString(b)
			}
		}
		f(ctx, msg)
	} else {
		ctx.TellFailure(msg, fmt.Errorf("can not found the function=%s", funcName))
	}
}

func (x *FunctionsNode) Destroy() {
}

func (x *FunctionsNode) getFunctionName(ctx types.RuleContext, msg types.RuleMsg) string {
	if x.HasVars {
		evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
		return str.ExecuteTemplate(x.Config.FunctionName, evn)
	} else {
		return x.Config.FunctionName
	}
}
