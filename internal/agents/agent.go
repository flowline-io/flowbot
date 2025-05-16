package agents

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/flowline-io/flowbot/pkg/config"
	"sync"
)

const (
	AgentChat              = "chat"
	AgentReact             = "react"
	AgentRepoReviewComment = "repo-review-comment"
	AgentNewsSummary       = "news-summary"
	AgentBillClassify      = "bill-classify"
	AgentExtractTags       = "extract-tags"
	AgentSimilarTags       = "similar-tags"
)

var agents = make(map[string]config.Agent)
var loadOnceAgents = sync.Once{}

func AgentModelName(name string) string {
	loadOnceAgents.Do(func() {
		for _, item := range config.App.Agents {
			agents[item.Name] = item
		}
	})
	a, ok := agents[name]
	if !ok || a.Enabled == false {
		return ""
	}
	return a.Model
}

func ReactAgent(ctx context.Context, modelName string, tools []tool.BaseTool) (*react.Agent, error) {
	llm, err := ChatModel(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("chat model failed, %w", err)
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: llm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("react agent failed, %w", err)
	}

	return agent, nil
}

func LLMGenerate(ctx context.Context, modelName, prompt string) (string, error) {
	messages, err := BaseTemplate().Format(ctx, map[string]any{
		"content": prompt,
	})

	if err != nil {
		return "", fmt.Errorf("prompt format failed, %w", err)
	}

	llm, err := ChatModel(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("chat model failed, %w", err)
	}

	resp, err := Generate(ctx, llm, messages)
	if err != nil {
		return "", fmt.Errorf("llm generate failed, %w", err)
	}

	if resp == nil {
		return "", nil
	}

	return resp.Content, nil
}
