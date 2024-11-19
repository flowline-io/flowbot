package dev

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers/lobehub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/langchain"
	"github.com/tmc/langchaingo/llms"
)

const (
	getCurrentTimeToolId    = "getCurrentTime"
	getWebsiteContentToolId = "getWebsiteContent"
)

var langchainRules = []langchain.Rule{
	{
		Id: getCurrentTimeToolId,
		Tool: llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        getCurrentTimeToolId,
				Description: "Get the current time",
				Parameters: map[string]any{
					"type": "format",
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "time layout format, default: RFC1123Z",
						},
					},
					"required": []string{},
				},
			},
		},
		Execute: func(ctx types.Context, args types.KV) (string, error) {
			return time.Now().Format(time.RFC1123Z), nil
		},
	},
	{
		Id: getWebsiteContentToolId,
		Tool: llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        getWebsiteContentToolId,
				Description: "Extract web content",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "The URL to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		},
		Execute: func(ctx types.Context, args types.KV) (string, error) {
			url, _ := args.String("url")
			if url == "" {
				return "", fmt.Errorf("empty url")
			}

			resp, err := lobehub.NewLobehub().WebCrawler(url)
			if err != nil {
				return "", fmt.Errorf("get website content failed, %w", err)
			}

			return resp.Content, nil
		},
	},
}
