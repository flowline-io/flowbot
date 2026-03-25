package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/config"
)

var (
	agents         = make(map[string]config.Agent)
	loadOnceAgents = sync.Once{}
)

func loadAgents() {
	loadOnceAgents.Do(func() {
		for _, item := range config.App.Agents {
			agents[item.Name] = item
		}
	})
}

func AgentModelName(name string) string {
	loadAgents()
	a, ok := agents[name]
	if !ok || !a.Enabled {
		return ""
	}
	return a.Model
}

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

// ReactAgent creates a ReAct agent instance with tools
func ReactAgent(ctx context.Context, modelName string, tools []BaseTool) (*ReactAgentInstance, error) {
	client, err := ChatModel(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("chat model failed, %w", err)
	}

	return &ReactAgentInstance{
		client: client,
		tools:  tools,
	}, nil
}

// ReactAgentInstance represents an agent that can call tools
type ReactAgentInstance struct {
	client *GenaiClient
	tools  []BaseTool
}

// Generate runs the agent with messages
func (a *ReactAgentInstance) Generate(ctx context.Context, messages []*Message) (*Message, error) {
	return Generate(ctx, a.client, messages)
}

func LLMGenerate(ctx context.Context, modelName, prompt string) (string, error) {
	messages, err := BaseTemplate().Format(ctx, map[string]any{
		"content": prompt,
	})
	if err != nil {
		return "", fmt.Errorf("prompt format failed, %w", err)
	}

	client, err := ChatModel(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("chat model failed, %w", err)
	}

	resp, err := Generate(ctx, client, messages)
	if err != nil {
		return "", fmt.Errorf("llm generate failed, %w", err)
	}

	if resp == nil {
		return "", nil
	}

	return resp.Content, nil
}
