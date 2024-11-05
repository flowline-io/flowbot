package expression

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/flog"
)

var globalEnv = map[string]any{
	"debug": func(ctx context.Context, v any) any { // Methods defined on the struct become functions.
		flog.Info("[expr] Debug: %+v", v)
		return true
	},
}

func LoadEnv(name string, f any) {
	globalEnv[name] = f
}
