package approval

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
)

// Verdict is the aux reviewer classification.
type Verdict string

const (
	// VerdictApprove silently allows the tool call.
	VerdictApprove Verdict = "APPROVE"
	// VerdictDeny blocks the tool call and injects reason to the agent.
	VerdictDeny Verdict = "DENY"
	// VerdictEscalate asks a human via ConfirmGate.
	VerdictEscalate Verdict = "ESCALATE"
)

// MaxReasonChars caps sanitized reviewer reasons injected into tool errors.
const MaxReasonChars = 200

// ReviewResult is a parsed aux reviewer response.
type ReviewResult struct {
	Verdict Verdict
	Reason  string
}

type reviewJSON struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// ParseReviewOutput parses and validates aux LLM JSON output.
func ParseReviewOutput(raw string) (ReviewResult, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ReviewResult{}, fmt.Errorf("approval: empty reviewer output")
	}
	// Allow fenced JSON or leading/trailing prose by extracting the first object.
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var parsed reviewJSON
	if err := sonic.Unmarshal([]byte(raw), &parsed); err != nil {
		return ReviewResult{}, fmt.Errorf("approval: parse reviewer json: %w", err)
	}
	verdict := Verdict(strings.ToUpper(strings.TrimSpace(parsed.Verdict)))
	switch verdict {
	case VerdictApprove, VerdictDeny, VerdictEscalate:
	default:
		return ReviewResult{}, fmt.Errorf("approval: invalid verdict %q", parsed.Verdict)
	}
	return ReviewResult{
		Verdict: verdict,
		Reason:  SanitizeReason(parsed.Reason),
	}, nil
}

// SanitizeReason trims, strips instruction-like prefixes, and caps length.
func SanitizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	reason = strings.ReplaceAll(reason, "\n", " ")
	reason = strings.Join(strings.Fields(reason), " ")
	lower := strings.ToLower(reason)
	for _, prefix := range []string{
		"ignore previous",
		"ignore all",
		"system:",
		"assistant:",
	} {
		if strings.HasPrefix(lower, prefix) {
			reason = strings.TrimSpace(reason[len(prefix):])
			lower = strings.ToLower(reason)
		}
	}
	if utf8.RuneCountInString(reason) <= MaxReasonChars {
		return reason
	}
	runes := []rune(reason)
	return string(runes[:MaxReasonChars])
}
