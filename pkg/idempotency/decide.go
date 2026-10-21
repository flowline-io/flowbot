package idempotency

import "github.com/flowline-io/flowbot/pkg/types"

// Outcome is what a caller does with an existing or new claim.
type Outcome int

const (
	// OutcomeRun means this call owns the claim and must execute the write.
	OutcomeRun Outcome = iota
	// OutcomeReplay means return the stored success body.
	OutcomeReplay
	// OutcomeInProgress means the first call is still executing.
	OutcomeInProgress
	// OutcomeUnknown means the side effect may have happened.
	OutcomeUnknown
	// OutcomeMismatch means this key was stored with a different payload hash.
	OutcomeMismatch
)

// Claim is the product view of one stored row for this request.
type Claim struct {
	ID            int64
	Outcome       Outcome
	ResultJSON    string
	CorrelationID string
}

// Decide selects an outcome from a stored claim and the hash of this request.
// retained is false when a success body was too large to store.
func Decide(status, storedHash, wantHash string, retained bool) Outcome {
	if storedHash != wantHash {
		return OutcomeMismatch
	}
	switch types.IdempotencyStatus(status) {
	case types.IdempotencyInProgress:
		return OutcomeInProgress
	case types.IdempotencySucceeded:
		if !retained {
			return OutcomeUnknown
		}
		return OutcomeReplay
	default:
		return OutcomeUnknown
	}
}
