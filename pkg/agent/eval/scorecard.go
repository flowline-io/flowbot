package eval

import (
	"math"
	"strings"
)

// DefaultLatencyBudgetMs is the per-trial latency budget for L3 LatencyScore.
const DefaultLatencyBudgetMs int64 = 8000

// DefaultTokenBudget is the per-trial token budget for L3 TokenScore.
const DefaultTokenBudget = 6000

// ScorecardOptions configures L3 budgets when building a scorecard.
type ScorecardOptions struct {
	// LatencyBudgetMs is the per-trial latency budget (default DefaultLatencyBudgetMs).
	LatencyBudgetMs int64
	// TokenBudget is the per-trial token budget (default DefaultTokenBudget).
	TokenBudget int
}

// CapabilityScorecard aggregates L1/L2/L3 layer scores into Total (0–100).
type CapabilityScorecard struct {
	// HardPassRate is Summary.Passed / Summary.Total (0–1).
	HardPassRate float64 `json:"hard_pass_rate"`
	// PassAtK mirrors report pass@k when set (0–1); appendix only.
	PassAtK *float64 `json:"pass_at_k,omitempty"`
	// PassHatK mirrors report pass^k when set (0–1); appendix only.
	PassHatK *float64 `json:"pass_hat_k,omitempty"`
	// Reliability is kept as appendix: 0.6*pass^k + 0.4*pass@k when both set; else HardPassRate.
	Reliability float64 `json:"reliability"`
	// PassAt1 is successful trials / (cases × k) (0–1).
	PassAt1 float64 `json:"pass_at_1"`
	// PassAt1CI is Wilson 95% CI for PassAt1 when trial counts are known.
	PassAt1CI *WilsonCI `json:"pass_at_1_ci,omitempty"`
	// L1Score is compliance rate (0–1).
	L1Score float64 `json:"l1_score"`
	// L2Score is (ToolCallAcc + RepairRate) / 2, or ToolCallAcc when no repair cases.
	L2Score float64 `json:"l2_score"`
	// L3Score is the equal-weight mean of available L3 dimensions.
	L3Score float64 `json:"l3_score"`
	// Total is 100 * (0.2*L1 + 0.5*L2 + 0.3*L3).
	Total float64 `json:"total"`
	// ToolCallAcc is tool hard-gate accuracy among eligible trials (0–1).
	ToolCallAcc *float64 `json:"tool_call_acc,omitempty"`
	// RepairRate is Pass@1 among tier=repair cases (0–1).
	RepairRate *float64 `json:"repair_rate,omitempty"`
	// RepairRateCI is Wilson 95% CI for RepairRate when set.
	RepairRateCI *WilsonCI `json:"repair_rate_ci,omitempty"`
	// LatencyScore is mean clamp(budget/actual) for trials with DurationMs > 0.
	LatencyScore *float64 `json:"latency_score,omitempty"`
	// TokenScore is mean clamp(budget/actual) for trials with TotalTokens > 0.
	TokenScore *float64 `json:"token_score,omitempty"`
	// NaturalRepairRate is appendix: pass rate among trials with ≥1 tool error.
	NaturalRepairRate *float64 `json:"natural_repair_rate,omitempty"`
	// ToolSelectionRate is appendix soft ExpectedTools order match rate.
	ToolSelectionRate *float64 `json:"tool_selection_rate,omitempty"`
	// CorrectnessAvg is mean judge correctness (1–5) when quality scoring is enabled.
	CorrectnessAvg *float64 `json:"correctness_avg,omitempty"`
	// FaithfulnessAvg is mean judge faithfulness (1–5).
	FaithfulnessAvg *float64 `json:"faithfulness_avg,omitempty"`
	// HelpfulnessAvg is mean judge helpfulness (1–5).
	HelpfulnessAvg *float64 `json:"helpfulness_avg,omitempty"`
	// SafetyAvg is mean judge safety (1–5).
	SafetyAvg *float64 `json:"safety_avg,omitempty"`
	// QualityAvg is mean of the four dimensions (1–5); appendix only (not in Total).
	QualityAvg *float64 `json:"quality_avg,omitempty"`
	// JudgeGoldAgreement mirrors report agreement when set (0–1).
	JudgeGoldAgreement *float64 `json:"judge_gold_agreement,omitempty"`
	// QualityEnabled is false for fake/scripted judges.
	QualityEnabled bool `json:"quality_enabled"`
	// TotalCI is set when suite-level repeats ≥ 2 were aggregated externally.
	TotalCI *MeanCI `json:"total_ci,omitempty"`
	// L1CI is suite-repeat CI when set.
	L1CI *MeanCI `json:"l1_ci,omitempty"`
	// L2CI is suite-repeat CI when set.
	L2CI *MeanCI `json:"l2_ci,omitempty"`
	// L3CI is suite-repeat CI when set.
	L3CI *MeanCI `json:"l3_ci,omitempty"`
	// Notes explains how to interpret the scorecard.
	Notes string `json:"notes,omitempty"`
}

