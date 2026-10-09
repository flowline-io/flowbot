package pipeline

import (
	"context"
	"errors"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// BlueprintLibrary persists user-imported blueprint templates.
type BlueprintLibrary interface {
	ListUserTemplates(ctx context.Context) ([]BlueprintTemplateRecord, error)
	GetUserTemplate(ctx context.Context, blueprintID string) (*BlueprintTemplateRecord, error)
	UpsertUserTemplate(ctx context.Context, rec BlueprintTemplateRecord) error
	DeleteUserTemplate(ctx context.Context, blueprintID string) error
}

// BlueprintOriginStore persists origin metadata on pipeline definitions.
type BlueprintOriginStore interface {
	CreateDefinition(ctx context.Context, name, description, createdBy string) error
	GetDefinitionByName(ctx context.Context, name string) (*model.PipelineDefinition, error)
	UpdateDefinitionDraft(ctx context.Context, name, yamlDraft string, version int) (*model.PipelineDefinition, error)
	PublishDefinition(ctx context.Context, name string, version int) (*model.PipelineDefinition, error)
	SetBlueprintOrigin(ctx context.Context, name string, origin BlueprintOrigin) error
	ClearBlueprintOrigin(ctx context.Context, name string) error
	ListLinkedPipelineNames(ctx context.Context, source, blueprintID string) ([]string, error)
	EnsureDefinitionCreatedBy(ctx context.Context, name, createdBy string) error
}

// BlueprintService orchestrates the template library and linked pipeline instances.
type BlueprintService struct {
	library   BlueprintLibrary
	instances BlueprintOriginStore
}

// NewBlueprintService creates a BlueprintService.
func NewBlueprintService(library BlueprintLibrary, instances BlueprintOriginStore) *BlueprintService {
	return &BlueprintService{library: library, instances: instances}
}

var activeBlueprintService *BlueprintService

// SetActiveBlueprintService wires the package-level blueprint service.
func SetActiveBlueprintService(svc *BlueprintService) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeBlueprintService = svc
}

// ActiveBlueprintService returns the wired blueprint service, or nil.
func ActiveBlueprintService() *BlueprintService {
	activeMu.Lock()
	defer activeMu.Unlock()
	return activeBlueprintService
}

