package eval

import "math"

// CapabilityScorecard aggregates reliability and quality for before/after optimization.
type CapabilityScorecard struct {
	// HardPassRate is Summary.Passed / Summary.Total (0–1).
	HardPassRate float64 `json:"hard_pass_rate"`
	// PassAtK mirrors report pass@k when set (0–1).
	PassAtK *float64 `json:"pass_at_k,omitempty"`
	// PassHatK mirrors report pass^k when set (0–1).
	PassHatK *float64 `json:"pass_hat_k,omitempty"`
	// Reliability is 0.6*pass^k + 0.4*pass@k when both set; else HardPassRate.
	Reliability float64 `json:"reliability"`
	// CorrectnessAvg is mean judge correctness (1–5) when quality scoring is enabled.
	CorrectnessAvg *float64 `json:"correctness_avg,omitempty"`
	// FaithfulnessAvg is mean judge faithfulness (1–5).
	FaithfulnessAvg *float64 `json:"faithfulness_avg,omitempty"`
	// HelpfulnessAvg is mean judge helpfulness (1–5).
	HelpfulnessAvg *float64 `json:"helpfulness_avg,omitempty"`
	// SafetyAvg is mean judge safety (1–5).
	SafetyAvg *float64 `json:"safety_avg,omitempty"`
	// QualityAvg is mean of the four dimensions (1–5).
	QualityAvg *float64 `json:"quality_avg,omitempty"`
	// JudgeGoldAgreement mirrors report agreement when set (0–1).
	JudgeGoldAgreement *float64 `json:"judge_gold_agreement,omitempty"`
	// CapabilityIndex is 0–100 composite for A/B: 50% reliability + 50% quality/5.
	// When quality is unavailable, index is 100 * reliability only.
	CapabilityIndex float64 `json:"capability_index"`
	// QualityEnabled is false for fake/scripted judges (scores omitted from index).
	QualityEnabled bool `json:"quality_enabled"`
	// Notes explains how to interpret the scorecard.
	Notes string `json:"notes,omitempty"`
}

// ScorecardFromReport builds a CapabilityScorecard from an EvalReport.
func ScorecardFromReport(report EvalReport) CapabilityScorecard {
	sc := CapabilityScorecard{
		PassAtK:            report.PassAtK,
		PassHatK:           report.PassHatK,
		JudgeGoldAgreement: report.JudgeGoldAgreement,
	}
	if report.Summary.Total > 0 {
		sc.HardPassRate = float64(report.Summary.Passed) / float64(report.Summary.Total)
	}
	sc.Reliability = reliabilityScore(sc.HardPassRate, report.PassAtK, report.PassHatK)
	sc.QualityEnabled = report.JudgeMode != "" && report.JudgeMode != "fake" && hasJudgeScores(report.Cases)
	if sc.QualityEnabled {
		avgs, ok := meanJudgeDims(report.Cases)
		if ok {
			sc.CorrectnessAvg = &avgs.Correctness
			sc.FaithfulnessAvg = &avgs.Faithfulness
			sc.HelpfulnessAvg = &avgs.Helpfulness
			sc.SafetyAvg = &avgs.Safety
			sc.QualityAvg = &avgs.Quality
			sc.CapabilityIndex = round2(100 * (0.5*sc.Reliability + 0.5*(avgs.Quality/5.0)))
			sc.Notes = "capability_index = 100 * (0.5*reliability + 0.5*quality_avg/5); reliability = 0.6*pass^k + 0.4*pass@k"
			return sc
		}
	}
	sc.CapabilityIndex = round2(100 * sc.Reliability)
	if report.JudgeMode == "fake" {
		sc.Notes = "quality disabled (judge-fake); capability_index = 100 * reliability only. Re-run with --judge-model X --judge-fake=false for dimension scores."
	} else {
		sc.Notes = "no live judge scores; capability_index = 100 * reliability only"
	}
	return sc
}

func reliabilityScore(hardPass float64, passAt, passHat *float64) float64 {
	if passAt != nil && passHat != nil {
		return round2(0.6*(*passHat) + 0.4*(*passAt))
	}
	return round2(hardPass)
}

func hasJudgeScores(cases []CaseResult) bool {
	for _, c := range cases {
		if c.Judge != nil && !c.Judge.Unknown {
			return true
		}
	}
	return false
}

type judgeAverages struct {
	Correctness, Faithfulness, Helpfulness, Safety, Quality float64
}

func meanJudgeDims(cases []CaseResult) (judgeAverages, bool) {
	var n int
	var sumC, sumF, sumH, sumS float64
	for _, item := range cases {
		if item.Judge == nil || item.Judge.Unknown {
			continue
		}
		if item.Judge.Correctness <= 0 && item.Judge.Faithfulness <= 0 &&
			item.Judge.Helpfulness <= 0 && item.Judge.Safety <= 0 {
			continue
		}
		sumC += float64(item.Judge.Correctness)
		sumF += float64(item.Judge.Faithfulness)
		sumH += float64(item.Judge.Helpfulness)
		sumS += float64(item.Judge.Safety)
		n++
	}
	if n == 0 {
		return judgeAverages{}, false
	}
	fn := float64(n)
	c, f, h, s := sumC/fn, sumF/fn, sumH/fn, sumS/fn
	q := (c + f + h + s) / 4
	return judgeAverages{
		Correctness: round2(c), Faithfulness: round2(f),
		Helpfulness: round2(h), Safety: round2(s), Quality: round2(q),
	}, true
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
