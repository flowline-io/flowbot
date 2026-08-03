package eval

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/tmc/langchaingo/llms"
)

// RunResult is one Fake or live scenario execution with scores.
type RunResult struct {
	// Messages is the full transcript.
	Messages []msg.AgentMessage
	// Metrics are scored outcomes.
	Metrics Metrics
	// Err is the loop error when present.
	Err error
}

// RunFakeScenario executes one scenario with FakeModel scripts.
func RunFakeScenario(ctx context.Context, scenario Scenario) (RunResult, error) {
	model := agentllm.NewFakeModel(scenario.Scripts...)
	return executeScenario(ctx, scenario, model)
}

// RunWithModel executes one scenario against an arbitrary llms.Model.
func RunWithModel(ctx context.Context, scenario Scenario, model llms.Model) (RunResult, error) {
	if model == nil {
		return RunResult{}, fmt.Errorf("eval: model is required")
	}
	return executeScenario(ctx, scenario, model)
}

func executeScenario(ctx context.Context, scenario Scenario, model llms.Model) (RunResult, error) {
	reg := tool.NewRegistry()
	for _, item := range scenario.Tools {
		if err := reg.Register(item); err != nil {
			return RunResult{}, fmt.Errorf("eval: register tool: %w", err)
		}
	}
	cfg := loop.DefaultConfig()
	cfg.ModelName = "eval"
	if scenario.Expect.MaxSteps > 0 {
		cfg.MaxSteps = scenario.Expect.MaxSteps
	}
	start := time.Now()
	messages, err := loop.RunLoop(ctx, []msg.AgentMessage{
		msg.NewUserMessage(scenario.Prompt),
	}, &msg.Context{}, cfg, loop.LoopDeps{Model: model, Registry: reg}, nil)
	metrics := ScoreWithWorkspace(messages, scenario.Expect, err, scenario.WorkspaceRoot)
	metrics.DurationMs = time.Since(start).Milliseconds()
	return RunResult{Messages: messages, Metrics: metrics, Err: err}, nil
}

// TranscriptSummary builds a short readable excerpt from messages.
func TranscriptSummary(messages []msg.AgentMessage, lineLimit int) string {
	if lineLimit <= 0 {
		lineLimit = 40
	}
	var lines []string
	for _, item := range messages {
		switch m := item.(type) {
		case msg.UserMessage:
			lines = append(lines, "user: "+trimOneLine(partsText(m.Parts), 200))
		case msg.AssistantMessage:
			if calls := m.ToolCalls(); len(calls) > 0 {
				names := make([]string, 0, len(calls))
				for _, c := range calls {
					names = append(names, c.Name)
				}
				lines = append(lines, "assistant.tools: "+strings.Join(names, ", "))
			}
			if text := strings.TrimSpace(m.TextContent()); text != "" {
				lines = append(lines, "assistant: "+trimOneLine(text, 200))
			}
		case msg.ToolResultMessage:
			errMark := ""
			if m.IsError {
				errMark = " ERROR"
			}
			lines = append(lines, "tool."+m.Name+errMark+": "+trimOneLine(partsText(m.Parts), 160))
		}
		if len(lines) >= lineLimit {
			lines = append(lines, "...(truncated)")
			break
		}
	}
	return strings.Join(lines, "\n")
}

func trimOneLine(s string, limit int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if limit > 0 && len(s) > limit {
		return s[:limit] + "..."
	}
	return s
}

func partsText(parts []msg.ContentPart) string {
	var b strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// CaseResultFromRun maps a RunResult into a report case.
// TranscriptSummary is attached only for failed cases (CI/report readability).
func CaseResultFromRun(name string, run RunResult) CaseResult {
	cr := CaseResult{
		Name:    name,
		Passed:  run.Metrics.Passed,
		Metrics: run.Metrics,
	}
	if !cr.Passed {
		cr.TranscriptSummary = TranscriptSummary(run.Messages, 40)
	}
	if run.Err != nil {
		cr.Error = run.Err.Error()
	}
	return cr
}
