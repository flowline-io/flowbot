// Package eval provides FakeModel-scripted harness evaluation scenarios and scoring.
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/tmc/langchaingo/llms"
)

// Expectation describes success criteria for one eval scenario.
type Expectation struct {
	// RequiredTools lists tool names that must each appear at least once (hard coverage).
	RequiredTools []string
	// ForbiddenTools lists tool names that must not appear (hard).
	ForbiddenTools []string
	// ExpectedTools lists tool names that should appear in order among executed tools.
	// Soft by default; hard only when StrictToolOrder is true.
	ExpectedTools []string
	// StrictToolOrder makes ExpectedTools order a hard gate.
	StrictToolOrder bool
	// RequiredArgs maps tool name to required argument keys that must be non-empty.
	RequiredArgs map[string][]string
	// MaxSteps bounds assistant turns (0 disables). Hard-fails unless SoftMaxSteps.
	MaxSteps int
	// SoftMaxSteps records MaxSteps overflow without failing the hard gate (capability live).
	SoftMaxSteps bool
	// RequireCompletion requires a non-error final assistant text response.
	RequireCompletion bool
	// Outcome holds additional outcome assertions (files, final text).
	Outcome OutcomeAsserts
}

// OutcomeAsserts describes environment / proxy outcome checks.
type OutcomeAsserts struct {
	// FinalTextContains requires each substring in the last assistant text.
	FinalTextContains []string
	// Files asserts workspace file outcomes (true outcome when tools wrote them).
	Files []FileAssert
}

// FileAssert checks one workspace-relative file after the run.
type FileAssert struct {
	// Path is relative to WorkspaceRoot.
	Path string
	// Contains requires the file to exist and include this substring.
	Contains string
	// Equals, when non-empty, requires exact file content match.
	Equals string
}

// Metrics captures scored outcomes for one scenario run.
type Metrics struct {
	// ToolSelectionCorrect is true when ExpectedTools appear in order (soft metric).
	ToolSelectionCorrect bool
	// RequiredToolsCovered is true when every RequiredTools entry was called.
	RequiredToolsCovered bool
	// ForbiddenToolsClear is true when no ForbiddenTools entry was called.
	ForbiddenToolsClear bool
	// ArgsValid is true when required tool arguments were present and non-empty.
	ArgsValid bool
	// OutcomeOK is true when OutcomeAsserts passed (or none were set).
	OutcomeOK bool
	// StepCount is the number of assistant turns observed.
	StepCount int
	// Completed is true when the run finished with a final assistant message and no error.
	Completed bool
	// ToolsCalled lists tool names executed during the run.
	ToolsCalled []string
	// FinalText is the last non-empty assistant text content.
	FinalText string
	// StepsWithinLimit is true when MaxSteps is 0 or StepCount <= MaxSteps.
	StepsWithinLimit bool
	// DurationMs is wall time for the scenario run.
	DurationMs int64
	// TotalTokens sums assistant Usage.TotalTokens when reported.
	TotalTokens int
	// Passed is the hard CI gate (required/forbidden/args/outcome/completion/max steps; order only if strict).
	Passed bool
}

// Scenario is one FakeModel-driven harness evaluation case.
type Scenario struct {
	// Name identifies the scenario in table tests.
	Name string
	// Suite is "regression" or "capability" (for reports).
	Suite string
	// Prompt is the user message.
	Prompt string
	// Scripts are FakeModel responses in order.
	Scripts []agentllm.ResponseScript
	// Tools are registered for the run.
	Tools []tool.Tool
	// WorkspaceRoot is the isolated workspace for file outcome checks and coding tools.
	WorkspaceRoot string
	// Expect defines scoring criteria.
	Expect Expectation
}

// Score derives metrics from a completed agent run.
func Score(messages []msg.AgentMessage, expect Expectation, runErr error) Metrics {
	return ScoreWithWorkspace(messages, expect, runErr, "")
}

