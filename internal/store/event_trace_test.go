package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestDataEventTraceColumns(t *testing.T) {
	t.Parallel()
	st := NewEventStore(getTestClient(t))
	ctx := t.Context()
	event := types.DataEvent{
		EventID:       "evt-trace-1",
		EventType:     "test.trace",
		Source:        "mcp",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		TraceID:       "4bf92f3577b34da6a3ce929d0e0e4736",
	}
	require.NoError(t, st.AppendDataEvent(ctx, event))

	row, err := st.GetDataEventByEventID(ctx, event.EventID)
	require.NoError(t, err)
	require.NotNil(t, row)
	got := DataEventFromRow(row)
	assert.Equal(t, "corr-1", got.CorrelationID)
	assert.Equal(t, "cause-1", got.CausationID)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", got.TraceID)

	require.NoError(t, st.AppendEventOutbox(ctx, event))
	pending, err := st.ListPendingDataEventOutbox(ctx, time.Now().Add(time.Minute), 10)
	require.NoError(t, err)
	var found bool
	for _, item := range pending {
		if item.EventID == event.EventID {
			found = true
			assert.Equal(t, event.CorrelationID, item.CorrelationID)
			assert.Equal(t, event.CausationID, item.CausationID)
			assert.Equal(t, event.TraceID, item.TraceID)
		}
	}
	assert.True(t, found)
}