// WilsonCI is a Wilson score interval for a Bernoulli proportion.
type WilsonCI struct {
	// Low is the lower bound (0–1).
	Low float64 `json:"low"`
	// High is the upper bound (0–1).
	High float64 `json:"high"`
}

// MeanCI is a normal-approximation interval around a mean of suite repeats.
type MeanCI struct {
	// Mean is the sample mean.
	Mean float64 `json:"mean"`
	// Low is mean − 1.96*SEM.
	Low float64 `json:"low"`
	// High is mean + 1.96*SEM.
	High float64 `json:"high"`
	// N is the number of suite repeats.
	N int `json:"n"`
}

// ScorecardFromReport builds a CapabilityScorecard using budgets stored on the report when set.
func ScorecardFromReport(report EvalReport) CapabilityScorecard {
	return ScorecardFromReportOpts(report, ScorecardOptions{
		LatencyBudgetMs: report.LatencyBudgetMs,
		TokenBudget:     report.TokenBudget,
	})
}

// ScorecardFromReportOpts builds a CapabilityScorecard with explicit budgets.
func ScorecardFromReportOpts(report EvalReport, opts ScorecardOptions) CapabilityScorecard {
	if opts.LatencyBudgetMs <= 0 {
		opts.LatencyBudgetMs = DefaultLatencyBudgetMs
	}
	if opts.TokenBudget <= 0 {
		opts.TokenBudget = DefaultTokenBudget
	}
	sc := CapabilityScorecard{
		PassAtK:            report.PassAtK,
		PassHatK:           report.PassHatK,
		JudgeGoldAgreement: report.JudgeGoldAgreement,
	}
	if report.Summary.Total > 0 {
		sc.HardPassRate = float64(report.Summary.Passed) / float64(report.Summary.Total)
	}
	sc.Reliability = reliabilityScore(sc.HardPassRate, report.PassAtK, report.PassHatK)

	agg := aggregateLayerStats(report.Cases, opts)
	sc.PassAt1 = agg.passAt1
	if agg.passTrials > 0 {
		ci := WilsonInterval(agg.passSuccesses, agg.passTrials)
		sc.PassAt1CI = &ci
	}
	sc.L1Score = agg.l1
	sc.ToolCallAcc = agg.toolCallAcc
	sc.RepairRate = agg.repairRate
	if agg.repairTrials > 0 {
		ci := WilsonInterval(agg.repairSuccesses, agg.repairTrials)
		sc.RepairRateCI = &ci
	}
	sc.LatencyScore = agg.latencyScore
	sc.TokenScore = agg.tokenScore
	sc.NaturalRepairRate = agg.naturalRepair
	sc.ToolSelectionRate = agg.toolSelection

	sc.L2Score = l2FromParts(agg.toolCallAcc, agg.repairRate)
	sc.L3Score = l3FromParts(agg.passAt1, agg.latencyScore, agg.tokenScore)
	sc.Total = round2(100 * (0.2*sc.L1Score + 0.5*sc.L2Score + 0.3*sc.L3Score))

	sc.QualityEnabled = report.JudgeMode != "" && report.JudgeMode != "fake" && hasJudgeScores(report.Cases)
	if sc.QualityEnabled {
		if avgs, ok := meanJudgeDims(report.Cases); ok {
			sc.CorrectnessAvg = &avgs.Correctness
			sc.FaithfulnessAvg = &avgs.Faithfulness
			sc.HelpfulnessAvg = &avgs.Helpfulness
			sc.SafetyAvg = &avgs.Safety
			sc.QualityAvg = &avgs.Quality
		}
	}
	sc.Notes = scorecardNotes(report.JudgeMode, sc)
	return sc
}

func l2FromParts(toolAcc, repair *float64) float64 {
	if toolAcc == nil && repair == nil {
		return 0
	}
	if repair == nil {
		return round2(*toolAcc)
	}
	if toolAcc == nil {
		return round2(*repair)
	}
	return round2((*toolAcc + *repair) / 2)
}

