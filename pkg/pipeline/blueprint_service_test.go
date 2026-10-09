package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type mockLibrary struct {
	templates map[string]BlueprintTemplateRecord
}

func (m *mockLibrary) ListUserTemplates(_ context.Context) ([]BlueprintTemplateRecord, error) {
	out := make([]BlueprintTemplateRecord, 0, len(m.templates))
	for _, rec := range m.templates {
		out = append(out, rec)
	}
	return out, nil
}

func (m *mockLibrary) GetUserTemplate(_ context.Context, id string) (*BlueprintTemplateRecord, error) {
	rec, ok := m.templates[id]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := rec
	return &cp, nil
}

func (m *mockLibrary) UpsertUserTemplate(_ context.Context, rec BlueprintTemplateRecord) error {
	if m.templates == nil {
		m.templates = map[string]BlueprintTemplateRecord{}
	}
	m.templates[rec.BlueprintID] = rec
	return nil
}

func (m *mockLibrary) DeleteUserTemplate(_ context.Context, id string) error {
	if _, ok := m.templates[id]; !ok {
		return types.ErrNotFound
	}
	delete(m.templates, id)
	return nil
}

func TestBlueprintServiceInstantiateAndTakeControl(t *testing.T) {
	lib := &mockLibrary{templates: map[string]BlueprintTemplateRecord{}}
	cat := newMockCatalog()
	svc := NewBlueprintService(lib, cat)

	require.NoError(t, hub.Default.Register(hub.Descriptor{Type: hub.CapCore, App: "core"}))
	t.Cleanup(func() { hub.Default.Unregister(hub.CapCore) })

	def, err := svc.Instantiate(t.Context(), InstantiateRequest{
		Source:       BlueprintSourceBuiltin,
		BlueprintID:  "webhook_notify",
		PipelineName: "wh1",
		Inputs: map[string]any{
			"webhook_path":   "hooks/a",
			"notify_channel": []string{"ntfy"},
			"template_id":    "inbox",
		},
		Enable:    false,
		CreatedBy: "user-1",
	})
	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Equal(t, "webhook_notify", def.BlueprintID)
	assert.Equal(t, BlueprintSourceBuiltin, def.BlueprintSource)
	assert.False(t, IsEnabledInYAML(*def.YamlPublished))

	require.NoError(t, svc.TakeControl(t.Context(), "wh1"))
	got, err := cat.GetDefinitionByName(t.Context(), "wh1")
	require.NoError(t, err)
	assert.Empty(t, got.BlueprintID)
	assert.NotEmpty(t, *got.YamlPublished)
}

func TestBlueprintServiceInstantiateRejectsMissingCapabilities(t *testing.T) {
	lib := &mockLibrary{templates: map[string]BlueprintTemplateRecord{}}
	svc := NewBlueprintService(lib, newMockCatalog())
	yamlBytes := []byte(`kind: pipeline_blueprint
id: missing_cap_probe
title: Missing cap probe
requires:
  capabilities:
    - not_a_registered_capability
definition:
  description: probe
  steps:
    - name: noop
      capability: core
      operation: notify_send
`)
	_, err := svc.ImportYAML(t.Context(), yamlBytes, "user-1")
	require.NoError(t, err)
	_, err = svc.Instantiate(t.Context(), InstantiateRequest{
		Source:       BlueprintSourceUpload,
		BlueprintID:  "missing_cap_probe",
		PipelineName: "probe1",
		CreatedBy:    "user-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing capabilities")
}

func TestBlueprintServiceImportRejectsOfficialID(t *testing.T) {
	lib := &mockLibrary{templates: map[string]BlueprintTemplateRecord{}}
	svc := NewBlueprintService(lib, newMockCatalog())
	b, err := LookupBuiltinBlueprint("webhook_notify")
	require.NoError(t, err)
	_, err = svc.ImportYAML(t.Context(), []byte(b.YAML), "user-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrAlreadyExists)
}

func TestApplyYAMLRejectsLinkedInstance(t *testing.T) {
	cat := newMockCatalog()
	svc := NewService(cat)
	require.NoError(t, cat.CreateDefinition(t.Context(), "linked_pipe", "", "user-1"))
	require.NoError(t, cat.SetBlueprintOrigin(t.Context(), "linked_pipe", BlueprintOrigin{
		Source: BlueprintSourceBuiltin,
		ID:     "webhook_notify",
		Hash:   "abc",
		YAML:   "kind: pipeline_blueprint\nid: webhook_notify\n",
	}))
	yamlText := `
name: linked_pipe
enabled: true
triggers:
  - type: webhook
    enabled: true
    webhook:
      path: hooks/x
steps:
  - name: notify
    capability: core
    operation: notify_send
    params:
      template_id: x
      channels: ["a"]
`
	_, err := svc.ApplyYAML(t.Context(), []byte(yamlText), "user-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrConflict)
}

func TestApplyYAMLRejectsBlueprint(t *testing.T) {
	svc := NewService(newMockCatalog())
	b, err := LookupBuiltinBlueprint("webhook_notify")
	require.NoError(t, err)
	_, err = svc.ApplyYAML(t.Context(), []byte(b.YAML), "user-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blueprint")
}
