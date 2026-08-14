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
	// JudgeMode is "fake", "none", or "model:<name>" for interpreting quality scores.
	JudgeMode string `json:"judge_mode,omitempty"`
	// LatencyBudgetMs is the L3 latency budget used for this report when set.
	LatencyBudgetMs int64 `json:"latency_budget_ms,omitempty"`
	// TokenBudget is the L3 token budget used for this report when set.
	TokenBudget int `json:"token_budget,omitempty"`
	// Scorecard aggregates L1/L2/L3 Total for optimization A/B.
	Scorecard *CapabilityScorecard `json:"scorecard,omitempty"`
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
	// Difficulty is easy, medium, or hard when set.
	Difficulty string `json:"difficulty,omitempty"`
	// Tier is basic, combo, system, or repair when set.
	Tier string `json:"tier,omitempty"`
	// Passed is the hard gate result.
	Passed bool `json:"passed"`
	// Metrics holds detailed scores (last trial for multi-trial cases).
	Metrics Metrics `json:"metrics"`
	// TrialMetrics holds per-trial metrics for multi-trial cases.
	TrialMetrics []Metrics `json:"trial_metrics,omitempty"`
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
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := sonic.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// WriteReportMarkdown writes a capability-oriented Markdown summary next to JSON reports.
func WriteReportMarkdown(path string, report EvalReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(formatReportMarkdown(report)), 0o644)
}

func formatReportMarkdown(report EvalReport) string {
	sc := report.Scorecard
	if sc == nil {
		tmp := ScorecardFromReport(report)
		sc = &tmp
	}
	var b strings.Builder
	writeReportHeader(&b, report, *sc)
	writeScorecardTable(&b, report, *sc)
	writeCasesTable(&b, report.Cases)
	writeFailedCaseDetails(&b, report.Cases)
	return b.String()
}

func writeReportHeader(b *strings.Builder, report EvalReport, sc CapabilityScorecard) {
	_, _ = b.WriteString("# Agent capability report\n\n")
	_, _ = fmt.Fprintf(b, "- suite: %s\n", report.Suite)
	_, _ = fmt.Fprintf(b, "- generated_at: %s\n", report.GeneratedAt)
	if report.JudgeMode != "" {
		_, _ = fmt.Fprintf(b, "- judge_mode: %s\n", report.JudgeMode)
	}
	if report.Trials > 0 {
		_, _ = fmt.Fprintf(b, "- trials (k): %d\n", report.Trials)
	}
	_, _ = fmt.Fprintf(b, "- hard_pass: %d/%d (%.1f%%)\n",
		report.Summary.Passed, report.Summary.Total, 100*sc.HardPassRate)
}

func writeScorecardTable(b *strings.Builder, report EvalReport, sc CapabilityScorecard) {
	_, _ = b.WriteString("\n## Scorecard\n\n")
	_, _ = b.WriteString("| Metric | Value |\n| --- | --- |\n")
	_, _ = fmt.Fprintf(b, "| **total** (0-100) | **%.2f** |\n", sc.Total)
	_, _ = fmt.Fprintf(b, "| L1 compliance | %.4f |\n", sc.L1Score)
	_, _ = fmt.Fprintf(b, "| L2 execution | %.4f |\n", sc.L2Score)
	_, _ = fmt.Fprintf(b, "| L3 efficiency | %.4f |\n", sc.L3Score)
	_, _ = fmt.Fprintf(b, "| pass@1 | %.4f |\n", sc.PassAt1)
	if sc.PassAt1CI != nil {
		_, _ = fmt.Fprintf(b, "| pass@1 Wilson CI | [%.4f, %.4f] |\n", sc.PassAt1CI.Low, sc.PassAt1CI.High)
	}
	if sc.ToolCallAcc != nil {
		_, _ = fmt.Fprintf(b, "| tool_call_acc | %.4f |\n", *sc.ToolCallAcc)
	}
	if sc.RepairRate != nil {
		_, _ = fmt.Fprintf(b, "| repair_rate | %.4f |\n", *sc.RepairRate)
	}
	if sc.LatencyScore != nil {
		_, _ = fmt.Fprintf(b, "| latency_score | %.4f |\n", *sc.LatencyScore)
	}
	if sc.TokenScore != nil {
		_, _ = fmt.Fprintf(b, "| token_score | %.4f |\n", *sc.TokenScore)
	}
	_, _ = fmt.Fprintf(b, "| reliability (appendix) | %.4f |\n", sc.Reliability)
	if sc.PassAtK != nil {
		_, _ = fmt.Fprintf(b, "| pass@k (appendix) | %.4f |\n", *sc.PassAtK)
	}
	if sc.PassHatK != nil {
		_, _ = fmt.Fprintf(b, "| pass^k (appendix) | %.4f |\n", *sc.PassHatK)
	}
	writeQualityRows(b, sc)
	if sc.JudgeGoldAgreement != nil {
		_, _ = fmt.Fprintf(b, "| judge_gold_agreement | %.4f |\n", *sc.JudgeGoldAgreement)
	}
	if sc.NaturalRepairRate != nil {
		_, _ = fmt.Fprintf(b, "| natural_repair_rate | %.4f |\n", *sc.NaturalRepairRate)
	}
	if report.TotalDurationMs > 0 {
		_, _ = fmt.Fprintf(b, "| total_duration_ms | %d |\n", report.TotalDurationMs)
	}
	if report.TotalTokens > 0 {
		_, _ = fmt.Fprintf(b, "| total_tokens | %d |\n", report.TotalTokens)
	}
	if sc.Notes != "" {
		_, _ = fmt.Fprintf(b, "\n_%s_\n", sc.Notes)
	}
}