func l3FromParts(passAt1 float64, latency, tokens *float64) float64 {
	sum := passAt1
	n := 1.0
	if latency != nil {
		sum += *latency
		n++
	}
	if tokens != nil {
		sum += *tokens
		n++
	}
	return round2(sum / n)
}

func scorecardNotes(judgeMode string, sc CapabilityScorecard) string {
	base := "total = 100*(0.2*L1+0.5*L2+0.3*L3); L1=compliance; L2=(ToolCallAcc+RepairRate)/2; L3=mean(Pass@1,Latency?,Token?); judge quality is appendix-only"
	if judgeMode == "fake" {
		return base + "; quality disabled (judge-fake)"
	}
	if !sc.QualityEnabled {
		return base + "; no live judge scores"
	}
	return base
}

type layerAgg struct {
	passAt1         float64
	passSuccesses   int
	passTrials      int
	l1              float64
	toolCallAcc     *float64
	repairRate      *float64
	repairSuccesses int
	repairTrials    int
	latencyScore    *float64
	tokenScore      *float64
	naturalRepair   *float64
	toolSelection   *float64
}

type layerCounters struct {
	l1OK, l1N         int
	toolOK, toolN     int
	repairOK, repairN int
	latSum            float64
	latN              int
	tokSum            float64
	tokN              int
	natOK, natN       int
	selOK, selN       int
}

func aggregateLayerStats(cases []CaseResult, opts ScorecardOptions) layerAgg {
	var agg layerAgg
	var ctr layerCounters
	for _, c := range cases {
		for _, m := range trialMetricsForCase(c) {
			accumulateTrial(&agg, &ctr, c.Tier, m, opts)
		}
	}
	finalizeLayerAgg(&agg, ctr)
	return agg
}

func accumulateTrial(agg *layerAgg, ctr *layerCounters, tier string, m Metrics, opts ScorecardOptions) {
	agg.passTrials++
	if m.Passed {
		agg.passSuccesses++
	}
	if m.ComplianceEligible {
		ctr.l1N++
		if m.CompliancePassed {
			ctr.l1OK++
		}
	}
	if m.ToolAccEligible {
		ctr.toolN++
		if m.ToolAccPassed {
			ctr.toolOK++
		}
	}
	if strings.EqualFold(tier, TierRepair) {
		ctr.repairN++
		if m.Passed {
			ctr.repairOK++
		}
	}
	if m.DurationMs > 0 {
		ctr.latSum += clamp01(float64(opts.LatencyBudgetMs) / float64(m.DurationMs))
		ctr.latN++
	}
	if m.TotalTokens > 0 {
		ctr.tokSum += clamp01(float64(opts.TokenBudget) / float64(m.TotalTokens))
		ctr.tokN++
	}
	if m.NaturalRepairEligible {
		ctr.natN++
		if m.NaturalRepairSuccess {
			ctr.natOK++
		}
	}
	if m.ToolSelectionTracked {
		ctr.selN++
		if m.ToolSelectionCorrect {
			ctr.selOK++
		}
	}
}

func finalizeLayerAgg(agg *layerAgg, ctr layerCounters) {
	if agg.passTrials > 0 {
		agg.passAt1 = float64(agg.passSuccesses) / float64(agg.passTrials)
	}
	if ctr.l1N > 0 {
		agg.l1 = float64(ctr.l1OK) / float64(ctr.l1N)
	}
	agg.toolCallAcc = ratePtr(ctr.toolOK, ctr.toolN)
	agg.repairSuccesses = ctr.repairOK
	agg.repairTrials = ctr.repairN
	agg.repairRate = ratePtr(ctr.repairOK, ctr.repairN)
	if ctr.latN > 0 {
		v := round2(ctr.latSum / float64(ctr.latN))
		agg.latencyScore = &v
	}
	if ctr.tokN > 0 {
		v := round2(ctr.tokSum / float64(ctr.tokN))
		agg.tokenScore = &v
	}
	agg.naturalRepair = ratePtr(ctr.natOK, ctr.natN)
	agg.toolSelection = ratePtr(ctr.selOK, ctr.selN)
}

func ratePtr(ok, n int) *float64 {
	if n <= 0 {
		return nil
	}
	v := float64(ok) / float64(n)
	return &v
}

