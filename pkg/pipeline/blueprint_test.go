package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/pipeline"
)

func TestParseAndMaterializeBuiltinWebhook(t *testing.T) {
	t.Parallel()
	b, err := pipeline.LookupBuiltinBlueprint("webhook_notify")
	require.NoError(t, err)
	yamlText, err := pipeline.MaterializeEditorYAML(&b.Document, "wh-notify-1", false, map[string]any{
		"webhook_path":   "hooks/notify",
		"notify_channel": []string{"ntfy"},
		"template_id":    "inbox",
	})
	require.NoError(t, err)
	ed, err := pipeline.ParseEditorYAML(yamlText)
	require.NoError(t, err)
	assert.Equal(t, "wh-notify-1", ed.Name)
	assert.False(t, ed.Enabled)
	require.Len(t, ed.Triggers, 1)
	require.NotNil(t, ed.Triggers[0].Webhook)
	assert.Equal(t, "hooks/notify", ed.Triggers[0].Webhook.Path)
	require.Len(t, ed.Steps, 1)
	channels, ok := ed.Steps[0].Params["channels"].([]any)
	if !ok {
		strList, ok2 := ed.Steps[0].Params["channels"].([]string)
		require.True(t, ok2)
		assert.Equal(t, []string{"ntfy"}, strList)
	} else {
		require.Len(t, channels, 1)
		assert.Equal(t, "ntfy", channels[0])
	}
	assert.Equal(t, "inbox", ed.Steps[0].Params["template_id"])
}

func TestLooksLikeBlueprint(t *testing.T) {
	t.Parallel()
	assert.True(t, pipeline.LooksLikeBlueprint([]byte("kind: pipeline_blueprint\nid: x\n")))
	assert.False(t, pipeline.LooksLikeBlueprint([]byte("name: rss_fetch_and_notify\nsteps: []\n")))
}

func TestBindRequiredInput(t *testing.T) {
	t.Parallel()
	defs := []pipeline.BlueprintInputDef{
		{Name: "template_id", Type: pipeline.BlueprintInputString, Required: true},
		{Name: "notify_channel", Type: pipeline.BlueprintInputNotifyChannel, Required: true},
	}
	_, err := pipeline.BindBlueprintInputs(defs, map[string]any{"template_id": "inbox"})
	require.Error(t, err)
	bound, err := pipeline.BindBlueprintInputs(defs, map[string]any{
		"template_id":    "inbox",
		"notify_channel": "slack",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"slack"}, bound["notify_channel"])
}

func TestLoadBuiltinBlueprints(t *testing.T) {
	t.Parallel()
	all, err := pipeline.LoadBuiltinBlueprints()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 4)
	ids, err := pipeline.BuiltinBlueprintIDs()
	require.NoError(t, err)
	_, ok := ids["webhook_notify"]
	assert.True(t, ok)
}

func TestMissingCapabilities(t *testing.T) {
	t.Parallel()
	missing := pipeline.MissingCapabilities([]string{"not_a_registered_capability"})
	assert.Equal(t, []string{"not_a_registered_capability"}, missing)
}
