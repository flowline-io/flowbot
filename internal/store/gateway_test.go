package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestGatewayStoreCreateClaimComplete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := sqlitetest.OpenClient(t, "gateway_claim")
	s := NewGatewayStore(client)

	job, err := s.Create(ctx, types.GatewayCreateJob{
		UID: "u1", CLI: types.GatewayCLICursor, Prompt: "fix tests",
	})
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, types.GatewayJobPending, job.Status)

	claimed, err := s.Claim(ctx, "worker-a", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, job.JobID, claimed.JobID)
	assert.Equal(t, types.GatewayJobRunning, claimed.Status)
	assert.Equal(t, "worker-a", claimed.WorkerID)

	empty, err := s.Claim(ctx, "worker-b", time.Minute)
	require.NoError(t, err)
	assert.Nil(t, empty)

	code := 0
	done, err := s.Complete(ctx, job.JobID, types.GatewayCompleteRequest{
		WorkerID: "worker-a", Status: types.GatewayJobSucceeded, Output: "ok", ExitCode: &code, DurationMs: 12,
	}, 100)
	require.NoError(t, err)
	require.NotNil(t, done)
	assert.Equal(t, types.GatewayJobSucceeded, done.Status)
	assert.Equal(t, "ok", done.Output)
	assert.Equal(t, int64(12), done.DurationMs)
}

func TestGatewayStoreRejectOpenCode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := NewGatewayStore(sqlitetest.OpenClient(t, "gateway_opencode"))
	_, err := s.Create(ctx, types.GatewayCreateJob{CLI: types.GatewayCLIOpenCode, Prompt: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidArgument)
}

func TestGatewayStoreCancelAndLeaseReclaim(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := NewGatewayStore(sqlitetest.OpenClient(t, "gateway_lease"))

	job, err := s.Create(ctx, types.GatewayCreateJob{CLI: types.GatewayCLICursor, Prompt: "long"})
	require.NoError(t, err)
	claimed, err := s.Claim(ctx, "w1", time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, claimed)

	time.Sleep(5 * time.Millisecond)
	require.NoError(t, s.ReclaimExpired(ctx))
	got, err := s.Get(ctx, job.JobID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, types.GatewayJobPending, got.Status)
	assert.Empty(t, got.WorkerID)

	reclaimed, err := s.Claim(ctx, "w2", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, reclaimed)
	assert.Equal(t, job.JobID, reclaimed.JobID)
	assert.Equal(t, "w2", reclaimed.WorkerID)

	job2, err := s.Create(ctx, types.GatewayCreateJob{CLI: types.GatewayCLICursor, Prompt: "cancel me"})
	require.NoError(t, err)
	canceled, err := s.Cancel(ctx, job2.JobID)
	require.NoError(t, err)
	assert.Equal(t, types.GatewayJobCanceled, canceled.Status)
}

func TestGatewayStoreFreshWorker(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := NewGatewayStore(sqlitetest.OpenClient(t, "gateway_worker"))

	fresh, err := s.HasFreshWorker(ctx, time.Minute)
	require.NoError(t, err)
	assert.False(t, fresh)

	require.NoError(t, s.TouchWorker(ctx, "w1"))
	fresh, err = s.HasFreshWorker(ctx, time.Minute)
	require.NoError(t, err)
	assert.True(t, fresh)
}

func TestTruncateBytes(t *testing.T) {
	t.Parallel()
	got, trunc := truncateBytes("hi", 0)
	assert.Equal(t, "hi", got)
	assert.False(t, trunc)
	got, trunc = truncateBytes("abcdef", 5)
	assert.Equal(t, "ab...", got)
	assert.True(t, trunc)
}

func TestGatewayStoreCompleteTruncated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := NewGatewayStore(sqlitetest.OpenClient(t, "gateway_trunc"))
	job, err := s.Create(ctx, types.GatewayCreateJob{CLI: types.GatewayCLICursor, Prompt: "p"})
	require.NoError(t, err)
	_, err = s.Claim(ctx, "w1", time.Minute)
	require.NoError(t, err)
	code := 0
	done, err := s.Complete(ctx, job.JobID, types.GatewayCompleteRequest{
		WorkerID: "w1", Status: types.GatewayJobSucceeded, Output: "abcdef", ExitCode: &code,
	}, 5)
	require.NoError(t, err)
	assert.True(t, done.Truncated)
	assert.Equal(t, "ab...", done.Output)
}