func writeQualityRows(b *strings.Builder, sc CapabilityScorecard) {
	if sc.QualityEnabled && sc.QualityAvg != nil {
		_, _ = fmt.Fprintf(b, "| quality_avg (1–5) | %.2f |\n", *sc.QualityAvg)
		_, _ = fmt.Fprintf(b, "| correctness_avg | %.2f |\n", *sc.CorrectnessAvg)
		_, _ = fmt.Fprintf(b, "| faithfulness_avg | %.2f |\n", *sc.FaithfulnessAvg)
		_, _ = fmt.Fprintf(b, "| helpfulness_avg | %.2f |\n", *sc.HelpfulnessAvg)
		_, _ = fmt.Fprintf(b, "| safety_avg | %.2f |\n", *sc.SafetyAvg)
		return
	}
	_, _ = b.WriteString("| quality_avg | n/a (enable with `--judge-model` + `--judge-fake=false`) |\n")
}

func writeCasesTable(b *strings.Builder, cases []CaseResult) {
	_, _ = b.WriteString("\n## Cases\n\n")
	_, _ = b.WriteString("| case | diff | hard | trials | corr | faith | help | safety | ms | tokens |\n")
	_, _ = b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, c := range cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		diff := c.Difficulty
		if diff == "" {
			diff = DifficultyEasy
		}
		corr, faith, help, safety := judgeCell(c.Judge)
		_, _ = fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s | %s | %d | %d |\n",
			c.Name, diff, status, trialPassLabel(c.TrialPasses), corr, faith, help, safety,
			c.Metrics.DurationMs, c.Metrics.TotalTokens)
	}
}

func judgeCell(j *JudgeScores) (corr, faith, help, safety string) {
	corr, faith, help, safety = "—", "—", "—", "—"
	if j == nil || j.Unknown {
		return corr, faith, help, safety
	}
	return fmt.Sprintf("%d", j.Correctness), fmt.Sprintf("%d", j.Faithfulness),
		fmt.Sprintf("%d", j.Helpfulness), fmt.Sprintf("%d", j.Safety)
}

func writeFailedCaseDetails(b *strings.Builder, cases []CaseResult) {
	for _, c := range cases {
		if c.Passed && c.Error == "" && c.TranscriptSummary == "" {
			continue
		}
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(b, "\n### %s (%s)\n\n", c.Name, status)
		if c.Error != "" {
			_, _ = fmt.Fprintf(b, "- error: %s\n", c.Error)
		}
		if c.Judge != nil && c.Judge.Reasoning != "" {
			_, _ = fmt.Fprintf(b, "- judge: %s\n", c.Judge.Reasoning)
		}
		if c.Gold != nil && c.Gold.Rationale != "" {
			_, _ = fmt.Fprintf(b, "- gold: %s\n", c.Gold.Rationale)
		}
		if c.TranscriptSummary != "" {
			_, _ = fmt.Fprintf(b, "```\n%s\n```\n", c.TranscriptSummary)
		}
	}
	_, _ = b.WriteString("\n")
}

func trialPassLabel(trials []bool) string {
	if len(trials) == 0 {
		return "—"
	}
	ok := 0
	for _, p := range trials {
		if p {
			ok++
		}
	}
	return fmt.Sprintf("%d/%d", ok, len(trials))
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
	BaselineSuite      string              `json:"baseline_suite"`
	CandidateSuite     string              `json:"candidate_suite"`
	Improved           []string            `json:"improved"`
	Regressed          []string            `json:"regressed"`
	UnchangedPass      []string            `json:"unchanged_pass"`
	UnchangedFail      []string            `json:"unchanged_fail"`
	OnlyBaseline       []string            `json:"only_baseline"`
	OnlyCandidate      []string            `json:"only_candidate"`
	BaselineScorecard  CapabilityScorecard `json:"baseline_scorecard"`
	CandidateScorecard CapabilityScorecard `json:"candidate_scorecard"`
	TotalDelta         float64             `json:"total_delta"`
	L1Delta            float64             `json:"l1_delta"`
	L2Delta            float64             `json:"l2_delta"`
	L3Delta            float64             `json:"l3_delta"`
	ReliabilityDelta   float64             `json:"reliability_delta"`
	QualityDelta       *float64            `json:"quality_delta,omitempty"`
}

