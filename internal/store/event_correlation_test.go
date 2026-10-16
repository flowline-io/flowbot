package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestAppendDataEvent_FillsCorrelation(t *testing.T) {
	t.Parallel()
	st := NewEventStore(getTestClient(t))
	ctx := t.Context()

	require.NoError(t, st.AppendDataEvent(ctx, types.DataEvent{
		EventID:   "evt-corr-mint",
		EventType: "item.created",
	}))
	row, err := st.GetDataEventByEventID(ctx, "evt-corr-mint")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, strings.HasPrefix(row.CorrelationID, "cor_"))
	assert.Empty(t, row.CausationID)

	parent := types.WithLineage(ctx, "cor_parent", "evt-parent")
	require.NoError(t, st.AppendDataEvent(parent, types.DataEvent{
		EventID:   "evt-corr-child",
		EventType: "item.created",
	}))
	child, err := st.GetDataEventByEventID(ctx, "evt-corr-child")
	require.NoError(t, err)
	require.NotNil(t, child)
	assert.Equal(t, "cor_parent", child.CorrelationID)
	assert.Equal(t, "evt-parent", child.CausationID)

	require.NoError(t, st.AppendDataEvent(ctx, types.DataEvent{
		EventID:       "evt-corr-keep",
		EventType:     "item.created",
		CorrelationID: "cor_kept",
		CausationID:   "evt-kept",
	}))
	kept, err := st.GetDataEventByEventID(ctx, "evt-corr-keep")
	require.NoError(t, err)
	require.NotNil(t, kept)
	assert.Equal(t, "cor_kept", kept.CorrelationID)
	assert.Equal(t, "evt-kept", kept.CausationID)
}
