package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/idempotencyrecord"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/idempotency"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestIdempotencyClaimReplayAndSweep(t *testing.T) {
	client := sqlitetest.OpenClient(t, "idempotency")
	s := NewIdempotencyStore(client)
	ctx := t.Context()
	ref := IdempotencyRef{TokenID: 7, Operation: "drone.deploy", Key: "key-1", ParamHash: "hash-a", CorrelationID: "cor_1"}

	assert.Equal(t, string(types.IdempotencyInProgress), string(idempotencyrecord.StatusInProgress))

	row, created, err := s.Begin(ctx, ref)
	require.NoError(t, err)
	first := IdempotencyView(row, ref.ParamHash, created)
	assert.Equal(t, idempotency.OutcomeRun, first.Outcome)
	assert.Equal(t, "cor_1", first.CorrelationID)

	againRow, created, err := s.Begin(ctx, ref)
	require.NoError(t, err)
	again := IdempotencyView(againRow, ref.ParamHash, created)
	assert.Equal(t, idempotency.OutcomeInProgress, again.Outcome)
	assert.Equal(t, "cor_1", again.CorrelationID)

	require.NoError(t, s.Finish(ctx, first.ID, []byte(`{"id":"1"}`), "cor_1", 100))
	lookupRow, found, err := s.Lookup(ctx, ref)
	require.NoError(t, err)
	require.True(t, found)
	lookup := IdempotencyView(lookupRow, ref.ParamHash, false)
	assert.Equal(t, idempotency.OutcomeReplay, lookup.Outcome)
	assert.JSONEq(t, `{"id":"1"}`, lookup.ResultJSON)
	assert.Equal(t, "cor_1", lookup.CorrelationID)

	mismatchRef := ref
	mismatchRef.ParamHash = "hash-b"
	mismatchRow, found, err := s.Lookup(ctx, mismatchRef)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, idempotency.OutcomeMismatch, IdempotencyView(mismatchRow, mismatchRef.ParamHash, false).Outcome)

	require.NoError(t, s.Fail(ctx, first.ID))
	unknownRow, found, err := s.Lookup(ctx, ref)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, idempotency.OutcomeUnknown, IdempotencyView(unknownRow, ref.ParamHash, false).Outcome)

	second := IdempotencyRef{TokenID: 7, Operation: "drone.deploy", Key: "key-2", ParamHash: "hash-a", CorrelationID: "cor_2"}
	secondRow, created, err := s.Begin(ctx, second)
	require.NoError(t, err)
	assert.True(t, created)
	require.NoError(t, s.SweepInProgress(ctx))
	swept, found, err := s.Lookup(ctx, second)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, idempotency.OutcomeUnknown, IdempotencyView(swept, second.ParamHash, false).Outcome)
	assert.Equal(t, secondRow.ID, swept.ID)
}
