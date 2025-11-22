package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/flowline-io/flowbot/pkg/config"
)

var (
	agents         = make(map[string]config.Agent)
	loadOnceAgents = sync.Once{}
)

// loadAgents loads and caches all agent configurations.
func loadAgents() {
	loadOnceAgents.Do(func() {
		for _, item := range config.App.Agents {
			agents[item.Name] = item
		}
	})
}

// AgentModelName returns the model name for the specified agent.
func AgentModelName(name string) string {
	loadAgents()
	a, ok := agents[name]
	if !ok || !a.Enabled {
		return ""
	}
	return a.Model
}

// AgentEnabled checks if the specified agent is enabled.
func AgentEnabled(name string) bool {
	loadAgents()
	a, ok := agents[name]
	if !ok || !a.Enabled {
		return false
	}
	if a.Model == "" {
		return false
	}
	return true
}

// ReactAgent creates a React agent instance.
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

// LLMGenerate generates a text response using the specified model.
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
