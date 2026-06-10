package tool

import (
	"github.com/tmc/langchaingo/llms"
)

// BuildLLMTools converts registered tools into langchaingo tool definitions.
func BuildLLMTools(tools []Tool) []llms.Tool {
	result := make([]llms.Tool, 0, len(tools))
	for _, t := range tools {
		result = append(result, llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return result
}
