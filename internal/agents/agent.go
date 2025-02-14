package agents

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func ReactAgent(ctx context.Context, tools []tool.BaseTool) (*react.Agent, error) {
	llm, err := ChatModel(ctx, ToolcallModel())
	if err != nil {
		return nil, fmt.Errorf("chat model failed, %w", err)
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: llm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("react agent failed, %w", err)
	}

	return agent, nil
}
