package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/idempotencyrecord"
	"github.com/flowline-io/flowbot/pkg/types"
)

// IdempotencyStore persists write claims.
type IdempotencyStore struct {
	client *gen.Client
}

// IdempotencyRef identifies one claim and the payload hash of this request.
type IdempotencyRef struct {
	TokenID       int64
	Operation     string
	Key           string
	ParamHash     string
	CorrelationID string
}

// NewIdempotencyStore returns a store backed by client.
func NewIdempotencyStore(client *gen.Client) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

// IdempotencyStoreFromDB returns a store using the global database client.
func IdempotencyStoreFromDB() *IdempotencyStore {
	return NewIdempotencyStore(ClientFromDB())
}

// Begin inserts an in-progress claim. created is false when the unique key already has a row.
func (s *IdempotencyStore) Begin(ctx context.Context, ref IdempotencyRef) (*gen.IdempotencyRecord, bool, error) {
	if s == nil || s.client == nil {
		return nil, false, types.Errorf(types.ErrUnavailable, "idempotency store unavailable")
	}
	row, err := s.client.IdempotencyRecord.Create().
		SetTokenID(ref.TokenID).
		SetOperation(ref.Operation).
		SetIdempotencyKey(ref.Key).
		SetParamHash(ref.ParamHash).
		SetStatus(idempotencyrecord.Status(types.IdempotencyInProgress)).
		SetCorrelationID(ref.CorrelationID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err == nil {
		return row, true, nil
	}
	if !gen.IsConstraintError(err) {
		return nil, false, fmt.Errorf("insert idempotency claim: %w", err)
	}
	existing, getErr := s.get(ctx, ref)
	if getErr != nil {
		return nil, false, getErr
	}
	return existing, false, nil
}

// Lookup returns the stored row. found is false when this key has no row.
func (s *IdempotencyStore) Lookup(ctx context.Context, ref IdempotencyRef) (*gen.IdempotencyRecord, bool, error) {
	if s == nil || s.client == nil {
		return nil, false, types.Errorf(types.ErrUnavailable, "idempotency store unavailable")
	}
	row, err := s.get(ctx, ref)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return row, true, nil
}

// Finish marks a claim succeeded and stores the response when it fits maxBytes.
func (s *IdempotencyStore) Finish(ctx context.Context, id int64, result []byte, correlationID string, maxBytes int) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "idempotency store unavailable")
	}
	upd := s.client.IdempotencyRecord.UpdateOneID(id).
		SetStatus(idempotencyrecord.Status(types.IdempotencySucceeded)).
		SetCorrelationID(correlationID)
	if maxBytes > 0 && len(result) > 0 && len(result) <= maxBytes {
		text := string(result)
		upd = upd.SetResultJSON(text)
	}
	if _, err := upd.Save(ctx); err != nil {
		return fmt.Errorf("finish idempotency claim: %w", err)
	}
	return nil
}

// Fail marks a claim unknown so a later call with the same key does not execute again.
func (s *IdempotencyStore) Fail(ctx context.Context, id int64) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "idempotency store unavailable")
	}
	if _, err := s.client.IdempotencyRecord.UpdateOneID(id).
		SetStatus(idempotencyrecord.Status(types.IdempotencyUnknown)).
		Save(ctx); err != nil {
		return fmt.Errorf("mark idempotency claim unknown: %w", err)
	}
	return nil
}

// Release deletes a claim that never started a side effect, so the key can be reused.
func (s *IdempotencyStore) Release(ctx context.Context, id int64) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "idempotency store unavailable")
	}
	if err := s.client.IdempotencyRecord.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("release idempotency claim: %w", err)
	}
	return nil
}

// SweepInProgress moves every in-progress claim to unknown.
func (s *IdempotencyStore) SweepInProgress(ctx context.Context) error {
	if s == nil || s.client == nil {
		return nil
	}
	if _, err := s.client.IdempotencyRecord.Update().
		Where(idempotencyrecord.StatusEQ(idempotencyrecord.Status(types.IdempotencyInProgress))).
		SetStatus(idempotencyrecord.Status(types.IdempotencyUnknown)).
		Save(ctx); err != nil {
		return fmt.Errorf("sweep idempotency claims: %w", err)
	}
	return nil
}

func (s *IdempotencyStore) get(ctx context.Context, ref IdempotencyRef) (*gen.IdempotencyRecord, error) {
	row, err := s.client.IdempotencyRecord.Query().
		Where(
			idempotencyrecord.TokenIDEQ(ref.TokenID),
			idempotencyrecord.OperationEQ(ref.Operation),
			idempotencyrecord.IdempotencyKeyEQ(ref.Key),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("load idempotency claim: %w", err)
	}
	return row, nil
}

// Ready reports whether the store can persist claims.
func (s *IdempotencyStore) Ready() bool {
	return s != nil && s.client != nil
}
