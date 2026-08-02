package pipeline

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// DefinitionCatalog loads and mutates pipeline definitions stored in the database.
type DefinitionCatalog interface {
	CreateDefinition(ctx context.Context, name, description, createdBy string) error
	GetDefinitionByName(ctx context.Context, name string) (*model.PipelineDefinition, error)
	UpdateDefinitionDraft(ctx context.Context, name, yamlDraft string, version int) (*model.PipelineDefinition, error)
	PublishDefinition(ctx context.Context, name string, version int) (*model.PipelineDefinition, error)
	EnsureDefinitionCreatedBy(ctx context.Context, name, createdBy string) error
	DeleteDefinitionByName(ctx context.Context, name string) (int64, error)
	ListPublishedDefinitions(ctx context.Context) ([]DefinitionRecord, error)
	GetRunsByParentName(ctx context.Context, parentName string) ([]*model.PipelineRun, error)
}

// ListInfo is a published pipeline summary for list APIs.
type ListInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Resumable   bool     `json:"resumable"`
	Triggers    []string `json:"triggers"`
}

// ApplyResult is returned after applying a pipeline YAML definition.
type ApplyResult struct {
	Name    string `json:"name"`
	ID      int64  `json:"id"`
	Enabled bool   `json:"enabled"`
	Version int    `json:"version"`
}

// Service orchestrates DB-backed pipeline apply/run and engine reload.
type Service struct {
	catalog DefinitionCatalog
}

// NewService creates a pipeline definition Service.
func NewService(catalog DefinitionCatalog) *Service {
	return &Service{catalog: catalog}
}

// ApplyYAML parses YAML, writes draft, publishes immediately, and reloads the engine.
func (s *Service) ApplyYAML(ctx context.Context, data []byte, createdBy string) (*ApplyResult, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	ed, err := ParseEditorYAML(string(data))
	if err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid pipeline YAML", err)
	}
	name := strings.TrimSpace(ed.Name)
	if err := ValidateName(name); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid pipeline name", err)
	}

	def, err := s.catalog.GetDefinitionByName(ctx, name)
	if err != nil {
		if !errors.Is(err, types.ErrNotFound) {
			return nil, err
		}
		if createErr := s.catalog.CreateDefinition(ctx, name, ed.Description, strings.TrimSpace(createdBy)); createErr != nil {
			return nil, createErr
		}
		def, err = s.catalog.GetDefinitionByName(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	if err := s.catalog.EnsureDefinitionCreatedBy(ctx, name, createdBy); err != nil {
		flog.Error(fmt.Errorf("ensure pipeline created_by: %w", err))
	}

	updated, err := s.catalog.UpdateDefinitionDraft(ctx, name, string(data), def.Version)
	if err != nil {
		return nil, err
	}
	published, err := s.catalog.PublishDefinition(ctx, name, updated.Version)
	if err != nil {
		return nil, err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(fmt.Errorf("reload pipeline engine after apply: %w", reloadErr))
	}
	return &ApplyResult{
		Name:    published.Name,
		ID:      published.ID,
		Enabled: IsEnabledInYAML(stringValue(published.YamlPublished)),
		Version: published.Version,
	}, nil
}

// List returns published pipeline summaries.
func (s *Service) List(ctx context.Context) ([]ListInfo, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	records, err := s.catalog.ListPublishedDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ListInfo, 0, len(records))
	for _, rec := range records {
		info := ListInfo{
			Name:        rec.Name,
			Description: rec.Description,
			Enabled:     IsEnabledInYAML(rec.YAML),
			Triggers:    []string{},
		}
		ed, parseErr := ParseEditorYAML(rec.YAML)
		if parseErr == nil && ed != nil {
			info.Resumable = ed.Resumable
			if ed.Description != "" {
				info.Description = ed.Description
			}
			info.Triggers = triggerSummaries(ed.Triggers)
		}
		items = append(items, info)
	}
	return items, nil
}