// ScoreWithWorkspace derives metrics and evaluates file outcomes under workspaceRoot.
func ScoreWithWorkspace(messages []msg.AgentMessage, expect Expectation, runErr error, workspaceRoot string) Metrics {
	m := Metrics{
		ArgsValid:            true,
		RequiredToolsCovered: true,
		ForbiddenToolsClear:  true,
		OutcomeOK:            true,
	}
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
	m.RequiredToolsCovered = toolsCovered(expect.RequiredTools, m.ToolsCalled)
	m.ForbiddenToolsClear = toolsForbiddenClear(expect.ForbiddenTools, m.ToolsCalled)
	m.StepsWithinLimit = expect.MaxSteps == 0 || m.StepCount <= expect.MaxSteps
	if expect.MaxSteps > 0 && m.StepCount > expect.MaxSteps && !expect.SoftMaxSteps {
		m.Completed = false
	}
	m.OutcomeOK = scoreOutcome(&m, expect.Outcome, workspaceRoot)
	m.Passed = hardPass(m, expect)
	return m
}

func hardPass(m Metrics, expect Expectation) bool {
	if !m.ArgsValid || !m.RequiredToolsCovered || !m.ForbiddenToolsClear || !m.OutcomeOK {
		return false
	}
	if expect.RequireCompletion && !m.Completed {
		return false
	}
	if expect.MaxSteps > 0 && m.StepCount > expect.MaxSteps && !expect.SoftMaxSteps {
		return false
	}
	if expect.StrictToolOrder && !m.ToolSelectionCorrect {
		return false
	}
	return true
}

func scoreOutcome(m *Metrics, outcome OutcomeAsserts, workspaceRoot string) bool {
	ok := true
	for _, want := range outcome.FinalTextContains {
		if !strings.Contains(m.FinalText, want) {
			ok = false
		}
	}
	for _, fileAssert := range outcome.Files {
		if !fileAssertOK(workspaceRoot, fileAssert) {
			ok = false
		}
	}
	return ok
}

func fileAssertOK(workspaceRoot string, fa FileAssert) bool {
	path := strings.TrimSpace(fa.Path)
	if path == "" {
		return false
	}
	full := path
	if workspaceRoot != "" {
		full = filepath.Join(workspaceRoot, filepath.FromSlash(path))
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return false
	}
	content := string(data)
	if fa.Equals != "" && content != fa.Equals {
		return false
	}
	if fa.Contains != "" && !strings.Contains(content, fa.Contains) {
		return false
	}
	return true
}

func scoreAssistant(m *Metrics, assistant msg.AssistantMessage, required map[string][]string, runErr error) {
	m.StepCount++
	if assistant.Usage != nil {
		m.TotalTokens += assistant.Usage.TotalTokens
	}
	for _, call := range assistant.ToolCalls() {
		m.ToolsCalled = append(m.ToolsCalled, call.Name)
		if !argsValidForTool(call, required[call.Name]) {
			m.ArgsValid = false
		}
	}
	text := strings.TrimSpace(assistant.TextContent())
	if text != "" {
		m.FinalText = text
	}
	if len(assistant.ToolCalls()) == 0 && text != "" && runErr == nil {
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

func toolsCovered(required, called []string) bool {
	if len(required) == 0 {
		return true
	}
	seen := make(map[string]struct{}, len(called))
	for _, name := range called {
		seen[name] = struct{}{}
	}
	for _, name := range required {
		if _, ok := seen[name]; !ok {
			return false
		}
	}
	return true
}

func toolsForbiddenClear(forbidden, called []string) bool {
	if len(forbidden) == 0 {
		return true
	}
	deny := make(map[string]struct{}, len(forbidden))
	for _, name := range forbidden {
		deny[name] = struct{}{}
	}
	for _, name := range called {
		if _, hit := deny[name]; hit {
			return false
		}
	}
	return true
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
