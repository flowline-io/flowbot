package eval

import (
	"context"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// ProgressEvent reports live/regression case progress (go test style).
type ProgressEvent struct {
	// Phase is "case_start", "trial", or "case_done".
	Phase string
	// CaseName is the scenario name.
	CaseName string
	// CaseIndex is 1-based index among scenarios.
	CaseIndex int
	// CaseTotal is the number of scenarios.
	CaseTotal int
	// Trial is 1-based trial index (live only; 0 when N/A).
	Trial int
	// Trials is k for live runs.
	Trials int
	// Passed is set for trial and case_done phases.
	Passed bool
	// Duration is wall time for the trial or whole case.
	Duration time.Duration
	// Detail is an optional failure reason.
	Detail string
}

// ProgressFunc receives progress events during evaluation.
type ProgressFunc func(ProgressEvent)

// LiveOptions configures multi-trial live evaluation.
type LiveOptions struct {
	// Trials is k (default 3).
	Trials int
	// ModelName is sent as llms.WithModel on each request.
	// Empty defaults to "eval" (safe for FakeModel); real providers need the configured name.
	ModelName string
	// JudgeModel scores open quality dimensions when non-nil.
	JudgeModel llms.Model
	// GoldByCase maps case name to gold scores for agreement.
	GoldByCase map[string]GoldScores
	// OnProgress reports case/trial progress when set.
	OnProgress ProgressFunc
	// JudgeMode labels quality scoring: "fake", "none", or "model:<name>".
	JudgeMode string
	// LatencyBudgetMs overrides default L3 latency budget when > 0.
	LatencyBudgetMs int64
	// TokenBudget overrides default L3 token budget when > 0.
	TokenBudget int
}

// RunLiveScenarios runs each scenario k times with a real (or fake) model and aggregates pass@k / pass^k.
func RunLiveScenarios(ctx context.Context, scenarios []Scenario, model llms.Model, opts LiveOptions) (EvalReport, error) {
	if model == nil {
		return EvalReport{}, fmt.Errorf("eval: live model is required")
	}
	k := opts.Trials
	if k <= 0 {
		k = 3
	}
	cases := make([]CaseResult, 0, len(scenarios))
	allTrials := make([][]bool, 0, len(scenarios))
	var agreeSum float64
	var agreeN int
	var totalDuration int64
	var totalTokens int

	for idx, scenario := range scenarios {
		cr, trialPasses, dur, tokens, err := runLiveCase(ctx, scenario, model, opts, idx, len(scenarios), k)
		if err != nil {
			return EvalReport{}, err
		}
		totalDuration += dur
		totalTokens += tokens
		allTrials = append(allTrials, trialPasses)
		if cr.Judge != nil {
			if gold, ok := opts.GoldByCase[scenario.Name]; ok {
				rate, compared := AgreementRate(*cr.Judge, gold)
				if compared > 0 {
					agreeSum += rate
					agreeN++
				}
			}
		}
		cases = append(cases, cr)
	}

	report := NewReport("capability", cases)
	report.Trials = k
	report.TotalDurationMs = totalDuration
	report.TotalTokens = totalTokens
	report.JudgeMode = opts.JudgeMode
	if report.JudgeMode == "" {
		if opts.JudgeModel == nil {
			report.JudgeMode = "none"
		} else {
			report.JudgeMode = "fake"
		}
	}
	report.LatencyBudgetMs = opts.LatencyBudgetMs
	report.TokenBudget = opts.TokenBudget
	passAt := PassAtK(allTrials)
	passHat := PassHatK(allTrials)
	report.PassAtK = &passAt
	report.PassHatK = &passHat
	if agreeN > 0 {
		avg := agreeSum / float64(agreeN)
		report.JudgeGoldAgreement = &avg
	}
	sc := ScorecardFromReportOpts(report, ScorecardOptions{
		LatencyBudgetMs: opts.LatencyBudgetMs,
		TokenBudget:     opts.TokenBudget,
	})
	report.Scorecard = &sc
	return report, nil
}

func runLiveCase(
	ctx context.Context,
	scenario Scenario,
	model llms.Model,
	opts LiveOptions,
	idx, total, k int,
) (CaseResult, []bool, int64, int, error) {
	caseStart := time.Now()
	emitProgress(opts.OnProgress, ProgressEvent{
		Phase: "case_start", CaseName: scenario.Name,
		CaseIndex: idx + 1, CaseTotal: total, Trials: k,
	})

	trialPasses := make([]bool, 0, k)
	trialMetrics := make([]Metrics, 0, k)
	var last RunResult
	var totalDuration int64
	var totalTokens int
	for i := range k {
		if err := ResetScenarioWorkspace(scenario); err != nil {
			return CaseResult{}, nil, 0, 0, err
		}
		run, err := RunWithModel(ctx, scenario, model, opts.ModelName)
		if err != nil {
			return CaseResult{}, nil, 0, 0, err
		}
		last = run
		totalDuration += run.Metrics.DurationMs
		totalTokens += run.Metrics.TotalTokens
		trialPasses = append(trialPasses, run.Metrics.Passed)
		trialMetrics = append(trialMetrics, run.Metrics)
		emitProgress(opts.OnProgress, ProgressEvent{
			Phase: "trial", CaseName: scenario.Name,
			CaseIndex: idx + 1, CaseTotal: total,
			Trial: i + 1, Trials: k, Passed: run.Metrics.Passed,
			Duration: time.Duration(run.Metrics.DurationMs) * time.Millisecond,
			Detail:   FailReason(run.Metrics, scenario.Expect),
		})
	}

	cr := CaseResultFromRun(scenario.Name, last)
	cr.Difficulty = NormalizeDifficulty(scenario.Difficulty)
	cr.Tier = NormalizeTier(scenario.Tier)
	cr.TrialPasses = trialPasses
	cr.TrialMetrics = trialMetrics
	cr.Passed = allTrialsPassed(trialPasses)
	if !cr.Passed && cr.TranscriptSummary == "" {
		cr.TranscriptSummary = TranscriptSummary(last.Messages, 40)
	}
	if !cr.Passed && cr.Error == "" {
		cr.Error = FailReason(last.Metrics, scenario.Expect)
	}
	if err := attachLiveJudge(ctx, &cr, scenario, last, opts); err != nil {
		return CaseResult{}, nil, 0, 0, err
	}
	emitProgress(opts.OnProgress, ProgressEvent{
		Phase: "case_done", CaseName: scenario.Name,
		CaseIndex: idx + 1, CaseTotal: total,
		Trials: k, Passed: cr.Passed, Duration: time.Since(caseStart),
		Detail: cr.Error,
	})
	return cr, trialPasses, totalDuration, totalTokens, nil
}

func allTrialsPassed(trials []bool) bool {
	for _, ok := range trials {
		if !ok {
			return false
		}
	}
	return len(trials) > 0
}

func attachLiveJudge(ctx context.Context, cr *CaseResult, scenario Scenario, last RunResult, opts LiveOptions) error {
	if opts.JudgeModel == nil {
		return nil
	}
	summary := TranscriptSummary(last.Messages, 40)
	judge, err := JudgeAll(ctx, opts.JudgeModel, scenario.Prompt, summary, last.Metrics.FinalText)
	if err != nil {
		return err
	}
	cr.Judge = &judge
	if gold, ok := opts.GoldByCase[scenario.Name]; ok {
		gg := gold
		cr.Gold = &gg
	}
	return nil
}

func emitProgress(fn ProgressFunc, ev ProgressEvent) {
	if fn != nil {
		fn(ev)
	}
}

// PassAtK estimates the fraction of tasks with at least one success in k trials.
func PassAtK(trialPasses [][]bool) float64 {
	if len(trialPasses) == 0 {
		return 0
	}
	success := 0
	for _, trials := range trialPasses {
		for _, ok := range trials {
			if ok {
				success++
				break
			}
		}
	}
	return float64(success) / float64(len(trialPasses))
}

// PassHatK estimates the fraction of tasks where all k trials succeeded.
func PassHatK(trialPasses [][]bool) float64 {
	if len(trialPasses) == 0 {
		return 0
	}
	success := 0
	for _, trials := range trialPasses {
		if allTrialsPassed(trials) {
			success++
		}
	}
	return float64(success) / float64(len(trialPasses))
}
