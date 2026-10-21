package store

import (
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/idempotency"
)

// IdempotencyView maps a stored row onto the outcome for this request.
// created is true only for the call that inserted the row.
func IdempotencyView(row *gen.IdempotencyRecord, wantHash string, created bool) idempotency.Claim {
	if row == nil {
		return idempotency.Claim{}
	}
	if created {
		return idempotency.Claim{
			ID:            row.ID,
			Outcome:       idempotency.OutcomeRun,
			CorrelationID: row.CorrelationID,
		}
	}
	retained := row.ResultJSON != nil && *row.ResultJSON != ""
	claim := idempotency.Claim{
		ID:            row.ID,
		Outcome:       idempotency.Decide(string(row.Status), row.ParamHash, wantHash, retained),
		CorrelationID: row.CorrelationID,
	}
	if retained {
		claim.ResultJSON = *row.ResultJSON
	}
	return claim
}
