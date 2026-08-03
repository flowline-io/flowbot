package eval

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

// LiveOptions configures multi-trial live evaluation.
type LiveOptions struct {
	// Trials is k (default 3).
	Trials int
	// JudgeModel scores open quality dimensions when non-nil.
	JudgeModel llms.Model
	// GoldByCase maps case name to gold scores for agreement.
	GoldByCase map[string]GoldScores
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

	for _, scenario := range scenarios {
		trialPasses := make([]bool, 0, k)
		var last RunResult
		for i := 0; i < k; i++ {
			run, err := RunWithModel(ctx, scenario, model)
			if err != nil {
				return EvalReport{}, err
			}
			last = run
			totalDuration += run.Metrics.DurationMs
			totalTokens += run.Metrics.TotalTokens
			trialPasses = append(trialPasses, run.Metrics.Passed)
		}
		allTrials = append(allTrials, trialPasses)
		cr := CaseResultFromRun(scenario.Name, last)
		cr.TrialPasses = trialPasses
		allOK := true
		for _, ok := range trialPasses {
			if !ok {
				allOK = false
				break
			}
		}
		cr.Passed = allOK
		if !cr.Passed && cr.TranscriptSummary == "" {
			cr.TranscriptSummary = TranscriptSummary(last.Messages, 40)
		}

		if opts.JudgeModel != nil {
			summary := TranscriptSummary(last.Messages, 40)
			judge, err := JudgeAll(ctx, opts.JudgeModel, scenario.Prompt, summary, last.Metrics.FinalText)
			if err != nil {
				return EvalReport{}, err
			}
			cr.Judge = &judge
			if gold, ok := opts.GoldByCase[scenario.Name]; ok {
				gg := gold
				cr.Gold = &gg
				rate, compared := AgreementRate(judge, gold)
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
	passAt := PassAtK(allTrials)
	passHat := PassHatK(allTrials)
	report.PassAtK = &passAt
	report.PassHatK = &passHat
	if agreeN > 0 {
		avg := agreeSum / float64(agreeN)
		report.JudgeGoldAgreement = &avg
	}
	return report, nil
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
		if len(trials) == 0 {
			continue
		}
		all := true
		for _, ok := range trials {
			if !ok {
				all = false
				break
			}
		}
		if all {
			success++
		}
	}
	return float64(success) / float64(len(trialPasses))
}
