package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
)

// EvalReport is the stable JSON artifact for one suite run.
type EvalReport struct {
	// Suite is regression or capability.
	Suite string `json:"suite"`
	// GeneratedAt is RFC3339 timestamp.
	GeneratedAt string `json:"generated_at"`
	// Cases holds per-case results.
	Cases []CaseResult `json:"cases"`
	// Summary aggregates hard pass counts.
	Summary ReportSummary `json:"summary"`
	// PassAtK is set for live multi-trial runs (optional).
	PassAtK *float64 `json:"pass_at_k,omitempty"`
	// PassHatK is set for live multi-trial runs (optional).
	PassHatK *float64 `json:"pass_hat_k,omitempty"`
	// Trials is k when multi-trial aggregation was used.
	Trials int `json:"trials,omitempty"`
	// TotalDurationMs sums trial wall times for the suite.
	TotalDurationMs int64 `json:"total_duration_ms,omitempty"`
	// TotalTokens sums reported token usage across trials.
	TotalTokens int `json:"total_tokens,omitempty"`
	// JudgeGoldAgreement is the fraction of gold dimensions within tolerance.
	JudgeGoldAgreement *float64 `json:"judge_gold_agreement,omitempty"`
}

// ReportSummary counts hard passes.
type ReportSummary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// CaseResult is one task/scenario outcome in a report.
type CaseResult struct {
	// Name identifies the case.
	Name string `json:"name"`
	// Passed is the hard gate result.
	Passed bool `json:"passed"`
	// Metrics holds detailed scores.
	Metrics Metrics `json:"metrics"`
	// TranscriptSummary is a short human-readable trajectory excerpt (failed cases).
	TranscriptSummary string `json:"transcript_summary,omitempty"`
	// Error is a run-level error message when present.
	Error string `json:"error,omitempty"`
	// Judge holds model-grader scores when used.
	Judge *JudgeScores `json:"judge,omitempty"`
	// Gold holds human gold scores when present.
	Gold *GoldScores `json:"gold,omitempty"`
	// TrialPasses lists per-trial hard pass for multi-trial cases.
	TrialPasses []bool `json:"trial_passes,omitempty"`
}

// NewReport builds a report from case results.
func NewReport(suite string, cases []CaseResult) EvalReport {
	summary := ReportSummary{Total: len(cases)}
	for _, c := range cases {
		if c.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}
	return EvalReport{
		Suite:       suite,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Cases:       cases,
		Summary:     summary,
	}
}

// WriteReportJSON writes the report as indented JSON to path.
func WriteReportJSON(path string, report EvalReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := sonic.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// WriteReportMarkdown writes a short Markdown summary next to JSON reports.
func WriteReportMarkdown(path string, report EvalReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(formatReportMarkdown(report)), 0o644)
}

func formatReportMarkdown(report EvalReport) string {
	var b strings.Builder
	_, _ = b.WriteString("# Agent eval report\n\n")
	_, _ = fmt.Fprintf(&b, "- suite: %s\n", report.Suite)
	_, _ = fmt.Fprintf(&b, "- generated_at: %s\n", report.GeneratedAt)
	_, _ = fmt.Fprintf(&b, "- passed: %d/%d\n", report.Summary.Passed, report.Summary.Total)
	if report.TotalDurationMs > 0 {
		_, _ = fmt.Fprintf(&b, "- total_duration_ms: %d\n", report.TotalDurationMs)
	}
	if report.TotalTokens > 0 {
		_, _ = fmt.Fprintf(&b, "- total_tokens: %d\n", report.TotalTokens)
	}
	if report.PassAtK != nil {
		_, _ = fmt.Fprintf(&b, "- pass@k: %.4f\n", *report.PassAtK)
	}
	if report.PassHatK != nil {
		_, _ = fmt.Fprintf(&b, "- pass^k: %.4f\n", *report.PassHatK)
	}
	if report.JudgeGoldAgreement != nil {
		_, _ = fmt.Fprintf(&b, "- judge_gold_agreement: %.4f\n", *report.JudgeGoldAgreement)
	}
	_, _ = b.WriteString("\n## Cases\n\n")
	for _, c := range report.Cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(&b, "### %s (%s)\n\n", c.Name, status)
		if c.Error != "" {
			_, _ = fmt.Fprintf(&b, "- error: %s\n", c.Error)
		}
		if c.TranscriptSummary != "" {
			_, _ = fmt.Fprintf(&b, "```\n%s\n```\n", c.TranscriptSummary)
		}
		_, _ = b.WriteString("\n")
	}
	return b.String()
}

// LoadReportJSON reads an EvalReport from path.
func LoadReportJSON(path string) (EvalReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EvalReport{}, err
	}
	var report EvalReport
	if err := sonic.Unmarshal(data, &report); err != nil {
		return EvalReport{}, err
	}
	return report, nil
}

