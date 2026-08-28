package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

type mockCatalog struct {
	defs map[string]*model.PipelineDefinition
	runs map[string][]*model.PipelineRun
}

func newMockCatalog() *mockCatalog {
	return &mockCatalog{
		defs: map[string]*model.PipelineDefinition{},
		runs: map[string][]*model.PipelineRun{},
	}
}

func (m *mockCatalog) CreateDefinition(_ context.Context, name, description, createdBy string) error {
	if _, ok := m.defs[name]; ok {
		return types.ErrAlreadyExists
	}
	m.defs[name] = &model.PipelineDefinition{
		ID:          1,
		Name:        name,
		Description: description,
		YamlDraft:   "",
		Version:     1,
		Status:      "draft",
		CreatedBy:   createdBy,
	}
	return nil
}

func (m *mockCatalog) GetDefinitionByName(_ context.Context, name string) (*model.PipelineDefinition, error) {
	def, ok := m.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *def
	return &cp, nil
}

func (m *mockCatalog) UpdateDefinitionDraft(_ context.Context, name, yamlDraft string, version int) (*model.PipelineDefinition, error) {
	def, ok := m.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	if def.Version != version {
		return nil, types.ErrConflict
	}
	def.YamlDraft = yamlDraft
	def.Version++
	cp := *def
	return &cp, nil
}

func (m *mockCatalog) PublishDefinition(_ context.Context, name string, version int) (*model.PipelineDefinition, error) {
	def, ok := m.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	if def.Version != version {
		return nil, types.ErrConflict
	}
	if def.YamlDraft == "" {
		return nil, types.ErrConflict
	}
	published := def.YamlDraft
	def.YamlPublished = &published
	def.Status = "published"
	def.Version++
	cp := *def
	return &cp, nil
}

func (m *mockCatalog) EnsureDefinitionCreatedBy(_ context.Context, name, createdBy string) error {
	def, ok := m.defs[name]
	if !ok {
		return types.ErrNotFound
	}
	if def.CreatedBy == "" {
		def.CreatedBy = createdBy
	}
	return nil
}

func (m *mockCatalog) DeleteDefinitionByName(_ context.Context, name string) (int64, error) {
	if _, ok := m.defs[name]; !ok {
		return 0, nil
	}
	n := int64(len(m.runs[name]))
	delete(m.defs, name)
	delete(m.runs, name)
	return n, nil
}

func (m *mockCatalog) ListPublishedDefinitions(_ context.Context) ([]DefinitionRecord, error) {
	var out []DefinitionRecord
	for _, def := range m.defs {
		if def.YamlPublished == nil {
			continue
		}
		out = append(out, DefinitionRecord{
			Name:        def.Name,
			Description: def.Description,
			YAML:        *def.YamlPublished,
			CreatedBy:   def.CreatedBy,
		})
	}
	return out, nil
}

func (m *mockCatalog) GetRunsByParentName(_ context.Context, parentName string) ([]*model.PipelineRun, error) {
	return m.runs[parentName], nil
}

func TestServiceApplyYAMLAndExport(t *testing.T) {
	cat := newMockCatalog()
	svc := NewService(cat)
	yamlText := `
name: demo_pipe
description: demo
enabled: true
resumable: false
triggers:
  - type: event
    enabled: true
    event: demo.created
steps:
  - name: noop
    capability: core
    operation: echo
    params: {}
`
	res, err := svc.ApplyYAML(context.Background(), []byte(yamlText), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "demo_pipe", res.Name)
	assert.True(t, res.Enabled)

	exported, err := svc.Export(context.Background(), "demo_pipe")
	require.NoError(t, err)
	assert.Contains(t, exported, "demo_pipe")

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "demo_pipe", list[0].Name)
	assert.Contains(t, list[0].Triggers, "event:demo.created")
}

type conflictOnUpdateCatalog struct {
	*mockCatalog
}

func (*conflictOnUpdateCatalog) UpdateDefinitionDraft(context.Context, string, string, int) (*model.PipelineDefinition, error) {
	return nil, types.ErrConflict
}