// CompareReports contrastively compares two eval reports by case name and scorecard.
func CompareReports(baseline, candidate EvalReport) CompareDiff {
	baseMap := casePassMap(baseline.Cases)
	candMap := casePassMap(candidate.Cases)
	baseSC := ScorecardFromReport(baseline)
	candSC := ScorecardFromReport(candidate)
	diff := CompareDiff{
		BaselineSuite:      baseline.Suite,
		CandidateSuite:     candidate.Suite,
		BaselineScorecard:  baseSC,
		CandidateScorecard: candSC,
		TotalDelta:         round2(candSC.Total - baseSC.Total),
		L1Delta:            round2(candSC.L1Score - baseSC.L1Score),
		L2Delta:            round2(candSC.L2Score - baseSC.L2Score),
		L3Delta:            round2(candSC.L3Score - baseSC.L3Score),
		ReliabilityDelta:   round2(candSC.Reliability - baseSC.Reliability),
	}
	if baseSC.QualityAvg != nil && candSC.QualityAvg != nil {
		d := round2(*candSC.QualityAvg - *baseSC.QualityAvg)
		diff.QualityDelta = &d
	}
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
func FormatCompareMarkdown(diff *CompareDiff) string {
	if diff == nil {
		return ""
	}
	var b strings.Builder
	_, _ = b.WriteString("# Eval compare\n\n")
	_, _ = fmt.Fprintf(&b, "- baseline: %s\n", diff.BaselineSuite)
	_, _ = fmt.Fprintf(&b, "- candidate: %s\n", diff.CandidateSuite)
	_, _ = b.WriteString("\n## Scorecard delta\n\n")
	_, _ = fmt.Fprintf(&b, "| Metric | Baseline | Candidate | Delta |\n| --- | --- | --- | --- |\n")
	_, _ = fmt.Fprintf(&b, "| total | %.2f | %.2f | %+.2f |\n",
		diff.BaselineScorecard.Total, diff.CandidateScorecard.Total, diff.TotalDelta)
	_, _ = fmt.Fprintf(&b, "| L1 | %.4f | %.4f | %+.4f |\n",
		diff.BaselineScorecard.L1Score, diff.CandidateScorecard.L1Score, diff.L1Delta)
	_, _ = fmt.Fprintf(&b, "| L2 | %.4f | %.4f | %+.4f |\n",
		diff.BaselineScorecard.L2Score, diff.CandidateScorecard.L2Score, diff.L2Delta)
	_, _ = fmt.Fprintf(&b, "| L3 | %.4f | %.4f | %+.4f |\n",
		diff.BaselineScorecard.L3Score, diff.CandidateScorecard.L3Score, diff.L3Delta)
	_, _ = fmt.Fprintf(&b, "| reliability | %.4f | %.4f | %+.4f |\n",
		diff.BaselineScorecard.Reliability, diff.CandidateScorecard.Reliability, diff.ReliabilityDelta)
	if diff.QualityDelta != nil && diff.BaselineScorecard.QualityAvg != nil && diff.CandidateScorecard.QualityAvg != nil {
		_, _ = fmt.Fprintf(&b, "| quality_avg | %.2f | %.2f | %+.2f |\n",
			*diff.BaselineScorecard.QualityAvg, *diff.CandidateScorecard.QualityAvg, *diff.QualityDelta)
	} else {
		_, _ = b.WriteString("| quality_avg | n/a | n/a | n/a |\n")
	}
	_, _ = fmt.Fprintf(&b, "\n## Case gate delta\n\n")
	_, _ = fmt.Fprintf(&b, "- improved (%d): %v\n", len(diff.Improved), diff.Improved)
	_, _ = fmt.Fprintf(&b, "- regressed (%d): %v\n", len(diff.Regressed), diff.Regressed)
	_, _ = fmt.Fprintf(&b, "- unchanged pass (%d)\n", len(diff.UnchangedPass))
	_, _ = fmt.Fprintf(&b, "- unchanged fail (%d)\n", len(diff.UnchangedFail))
	_, _ = fmt.Fprintf(&b, "- only baseline (%d): %v\n", len(diff.OnlyBaseline), diff.OnlyBaseline)
	_, _ = fmt.Fprintf(&b, "- only candidate (%d): %v\n", len(diff.OnlyCandidate), diff.OnlyCandidate)
	return b.String()
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
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
