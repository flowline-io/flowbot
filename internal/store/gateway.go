// Package store provides database storage implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/gatewayjob"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/gatewayworker"
	"github.com/flowline-io/flowbot/pkg/types"
)

const maxClaimAttempts = 8

// GatewayStore persists local-CLI gateway jobs and worker heartbeats.
type GatewayStore struct {
	client *gen.Client
}

// NewGatewayStore creates a GatewayStore with the given ent client.
func NewGatewayStore(client *gen.Client) *GatewayStore {
	return &GatewayStore{client: client}
}

// GatewayStoreFromDB returns a GatewayStore using the global database client.
func GatewayStoreFromDB() *GatewayStore {
	return NewGatewayStore(ClientFromDB())
}

func (s *GatewayStore) ready() bool {
	return s != nil && s.client != nil
}

// Create inserts a pending gateway job and returns its view.
func (s *GatewayStore) Create(ctx context.Context, in types.GatewayCreateJob) (*types.GatewayJob, error) {
	if !s.ready() {
		return nil, types.ErrUnavailable
	}
	cli := types.GatewayCLI(strings.TrimSpace(string(in.CLI)))
	if cli == "" {
		cli = types.GatewayCLICursor
	}
	if cli != types.GatewayCLICursor {
		return nil, types.Errorf(types.ErrInvalidArgument, "unsupported gateway cli %q", cli)
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "prompt is required")
	}
	jobID := types.Id()
	row, err := s.client.GatewayJob.Create().
		SetJobID(jobID).
		SetUID(strings.TrimSpace(in.UID)).
		SetCli(string(cli)).
		SetPrompt(prompt).
		SetCwd(strings.TrimSpace(in.Cwd)).
		SetStatus(string(types.GatewayJobPending)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("gateway: create job: %w", err)
	}
	return gatewayJobFromEnt(row), nil
}

// Get returns a job by job_id, or nil when not found.
func (s *GatewayStore) Get(ctx context.Context, jobID string) (*types.GatewayJob, error) {
	if !s.ready() {
		return nil, types.ErrUnavailable
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "job_id is required")
	}
	row, err := s.client.GatewayJob.Query().Where(gatewayjob.JobIDEQ(jobID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("gateway: get job: %w", err)
	}
	return gatewayJobFromEnt(row), nil
}

// Claim atomically takes the oldest pending job for workerID with a lease.
func (s *GatewayStore) Claim(ctx context.Context, workerID string, leaseTTL time.Duration) (*types.GatewayJob, error) {
	if !s.ready() {
		return nil, types.ErrUnavailable
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "worker_id is required")
	}
	if leaseTTL <= 0 {
		leaseTTL = 90 * time.Second
	}
	if err := s.ReclaimExpired(ctx); err != nil {
		return nil, err
	}
	if err := s.TouchWorker(ctx, workerID); err != nil {
		return nil, err
	}
	return s.claimInTx(ctx, workerID, leaseTTL)
}

func (s *GatewayStore) claimInTx(ctx context.Context, workerID string, leaseTTL time.Duration) (*types.GatewayJob, error) {
	for range maxClaimAttempts {
		job, lost, err := s.tryClaimOnce(ctx, workerID, leaseTTL)
		if err != nil {
			return nil, err
		}
		if !lost {
			return job, nil
		}
	}
	return nil, nil
}