func TestServiceApplyYAMLConflict(t *testing.T) {
	base := newMockCatalog()
	require.NoError(t, base.CreateDefinition(context.Background(), "conflict_pipe", "", ""))
	svc := NewService(&conflictOnUpdateCatalog{mockCatalog: base})
	yamlText := `
name: conflict_pipe
enabled: true
triggers:
  - type: cron
    enabled: true
    cron: "@daily"
steps: []
`
	_, err := svc.ApplyYAML(context.Background(), []byte(yamlText), "")
	require.ErrorIs(t, err, types.ErrConflict)
}

func TestServiceExportNotPublished(t *testing.T) {
	cat := newMockCatalog()
	require.NoError(t, cat.CreateDefinition(context.Background(), "draft_only", "", ""))
	svc := NewService(cat)
	_, err := svc.Export(context.Background(), "draft_only")
	require.ErrorIs(t, err, types.ErrNotFound)
}

func TestServiceDeleteAndListRuns(t *testing.T) {
	cat := newMockCatalog()
	svc := NewService(cat)
	yamlText := `
name: del_pipe
enabled: true
triggers:
  - type: event
    enabled: true
    event: x.created
steps: []
`
	_, err := svc.ApplyYAML(context.Background(), []byte(yamlText), "")
	require.NoError(t, err)
	cat.runs["del_pipe"] = []*model.PipelineRun{{ID: 9, PipelineName: "del_pipe__trigger_event_0"}}

	runs, err := svc.ListRuns(context.Background(), "del_pipe")
	require.NoError(t, err)
	require.Len(t, runs, 1)

	require.NoError(t, svc.Delete(context.Background(), "del_pipe"))
	_, err = svc.Export(context.Background(), "del_pipe")
	require.ErrorIs(t, err, types.ErrNotFound)
}

func TestServiceStartRunAsync(t *testing.T) {
	cat := newMockCatalog()
	svc := NewService(cat)
	yamlText := `
name: run_pipe
enabled: true
triggers:
  - type: event
    enabled: true
    event: run.created
steps: []
`
	_, err := svc.ApplyYAML(context.Background(), []byte(yamlText), "owner-1")
	require.NoError(t, err)

	t.Run("engine not ready", func(t *testing.T) {
		SetActiveEngine(nil)
		_, err := svc.StartRunAsync(context.Background(), "run_pipe", map[string]any{"event_type": "run.created"}, "")
		require.ErrorIs(t, err, types.ErrUnavailable)
	})

	t.Run("disabled", func(t *testing.T) {
		disabled := `
name: run_pipe
enabled: false
triggers:
  - type: event
    enabled: true
    event: run.created
steps: []
`
		_, err := svc.ApplyYAML(context.Background(), []byte(disabled), "")
		require.NoError(t, err)
		// Engine pointer only needs to be non-nil; disabled fails before ExecuteManual.
		SetActiveEngine(&Engine{})
		t.Cleanup(func() { SetActiveEngine(nil) })
		_, err = svc.StartRunAsync(context.Background(), "run_pipe", nil, "")
		require.ErrorIs(t, err, types.ErrInvalidArgument)
	})

	t.Run("success mints unique event id", func(t *testing.T) {
		enabled := `
name: run_pipe
enabled: true
triggers:
  - type: event
    enabled: true
    event: run.created
steps: []
`
		_, err := svc.ApplyYAML(context.Background(), []byte(enabled), "owner-1")
		require.NoError(t, err)

		defs := []Definition{{
			Name: "run_pipe__trigger_event_0", ParentName: "run_pipe", Enabled: true,
			Trigger: Trigger{Event: "run.created"}, UID: "owner-1",
		}}
		store := newMockPipelineStore()
		eng := NewEngine(defs, store, nil, nil, nil)
		SetActiveEngine(eng)
		t.Cleanup(func() {
			SetActiveEngine(nil)
			go eng.Stop()
		})

		runID, err := svc.StartRunAsync(context.Background(), "run_pipe", map[string]any{
			"event_type": "run.created",
			"event_id":   "caller-should-not-win",
			"title":      "hi",
		}, "token-uid")
		require.NoError(t, err)
		require.NotZero(t, runID)

		store.mu.Lock()
		run := store.runs[runID]
		store.mu.Unlock()
		require.NotNil(t, run)
		assert.True(t, strings.HasPrefix(run.EventID, "manual-"))
		assert.NotEqual(t, "caller-should-not-win", run.EventID)
		assert.Equal(t, "manual", string(run.TriggerSource))
	})
}
