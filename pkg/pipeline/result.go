package pipeline

import (
	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// StepResultFromInvoke converts an ability invoke result into a step output map for templates.
func StepResultFromInvoke(res *ability.InvokeResult) map[string]any {
	if res == nil {
		return map[string]any{}
	}
	if res.Data == nil {
		return map[string]any{}
	}
	if m, ok := res.Data.(map[string]any); ok {
		return m
	}
	dataJSON, err := sonic.Marshal(res.Data)
	if err != nil {
		return map[string]any{"result": res.Data}
	}
	var stepResult any
	if err := sonic.Unmarshal(dataJSON, &stepResult); err != nil {
		return map[string]any{"result": res.Data}
	}
	if m, ok := stepResult.(map[string]any); ok {
		return m
	}
	return map[string]any{"items": stepResult}
}
