package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/functiondefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/functiondefinitionversion"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/functionrun"
	"github.com/flowline-io/flowbot/pkg/types"
)

// FunctionStore persists named function definitions, published versions, and runs.
type FunctionStore struct {
	client *gen.Client
}

// NewFunctionStore returns a FunctionStore backed by the given ent client.
func NewFunctionStore(client *gen.Client) *FunctionStore {
	return &FunctionStore{client: client}
}

// FunctionStoreFromDB returns a FunctionStore using the global database client.
func FunctionStoreFromDB() *FunctionStore {
	return NewFunctionStore(ClientFromDB())
}

// Create creates a new function definition in draft status at version 1 with draft fields set.
// createdBy is the Web UI user UID that created the function (may be empty in tests).
func (s *FunctionStore) Create(ctx context.Context, name, metadata, entrypoint, source, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	_, err := s.client.FunctionDefinition.Create().
		SetName(name).
		SetMetadataDraft(metadata).
		SetEntrypointDraft(entrypoint).
		SetSourceDraft(source).
		SetNillableMetadataPublished(nil).
		SetNillableEntrypointPublished(nil).
		SetNillableSourcePublished(nil).
		SetVersion(1).
		SetStatus(functiondefinition.StatusDraft).
		SetCreatedBy(strings.TrimSpace(createdBy)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if gen.IsConstraintError(err) {
			return fmt.Errorf("function %q %w", name, types.ErrAlreadyExists)
		}
		return err
	}
	return nil
}

// CreateDefinition creates an empty draft definition (tests / callers that fill draft later).
func (s *FunctionStore) CreateDefinition(ctx context.Context, name, createdBy string) error {
	return s.Create(ctx, name, "", "", "", createdBy)
}

