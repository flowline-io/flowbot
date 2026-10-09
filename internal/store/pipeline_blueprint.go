package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelineblueprinttemplate"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ListUserTemplates returns imported blueprint templates ordered by id.
func (s *PipelineStore) ListUserTemplates(ctx context.Context) ([]pipeline.BlueprintTemplateRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.PipelineBlueprintTemplate.Query().
		Order(gen.Asc(pipelineblueprinttemplate.FieldBlueprintID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list blueprint templates: %w", err)
	}
	out := make([]pipeline.BlueprintTemplateRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapBlueprintTemplate(row))
	}
	return out, nil
}

// GetUserTemplate returns one imported blueprint template.
func (s *PipelineStore) GetUserTemplate(ctx context.Context, blueprintID string) (*pipeline.BlueprintTemplateRecord, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	row, err := s.client.PipelineBlueprintTemplate.Query().
		Where(pipelineblueprinttemplate.BlueprintID(blueprintID)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint template: %w", err)
	}
	rec := mapBlueprintTemplate(row)
	return &rec, nil
}

// UpsertUserTemplate creates or updates an imported blueprint template.
func (s *PipelineStore) UpsertUserTemplate(ctx context.Context, rec pipeline.BlueprintTemplateRecord) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline store not ready")
	}
	id := strings.TrimSpace(rec.BlueprintID)
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "blueprint id is required")
	}
	now := time.Now()
	existing, err := s.client.PipelineBlueprintTemplate.Query().
		Where(pipelineblueprinttemplate.BlueprintID(id)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("lookup blueprint template: %w", err)
	}
	if gen.IsNotFound(err) {
		_, err = s.client.PipelineBlueprintTemplate.Create().
			SetBlueprintID(id).
			SetTitle(rec.Title).
			SetDescription(rec.Description).
			SetYaml(rec.YAML).
			SetContentHash(rec.Hash).
			SetCreatedBy(strings.TrimSpace(rec.CreatedBy)).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create blueprint template: %w", err)
		}
		return nil
	}
	_, err = s.client.PipelineBlueprintTemplate.UpdateOneID(existing.ID).
		SetTitle(rec.Title).
		SetDescription(rec.Description).
		SetYaml(rec.YAML).
		SetContentHash(rec.Hash).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update blueprint template: %w", err)
	}
	return nil
}

// DeleteUserTemplate removes an imported blueprint template.
func (s *PipelineStore) DeleteUserTemplate(ctx context.Context, blueprintID string) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline store not ready")
	}
	n, err := s.client.PipelineBlueprintTemplate.Delete().
		Where(pipelineblueprinttemplate.BlueprintID(blueprintID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete blueprint template: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// SetBlueprintOrigin writes linked-instance origin fields.
func (s *PipelineStore) SetBlueprintOrigin(ctx context.Context, name string, origin pipeline.BlueprintOrigin) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline store not ready")
	}
	inputs := origin.Inputs
	if inputs == nil {
		inputs = map[string]any{}
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(pipelinedefinition.Name(name)).
		SetBlueprintSource(origin.Source).
		SetBlueprintID(origin.ID).
		SetBlueprintHash(origin.Hash).
		SetBlueprintYaml(origin.YAML).
		SetBlueprintInputs(inputs).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("set blueprint origin: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ClearBlueprintOrigin detaches a pipeline from its blueprint.
func (s *PipelineStore) ClearBlueprintOrigin(ctx context.Context, name string) error {
	if s == nil || s.client == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline store not ready")
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(pipelinedefinition.Name(name)).
		SetBlueprintSource("").
		SetBlueprintID("").
		SetBlueprintHash("").
		SetBlueprintYaml("").
		SetBlueprintInputs(map[string]any{}).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("clear blueprint origin: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListLinkedPipelineNames returns pipeline names linked to a blueprint source and id.
func (s *PipelineStore) ListLinkedPipelineNames(ctx context.Context, source, blueprintID string) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.PipelineDefinition.Query().
		Where(
			pipelinedefinition.BlueprintSourceEQ(source),
			pipelinedefinition.BlueprintIDEQ(blueprintID),
		).
		Select(pipelinedefinition.FieldName).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list linked pipelines: %w", err)
	}
	return rows, nil
}

func mapBlueprintTemplate(row *gen.PipelineBlueprintTemplate) pipeline.BlueprintTemplateRecord {
	if row == nil {
		return pipeline.BlueprintTemplateRecord{}
	}
	return pipeline.BlueprintTemplateRecord{
		BlueprintID: row.BlueprintID,
		Title:       row.Title,
		Description: row.Description,
		YAML:        row.Yaml,
		Hash:        row.ContentHash,
		CreatedBy:   row.CreatedBy,
	}
}