func (s *GatewayStore) tryClaimOnce(ctx context.Context, workerID string, leaseTTL time.Duration) (*types.GatewayJob, bool, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("gateway: begin claim tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	row, err := tx.GatewayJob.Query().
		Where(gatewayjob.StatusEQ(string(types.GatewayJobPending))).
		Order(gen.Asc(gatewayjob.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			if err := tx.Commit(); err != nil {
				return nil, false, fmt.Errorf("gateway: commit empty claim: %w", err)
			}
			committed = true
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("gateway: query pending: %w", err)
	}

	now := time.Now()
	n, err := tx.GatewayJob.Update().
		Where(gatewayjob.IDEQ(row.ID), gatewayjob.StatusEQ(string(types.GatewayJobPending))).
		SetStatus(string(types.GatewayJobRunning)).
		SetWorkerID(workerID).
		SetClaimedAt(now).
		SetLeaseUntil(now.Add(leaseTTL)).
		Save(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("gateway: claim update: %w", err)
	}
	if n == 0 {
		if err := tx.Commit(); err != nil {
			return nil, false, fmt.Errorf("gateway: commit lost claim: %w", err)
		}
		committed = true
		return nil, true, nil
	}
	updated, err := tx.GatewayJob.Get(ctx, row.ID)
	if err != nil {
		return nil, false, fmt.Errorf("gateway: reload claimed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("gateway: commit claim: %w", err)
	}
	committed = true
	return gatewayJobFromEnt(updated), false, nil
}

// Complete writes a terminal result for a running (or already canceled) job.
func (s *GatewayStore) Complete(ctx context.Context, jobID string, in types.GatewayCompleteRequest, maxOutputBytes int) (*types.GatewayJob, error) {
	if !s.ready() {
		return nil, types.ErrUnavailable
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "job_id is required")
	}
	if in.Status != types.GatewayJobSucceeded && in.Status != types.GatewayJobFailed {
		return nil, types.Errorf(types.ErrInvalidArgument, "complete status must be succeeded or failed")
	}
	row, err := s.client.GatewayJob.Query().Where(gatewayjob.JobIDEQ(jobID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.Errorf(types.ErrNotFound, "job not found")
		}
		return nil, fmt.Errorf("gateway: get for complete: %w", err)
	}
	if err := validateCompleteRow(row, in.WorkerID); err != nil {
		if errors.Is(err, errAlreadyCanceled) {
			return gatewayJobFromEnt(row), nil
		}
		return nil, err
	}
	return s.applyComplete(ctx, row, in, maxOutputBytes)
}

var errAlreadyCanceled = errors.New("already canceled")

func validateCompleteRow(row *gen.GatewayJob, workerID string) error {
	cur := types.GatewayJobStatus(row.Status)
	if cur == types.GatewayJobCanceled {
		return errAlreadyCanceled
	}
	if cur != types.GatewayJobRunning {
		return types.Errorf(types.ErrConflict, "job is %s", cur)
	}
	if wid := strings.TrimSpace(workerID); wid != "" && row.WorkerID != "" && row.WorkerID != wid {
		return types.Errorf(types.ErrForbidden, "job claimed by another worker")
	}
	return nil
}

func (s *GatewayStore) applyComplete(ctx context.Context, row *gen.GatewayJob, in types.GatewayCompleteRequest, maxOutputBytes int) (*types.GatewayJob, error) {
	now := time.Now()
	out, truncated := truncateBytes(in.Output, maxOutputBytes)
	upd := s.client.GatewayJob.Update().
		Where(gatewayjob.IDEQ(row.ID), gatewayjob.StatusEQ(string(types.GatewayJobRunning))).
		SetStatus(string(in.Status)).
		SetOutput(out).
		SetTruncated(truncated).
		SetErrorText(strings.TrimSpace(in.Error)).
		SetDurationMs(in.DurationMs).
		SetFinishedAt(now).
		ClearLeaseUntil()
	if strings.TrimSpace(in.Cwd) != "" {
		upd.SetCwd(strings.TrimSpace(in.Cwd))
	}
	if in.ExitCode != nil {
		upd.SetExitCode(*in.ExitCode)
	}
	n, err := upd.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("gateway: complete: %w", err)
	}
	if n == 0 {
		again, gerr := s.Get(ctx, row.JobID)
		if gerr != nil {
			return nil, gerr
		}
		if again != nil && again.Status == types.GatewayJobCanceled {
			return again, nil
		}
		return nil, types.Errorf(types.ErrConflict, "job no longer running")
	}
	return s.Get(ctx, row.JobID)
}

// Cancel marks a non-terminal job as canceled.
func (s *GatewayStore) Cancel(ctx context.Context, jobID string) (*types.GatewayJob, error) {
	if !s.ready() {
		return nil, types.ErrUnavailable
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "job_id is required")
	}
	row, err := s.client.GatewayJob.Query().Where(gatewayjob.JobIDEQ(jobID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.Errorf(types.ErrNotFound, "job not found")
		}
		return nil, fmt.Errorf("gateway: get for cancel: %w", err)
	}
	if types.GatewayJobTerminal(types.GatewayJobStatus(row.Status)) {
		return gatewayJobFromEnt(row), nil
	}
	now := time.Now()
	_, err = s.client.GatewayJob.UpdateOneID(row.ID).
		SetStatus(string(types.GatewayJobCanceled)).
		SetFinishedAt(now).
		ClearLeaseUntil().
		SetErrorText("canceled").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("gateway: cancel: %w", err)
	}
	return s.Get(ctx, jobID)
}

