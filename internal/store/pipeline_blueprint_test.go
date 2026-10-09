package store_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestPipelineBlueprintTemplateCRUD(t *testing.T) {
	client := sqlitetest.OpenClient(t, t.Name())
	s := store.NewPipelineStore(client)
	ctx := t.Context()

	b, err := pipeline.LookupBuiltinBlueprint("webhook_notify")
	require.NoError(t, err)

	rec := pipeline.BlueprintTemplateRecord{
		BlueprintID: "user_webhook",
		Title:       "User webhook",
		Description: "copy",
		YAML:        b.YAML,
		Hash:        pipeline.ContentHash([]byte(b.YAML)),
		CreatedBy:   "user-1",
	}
	require.NoError(t, s.UpsertUserTemplate(ctx, rec))

	got, err := s.GetUserTemplate(ctx, "user_webhook")
	require.NoError(t, err)
	assert.Equal(t, "User webhook", got.Title)

	rec.Title = "Updated"
	require.NoError(t, s.UpsertUserTemplate(ctx, rec))
	got, err = s.GetUserTemplate(ctx, "user_webhook")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)

	list, err := s.ListUserTemplates(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, s.CreateDefinition(ctx, "from-bp", "d", "user-1"))
	require.NoError(t, s.SetBlueprintOrigin(ctx, "from-bp", pipeline.BlueprintOrigin{
		Source: pipeline.BlueprintSourceUpload,
		ID:     "user_webhook",
		Hash:   rec.Hash,
		YAML:   rec.YAML,
		Inputs: map[string]any{"template_id": "x"},
	}))
	names, err := s.ListLinkedPipelineNames(ctx, pipeline.BlueprintSourceUpload, "user_webhook")
	require.NoError(t, err)
	assert.Equal(t, []string{"from-bp"}, names)

	require.NoError(t, s.ClearBlueprintOrigin(ctx, "from-bp"))
	names, err = s.ListLinkedPipelineNames(ctx, pipeline.BlueprintSourceUpload, "user_webhook")
	require.NoError(t, err)
	assert.Empty(t, names)

	require.NoError(t, s.DeleteUserTemplate(ctx, "user_webhook"))
	_, err = s.GetUserTemplate(ctx, "user_webhook")
	assert.ErrorIs(t, err, types.ErrNotFound)
}
