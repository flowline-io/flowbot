package dcg

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
)

// Decision is the outcome of a dcg command evaluation.
type Decision struct {
	// Allow is true when dcg permits the command.
	Allow bool
	// Reason explains a denial when Allow is false.
	Reason string
	// RuleID is the matching dcg rule identifier when present.
	RuleID string
	// PackID is the matching dcg pack identifier when present.
	PackID string
}

type robotOutput struct {
	Command  string `json:"command"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	RuleID   string `json:"rule_id"`
	PackID   string `json:"pack_id"`
}

// parseRobotDecision maps dcg --robot test stdout and exit code to a Decision.
func parseRobotDecision(stdout string, exitCode int) (Decision, error) {
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return Decision{}, fmt.Errorf("dcg: empty robot output (exit %d)", exitCode)
	}
	var out robotOutput
	if err := sonic.Unmarshal([]byte(stdout), &out); err != nil {
		return Decision{}, fmt.Errorf("dcg: parse robot json: %w", err)
	}
	decision := strings.ToLower(strings.TrimSpace(out.Decision))
	switch exitCode {
	case 0:
		if decision != "allow" && decision != "" {
			return Decision{}, fmt.Errorf("dcg: exit 0 with decision %q", out.Decision)
		}
		return Decision{Allow: true}, nil
	case 1:
		if decision != "deny" && decision != "block" {
			return Decision{}, fmt.Errorf("dcg: exit 1 with decision %q", out.Decision)
		}
		reason := strings.TrimSpace(out.Reason)
		if reason == "" {
			reason = ReasonBlocked
		}
		return Decision{
			Allow:  false,
			Reason: reason,
			RuleID: out.RuleID,
			PackID: out.PackID,
		}, nil
	default:
		return Decision{}, fmt.Errorf("dcg: unexpected exit code %d", exitCode)
	}
}
