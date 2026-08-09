package eval

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// AnnotateLayerMetrics fills L1/L2/natural-repair fields after hard scoring.
func AnnotateLayerMetrics(m *Metrics, caseName, tier string, tags []string, expect Expectation) {
	if m == nil {
		return
	}
	_ = tier
	m.ToolSelectionTracked = len(expect.ExpectedTools) > 0
	m.NaturalRepairEligible = m.ToolErrorCount >= 1
	m.NaturalRepairSuccess = m.NaturalRepairEligible && m.Passed
	m.ComplianceEligible = complianceEligible(caseName, tags, expect)
	m.CompliancePassed = m.ComplianceEligible && compliancePassed(*m, expect)
	m.ToolAccEligible = toolAccEligible(expect)
	m.ToolAccPassed = m.ToolAccEligible && toolAccPassed(*m, expect)
}

func compliancePassed(m Metrics, expect Expectation) bool {
	if !m.ForbiddenToolsClear {
		return false
	}
	if hasTextGuard(expect) && !m.TextOutcomeOK {
		return false
	}
	if expect.RequireCompletion && !m.Completed {
		return false
	}
	return true
}

func toolAccPassed(m Metrics, expect Expectation) bool {
	if !m.RequiredToolsCovered || !m.ArgsValid {
		return false
	}
	if len(expect.ExpectedTools) == 0 {
		return true
	}
	if expect.StrictToolOrder {
		return m.ToolSelectionCorrect
	}
	if len(expect.RequiredTools) == 0 && len(expect.RequiredArgs) == 0 {
		return m.ToolSelectionCorrect
	}
	return true
}

func complianceEligible(caseName string, tags []string, _ Expectation) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), MetricCompliance) {
			return true
		}
	}
	name := strings.ToLower(caseName)
	if strings.Contains(name, "refuse_") || strings.Contains(name, "admit_") {
		return true
	}
	if strings.Contains(name, "_schema") || strings.Contains(name, "schema_") {
		return true
	}
	return false
}

func hasTextGuard(expect Expectation) bool {
	return len(expect.Outcome.FinalTextContains) > 0 || len(expect.Outcome.FinalTextContainsAny) > 0
}

func toolAccEligible(expect Expectation) bool {
	return len(expect.RequiredTools) > 0 || len(expect.ExpectedTools) > 0 || len(expect.RequiredArgs) > 0
}

// CountToolErrors returns how many tool results in messages have IsError.
func CountToolErrors(messages []msg.AgentMessage) int {
	n := 0
	for _, item := range messages {
		tr, ok := item.(msg.ToolResultMessage)
		if ok && tr.IsError {
			n++
		}
	}
	return n
}

// NormalizeTier maps empty/unknown tier to basic.
func NormalizeTier(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", TierBasic:
		return TierBasic
	case TierCombo:
		return TierCombo
	case TierSystem:
		return TierSystem
	case TierRepair:
		return TierRepair
	default:
		return TierBasic
	}
}