// ListCatalog returns official plus user templates with instance counts and missing caps.
func (s *BlueprintService) ListCatalog(ctx context.Context) ([]BlueprintCatalogEntry, error) {
	if s == nil || s.library == nil || s.instances == nil {
		return nil, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	builtin, err := LoadBuiltinBlueprints()
	if err != nil {
		return nil, err
	}
	out := make([]BlueprintCatalogEntry, 0, len(builtin)+8)
	for _, b := range builtin {
		count, err := s.instanceCount(ctx, BlueprintSourceBuiltin, b.Document.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, BlueprintCatalogEntry{
			Source:        BlueprintSourceBuiltin,
			Document:      b.Document,
			YAML:          b.YAML,
			Hash:          b.Hash,
			Missing:       MissingCapabilities(b.Document.Requires.Capabilities),
			InstanceCount: count,
		})
	}
	user, err := s.library.ListUserTemplates(ctx)
	if err != nil {
		return nil, err
	}
	for _, rec := range user {
		doc, err := ParseBlueprintYAML([]byte(rec.YAML))
		if err != nil {
			flog.Error(err)
			continue
		}
		count, err := s.instanceCount(ctx, BlueprintSourceUpload, rec.BlueprintID)
		if err != nil {
			return nil, err
		}
		out = append(out, BlueprintCatalogEntry{
			Source:        BlueprintSourceUpload,
			Document:      *doc,
			YAML:          rec.YAML,
			Hash:          rec.Hash,
			Missing:       MissingCapabilities(doc.Requires.Capabilities),
			InstanceCount: count,
		})
	}
	return out, nil
}

// ResolveTemplate loads a builtin or user template.
func (s *BlueprintService) ResolveTemplate(ctx context.Context, source, id string) (*BlueprintCatalogEntry, error) {
	if s == nil || s.library == nil {
		return nil, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	source = strings.TrimSpace(source)
	id = strings.TrimSpace(id)
	if err := ValidateBlueprintID(id); err != nil {
		return nil, err
	}
	switch source {
	case BlueprintSourceBuiltin:
		b, err := LookupBuiltinBlueprint(id)
		if err != nil {
			return nil, err
		}
		return &BlueprintCatalogEntry{
			Source:   BlueprintSourceBuiltin,
			Document: b.Document,
			YAML:     b.YAML,
			Hash:     b.Hash,
			Missing:  MissingCapabilities(b.Document.Requires.Capabilities),
		}, nil
	case BlueprintSourceUpload:
		rec, err := s.library.GetUserTemplate(ctx, id)
		if err != nil {
			return nil, err
		}
		doc, err := ParseBlueprintYAML([]byte(rec.YAML))
		if err != nil {
			return nil, err
		}
		return &BlueprintCatalogEntry{
			Source:   BlueprintSourceUpload,
			Document: *doc,
			YAML:     rec.YAML,
			Hash:     rec.Hash,
			Missing:  MissingCapabilities(doc.Requires.Capabilities),
		}, nil
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "unknown blueprint source %s", source)
	}
}

// ImportYAML adds a user template. Ids that collide with official or existing user templates are rejected.
func (s *BlueprintService) ImportYAML(ctx context.Context, yamlBytes []byte, createdBy string) (*BlueprintTemplateRecord, error) {
	if s == nil || s.library == nil {
		return nil, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	doc, err := ParseBlueprintYAML(yamlBytes)
	if err != nil {
		return nil, err
	}
	ids, err := BuiltinBlueprintIDs()
	if err != nil {
		return nil, err
	}
	if _, ok := ids[doc.ID]; ok {
		return nil, types.Errorf(types.ErrAlreadyExists, "blueprint id %s is reserved by an official template", doc.ID)
	}
	if _, err := s.library.GetUserTemplate(ctx, doc.ID); err == nil {
		return nil, types.Errorf(types.ErrAlreadyExists, "blueprint %s already imported; replace it instead", doc.ID)
	} else if !errors.Is(err, types.ErrNotFound) {
		return nil, err
	}
	rec := BlueprintTemplateRecord{
		BlueprintID: doc.ID,
		Title:       doc.Title,
		Description: doc.Description,
		YAML:        string(yamlBytes),
		Hash:        ContentHash(yamlBytes),
		CreatedBy:   strings.TrimSpace(createdBy),
	}
	if err := s.library.UpsertUserTemplate(ctx, rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// ReplaceYAML updates a user template in place. Id in YAML must match.
func (s *BlueprintService) ReplaceYAML(ctx context.Context, blueprintID string, yamlBytes []byte) (*BlueprintTemplateRecord, error) {
	if s == nil || s.library == nil {
		return nil, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	if err := ValidateBlueprintID(blueprintID); err != nil {
		return nil, err
	}
	doc, err := ParseBlueprintYAML(yamlBytes)
	if err != nil {
		return nil, err
	}
	if doc.ID != blueprintID {
		return nil, types.Errorf(types.ErrInvalidArgument, "YAML id %s does not match %s", doc.ID, blueprintID)
	}
	existing, err := s.library.GetUserTemplate(ctx, blueprintID)
	if err != nil {
		return nil, err
	}
	rec := BlueprintTemplateRecord{
		BlueprintID: blueprintID,
		Title:       doc.Title,
		Description: doc.Description,
		YAML:        string(yamlBytes),
		Hash:        ContentHash(yamlBytes),
		CreatedBy:   existing.CreatedBy,
	}
	if err := s.library.UpsertUserTemplate(ctx, rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// DeleteUserTemplate removes a user template when no linked instances remain.
func (s *BlueprintService) DeleteUserTemplate(ctx context.Context, blueprintID string) error {
	if s == nil || s.library == nil || s.instances == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	if err := ValidateBlueprintID(blueprintID); err != nil {
		return err
	}
	names, err := s.instances.ListLinkedPipelineNames(ctx, BlueprintSourceUpload, blueprintID)
	if err != nil {
		return err
	}
	if len(names) > 0 {
		return types.Errorf(types.ErrConflict, "blueprint %s is used by %d pipeline(s)", blueprintID, len(names))
	}
	return s.library.DeleteUserTemplate(ctx, blueprintID)
}

// InstantiateRequest creates a linked pipeline from a catalog template.
type InstantiateRequest struct {
	Source       string
	BlueprintID  string
	PipelineName string
	Inputs       map[string]any
	Enable       bool
	CreatedBy    string
}

// Instantiate creates a new linked pipeline instance.
func (s *BlueprintService) Instantiate(ctx context.Context, req InstantiateRequest) (*model.PipelineDefinition, error) {
	if s == nil || s.instances == nil {
		return nil, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	entry, err := s.ResolveTemplate(ctx, req.Source, req.BlueprintID)
	if err != nil {
		return nil, err
	}
	if len(entry.Missing) > 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "missing capabilities: %s", strings.Join(entry.Missing, ", "))
	}
	yamlText, err := MaterializeEditorYAML(&entry.Document, req.PipelineName, req.Enable, req.Inputs)
	if err != nil {
		return nil, err
	}
	bound, err := BindBlueprintInputs(entry.Document.Inputs, req.Inputs)
	if err != nil {
		return nil, err
	}
	if err := s.instances.CreateDefinition(ctx, req.PipelineName, entry.Document.Description, req.CreatedBy); err != nil {
		return nil, err
	}
	def, err := s.instances.GetDefinitionByName(ctx, req.PipelineName)
	if err != nil {
		return nil, err
	}
	updated, err := s.instances.UpdateDefinitionDraft(ctx, req.PipelineName, yamlText, def.Version)
	if err != nil {
		return nil, err
	}
	if err := s.instances.SetBlueprintOrigin(ctx, req.PipelineName, BlueprintOrigin{
		Source: req.Source,
		ID:     entry.Document.ID,
		Hash:   entry.Hash,
		YAML:   entry.YAML,
		Inputs: bound,
	}); err != nil {
		return nil, err
	}
	published, err := s.instances.PublishDefinition(ctx, req.PipelineName, updated.Version)
	if err != nil {
		return nil, err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(reloadErr)
	}
	return published, nil
}

// UpdateInstanceInputs rematerializes a linked pipeline with new input values.
// When enable is non-nil, the published enabled flag is set to that value.
func (s *BlueprintService) UpdateInstanceInputs(ctx context.Context, pipelineName string, inputs map[string]any, enable *bool) (*model.PipelineDefinition, error) {
	def, origin, err := s.requireLinked(ctx, pipelineName)
	if err != nil {
		return nil, err
	}
	doc, err := ParseBlueprintYAML([]byte(origin.YAML))
	if err != nil {
		return nil, err
	}
	enabled := instanceEnabled(def)
	if enable != nil {
		enabled = *enable
	}
	yamlText, err := MaterializeEditorYAML(doc, pipelineName, enabled, inputs)
	if err != nil {
		return nil, err
	}
	bound, err := BindBlueprintInputs(doc.Inputs, inputs)
	if err != nil {
		return nil, err
	}
	updated, err := s.instances.UpdateDefinitionDraft(ctx, pipelineName, yamlText, def.Version)
	if err != nil {
		return nil, err
	}
	origin.Inputs = bound
	if err := s.instances.SetBlueprintOrigin(ctx, pipelineName, origin); err != nil {
		return nil, err
	}
	published, err := s.instances.PublishDefinition(ctx, pipelineName, updated.Version)
	if err != nil {
		return nil, err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(reloadErr)
	}
	return published, nil
}

// ConfirmInstanceUpdate rematerializes from the current catalog template using stored inputs.
func (s *BlueprintService) ConfirmInstanceUpdate(ctx context.Context, pipelineName string) (*model.PipelineDefinition, error) {
	def, origin, err := s.requireLinked(ctx, pipelineName)
	if err != nil {
		return nil, err
	}
	entry, err := s.ResolveTemplate(ctx, origin.Source, origin.ID)
	if err != nil {
		return nil, err
	}
	if entry.Hash == origin.Hash {
		return def, nil
	}
	enabled := instanceEnabled(def)
	yamlText, err := MaterializeEditorYAML(&entry.Document, pipelineName, enabled, origin.Inputs)
	if err != nil {
		return nil, err
	}
	bound, err := BindBlueprintInputs(entry.Document.Inputs, origin.Inputs)
	if err != nil {
		return nil, err
	}
	updated, err := s.instances.UpdateDefinitionDraft(ctx, pipelineName, yamlText, def.Version)
	if err != nil {
		return nil, err
	}
	origin.YAML = entry.YAML
	origin.Hash = entry.Hash
	origin.Inputs = bound
	if err := s.instances.SetBlueprintOrigin(ctx, pipelineName, origin); err != nil {
		return nil, err
	}
	published, err := s.instances.PublishDefinition(ctx, pipelineName, updated.Version)
	if err != nil {
		return nil, err
	}
	if reloadErr := ReloadDefinitions(ctx); reloadErr != nil {
		flog.Error(reloadErr)
	}
	return published, nil
}

// TakeControl irreversibly detaches origin from a linked pipeline.
func (s *BlueprintService) TakeControl(ctx context.Context, pipelineName string) error {
	_, _, err := s.requireLinked(ctx, pipelineName)
	if err != nil {
		return err
	}
	return s.instances.ClearBlueprintOrigin(ctx, pipelineName)
}

// InstanceUpdateAvailable reports whether the catalog template hash differs from the instance.
func (s *BlueprintService) InstanceUpdateAvailable(ctx context.Context, def *model.PipelineDefinition) (bool, error) {
	if def == nil || !originLinked(def.BlueprintSource, def.BlueprintID) {
		return false, nil
	}
	entry, err := s.ResolveTemplate(ctx, def.BlueprintSource, def.BlueprintID)
	if err != nil {
		return false, err
	}
	return entry.Hash != def.BlueprintHash, nil
}

func (s *BlueprintService) requireLinked(ctx context.Context, pipelineName string) (*model.PipelineDefinition, BlueprintOrigin, error) {
	if s == nil || s.instances == nil {
		return nil, BlueprintOrigin{}, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	def, err := s.instances.GetDefinitionByName(ctx, pipelineName)
	if err != nil {
		return nil, BlueprintOrigin{}, err
	}
	origin, ok := OriginFromDefinition(def)
	if !ok {
		return nil, BlueprintOrigin{}, types.Errorf(types.ErrInvalidArgument, "pipeline %s is not a blueprint instance", pipelineName)
	}
	return def, origin, nil
}

func instanceEnabled(def *model.PipelineDefinition) bool {
	if def == nil {
		return false
	}
	if def.YamlPublished != nil && *def.YamlPublished != "" {
		return IsEnabledInYAML(*def.YamlPublished)
	}
	return IsEnabledInYAML(def.YamlDraft)
}

func (s *BlueprintService) instanceCount(ctx context.Context, source, id string) (int, error) {
	names, err := s.instances.ListLinkedPipelineNames(ctx, source, id)
	if err != nil {
		return 0, err
	}
	return len(names), nil
}
