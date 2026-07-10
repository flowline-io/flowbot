// Package eval provides FakeModel-scripted harness evaluation scenarios.
package eval

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/tmc/langchaingo/llms"
)

// Expectation describes success criteria for one eval scenario.
type Expectation struct {
	// ExpectedTools lists tool names that must appear in order among executed tools.
	ExpectedTools []string
	// RequiredArgs maps tool name to required argument keys that must be non-empty.
	RequiredArgs map[string][]string
	// MaxSteps fails the scenario when the loop exceeds this step count (0 disables).
	MaxSteps int
	// RequireCompletion requires a non-error final assistant text response.
	RequireCompletion bool
}

// Metrics captures scored outcomes for one scenario run.
type Metrics struct {
	// ToolSelectionCorrect is true when expected tools were selected in order.
	ToolSelectionCorrect bool
	// ArgsValid is true when required tool arguments were present and non-empty.
	ArgsValid bool
	// StepCount is the number of assistant turns observed.
	StepCount int
	// Completed is true when the run finished with a final assistant message and no error.
	Completed bool
	// ToolsCalled lists tool names executed during the run.
	ToolsCalled []string
}

// Scenario is one FakeModel-driven harness evaluation case.
type Scenario struct {
	// Name identifies the scenario in table tests.
	Name string
	// Prompt is the user message.
	Prompt string
	// Scripts are FakeModel responses in order.
	Scripts []agentllm.ResponseScript
	// Tools are registered for the run.
	Tools []tool.Tool
	// Expect defines scoring criteria.
	Expect Expectation
}

// Score derives metrics from a completed agent run.
func Score(messages []msg.AgentMessage, expect Expectation, runErr error) Metrics {
	m := Metrics{ArgsValid: true}
	required := expect.RequiredArgs
	if required == nil {
		required = map[string][]string{}
	}

	for _, item := range messages {
		assistant, ok := item.(msg.AssistantMessage)
		if !ok {
			continue
		}
		scoreAssistant(&m, assistant, required, runErr)
	}
	if expect.RequireCompletion && runErr != nil {
		m.Completed = false
	}
	m.ToolSelectionCorrect = toolsMatch(expect.ExpectedTools, m.ToolsCalled)
	if expect.MaxSteps > 0 && m.StepCount > expect.MaxSteps {
		m.Completed = false
	}
	return m
}

func scoreAssistant(m *Metrics, assistant msg.AssistantMessage, required map[string][]string, runErr error) {
	m.StepCount++
	for _, call := range assistant.ToolCalls() {
		m.ToolsCalled = append(m.ToolsCalled, call.Name)
		if !argsValidForTool(call, required[call.Name]) {
			m.ArgsValid = false
		}
	}
	if len(assistant.ToolCalls()) == 0 && strings.TrimSpace(assistant.TextContent()) != "" && runErr == nil {
		m.Completed = true
	}
}

func argsValidForTool(call msg.ToolCallPart, keys []string) bool {
	if len(keys) == 0 {
		return true
	}
	args, err := parseArgs(call.Arguments)
	if err != nil {
		return false
	}
	for _, key := range keys {
		if strings.TrimSpace(fmt.Sprint(args[key])) == "" {
			return false
		}
	}
	return true
}

func toolsMatch(expected, called []string) bool {
	if len(expected) == 0 {
		return true
	}
	if len(called) < len(expected) {
		return false
	}
	idx := 0
	for _, name := range called {
		if name == expected[idx] {
			idx++
			if idx == len(expected) {
				return true
			}
		}
	}
	return false
}

func parseArgs(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := sonic.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// ToolCallScript builds a FakeModel script that requests one tool call.
func ToolCallScript(id, name, argsJSON string) agentllm.ResponseScript {
	return agentllm.ResponseScript{
		ToolCalls: []llms.ToolCall{{
			ID:   id,
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      name,
				Arguments: argsJSON,
			},
		}},
	}
}

// TextScript builds a FakeModel script that returns plain assistant text.
func TextScript(content string) agentllm.ResponseScript {
	return agentllm.ResponseScript{Content: content}
}
