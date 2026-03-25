package agents

import (
	"context"
	"encoding/json"
)

// Tool interface for agent tools
type Tool interface {
	Name() string
	Description() string
	Parameters() *ParamsOneOf
	Execute(ctx context.Context, input string) (string, error)
}

// BaseTool is the base interface for all tools
type BaseTool interface {
	Info(ctx context.Context) (*ToolInfo, error)
	InvokableRun(ctx context.Context, input string) (string, error)
}

// ToolInfo contains tool metadata
type ToolInfo struct {
	Name        string
	Desc        string
	ParamsOneOf *ParamsOneOf
}

// InvokableTool is a tool that can be invoked
type InvokableTool interface {
	BaseTool
}

// FunctionTool implements InvokableTool for function-based tools
type FunctionTool struct {
	Name        string
	Description string
	Parameters  *ParamsOneOf
	Execute     func(ctx context.Context, input string) (string, error)
}

// Info returns tool metadata
func (f *FunctionTool) Info(ctx context.Context) (*ToolInfo, error) {
	return &ToolInfo{
		Name:        f.Name,
		Desc:        f.Description,
		ParamsOneOf: f.Parameters,
	}, nil
}

// InvokableRun executes the tool
func (f *FunctionTool) InvokableRun(ctx context.Context, input string) (string, error) {
	return f.Execute(ctx, input)
}

// ConvertFromString converts string input to map if needed
func ConvertFromString(input string) (map[string]any, error) {
	if input == "" {
		return make(map[string]any), nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(input), &result); err != nil {
		return nil, err
	}
	return result, nil
}