// CompareDiff summarizes differences between a baseline and candidate report.
type CompareDiff struct {
	BaselineSuite  string   `json:"baseline_suite"`
	CandidateSuite string   `json:"candidate_suite"`
	Improved       []string `json:"improved"`
	Regressed      []string `json:"regressed"`
	UnchangedPass  []string `json:"unchanged_pass"`
	UnchangedFail  []string `json:"unchanged_fail"`
	OnlyBaseline   []string `json:"only_baseline"`
	OnlyCandidate  []string `json:"only_candidate"`
}

// CompareReports contrastively compares two eval reports by case name.
func CompareReports(baseline, candidate EvalReport) CompareDiff {
	baseMap := casePassMap(baseline.Cases)
	candMap := casePassMap(candidate.Cases)
	diff := CompareDiff{BaselineSuite: baseline.Suite, CandidateSuite: candidate.Suite}
	for _, name := range unionSortedKeys(baseMap, candMap) {
		classifyCase(&diff, name, baseMap, candMap)
	}
	return diff
}

func casePassMap(cases []CaseResult) map[string]bool {
	out := make(map[string]bool, len(cases))
	for _, c := range cases {
		out[c.Name] = c.Passed
	}
	return out
}

func unionSortedKeys(a, b map[string]bool) []string {
	names := make(map[string]struct{}, len(a)+len(b))
	for n := range a {
		names[n] = struct{}{}
	}
	for n := range b {
		names[n] = struct{}{}
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, n)
	}
	slices.Sort(ordered)
	return ordered
}

func classifyCase(diff *CompareDiff, name string, baseMap, candMap map[string]bool) {
	bp, bok := baseMap[name]
	cp, cok := candMap[name]
	if bok && !cok {
		diff.OnlyBaseline = append(diff.OnlyBaseline, name)
		return
	}
	if !bok && cok {
		diff.OnlyCandidate = append(diff.OnlyCandidate, name)
		return
	}
	switch {
	case bp && cp:
		diff.UnchangedPass = append(diff.UnchangedPass, name)
	case !bp && !cp:
		diff.UnchangedFail = append(diff.UnchangedFail, name)
	case !bp && cp:
		diff.Improved = append(diff.Improved, name)
	default:
		diff.Regressed = append(diff.Regressed, name)
	}
}

// FormatCompareMarkdown renders a compare diff as Markdown.
func FormatCompareMarkdown(diff CompareDiff) string {
	return fmt.Sprintf(`# Eval compare

- baseline: %s
- candidate: %s
- improved (%d): %v
- regressed (%d): %v
- unchanged pass (%d)
- unchanged fail (%d)
- only baseline (%d): %v
- only candidate (%d): %v
`,
		diff.BaselineSuite, diff.CandidateSuite,
		len(diff.Improved), diff.Improved,
		len(diff.Regressed), diff.Regressed,
		len(diff.UnchangedPass),
		len(diff.UnchangedFail),
		len(diff.OnlyBaseline), diff.OnlyBaseline,
		len(diff.OnlyCandidate), diff.OnlyCandidate,
	)
}

// TaskDraft is a YAML-serializable draft task exported from a failed run.
type TaskDraft struct {
	Name              string `yaml:"name"`
	Suite             string `yaml:"suite"`
	Prompt            string `yaml:"prompt"`
	Notes             string `yaml:"notes"`
	TranscriptExcerpt string `yaml:"transcript_excerpt"`
	SourceError       string `yaml:"source_error,omitempty"`
	ExportedAt        string `yaml:"exported_at"`
}

// ExportTaskDraft builds a draft from a failed case result.
func ExportTaskDraft(caseResult CaseResult, prompt string) TaskDraft {
	name := caseResult.Name
	if strings.TrimSpace(name) == "" {
		name = "exported_case"
	}
	return TaskDraft{
		Name:              name + "_from_failure",
		Suite:             "capability",
		Prompt:            prompt,
		Notes:             "Human review required before promoting into regression suite.",
		TranscriptExcerpt: caseResult.TranscriptSummary,
		SourceError:       caseResult.Error,
		ExportedAt:        time.Now().UTC().Format(time.RFC3339),
	}
}

// WriteTaskDraftYAML writes a task draft YAML file under dir.
func WriteTaskDraftYAML(dir string, draft TaskDraft) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, sanitizeFileName(draft.Name)+".yaml")
	data, err := yaml.Marshal(draft)
	if err != nil {
		return "", fmt.Errorf("eval: marshal task draft: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "task"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			_, _ = b.WriteRune(r)
		case r == ' ' || r == '/':
			_ = b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "task"
	}
	return out
}
