// Package store provides database storage implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventconsumption"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinitionversion"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinesteprun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/predicate"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ---------------------------------------------------------------------------
// PipelineStore
// ---------------------------------------------------------------------------

// PipelineStore persists pipeline definitions, runs, step runs, and event consumptions.
type PipelineStore struct {
	client *gen.Client
}

func NewPipelineStore(client *gen.Client) *PipelineStore {
	return &PipelineStore{client: client}
}

// PipelineStoreFromDB returns a PipelineStore using the global database client.
func PipelineStoreFromDB() *PipelineStore {
	return NewPipelineStore(ClientFromDB())
}

func (s *PipelineStore) CreateRun(ctx context.Context, pipelineName, eventID, eventType, triggerSource string) (*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	run, err := s.client.PipelineRun.Create().
		SetPipelineName(pipelineName).
		SetEventID(eventID).
		SetEventType(eventType).
		SetTriggerSource(pipelinerun.TriggerSource(triggerSource)).
		SetStatus(int(schema.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *PipelineStore) UpdateRunStatus(ctx context.Context, runID int64, status int, errMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineRun.UpdateOneID(runID).
		SetStatus(int(status)).
		SetCompletedAt(time.Now())
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) CreateStepRun(ctx context.Context, runID int64, stepName, capability, operation string, params map[string]any, attempt int) (*gen.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	sr, err := s.client.PipelineStepRun.Create().
		SetPipelineRunID(runID).
		SetStepName(stepName).
		SetCapability(capability).
		SetOperation(operation).
		SetParams(map[string]any(params)).
		SetAttempt(attempt).
		SetStatus(int(schema.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return sr, nil
}

func (s *PipelineStore) UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineStepRun.UpdateOneID(stepRunID).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == int(schema.PipelineDone) || status == int(schema.PipelineCancel) {
		now := time.Now()
		upd = upd.SetCompletedAt(now)
	}
	if result != nil {
		upd = upd.SetResult(result)
	}
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) RecordConsumption(ctx context.Context, consumerName, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.EventConsumption.Create().
		SetConsumerName(consumerName).
		SetEventID(eventID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error) {
	if s == nil || s.client == nil {
		return false, nil
	}
	count, err := s.client.EventConsumption.Query().
		Where(
			eventconsumption.ConsumerName(consumerName),
			eventconsumption.EventID(eventID),
		).
		Count(ctx)
	return count > 0, err
}

// SaveCheckpoint persists the intermediate pipeline run state.
func (s *PipelineStore) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	if s == nil || s.client == nil {
		return nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	var cp map[string]any
	if err := sonic.Unmarshal(raw, &cp); err != nil {
		return err
	}
	_, err = s.client.PipelineRun.UpdateOneID(runID).
		SetCheckpointData(cp).
		Save(ctx)
	return err
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running pipeline.
func (s *PipelineStore) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.PipelineRun.UpdateOneID(runID).
		SetLastHeartbeat(time.Now()).
		Save(ctx)
	return err
}

// GetIncompleteRuns returns pipeline runs that are in Start state and may need recovery.
func (s *PipelineStore) GetIncompleteRuns(ctx context.Context) ([]*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.Status(int(schema.PipelineStart))).
		Order(pipelinerun.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return runs, nil
}

// GetCheckpoint loads the checkpoint data for a pipeline run.
func (s *PipelineStore) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	if s == nil || s.client == nil {
		return nil
	}
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Select(pipelinerun.FieldCheckpointData).
		Only(ctx)
	if err != nil {
		return err
	}
	if run.CheckpointData == nil {
		return nil
	}
	raw, err := sonic.Marshal(run.CheckpointData)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(raw, target)
}

// GetRun returns a pipeline run by ID.
func (s *PipelineStore) GetRun(ctx context.Context, runID int64) (*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// RecordResourceLink inserts a resource link with UPSERT semantics.
func (s *PipelineStore) RecordResourceLink(ctx context.Context, link *gen.ResourceLink) error {
	if s == nil || s.client == nil {
		return nil
	}
	err := s.client.ResourceLink.Create().
		SetSourceEventID(link.SourceEventID).
		SetTargetEventID(link.TargetEventID).
		SetSourceApp(link.SourceApp).
		SetTargetApp(link.TargetApp).
		SetSourceCapability(link.SourceCapability).
		SetTargetCapability(link.TargetCapability).
		SetSourceEntityID(link.SourceEntityID).
		SetTargetEntityID(link.TargetEntityID).
		SetPipelineRunID(link.PipelineRunID).
		SetPipelineName(link.PipelineName).
		SetCreatedAt(time.Now()).
		OnConflictColumns(
			resourcelink.FieldSourceEventID,
			resourcelink.FieldTargetEventID,
		).
		Ignore().
		Exec(ctx)
	return err
}

// CreateDefinition creates a new pipeline definition with initial yaml_draft and version 1.
// createdBy is the Web UI user UID that created the pipeline (may be empty in tests).
func (s *PipelineStore) CreateDefinition(ctx context.Context, name, description, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	_, err := s.client.PipelineDefinition.Create().
		SetName(name).
		SetDescription(description).
		SetYamlDraft("").
		SetNillableYamlPublished(nil).
		SetVersion(1).
		SetStatus("draft").
		SetCreatedBy(strings.TrimSpace(createdBy)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if gen.IsConstraintError(err) {
			return fmt.Errorf("pipeline %q %w", name, types.ErrAlreadyExists)
		}
		return err
	}
	return nil
}

// EnsureDefinitionCreatedBy sets created_by when it is currently empty.
// Used to backfill owner UID for pipelines created before the field existed.
func (s *PipelineStore) EnsureDefinitionCreatedBy(ctx context.Context, name, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return nil
	}
	_, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.CreatedByEQ(""),
		).
		SetCreatedBy(createdBy).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("ensure pipeline created_by: %w", err)
	}
	return nil
}

// GetDefinitionByName returns a pipeline definition by name.
func (s *PipelineStore) GetDefinitionByName(ctx context.Context, name string) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.PipelineDefinition.Query().
		Where(pipelinedefinition.Name(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// ListDefinitionVersions returns all published version snapshots for a pipeline,
// ordered by version descending (newest first).
func (s *PipelineStore) ListDefinitionVersions(ctx context.Context, name string) ([]*gen.PipelineDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineDefinitionVersion.Query().
		Where(pipelinedefinitionversion.PipelineName(name)).
		Order(gen.Desc(pipelinedefinitionversion.FieldVersion)).
		All(ctx)
}

// GetDefinitionVersion returns a single version snapshot by pipeline name and version number.
func (s *PipelineStore) GetDefinitionVersion(ctx context.Context, name string, version int) (*gen.PipelineDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.PipelineDefinitionVersion.Query().
		Where(
			pipelinedefinitionversion.PipelineName(name),
			pipelinedefinitionversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// ListDefinitions returns all pipeline definitions ordered by updated_at desc.
func (s *PipelineStore) ListDefinitions(ctx context.Context) ([]*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineDefinition.Query().
		Order(gen.Desc(pipelinedefinition.FieldUpdatedAt)).
		All(ctx)
}

// UpdateDefinitionDraft updates the yaml_draft with atomic optimistic locking.
// Uses conditional UPDATE WHERE version=X. Returns ErrConflict if no row matched.
func (s *PipelineStore) UpdateDefinitionDraft(ctx context.Context, name, yamlDraft string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
		).
		SetYamlDraft(yamlDraft).
		SetVersion(version + 1).
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

// PublishDefinition copies yaml_draft to yaml_published with atomic optimistic locking.
// Also inserts a version snapshot into pipeline_definition_versions.
func (s *PipelineStore) PublishDefinition(ctx context.Context, name string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	def, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.YamlDraft == "" {
		return nil, types.ErrConflict
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
		).
		SetYamlPublished(def.YamlDraft).
		SetVersion(version + 1).
		SetStatus("published").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}

	if _, err := s.client.PipelineDefinitionVersion.Create().
		SetPipelineName(name).
		SetVersion(version + 1).
		SetYaml(def.YamlDraft).
		SetCreatedAt(time.Now()).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("publish: insert version snapshot: %w", err)
	}

	return s.GetDefinitionByName(ctx, name)
}

// SetDefinitionEnabled toggles the top-level enabled flag in draft and published YAML.
// Only published pipelines can be paused at runtime.
func (s *PipelineStore) SetDefinitionEnabled(ctx context.Context, name string, enabled bool) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	def, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.Status != pipelinedefinition.StatusPublished || def.YamlPublished == nil || *def.YamlPublished == "" {
		return nil, fmt.Errorf("pipeline %q is not published: %w", name, types.ErrInvalidArgument)
	}

	newPublished, err := pipeline.SetEnabledInYAML(*def.YamlPublished, enabled)
	if err != nil {
		return nil, fmt.Errorf("set enabled on published yaml: %w", err)
	}

	update := s.client.PipelineDefinition.Update().
		Where(pipelinedefinition.Name(name)).
		SetYamlPublished(newPublished).
		SetUpdatedAt(time.Now())

	if def.YamlDraft != "" {
		newDraft, draftErr := pipeline.SetEnabledInYAML(def.YamlDraft, enabled)
		if draftErr != nil {
			return nil, fmt.Errorf("set enabled on draft yaml: %w", draftErr)
		}
		update = update.SetYamlDraft(newDraft)
	}

	if _, err := update.Save(ctx); err != nil {
		return nil, err
	}
	return s.GetDefinitionByName(ctx, name)
}

// RenameDefinition renames a pipeline definition and cascades the new name to
// version snapshots, runs (including compound trigger names), resource links,
// and top-level name fields in draft/published YAML.
func (s *PipelineStore) RenameDefinition(ctx context.Context, oldName, newName string) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if err := pipeline.ValidateName(newName); err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidArgument, err.Error())
	}
	if oldName == newName {
		return s.GetDefinitionByName(ctx, oldName)
	}

	def, err := s.GetDefinitionByName(ctx, oldName)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRenameTargetAvailable(ctx, newName); err != nil {
		return nil, err
	}

	draftYAML, publishedYAML, err := renameDefinitionYAML(def, newName)
	if err != nil {
		return nil, err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("rename pipeline begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := renameDefinitionInTx(ctx, tx, oldName, newName, draftYAML, publishedYAML); err != nil {
		return nil, err
	}
	if err := renamePipelineReferencesInTx(ctx, tx, oldName, newName); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("rename pipeline commit: %w", err)
	}
	committed = true
	return s.GetDefinitionByName(ctx, newName)
}

