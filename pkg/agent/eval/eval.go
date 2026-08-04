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
	// FinalTextContains requires each substring in the last assistant text (case-sensitive).
	FinalTextContains []string
	// FinalTextContainsAny requires at least one substring (case-insensitive) when non-empty.
	FinalTextContainsAny []string
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

// Tier labels for the golden dataset ladder.
const (
	TierBasic  = "basic"
	TierCombo  = "combo"
	TierSystem = "system"
	TierRepair = "repair"
)

// MetricCompliance tags a case as L1 compliance-eligible regardless of name heuristics.
const MetricCompliance = "compliance"

// Metrics captures scored outcomes for one scenario run.
type Metrics struct {
	// ToolSelectionCorrect is true when ExpectedTools appear in order (soft metric).
	ToolSelectionCorrect bool
	// ToolSelectionTracked is true when ExpectedTools was non-empty (soft metric denominator).
	ToolSelectionTracked bool
	// RequiredToolsCovered is true when every RequiredTools entry was called.
	RequiredToolsCovered bool
	// ForbiddenToolsClear is true when no ForbiddenTools entry was called.
	ForbiddenToolsClear bool
	// ArgsValid is true when required tool arguments were present and non-empty.
	ArgsValid bool
	// OutcomeOK is true when OutcomeAsserts passed (or none were set).
	OutcomeOK bool
	// TextOutcomeOK is true when final-text asserts passed (or none were set).
	TextOutcomeOK bool
	// FileOutcomeOK is true when file asserts passed (or none were set).
	FileOutcomeOK bool
	// StepCount is the number of assistant turns observed.
	StepCount int
	// Completed is true when the run finished with a final assistant message and no error.
	Completed bool
	// ToolsCalled lists tool names executed during the run.
	ToolsCalled []string
	// FinalText is the last non-empty assistant text content.
	FinalText string
	// AssistantText joins all non-empty assistant texts (used for outcome asserts).
	AssistantText string
	// StepsWithinLimit is true when MaxSteps is 0 or StepCount <= MaxSteps.
	StepsWithinLimit bool
	// DurationMs is wall time for the scenario run.
	DurationMs int64
	// TotalTokens sums assistant Usage.TotalTokens when reported.
	TotalTokens int
	// ToolErrorCount is the number of tool results with IsError.
	ToolErrorCount int
	// NaturalRepairEligible is true when ToolErrorCount >= 1.
	NaturalRepairEligible bool
	// NaturalRepairSuccess is true when eligible and Passed.
	NaturalRepairSuccess bool
	// ComplianceEligible is true when this trial counts toward L1.
	ComplianceEligible bool
	// CompliancePassed is the L1 gate result for this trial.
	CompliancePassed bool
	// ToolAccEligible is true when this trial counts toward ToolCallAcc.
	ToolAccEligible bool
	// ToolAccPassed is the ToolCallAcc hard-gate result for this trial.
	ToolAccPassed bool
	// Passed is the hard CI gate (required/forbidden/args/outcome/completion/max steps; order only if strict).
	Passed bool
}

// Scenario is one FakeModel-driven harness evaluation case.
type Scenario struct {
	// Name identifies the scenario in table tests.
	Name string
	// Suite is "regression" or "capability" (for reports).
	Suite string
	// Difficulty is easy, medium, or hard (capability ladder).
	Difficulty string
	// Tier is basic, combo, system, or repair.
	Tier string
	// MetricsTags lists optional metric tags (e.g. compliance).
	MetricsTags []string
	// Prompt is the user message.
	Prompt string
	// Scripts are FakeModel responses in order.
	Scripts []agentllm.ResponseScript
	// Tools are registered for the run.
	Tools []tool.Tool
	// WorkspaceRoot is the isolated workspace for file outcome checks and coding tools.
	WorkspaceRoot string
	// Fixtures seed WorkspaceRoot before each trial (live multi-trial isolation).
	Fixtures []WorkspaceFixture
	// Expect defines scoring criteria.
	Expect Expectation
}

// WorkspaceFixture seeds one file into an isolated workspace.
type WorkspaceFixture struct {
	// Path is relative to WorkspaceRoot.
	Path string
	// Content is the file body.
	Content string
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
		TextOutcomeOK:        true,
		FileOutcomeOK:        true,
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
	m.ToolErrorCount = CountToolErrors(messages)
	if expect.RequireCompletion && runErr != nil {
		m.Completed = false
	}
	m.ToolSelectionCorrect = toolsMatch(expect.ExpectedTools, m.ToolsCalled)
	m.ToolSelectionTracked = len(expect.ExpectedTools) > 0
	m.RequiredToolsCovered = toolsCovered(expect.RequiredTools, m.ToolsCalled)
	m.ForbiddenToolsClear = toolsForbiddenClear(expect.ForbiddenTools, m.ToolsCalled)
	m.StepsWithinLimit = expect.MaxSteps == 0 || m.StepCount <= expect.MaxSteps
	if expect.MaxSteps > 0 && m.StepCount > expect.MaxSteps && !expect.SoftMaxSteps {
		m.Completed = false
	}
	m.TextOutcomeOK, m.FileOutcomeOK = scoreOutcomeParts(m, expect.Outcome, workspaceRoot)
	m.OutcomeOK = m.TextOutcomeOK && m.FileOutcomeOK
	m.Passed = hardPass(m, expect)
	m.NaturalRepairEligible = m.ToolErrorCount >= 1
	m.NaturalRepairSuccess = m.NaturalRepairEligible && m.Passed
	return m
}

