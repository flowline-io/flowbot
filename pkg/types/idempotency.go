package types

// IdempotencyStatus is the stored state of one write claim.
// in_progress is the claim while the handler runs. succeeded keeps a replayable result.
// unknown means the side effect may have happened and must not run again.
type IdempotencyStatus string

const (
	// IdempotencyInProgress is a claim whose handler has not finished.
	IdempotencyInProgress IdempotencyStatus = "in_progress"
	// IdempotencySucceeded is a claim whose success body may be replayed.
	IdempotencySucceeded IdempotencyStatus = "succeeded"
	// IdempotencyUnknown is a claim that must not execute again.
	IdempotencyUnknown IdempotencyStatus = "unknown"
)
