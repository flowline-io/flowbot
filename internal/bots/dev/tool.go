package dev

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/tool"
	"github.com/flowline-io/flowbot/version"
)

var toolRules = []tool.Rule{
	func(ctx types.Context) (agents.InvokableTool, error) {
		return &agents.FunctionTool{
			Name:        "getCurrentTime",
			Description: "Get the current time",
			Parameters: &agents.ParamsOneOf{
				OneOf: []agents.Schema{},
			},
			Execute: func(_ context.Context, input string) (string, error) {
				return time.Now().Format(time.DateTime), nil
			},
		}, nil
	},
	func(ctx types.Context) (agents.InvokableTool, error) {
		return &agents.FunctionTool{
			Name:        "getCurrentVersion",
			Description: "Get the current version",
			Parameters: &agents.ParamsOneOf{
				OneOf: []agents.Schema{},
			},
			Execute: func(_ context.Context, input string) (string, error) {
				return version.Buildtags, nil
			},
		}, nil
	},
}