// ensureRenameTargetAvailable reports whether newName can be used for a rename.
func (s *PipelineStore) ensureRenameTargetAvailable(ctx context.Context, newName string) error {
	_, err := s.GetDefinitionByName(ctx, newName)
	if err == nil {
		return fmt.Errorf("pipeline %q %w", newName, types.ErrAlreadyExists)
	}
	if !errors.Is(err, types.ErrNotFound) {
		return err
	}
	return nil
}

// renameDefinitionYAML rewrites top-level name fields in draft and published YAML.
func renameDefinitionYAML(def *gen.PipelineDefinition, newName string) (string, *string, error) {
	draftYAML := def.YamlDraft
	if draftYAML != "" {
		updated, err := pipeline.SetNameInYAML(draftYAML, newName)
		if err != nil {
			return "", nil, fmt.Errorf("rename pipeline draft yaml: %w", err)
		}
		draftYAML = updated
	}
	var publishedYAML *string
	if def.YamlPublished != nil && *def.YamlPublished != "" {
		updated, err := pipeline.SetNameInYAML(*def.YamlPublished, newName)
		if err != nil {
			return "", nil, fmt.Errorf("rename pipeline published yaml: %w", err)
		}
		publishedYAML = &updated
	}
	return draftYAML, publishedYAML, nil
}

