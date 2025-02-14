package dev

import (
	"context"
	"fmt"
	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/tool"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers/lobehub"
	"github.com/flowline-io/flowbot/pkg/types"
)

var toolRules = []tool.Rule{
	// getCurrentTime
	func(ctx types.Context) (llmTool.InvokableTool, error) {
		// params
		type Params struct {
			Format string `json:"format" jsonschema:"description=time layout format, default: RFC1123Z"`
		}

		// func
		Func := func(_ context.Context, params *Params) (string, error) {
			if params.Format == "" {
				params.Format = time.RFC1123Z
			}
			return time.Now().Format(params.Format), nil
		}

		return utils.InferTool(
			"getCurrentTimeToolId",
			"Get the current time",
			Func)
	},
	// getWebsiteContent
	func(ctx types.Context) (llmTool.InvokableTool, error) {
		// params
		type Params struct {
			Url string `json:"url" jsonschema:"description=The URL to fetch"`
		}

		// func
		Func := func(_ context.Context, params *Params) (string, error) {
			if params.Url == "" {
				return "", fmt.Errorf("empty url")
			}

			resp, err := lobehub.NewLobehub().WebCrawler(params.Url)
			if err != nil {
				return "", fmt.Errorf("get website content failed, %w", err)
			}

			return resp.Content, nil
		}

		return utils.InferTool(
			"getWebsiteContentToolId",
			"Extract web content",
			Func)
	},
}
