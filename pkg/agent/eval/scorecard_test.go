package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScorecardL123_weightsAndZeroActual(t *testing.T) {
	t.Parallel()
	report := EvalReport{
		Summary: ReportSummary{Total: 2, Passed: 2},
		Trials:  1,
		Cases: []CaseResult{
			{
				Name:   "openqa_refuse_secrets",
				Tier:   TierBasic,
				Passed: true,
				TrialMetrics: []Metrics{{
					Passed:              true,
					ComplianceEligible:  true,
					CompliancePassed:    true,
					ForbiddenToolsClear: true,
					TextOutcomeOK:       true,
					Completed:           true,
				}},
			},
			{
				Name:   "tools_basic_read",
				Tier:   TierBasic,
				Passed: true,
				TrialMetrics: []Metrics{{
					Passed:          true,
					ToolAccEligible: true,
					ToolAccPassed:   true,
					DurationMs:      0,
					TotalTokens:     0,
				}},
			},
		},
	}
	sc := ScorecardFromReport(report)
	assert.InDelta(t, 1.0, sc.L1Score, 1e-9)
	require.NotNil(t, sc.ToolCallAcc)
	assert.InDelta(t, 1.0, *sc.ToolCallAcc, 1e-9)
	assert.Nil(t, sc.RepairRate)
	assert.InDelta(t, 1.0, sc.L2Score, 1e-9) // ToolCallAcc only
	assert.InDelta(t, 1.0, sc.PassAt1, 1e-9)
	assert.Nil(t, sc.LatencyScore)
	assert.Nil(t, sc.TokenScore)
	assert.InDelta(t, 1.0, sc.L3Score, 1e-9) // Pass@1 only
	assert.InDelta(t, 100.0, sc.Total, 1e-9)
	assert.False(t, sc.QualityEnabled)
	assert.Contains(t, sc.Notes, "0.2*L1")
}

func TestScorecard_repairRateAndBudgets(t *testing.T) {
	t.Parallel()
	report := EvalReport{
		Summary: ReportSummary{Total: 1, Passed: 1},
		Cases: []CaseResult{{
			Name:   "repair_wrong_path_then_read",
			Tier:   TierRepair,
			Passed: true,
			TrialMetrics: []Metrics{
				{
					Passed:                true,
					ToolAccEligible:       true,
					ToolAccPassed:         true,
					ToolErrorCount:        1,
					NaturalRepairEligible: true,
					NaturalRepairSuccess:  true,
					DurationMs:            4000,
					TotalTokens:           3000,
				},
				{
					Passed:          false,
					ToolAccEligible: true,
					ToolAccPassed:   false,
					DurationMs:      16000,
					TotalTokens:     12000,
				},
			},
			TrialPasses: []bool{true, false},
		}},
	}
	sc := ScorecardFromReportOpts(report, ScorecardOptions{
		LatencyBudgetMs: 8000,
		TokenBudget:     6000,
	})
	require.NotNil(t, sc.RepairRate)
	assert.InDelta(t, 0.5, *sc.RepairRate, 1e-9)
	require.NotNil(t, sc.PassAt1CI)
	require.NotNil(t, sc.LatencyScore)
	assert.InDelta(t, 0.75, *sc.LatencyScore, 1e-9) // mean(1.0, 0.5)
	require.NotNil(t, sc.TokenScore)
	assert.InDelta(t, 0.75, *sc.TokenScore, 1e-9) // mean(1.0, 0.5)
	assert.InDelta(t, 0.5, sc.PassAt1, 1e-9)
}

func TestWilsonInterval_bounds(t *testing.T) {
	t.Parallel()
	ci := WilsonInterval(10, 10)
	assert.Greater(t, ci.Low, 0.5)
	assert.LessOrEqual(t, ci.High, 1.0)
	ci0 := WilsonInterval(0, 10)
	assert.GreaterOrEqual(t, ci0.Low, 0.0)
	assert.Less(t, ci0.High, 0.5)
}

func TestMergeRepeatScorecards_ci(t *testing.T) {
	t.Parallel()
	p1, p2 := 0.7, 0.9
	r1, r2 := 0.4, 0.6
	cards := []CapabilityScorecard{
		{Total: 80, L1Score: 1, L2Score: 0.8, L3Score: 0.7, PassAt1: 0.5, ToolCallAcc: &p1, RepairRate: &r1, PassAt1CI: &WilsonCI{Low: 0.1, High: 0.9}},
		{Total: 90, L1Score: 1, L2Score: 0.9, L3Score: 0.8, PassAt1: 0.7, ToolCallAcc: &p2, RepairRate: &r2, PassAt1CI: &WilsonCI{Low: 0.2, High: 0.95}},
	}
	merged := MergeRepeatScorecards(cards)
	assert.InDelta(t, 85.0, merged.Total, 1e-9)
	assert.InDelta(t, 0.6, merged.PassAt1, 1e-9)
	require.NotNil(t, merged.TotalCI)
	assert.Equal(t, 2, merged.TotalCI.N)
	require.NotNil(t, merged.ToolCallAcc)
	assert.InDelta(t, 0.8, *merged.ToolCallAcc, 1e-9)
	require.NotNil(t, merged.RepairRate)
	assert.InDelta(t, 0.5, *merged.RepairRate, 1e-9)
	assert.Nil(t, merged.PassAt1CI)
}

func TestAnnotateLayerMetrics_complianceVsToolAcc(t *testing.T) {
	t.Parallel()
	m := Metrics{
		ForbiddenToolsClear:  true,
		TextOutcomeOK:        true,
		Completed:            true,
		RequiredToolsCovered: true,
		ArgsValid:            true,
		Passed:               true,
	}
	expect := Expectation{
		ForbiddenTools:    []string{"run_terminal"},
		RequiredTools:     []string{"read_file"},
		RequireCompletion: true,
		Outcome:           OutcomeAsserts{FinalTextContainsAny: []string{"cannot"}},
	}
	AnnotateLayerMetrics(&m, "openqa_refuse_secrets", TierBasic, []string{MetricCompliance}, expect)
	assert.True(t, m.ComplianceEligible)
	assert.True(t, m.CompliancePassed)
	assert.True(t, m.ToolAccEligible)
	assert.True(t, m.ToolAccPassed)
}

func TestComplianceEligibleDoesNotAutoIncludeForbiddenToolsOnly(t *testing.T) {
	t.Parallel()
	expect := Expectation{ForbiddenTools: []string{"run_terminal"}}
	assert.False(t, complianceEligible("tools_edit_listen_port", nil, expect))
	assert.True(t, complianceEligible("openqa_refuse_shell", nil, expect))
}
