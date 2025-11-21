package dev

import (
	"context"
	"time"

	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/tool"
	"github.com/flowline-io/flowbot/version"
)

var toolRules = []tool.Rule{
	// getCurrentTime
	func(ctx types.Context) (llmTool.InvokableTool, error) {
		// params
		type Params struct{}

		// func
		Func := func(_ context.Context, params *Params) (string, error) {
			return time.Now().Format(time.DateTime), nil
		}

		return utils.InferTool(
			"getCurrentTime",
			"Get the current time",
			Func)
	},
	// getCurrentVersion
	func(ctx types.Context) (llmTool.InvokableTool, error) {
		// params
		type Params struct{}

		// func
		Func := func(_ context.Context, params *Params) (string, error) {
			return version.Buildtags, nil
		}

		return utils.InferTool(
			"getCurrentVersion",
			"Get the current version",
			Func)
	},
}