// TouchWorker upserts worker last-seen and optionally renews a running job lease.
func (s *GatewayStore) TouchWorker(ctx context.Context, workerID string) error {
	return s.TouchWorkerLease(ctx, workerID, "", 0)
}

// TouchWorkerLease updates worker last-seen and renews lease for jobID when set.
func (s *GatewayStore) TouchWorkerLease(ctx context.Context, workerID, jobID string, leaseTTL time.Duration) error {
	if !s.ready() {
		return types.ErrUnavailable
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return types.Errorf(types.ErrInvalidArgument, "worker_id is required")
	}
	if err := s.ReclaimExpired(ctx); err != nil {
		return err
	}
	now := time.Now()
	err := s.client.GatewayWorker.Create().
		SetWorkerID(workerID).
		SetLastSeenAt(now).
		OnConflictColumns(gatewayworker.FieldWorkerID).
		UpdateLastSeenAt().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("gateway: touch worker: %w", err)
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil
	}
	if leaseTTL <= 0 {
		leaseTTL = 90 * time.Second
	}
	_, err = s.client.GatewayJob.Update().
		Where(
			gatewayjob.JobIDEQ(jobID),
			gatewayjob.StatusEQ(string(types.GatewayJobRunning)),
			gatewayjob.WorkerIDEQ(workerID),
		).
		SetLeaseUntil(now.Add(leaseTTL)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("gateway: renew lease: %w", err)
	}
	return nil
}

// HasFreshWorker reports whether any worker heartbeated within staleAfter.
func (s *GatewayStore) HasFreshWorker(ctx context.Context, staleAfter time.Duration) (bool, error) {
	if !s.ready() {
		return false, types.ErrUnavailable
	}
	if err := s.ReclaimExpired(ctx); err != nil {
		return false, err
	}
	if staleAfter <= 0 {
		staleAfter = 60 * time.Second
	}
	cutoff := time.Now().Add(-staleAfter)
	n, err := s.client.GatewayWorker.Query().
		Where(gatewayworker.LastSeenAtGTE(cutoff)).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("gateway: fresh worker: %w", err)
	}
	return n > 0, nil
}

// ReclaimExpired returns running jobs with expired leases to pending for re-claim.
func (s *GatewayStore) ReclaimExpired(ctx context.Context) error {
	if !s.ready() {
		return types.ErrUnavailable
	}
	now := time.Now()
	_, err := s.client.GatewayJob.Update().
		Where(
			gatewayjob.StatusEQ(string(types.GatewayJobRunning)),
			gatewayjob.LeaseUntilNotNil(),
			gatewayjob.LeaseUntilLT(now),
		).
		SetStatus(string(types.GatewayJobPending)).
		SetWorkerID("").
		ClearClaimedAt().
		ClearLeaseUntil().
		ClearFinishedAt().
		SetErrorText("").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("gateway: reclaim expired: %w", err)
	}
	return nil
}

func gatewayJobFromEnt(row *gen.GatewayJob) *types.GatewayJob {
	if row == nil {
		return nil
	}
	j := &types.GatewayJob{
		JobID:      row.JobID,
		UID:        row.UID,
		CLI:        types.GatewayCLI(row.Cli),
		Prompt:     row.Prompt,
		Cwd:        row.Cwd,
		Status:     types.GatewayJobStatus(row.Status),
		Output:     row.Output,
		Truncated:  row.Truncated,
		Error:      row.ErrorText,
		DurationMs: row.DurationMs,
		WorkerID:   row.WorkerID,
		CreatedAt:  row.CreatedAt,
		ClaimedAt:  row.ClaimedAt,
		FinishedAt: row.FinishedAt,
		LeaseUntil: row.LeaseUntil,
	}
	if row.ExitCode != nil {
		code := *row.ExitCode
		j.ExitCode = &code
	}
	return j
}

func truncateBytes(s string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s, false
	}
	if maxBytes < 3 {
		return s[:maxBytes], true
	}
	cut := maxBytes - 3
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "...", true
}
