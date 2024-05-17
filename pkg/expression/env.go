package expression

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/flog"
)

var Env = map[string]any{
	"debug": func(ctx context.Context, v any) any { // Methods defined on the struct become functions.
		flog.Info("[expr] Debug: %+v", v)
		return true
	},
}

type Env2 struct {
	Ctx   context.Context `expr:"ctx"`
	Input map[string]any  `expr:"input"`
	Lib   map[string]any  `expr:"lib"`
}

var Lib = map[string]any{
	"toInt": func(v any) int { return v.(int) + 1000 },
}

func LoadLib(name string, f any) {
	Env[name] = f
}

func NewEnv(ctx context.Context, val map[string]any) map[string]any {
	Env["ctx"] = ctx
	for k, v := range val {
		Env[k] = v
	}
	return Env
}