// GetDefinitionByName returns a function definition by name.
func (s *FunctionStore) GetDefinitionByName(ctx context.Context, name string) (*gen.FunctionDefinition, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.FunctionDefinition.Query().
		Where(functiondefinition.Name(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// UpdateDefinitionDraft updates draft metadata/entrypoint/source with optimistic locking.
// Uses conditional UPDATE WHERE version=expectedVersion. Returns ErrConflict if no row matched.
func (s *FunctionStore) UpdateDefinitionDraft(ctx context.Context, name, metadata, entrypoint, source string, expectedVersion int) (*gen.FunctionDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	n, err := s.client.FunctionDefinition.Update().
		Where(
			functiondefinition.Name(name),
			functiondefinition.Version(expectedVersion),
		).
		SetMetadataDraft(metadata).
		SetEntrypointDraft(entrypoint).
		SetSourceDraft(source).
		SetVersion(expectedVersion + 1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}
	return s.GetDefinitionByName(ctx, name)
}

// PublishDefinition copies draft fields to published with optimistic locking and inserts a version snapshot.
func (s *FunctionStore) PublishDefinition(ctx context.Context, name string, version int) (*gen.FunctionDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	def, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.MetadataDraft == "" || def.EntrypointDraft == "" || def.SourceDraft == "" {
		return nil, types.ErrConflict
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("publish function begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	n, err := tx.FunctionDefinition.Update().
		Where(
			functiondefinition.Name(name),
			functiondefinition.Version(version),
		).
		SetMetadataPublished(def.MetadataDraft).
		SetEntrypointPublished(def.EntrypointDraft).
		SetSourcePublished(def.SourceDraft).
		SetVersion(version + 1).
		SetStatus(functiondefinition.StatusPublished).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}

	if _, err := tx.FunctionDefinitionVersion.Create().
		SetFunctionName(name).
		SetVersion(version + 1).
		SetMetadata(def.MetadataDraft).
		SetEntrypoint(def.EntrypointDraft).
		SetSource(def.SourceDraft).
		SetCreatedAt(time.Now()).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("publish: insert version snapshot: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("publish function commit: %w", err)
	}
	committed = true
	return s.GetDefinitionByName(ctx, name)
}

// DeleteDefinitionByName removes a function definition, its version snapshots, and runs.
// Returns the number of runs deleted.
func (s *FunctionStore) DeleteDefinitionByName(ctx context.Context, name string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	runCount, err := s.client.FunctionRun.Delete().
		Where(functionrun.FunctionName(name)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete runs for %s: %w", name, err)
	}
	if _, err := s.client.FunctionDefinitionVersion.Delete().
		Where(functiondefinitionversion.FunctionName(name)).
		Exec(ctx); err != nil {
		return int64(runCount), fmt.Errorf("delete versions for %s: %w", name, err)
	}
	n, err := s.client.FunctionDefinition.Delete().
		Where(functiondefinition.Name(name)).
		Exec(ctx)
	if err != nil {
		return int64(runCount), fmt.Errorf("delete definition %s: %w", name, err)
	}
	if n == 0 {
		return int64(runCount), types.ErrNotFound
	}
	return int64(runCount), nil
}

// ListPublishedDefinitions returns published function definitions ordered by name.
func (s *FunctionStore) ListPublishedDefinitions(ctx context.Context) ([]*gen.FunctionDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.FunctionDefinition.Query().
		Where(
			functiondefinition.StatusEQ(functiondefinition.StatusPublished),
			functiondefinition.SourcePublishedNotNil(),
		).
		Order(gen.Asc(functiondefinition.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published function definitions: %w", err)
	}
	return rows, nil
}

// ListAllDefinitions returns all function definitions ordered by name.
func (s *FunctionStore) ListAllDefinitions(ctx context.Context) ([]*gen.FunctionDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.FunctionDefinition.Query().
		Order(gen.Asc(functiondefinition.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all function definitions: %w", err)
	}
	return rows, nil
}

// GetPublishedVersion returns a published version snapshot by function name and version number.
func (s *FunctionStore) GetPublishedVersion(ctx context.Context, name string, version int) (*gen.FunctionDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	row, err := s.client.FunctionDefinitionVersion.Query().
		Where(
			functiondefinitionversion.FunctionName(name),
			functiondefinitionversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

// GetLatestPublished returns the newest published version snapshot for a function name.
func (s *FunctionStore) GetLatestPublished(ctx context.Context, name string) (*gen.FunctionDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	row, err := s.client.FunctionDefinitionVersion.Query().
		Where(functiondefinitionversion.FunctionName(name)).
		Order(gen.Desc(functiondefinitionversion.FieldVersion)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

// CreateRun inserts a new function run. Empty idempotencyKey is stored as NULL so empty keys do not collide.
func (s *FunctionStore) CreateRun(ctx context.Context, functionName string, version int, status string, idempotencyKey string) (*gen.FunctionRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	create := s.client.FunctionRun.Create().
		SetFunctionName(functionName).
		SetVersion(version).
		SetCreatedAt(time.Now())
	if status != "" {
		create = create.SetStatus(functionrun.Status(status))
	}
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		create = create.SetIdempotencyKey(key)
	}
	run, err := create.Save(ctx)
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, fmt.Errorf("function run %q key %q %w", functionName, idempotencyKey, types.ErrConflict)
		}
		return nil, err
	}
	return run, nil
}

// UpdateRun updates status and result fields for an existing function run.
func (s *FunctionStore) UpdateRun(ctx context.Context, runID int64, status string, durationMs int64, exitCode *int, errMsg string, resultJSON *string) (*gen.FunctionRun, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	upd := s.client.FunctionRun.UpdateOneID(runID).
		SetDurationMs(durationMs).
		SetError(errMsg).
		SetNillableExitCode(exitCode).
		SetNillableResultJSON(resultJSON)
	if status != "" {
		upd = upd.SetStatus(functionrun.Status(status))
	}
	run, err := upd.Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return run, nil
}

// GetRunByIdempotencyKey returns a run for the given function name and non-empty idempotency key.
func (s *FunctionStore) GetRunByIdempotencyKey(ctx context.Context, functionName, idempotencyKey string) (*gen.FunctionRun, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return nil, types.ErrNotFound
	}
	run, err := s.client.FunctionRun.Query().
		Where(
			functionrun.FunctionName(functionName),
			functionrun.IdempotencyKeyEQ(key),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return run, nil
}

// ListRunsByName returns recent runs for a function name, newest first.
func (s *FunctionStore) ListRunsByName(ctx context.Context, functionName string) ([]*gen.FunctionRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.FunctionRun.Query().
		Where(functionrun.FunctionName(functionName)).
		Order(gen.Desc(functionrun.FieldCreatedAt)).
		Limit(100).
		All(ctx)
}
