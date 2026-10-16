package types

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCorrelation(t *testing.T) {
	t.Parallel()

	kept := EnsureCorrelation(context.Background(), DataEvent{
		CorrelationID: "cor_existing",
		CausationID:   "evt-parent",
	})
	assert.Equal(t, "cor_existing", kept.CorrelationID)
	assert.Equal(t, "evt-parent", kept.CausationID)

	minted := EnsureCorrelation(context.Background(), DataEvent{EventID: "evt-1"})
	require.True(t, strings.HasPrefix(minted.CorrelationID, "cor_"))
	assert.Len(t, minted.CorrelationID, len("cor_")+26)
	assert.Empty(t, minted.CausationID)

	ctx := WithLineage(context.Background(), "cor_from_ctx", "evt-trigger")
	inherited := EnsureCorrelation(ctx, DataEvent{EventID: "evt-child"})
	assert.Equal(t, "cor_from_ctx", inherited.CorrelationID)
	assert.Equal(t, "evt-trigger", inherited.CausationID)

	alreadyCaused := EnsureCorrelation(ctx, DataEvent{CausationID: "evt-other"})
	assert.Equal(t, "cor_from_ctx", alreadyCaused.CorrelationID)
	assert.Equal(t, "evt-other", alreadyCaused.CausationID)
}

func TestWithTriggerLineageMintsOnce(t *testing.T) {
	t.Parallel()
	ctx := WithTriggerLineage(context.Background(), DataEvent{EventID: "evt-cron"})
	lin, ok := LineageFrom(ctx)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(lin.CorrelationID, "cor_"))
	assert.Equal(t, "evt-cron", lin.EventID)

	child := EnsureCorrelation(ctx, DataEvent{EventID: "evt-step"})
	assert.Equal(t, lin.CorrelationID, child.CorrelationID)
	assert.Equal(t, "evt-cron", child.CausationID)

	boundCtx, bound := BindTriggerLineage(context.Background(), DataEvent{EventID: "evt-cron"})
	boundLin, ok := LineageFrom(boundCtx)
	require.True(t, ok)
	assert.Equal(t, bound.CorrelationID, boundLin.CorrelationID)
	assert.Equal(t, "evt-cron", boundLin.EventID)
	childOfBound := EnsureCorrelation(boundCtx, DataEvent{EventID: "evt-step"})
	assert.Equal(t, bound.CorrelationID, childOfBound.CorrelationID)
	assert.Equal(t, "evt-cron", childOfBound.CausationID)

	again := WithTriggerLineage(context.Background(), DataEvent{
		EventID:       "evt-hook",
		CorrelationID: "cor_hook",
	})
	hook, ok := LineageFrom(again)
	require.True(t, ok)
	assert.Equal(t, "cor_hook", hook.CorrelationID)
}