// ScoreScenario scores a run and annotates L1/L2 layer fields from scenario metadata.
func ScoreScenario(messages []msg.AgentMessage, sc Scenario, runErr error) Metrics {
	m := ScoreWithWorkspace(messages, sc.Expect, runErr, sc.WorkspaceRoot)
	AnnotateLayerMetrics(&m, sc.Name, sc.Tier, sc.MetricsTags, sc.Expect)
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
	textOK, fileOK := scoreOutcomeParts(*m, outcome, workspaceRoot)
	return textOK && fileOK
}

func scoreOutcomeParts(m Metrics, outcome OutcomeAsserts, workspaceRoot string) (textOK, fileOK bool) {
	text := outcomeSearchText(m)
	textOK = true
	norm := normalizeOutcomeText(text)
	for _, want := range outcome.FinalTextContains {
		if !strings.Contains(norm, normalizeOutcomeText(want)) {
			textOK = false
		}
	}
	if len(outcome.FinalTextContainsAny) > 0 && !textContainsAnyFold(text, outcome.FinalTextContainsAny) {
		textOK = false
	}
	fileOK = true
	for _, fileAssert := range outcome.Files {
		if !fileAssertOK(workspaceRoot, fileAssert) {
			fileOK = false
		}
	}
	return textOK, fileOK
}

func outcomeSearchText(m Metrics) string {
	if strings.TrimSpace(m.AssistantText) != "" {
		return m.AssistantText
	}
	return m.FinalText
}

func textContainsAnyFold(text string, wants []string) bool {
	lower := strings.ToLower(normalizeOutcomeText(text))
	for _, want := range wants {
		if want != "" && strings.Contains(lower, strings.ToLower(normalizeOutcomeText(want))) {
			return true
		}
	}
	return false
}

// normalizeOutcomeText maps typographic quotes and collapses JSON-insignificant whitespace
// outside of strings so `"id": "a"` matches a grader needle `"id":"a"`.
func normalizeOutcomeText(s string) string {
	return compactJSONOutsideStrings(normalizeQuotes(s))
}

// normalizeQuotes maps common typographic apostrophes/quotes to ASCII so live models match.
func normalizeQuotes(s string) string {
	return strings.NewReplacer(
		"\u2018", "'", // ‘
		"\u2019", "'", // ’
		"\u201A", "'", // ‚
		"\u02BC", "'", // ʼ
		"\u00B4", "'", // ´
		"\u201C", `"`, // “
		"\u201D", `"`, // ”
	).Replace(s)
}

func compactJSONOutsideStrings(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	escape := false
	for _, r := range s {
		if escape {
			_, _ = b.WriteRune(r)
			escape = false
			continue
		}
		if inString {
			if r == '\\' {
				escape = true
			} else if r == '"' {
				inString = false
			}
			_, _ = b.WriteRune(r)
			continue
		}
		if r == '"' {
			inString = true
			_, _ = b.WriteRune(r)
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		_, _ = b.WriteRune(r)
	}
	return b.String()
}

// FailReason returns a short explanation when metrics did not hard-pass.
func FailReason(m Metrics, expect Expectation) string {
	if m.Passed {
		return ""
	}
	var parts []string
	if !m.RequiredToolsCovered {
		parts = append(parts, "required tools missing")
	}
	if !m.ForbiddenToolsClear {
		parts = append(parts, "forbidden tool used")
	}
	if !m.ArgsValid {
		parts = append(parts, "invalid tool args")
	}
	if !m.OutcomeOK {
		parts = append(parts, outcomeFailDetail(outcomeSearchText(m), expect.Outcome))
	}
	if expect.RequireCompletion && !m.Completed {
		parts = append(parts, "incomplete")
	}
	if expect.MaxSteps > 0 && m.StepCount > expect.MaxSteps && !expect.SoftMaxSteps {
		parts = append(parts, "max steps exceeded")
	}
	if expect.StrictToolOrder && !m.ToolSelectionCorrect {
		parts = append(parts, "tool order mismatch")
	}
	if len(parts) == 0 {
		return "failed"
	}
	return strings.Join(parts, "; ")
}

func outcomeFailDetail(finalText string, outcome OutcomeAsserts) string {
	var missing []string
	norm := normalizeOutcomeText(finalText)
	for _, want := range outcome.FinalTextContains {
		if !strings.Contains(norm, normalizeOutcomeText(want)) {
			missing = append(missing, fmt.Sprintf("%q", want))
		}
	}
	if len(outcome.FinalTextContainsAny) > 0 && !textContainsAnyFold(finalText, outcome.FinalTextContainsAny) {
		quoted := make([]string, 0, len(outcome.FinalTextContainsAny))
		for _, want := range outcome.FinalTextContainsAny {
			quoted = append(quoted, fmt.Sprintf("%q", want))
		}
		missing = append(missing, "any of ["+strings.Join(quoted, ", ")+"]")
	}
	if len(missing) == 0 {
		return "outcome assert failed"
	}
	return "final text missing " + strings.Join(missing, ", ")
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
	if fa.Equals != "" && strings.TrimRight(content, "\r\n") != strings.TrimRight(fa.Equals, "\r\n") {
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
		if m.AssistantText != "" {
			m.AssistantText += "\n"
		}
		m.AssistantText += text
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