// renameDefinitionInTx updates the pipeline_definitions row inside a transaction.
func renameDefinitionInTx(ctx context.Context, tx *gen.Tx, oldName, newName, draftYAML string, publishedYAML *string) error {
	update := tx.PipelineDefinition.Update().
		Where(pipelinedefinition.Name(oldName)).
		SetName(newName).
		SetYamlDraft(draftYAML).
		SetUpdatedAt(time.Now())
	if publishedYAML != nil {
		update = update.SetYamlPublished(*publishedYAML)
	}
	n, err := update.Save(ctx)
	if err != nil {
		if gen.IsConstraintError(err) {
			return fmt.Errorf("pipeline %q %w", newName, types.ErrAlreadyExists)
		}
		return fmt.Errorf("rename pipeline definition: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// renamePipelineReferencesInTx cascades pipeline_name updates to versions, runs, and links.
func renamePipelineReferencesInTx(ctx context.Context, tx *gen.Tx, oldName, newName string) error {
	if _, err := tx.PipelineDefinitionVersion.Update().
		Where(pipelinedefinitionversion.PipelineName(oldName)).
		SetPipelineName(newName).
		Save(ctx); err != nil {
		return fmt.Errorf("rename pipeline versions: %w", err)
	}
	return renameExactAndCompoundNames(ctx, tx, oldName, newName)
}

// renameExactAndCompoundNames rewrites exact and __trigger_ compound pipeline_name values.
func renameExactAndCompoundNames(ctx context.Context, tx *gen.Tx, oldName, newName string) error {
	if _, err := tx.PipelineRun.Update().
		Where(pipelinerun.PipelineName(oldName)).
		SetPipelineName(newName).
		Save(ctx); err != nil {
		return fmt.Errorf("rename pipeline runs: %w", err)
	}
	if err := renameCompoundPipelineRuns(ctx, tx, oldName, newName); err != nil {
		return err
	}
	if _, err := tx.ResourceLink.Update().
		Where(resourcelink.PipelineName(oldName)).
		SetPipelineName(newName).
		Save(ctx); err != nil {
		return fmt.Errorf("rename pipeline resource links: %w", err)
	}
	return renameCompoundResourceLinks(ctx, tx, oldName, newName)
}

// renameCompoundPipelineRuns rewrites compound trigger run names to the new parent.
func renameCompoundPipelineRuns(ctx context.Context, tx *gen.Tx, oldName, newName string) error {
	oldTriggerPrefix := oldName + "__trigger_"
	newTriggerPrefix := newName + "__trigger_"
	compoundRuns, err := tx.PipelineRun.Query().
		Where(pipelinerun.PipelineNameHasPrefix(oldTriggerPrefix)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list compound pipeline runs: %w", err)
	}
	for _, run := range compoundRuns {
		renamed := newTriggerPrefix + strings.TrimPrefix(run.PipelineName, oldTriggerPrefix)
		if _, err := tx.PipelineRun.UpdateOneID(run.ID).
			SetPipelineName(renamed).
			Save(ctx); err != nil {
			return fmt.Errorf("rename compound pipeline run %d: %w", run.ID, err)
		}
	}
	return nil
}

// renameCompoundResourceLinks rewrites compound trigger resource link names to the new parent.
func renameCompoundResourceLinks(ctx context.Context, tx *gen.Tx, oldName, newName string) error {
	oldTriggerPrefix := oldName + "__trigger_"
	newTriggerPrefix := newName + "__trigger_"
	compoundLinks, err := tx.ResourceLink.Query().
		Where(resourcelink.PipelineNameHasPrefix(oldTriggerPrefix)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list compound resource links: %w", err)
	}
	for _, link := range compoundLinks {
		renamed := newTriggerPrefix + strings.TrimPrefix(link.PipelineName, oldTriggerPrefix)
		if _, err := tx.ResourceLink.UpdateOneID(link.ID).
			SetPipelineName(renamed).
			Save(ctx); err != nil {
			return fmt.Errorf("rename compound resource link %d: %w", link.ID, err)
		}
	}
	return nil
}

// DeleteDefinitionByName removes a pipeline definition and its associated runs.
// Returns the number of pipeline runs that were deleted.
func (s *PipelineStore) DeleteDefinitionByName(ctx context.Context, name string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	runCount, err := s.client.PipelineRun.Delete().
		Where(pipelinerun.PipelineName(name)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete runs for %s: %w", name, err)
	}
	_, err = s.client.PipelineDefinition.Delete().
		Where(pipelinedefinition.Name(name)).
		Exec(ctx)
	if err != nil {
		return int64(runCount), fmt.Errorf("delete definition %s: %w", name, err)
	}
	return int64(runCount), nil
}

// pipelineRunByParentName matches exact parent name and compound trigger engine names.
func pipelineRunByParentName(parentName string) predicate.PipelineRun {
	return pipelinerun.Or(
		pipelinerun.PipelineName(parentName),
		pipelinerun.PipelineNameHasPrefix(parentName+"__trigger_"),
	)
}

// GetRunsByParentName returns pipeline runs matching a parent pipeline name.
// Matches both exact name and compound trigger names (name__trigger_*).
func (s *PipelineStore) GetRunsByParentName(ctx context.Context, parentName string) ([]*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineRun.Query().
		Where(pipelineRunByParentName(parentName)).
		Order(gen.Desc(pipelinerun.FieldCreatedAt)).
		Limit(100).
		All(ctx)
}

// RunLatencyStatsByParentNames returns success rate and P50/P95 duration stats per parent pipeline name.
// Matches exact pipeline_name and compound trigger names (name__trigger_*).
// Only completed runs with started_at >= since (when since is non-zero) are included.
// Names without qualifying runs are omitted from the result.
func (s *PipelineStore) RunLatencyStatsByParentNames(ctx context.Context, names []string, since time.Time) (map[string]types.RunLatencyStats, error) {
	result := make(map[string]types.RunLatencyStats)
	if s == nil || s.client == nil || len(names) == 0 {
		return result, nil
	}
	runs, err := s.fetchCompletedRunsForParents(ctx, names, since)
	if err != nil {
		return nil, err
	}
	byParent := groupPipelineLatencyOutcomes(runs, names)
	for parent, outcomes := range byParent {
		result[parent] = computeRunLatencyStats(outcomes)
	}
	return result, nil
}

func (s *PipelineStore) fetchCompletedRunsForParents(ctx context.Context, names []string, since time.Time) ([]*gen.PipelineRun, error) {
	preds := make([]predicate.PipelineRun, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		preds = append(preds, pipelineRunByParentName(name))
	}
	if len(preds) == 0 {
		return nil, nil
	}
	q := s.client.PipelineRun.Query().
		Where(
			pipelinerun.Or(preds...),
			pipelinerun.CompletedAtNotNil(),
		)
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}
	runs, err := q.Select(
		pipelinerun.FieldPipelineName,
		pipelinerun.FieldStatus,
		pipelinerun.FieldStartedAt,
		pipelinerun.FieldCompletedAt,
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("run latency stats by parent names: %w", err)
	}
	return runs, nil
}

func groupPipelineLatencyOutcomes(runs []*gen.PipelineRun, names []string) map[string][]runLatencyOutcome {
	byParent := make(map[string][]runLatencyOutcome)
	for _, run := range runs {
		outcome, parent, ok := pipelineRunLatencyOutcome(run, names)
		if !ok {
			continue
		}
		byParent[parent] = append(byParent[parent], outcome)
	}
	return byParent
}

func pipelineRunLatencyOutcome(run *gen.PipelineRun, names []string) (runLatencyOutcome, string, bool) {
	if run == nil || run.CompletedAt == nil || run.StartedAt.IsZero() {
		return runLatencyOutcome{}, "", false
	}
	parent := matchParentPipelineName(run.PipelineName, names)
	if parent == "" {
		return runLatencyOutcome{}, "", false
	}
	dur := max(run.CompletedAt.Sub(run.StartedAt).Milliseconds(), 0)
	return runLatencyOutcome{
		durationMs: dur,
		success:    run.Status == int(schema.PipelineDone),
	}, parent, true
}

// LatestRunStartedAtByParentNames returns the latest started_at for each parent pipeline name.
// Matches exact pipeline_name and compound trigger names (name__trigger_*).
// Names without runs are omitted from the result.
func (s *PipelineStore) LatestRunStartedAtByParentNames(ctx context.Context, names []string) (map[string]time.Time, error) {
	result := make(map[string]time.Time)
	if s == nil || s.client == nil || len(names) == 0 {
		return result, nil
	}
	preds := make([]predicate.PipelineRun, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		preds = append(preds, pipelineRunByParentName(name))
	}
	if len(preds) == 0 {
		return result, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.Or(preds...)).
		Select(pipelinerun.FieldPipelineName, pipelinerun.FieldStartedAt).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("latest run started_at by parent names: %w", err)
	}
	for _, run := range runs {
		parent := matchParentPipelineName(run.PipelineName, names)
		if parent == "" {
			continue
		}
		if prev, ok := result[parent]; !ok || run.StartedAt.After(prev) {
			result[parent] = run.StartedAt
		}
	}
	return result, nil
}

// matchParentPipelineName maps a run pipeline_name back to a parent definition name.
func matchParentPipelineName(runName string, parents []string) string {
	best := ""
	for _, parent := range parents {
		if parent == "" {
			continue
		}
		if runName == parent {
			return parent
		}
		if strings.HasPrefix(runName, parent+"__trigger_") && len(parent) > len(best) {
			best = parent
		}
	}
	return best
}

// GetStepRunsByRunID returns all step runs for a given pipeline run, ordered by ID.
func (s *PipelineStore) GetStepRunsByRunID(ctx context.Context, runID int64) ([]*gen.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineStepRun.Query().
		Where(pipelinesteprun.PipelineRunIDEQ(runID)).
		Order(gen.Asc(pipelinesteprun.FieldID)).
		Limit(200).
		All(ctx)
}

// GetRunByID returns a pipeline run by its database ID.
func (s *PipelineStore) GetRunByID(ctx context.Context, id int64) (*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	run, err := s.client.PipelineRun.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return run, nil
}

// ListStepRunsByRunID returns all step runs for a pipeline run, ordered by creation time.
func (s *PipelineStore) ListStepRunsByRunID(ctx context.Context, runID int64) ([]*gen.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	steps, err := s.client.PipelineStepRun.Query().
		Where(pipelinesteprun.PipelineRunIDEQ(runID)).
		Order(gen.Asc(pipelinesteprun.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return steps, nil
}

// ListPublishedDefinitions returns all pipeline definitions that are published
// and have a non-nil yaml_published field.
func (s *PipelineStore) ListPublishedDefinitions(ctx context.Context) ([]pipeline.DefinitionRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.PipelineDefinition.Query().
		Where(
			pipelinedefinition.StatusEQ(pipelinedefinition.Status("published")),
			pipelinedefinition.YamlPublishedNotNil(),
		).
		Order(gen.Asc(pipelinedefinition.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published pipeline definitions: %w", err)
	}
	records := make([]pipeline.DefinitionRecord, 0, len(rows))
	for _, row := range rows {
		yamlContent := ""
		if row.YamlPublished != nil {
			yamlContent = *row.YamlPublished
		}
		records = append(records, pipeline.DefinitionRecord{
			Name:        row.Name,
			Description: row.Description,
			YAML:        yamlContent,
			CreatedBy:   row.CreatedBy,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return records, nil
}

// PipelineStats returns aggregated pipeline run statistics for chart rendering.
// name empty = all pipelines. since zero = no time filter. groupBy = "day"|"week"|"month".
func (s *PipelineStore) PipelineStats(ctx context.Context, name string, since time.Time, groupBy string) (*types.PipelineStats, error) {
	if s == nil || s.client == nil {
		return emptyPipelineStats(), nil
	}
	stats := &types.PipelineStats{}

	var err error
	stats.Summary, err = s.loadPipelineStatsSummary(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}
	stats.SuccessRateTrend, err = s.loadSuccessRate(ctx, name, since, groupBy)
	if err != nil {
		return nil, fmt.Errorf("success rate: %w", err)
	}
	stats.DurationDistribution.Pipeline, err = s.loadDurationBuckets(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("pipeline duration: %w", err)
	}
	stats.DurationDistribution.Step, err = s.loadStepDurationBuckets(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("step duration: %w", err)
	}
	stats.TriggerSourcePie, err = s.loadTriggerSources(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("trigger sources: %w", err)
	}
	return stats, nil
}

// loadPipelineStatsSummary returns headline counters for the pipelines overview.
func (s *PipelineStore) loadPipelineStatsSummary(ctx context.Context, name string, since time.Time) (types.PipelineStatsSummary, error) {
	summary := types.PipelineStatsSummary{}
	if name == "" {
		count, err := s.client.PipelineDefinition.Query().Count(ctx)
		if err != nil {
			return summary, err
		}
		summary.TotalPipelines = int64(count)
	}

	successful, err := s.countCompletedRunsByStatus(ctx, name, since, int(schema.PipelineDone))
	if err != nil {
		return summary, err
	}
	failed, err := s.countCompletedRunsByStatus(ctx, name, since, int(schema.PipelineFailed))
	if err != nil {
		return summary, err
	}
	summary.SuccessfulRuns = successful
	summary.FailedRuns = failed
	return summary, nil
}

// countCompletedRunsByStatus counts completed runs filtered by pipeline, time range, and status.
func (s *PipelineStore) countCompletedRunsByStatus(ctx context.Context, name string, since time.Time, status int) (int64, error) {
	q := s.client.PipelineRun.Query().
		Where(
			pipelinerun.CompletedAtNotNil(),
			pipelinerun.StatusEQ(status),
		)
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}
	count, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

// loadSuccessRate fetches completed runs and computes success rate points in Go.
// Ent v0.14.6 does not expose Modify() on query builders, so custom SQL GROUP BY
// expressions are handled via in-memory aggregation after fetching raw data.
func (s *PipelineStore) loadSuccessRate(ctx context.Context, name string, since time.Time, groupBy string) ([]types.SuccessRatePoint, error) {
	runs, err := s.fetchCompletedRuns(ctx, name, since)
	if err != nil {
		return nil, err
	}
	return computeSuccessRate(runs, groupBy), nil
}

// fetchCompletedRuns returns completed pipeline runs filtered by name and since time.
func (s *PipelineStore) fetchCompletedRuns(ctx context.Context, name string, since time.Time) ([]*gen.PipelineRun, error) {
	q := s.client.PipelineRun.Query().Where(pipelinerun.CompletedAtNotNil())
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}
	return q.All(ctx)
}

// computeSuccessRate aggregates completed runs into success rate points grouped by date.
func computeSuccessRate(runs []*gen.PipelineRun, groupBy string) []types.SuccessRatePoint {
	type dayStats struct {
		total   int64
		success int64
	}
	buckets := make(map[string]*dayStats)
	for _, r := range runs {
		if r.CompletedAt == nil {
			continue
		}
		key := dateGroupKey(*r.CompletedAt, groupBy)
		if buckets[key] == nil {
			buckets[key] = &dayStats{}
		}
		buckets[key].total++
		if r.Status == int(schema.PipelineDone) {
			buckets[key].success++
		}
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	points := make([]types.SuccessRatePoint, 0, len(keys))
	for _, k := range keys {
		s := buckets[k]
		rate := float64(0)
		if s.total > 0 {
			rate = float64(s.success) / float64(s.total)
		}
		points = append(points, types.SuccessRatePoint{
			Date: k, Total: s.total, Success: s.success, Rate: rate,
		})
	}
	if points == nil {
		points = []types.SuccessRatePoint{}
	}
	return points
}

// dateGroupKey returns a date grouping key based on the requested granularity.
func dateGroupKey(t time.Time, groupBy string) string {
	switch groupBy {
	case "week":
		y, w := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case "month":
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02")
	}
}

// loadDurationBuckets fetches completed runs and buckets by duration in Go.
func (s *PipelineStore) loadDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
	q := s.client.PipelineRun.Query().Where(pipelinerun.CompletedAtNotNil())
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}

	runs, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := emptyDurationBuckets()
	for _, r := range runs {
		if r.StartedAt.IsZero() || r.CompletedAt == nil {
			continue
		}
		incrementDurationBucket(result, r.CompletedAt.Sub(r.StartedAt))
	}
	return result, nil
}

// loadStepDurationBuckets fetches completed step runs and buckets by duration.
func (s *PipelineStore) loadStepDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
	q := s.client.PipelineStepRun.Query().Where(pipelinesteprun.CompletedAtNotNil())
	if name != "" {
		runIDs, err := s.client.PipelineRun.Query().
			Where(pipelineRunByParentName(name)).
			IDs(ctx)
		if err != nil {
			return nil, err
		}
		if len(runIDs) == 0 {
			return emptyDurationBuckets(), nil
		}
		q = q.Where(pipelinesteprun.PipelineRunIDIn(runIDs...))
	}
	if !since.IsZero() {
		q = q.Where(pipelinesteprun.StartedAtGTE(since))
	}

	steps, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := emptyDurationBuckets()
	for _, st := range steps {
		if st.StartedAt.IsZero() || st.CompletedAt == nil {
			continue
		}
		incrementDurationBucket(result, st.CompletedAt.Sub(st.StartedAt))
	}
	return result, nil
}

// loadTriggerSources counts runs grouped by trigger_source using ent GroupBy.
func (s *PipelineStore) loadTriggerSources(ctx context.Context, name string, since time.Time) ([]types.TriggerSourceCount, error) {
	q := s.client.PipelineRun.Query()
	if name != "" {
		q = q.Where(pipelineRunByParentName(name))
	}
	if !since.IsZero() {
		q = q.Where(pipelinerun.StartedAtGTE(since))
	}

	type row struct {
		Source string `sql:"trigger_source"`
		Count  int64  `sql:"count"`
	}
	var rows []row

	err := q.GroupBy(pipelinerun.FieldTriggerSource).
		Aggregate(gen.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	result := map[string]int64{"event": 0, "webhook": 0, "cron": 0, "manual": 0}
	for _, r := range rows {
		result[r.Source] = r.Count
	}
	return []types.TriggerSourceCount{
		{Source: "event", Count: result["event"]},
		{Source: "webhook", Count: result["webhook"]},
		{Source: "cron", Count: result["cron"]},
		{Source: "manual", Count: result["manual"]},
	}, nil
}

func emptyPipelineStats() *types.PipelineStats {
	return &types.PipelineStats{
		Summary: types.PipelineStatsSummary{},
		TriggerSourcePie: []types.TriggerSourceCount{
			{Source: "event"}, {Source: "webhook"}, {Source: "cron"}, {Source: "manual"},
		},
		DurationDistribution: types.DurationDistribution{
			Pipeline: emptyDurationBuckets(),
			Step:     emptyDurationBuckets(),
		},
	}
}

func emptyDurationBuckets() []types.DurationEntry {
	return []types.DurationEntry{
		{Bucket: "0-1s"}, {Bucket: "1-5s"}, {Bucket: "5-30s"}, {Bucket: "30s+"},
	}
}

func incrementDurationBucket(result []types.DurationEntry, dur time.Duration) {
	switch {
	case dur < time.Second:
		result[0].Count++
	case dur < 5*time.Second:
		result[1].Count++
	case dur < 30*time.Second:
		result[2].Count++
	default:
		result[3].Count++
	}
}