func trialMetricsForCase(c CaseResult) []Metrics {
	if len(c.TrialMetrics) > 0 {
		return c.TrialMetrics
	}
	if len(c.TrialPasses) > 0 {
		out := make([]Metrics, len(c.TrialPasses))
		for i, ok := range c.TrialPasses {
			m := c.Metrics
			m.Passed = ok
			out[i] = m
		}
		return out
	}
	return []Metrics{c.Metrics}
}

// WilsonInterval returns the Wilson score 95% confidence interval for k successes in n trials.
func WilsonInterval(successes, n int) WilsonCI {
	if n <= 0 {
		return WilsonCI{}
	}
	const z = 1.959963984540054 // ~95%
	nf := float64(n)
	phat := float64(successes) / nf
	z2 := z * z
	den := 1 + z2/nf
	center := phat + z2/(2*nf)
	margin := z * math.Sqrt((phat*(1-phat)+z2/(4*nf))/nf)
	return WilsonCI{
		Low:  round4(clamp01((center - margin) / den)),
		High: round4(clamp01((center + margin) / den)),
	}
}

// MeanInterval returns mean ± 1.96*SEM for suite-level repeats (N≥2).
func MeanInterval(values []float64) MeanCI {
	n := len(values)
	if n == 0 {
		return MeanCI{}
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)
	if n < 2 {
		return MeanCI{Mean: round2(mean), Low: round2(mean), High: round2(mean), N: n}
	}
	var ss float64
	for _, v := range values {
		d := v - mean
		ss += d * d
	}
	sem := math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
	margin := 1.96 * sem
	return MeanCI{
		Mean: round2(mean),
		Low:  round2(mean - margin),
		High: round2(mean + margin),
		N:    n,
	}
}

// MergeRepeatScorecards averages layer totals across suite repeats and attaches MeanCI when N≥2.
func MergeRepeatScorecards(cards []CapabilityScorecard) CapabilityScorecard {
	if len(cards) == 0 {
		return CapabilityScorecard{}
	}
	if len(cards) == 1 {
		return cards[0]
	}
	out := cards[0]
	totals := make([]float64, len(cards))
	l1s := make([]float64, len(cards))
	l2s := make([]float64, len(cards))
	l3s := make([]float64, len(cards))
	passAt1s := make([]float64, len(cards))
	for i, c := range cards {
		totals[i] = c.Total
		l1s[i] = c.L1Score
		l2s[i] = c.L2Score
		l3s[i] = c.L3Score
		passAt1s[i] = c.PassAt1
	}
	tCI := MeanInterval(totals)
	l1CI := MeanInterval(l1s)
	l2CI := MeanInterval(l2s)
	l3CI := MeanInterval(l3s)
	passAt1CI := MeanInterval(passAt1s)
	out.Total = tCI.Mean
	out.L1Score = l1CI.Mean
	out.L2Score = l2CI.Mean
	out.L3Score = l3CI.Mean
	out.PassAt1 = passAt1CI.Mean
	out.TotalCI = &tCI
	out.L1CI = &l1CI
	out.L2CI = &l2CI
	out.L3CI = &l3CI
	out.PassAt1CI = nil
	out.RepairRateCI = nil
	out.ToolCallAcc = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.ToolCallAcc })
	out.RepairRate = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.RepairRate })
	out.LatencyScore = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.LatencyScore })
	out.TokenScore = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.TokenScore })
	out.NaturalRepairRate = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.NaturalRepairRate })
	out.ToolSelectionRate = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.ToolSelectionRate })
	out.PassAtK = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.PassAtK })
	out.PassHatK = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.PassHatK })
	out.JudgeGoldAgreement = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.JudgeGoldAgreement })
	out.QualityAvg = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.QualityAvg })
	out.CorrectnessAvg = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.CorrectnessAvg })
	out.FaithfulnessAvg = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.FaithfulnessAvg })
	out.HelpfulnessAvg = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.HelpfulnessAvg })
	out.SafetyAvg = meanPtr(cards, func(c CapabilityScorecard) *float64 { return c.SafetyAvg })
	out.Notes = out.Notes + "; suite repeats N=" + itoa(len(cards))
	return out
}

func meanPtr(cards []CapabilityScorecard, pick func(CapabilityScorecard) *float64) *float64 {
	var sum float64
	var n int
	for _, c := range cards {
		v := pick(c)
		if v == nil {
			continue
		}
		sum += *v
		n++
	}
	if n == 0 {
		return nil
	}
	m := round2(sum / float64(n))
	return &m
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
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

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
