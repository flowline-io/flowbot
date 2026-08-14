package store

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// FunctionCatalogAdapter adapts FunctionStore to functions.Catalog (model DTOs).
type FunctionCatalogAdapter struct {
	S *FunctionStore
}

// NewFunctionCatalogAdapter wraps a FunctionStore as functions.Catalog.
func NewFunctionCatalogAdapter(s *FunctionStore) functions.Catalog {
	if s == nil {
		return nil
	}
	return FunctionCatalogAdapter{S: s}
}

// Create implements functions.Catalog.
func (a FunctionCatalogAdapter) Create(ctx context.Context, name, metadata, entrypoint, source, createdBy string) error {
	return a.S.Create(ctx, name, metadata, entrypoint, source, createdBy)
}

// GetByName implements functions.Catalog.
func (a FunctionCatalogAdapter) GetByName(ctx context.Context, name string) (*model.FunctionDefinition, error) {
	row, err := a.S.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return mapFunctionDefinitionDTO(row), nil
}

// UpdateDraft implements functions.Catalog.
func (a FunctionCatalogAdapter) UpdateDraft(ctx context.Context, name, metadata, entrypoint, source string, version int) (*model.FunctionDefinition, error) {
	row, err := a.S.UpdateDefinitionDraft(ctx, name, metadata, entrypoint, source, version)
	if err != nil {
		return nil, err
	}
	return mapFunctionDefinitionDTO(row), nil
}

// Publish implements functions.Catalog.
func (a FunctionCatalogAdapter) Publish(ctx context.Context, name string, version int) (*model.FunctionDefinition, error) {
	row, err := a.S.PublishDefinition(ctx, name, version)
	if err != nil {
		return nil, err
	}
	return mapFunctionDefinitionDTO(row), nil
}

// Delete implements functions.Catalog.
func (a FunctionCatalogAdapter) Delete(ctx context.Context, name string) (int64, error) {
	return a.S.DeleteDefinitionByName(ctx, name)
}

// ListPublished implements functions.Catalog.
func (a FunctionCatalogAdapter) ListPublished(ctx context.Context) ([]*model.FunctionDefinition, error) {
	rows, err := a.S.ListPublishedDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.FunctionDefinition, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapFunctionDefinitionDTO(row))
	}
	return out, nil
}

// ListAll implements functions.Catalog.
func (a FunctionCatalogAdapter) ListAll(ctx context.Context) ([]*model.FunctionDefinition, error) {
	rows, err := a.S.ListAllDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.FunctionDefinition, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapFunctionDefinitionDTO(row))
	}
	return out, nil
}

// GetVersion implements functions.Catalog.
func (a FunctionCatalogAdapter) GetVersion(ctx context.Context, name string, version int) (*model.FunctionDefinitionVersion, error) {
	row, err := a.S.GetPublishedVersion(ctx, name, version)
	if err != nil {
		return nil, err
	}
	return mapFunctionDefinitionVersionDTO(row), nil
}

// GetLatestPublished implements functions.Catalog.
func (a FunctionCatalogAdapter) GetLatestPublished(ctx context.Context, name string) (*model.FunctionDefinitionVersion, error) {
	row, err := a.S.GetLatestPublished(ctx, name)
	if err != nil {
		return nil, err
	}
	return mapFunctionDefinitionVersionDTO(row), nil
}

// CreateRun implements functions.Catalog.
func (a FunctionCatalogAdapter) CreateRun(ctx context.Context, name string, version int, idempotencyKey *string) (*model.FunctionRun, error) {
	key := ""
	if idempotencyKey != nil {
		key = *idempotencyKey
	}
	row, err := a.S.CreateRun(ctx, name, version, string(types.FunctionRunRunning), key)
	if err != nil {
		return nil, err
	}
	return mapFunctionRunDTO(row), nil
}

// GetRunByIdempotency implements functions.Catalog.
func (a FunctionCatalogAdapter) GetRunByIdempotency(ctx context.Context, name, key string) (*model.FunctionRun, error) {
	row, err := a.S.GetRunByIdempotencyKey(ctx, name, key)
	if err != nil {
		return nil, err
	}
	return mapFunctionRunDTO(row), nil
}

// CompleteRun implements functions.Catalog.
func (a FunctionCatalogAdapter) CompleteRun(ctx context.Context, id int64, status string, durationMs int64, exitCode *int, errMsg string, resultJSON *string) (*model.FunctionRun, error) {
	row, err := a.S.UpdateRun(ctx, id, status, durationMs, exitCode, errMsg, resultJSON)
	if err != nil {
		return nil, err
	}
	return mapFunctionRunDTO(row), nil
}

// ListRuns implements functions.Catalog.
func (a FunctionCatalogAdapter) ListRuns(ctx context.Context, name string) ([]*model.FunctionRun, error) {
	rows, err := a.S.ListRunsByName(ctx, name)
	if err != nil {
		return nil, err
	}
	out := make([]*model.FunctionRun, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapFunctionRunDTO(row))
	}
	return out, nil
}

func mapFunctionDefinitionDTO(row *gen.FunctionDefinition) *model.FunctionDefinition {
	if row == nil {
		return nil
	}
	return &model.FunctionDefinition{
		ID:                  row.ID,
		Name:                row.Name,
		Status:              string(row.Status),
		Version:             row.Version,
		CreatedBy:           row.CreatedBy,
		MetadataDraft:       row.MetadataDraft,
		EntrypointDraft:     row.EntrypointDraft,
		SourceDraft:         row.SourceDraft,
		MetadataPublished:   row.MetadataPublished,
		EntrypointPublished: row.EntrypointPublished,
		SourcePublished:     row.SourcePublished,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func mapFunctionDefinitionVersionDTO(row *gen.FunctionDefinitionVersion) *model.FunctionDefinitionVersion {
	if row == nil {
		return nil
	}
	return &model.FunctionDefinitionVersion{
		ID:           row.ID,
		FunctionName: row.FunctionName,
		Version:      row.Version,
		Metadata:     row.Metadata,
		Entrypoint:   row.Entrypoint,
		Source:       row.Source,
		CreatedAt:    row.CreatedAt,
	}
}

func mapFunctionRunDTO(row *gen.FunctionRun) *model.FunctionRun {
	if row == nil {
		return nil
	}
	return &model.FunctionRun{
		ID:             row.ID,
		FunctionName:   row.FunctionName,
		Version:        row.Version,
		Status:         string(row.Status),
		DurationMs:     row.DurationMs,
		ExitCode:       row.ExitCode,
		Error:          row.Error,
		ResultJSON:     row.ResultJSON,
		IdempotencyKey: row.IdempotencyKey,
		CreatedAt:      row.CreatedAt,
	}
}
