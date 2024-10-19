package dev

import (
	"github.com/flowline-io/flowbot/internal/ruleset/langchain"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/tmc/langchaingo/llms"
	"time"
)

const (
	getCurrentTimeToolId = "getCurrentTime"
	getUrlContentToolId  = "getUrlContent"
)

var langchainRules = []langchain.Rule{
	{
		Id: getCurrentTimeToolId,
		Tool: llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        getCurrentTimeToolId,
				Description: "Get the current time",
			},
		},
		Execute: func(ctx types.Context, args types.KV) (string, error) {
			return time.Now().Format(time.RFC1123Z), nil
		},
	},
	{
		Id: getUrlContentToolId,
		Tool: llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        getUrlContentToolId,
				Description: "Get the website content of url",
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
			return "<html><title>test</title></html>", nil
		},
	},
}