// Get returns parsed metadata for a published pipeline.
func (s *Service) Get(ctx context.Context, name string) (map[string]any, error) {
	yamlText, def, err := s.publishedYAML(ctx, name)
	if err != nil {
		return nil, err
	}
	ed, err := ParseEditorYAML(yamlText)
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "parse published pipeline YAML", err)
	}
	return map[string]any{
		"id":          def.ID,
		"name":        ed.Name,
		"description": ed.Description,
		"enabled":     ed.Enabled,
		"resumable":   ed.Resumable,
		"version":     def.Version,
		"status":      def.Status,
		"triggers":    ed.Triggers,
		"steps":       ed.Steps,
	}, nil
}

// Export returns published YAML for a pipeline.
func (s *Service) Export(ctx context.Context, name string) (string, error) {
	yamlText, _, err := s.publishedYAML(ctx, name)
	return yamlText, err
}

// Delete removes a pipeline definition (and parent/compound runs) and reloads the engine.
func (s *Service) Delete(ctx context.Context, name string) error {
	if s == nil || s.catalog == nil {
		return types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	if _, err := s.catalog.GetDefinitionByName(ctx, name); err != nil {
		return err
	}
	if _, err := s.catalog.DeleteDefinitionByName(ctx, name); err != nil {
		return err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(fmt.Errorf("reload pipeline engine after delete: %w", reloadErr))
	}
	return nil
}

// StartRunAsync starts a manual pipeline run and returns the run ID.
func (s *Service) StartRunAsync(ctx context.Context, name string, eventData map[string]any, uid string) (int64, error) {
	if s == nil {
		return 0, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	engine := ActiveEngine()
	if engine == nil {
		return 0, types.Errorf(types.ErrUnavailable, "pipeline engine not ready")
	}

	yamlText, def, err := s.publishedYAML(ctx, name)
	if err != nil {
		return 0, err
	}
	if !IsEnabledInYAML(yamlText) {
		return 0, types.Errorf(types.ErrInvalidArgument, "pipeline %s is disabled", name)
	}

	event := types.DataEvent{Data: map[string]any{}}
	if eventData != nil {
		maps.Copy(event.Data, eventData)
	}
	// Always mint a unique EventID for dedup; caller may still put event_id in Data for templates.
	event.EventID = "manual-" + types.Id()
	if et, ok := event.Data["event_type"].(string); ok && strings.TrimSpace(et) != "" {
		event.EventType = strings.TrimSpace(et)
	}
	uid = strings.TrimSpace(uid)
	if uid != "" {
		event.UID = uid
	} else if def != nil && strings.TrimSpace(def.CreatedBy) != "" {
		event.UID = strings.TrimSpace(def.CreatedBy)
	}

	return engine.ExecuteManual(ctx, name, event)
}

// ListRuns returns recent runs for a parent pipeline name.
func (s *Service) ListRuns(ctx context.Context, name string) ([]*model.PipelineRun, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	return s.catalog.GetRunsByParentName(ctx, name)
}

func (s *Service) publishedYAML(ctx context.Context, name string) (string, *model.PipelineDefinition, error) {
	if s == nil || s.catalog == nil {
		return "", nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, types.Errorf(types.ErrInvalidArgument, "pipeline name is required")
	}
	def, err := s.catalog.GetDefinitionByName(ctx, name)
	if err != nil {
		return "", nil, err
	}
	if def.YamlPublished == nil || strings.TrimSpace(*def.YamlPublished) == "" {
		return "", nil, types.Errorf(types.ErrNotFound, "pipeline %s is not published", name)
	}
	return *def.YamlPublished, def, nil
}

func triggerSummaries(triggers []TriggerEntry) []string {
	out := make([]string, 0, len(triggers))
	for _, t := range triggers {
		switch t.Type {
		case "event":
			if t.Event != "" {
				out = append(out, "event:"+t.Event)
			} else {
				out = append(out, "event")
			}
		case "cron":
			if t.Cron != "" {
				out = append(out, "cron:"+t.Cron)
			} else {
				out = append(out, "cron")
			}
		case "webhook":
			out = append(out, "webhook")
		default:
			if t.Type != "" {
				out = append(out, t.Type)
			}
		}
	}
	return out
}

func stringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
