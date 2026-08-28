package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestFindDefsByParentName(t *testing.T) {
	defs := []Definition{
		{Name: "p__trigger_event_0", ParentName: "p", Trigger: Trigger{Event: "a.created"}},
		{Name: "p__trigger_cron_1", ParentName: "p", Trigger: Trigger{Cron: "@daily"}},
		{Name: "other", ParentName: "other"},
	}
	got := FindDefsByParentName(defs, "p")
	require.Len(t, got, 2)
	assert.Empty(t, FindDefsByParentName(defs, "missing"))
	assert.Empty(t, FindDefsByParentName(defs, ""))
}

func TestSelectManualDef(t *testing.T) {
	defs := []Definition{
		{Name: "p__trigger_event_0", ParentName: "p", Trigger: Trigger{Event: "item.created"}},
		{Name: "p__trigger_cron_1", ParentName: "p", Trigger: Trigger{Cron: "@hourly"}},
	}

	t.Run("prefers matching event type", func(t *testing.T) {
		def, err := SelectManualDef(defs, "p", "item.created")
		require.NoError(t, err)
		assert.Equal(t, "p__trigger_event_0", def.Name)
	})

	t.Run("falls back to first when event type missing", func(t *testing.T) {
		def, err := SelectManualDef(defs, "p", "")
		require.NoError(t, err)
		assert.Equal(t, "p__trigger_event_0", def.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := SelectManualDef(defs, "missing", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, types.ErrNotFound)
	})
}

func TestExecuteManual(t *testing.T) {
	defs := []Definition{
		{Name: "demo__trigger_event_0", ParentName: "demo", Enabled: true, Trigger: Trigger{Event: "demo.created"}, Steps: nil},
	}
	store := newMockPipelineStore()
	eng := NewEngine(defs, store, nil, nil, nil)
	t.Cleanup(func() {
		// Best-effort; do not block the suite on cron drain.
		go eng.Stop()
	})

	t.Run("requires event id", func(t *testing.T) {
		_, err := eng.ExecuteManual(context.Background(), "demo", types.DataEvent{})
		require.ErrorIs(t, err, types.ErrInvalidArgument)
	})

	t.Run("returns run id and rejects dedup", func(t *testing.T) {
		event := types.DataEvent{EventID: "manual-1", EventType: "demo.created"}
		runID, err := eng.ExecuteManual(context.Background(), "demo", event)
		require.NoError(t, err)
		require.NotZero(t, runID)

		runID2, err := eng.ExecuteManual(context.Background(), "demo", event)
		require.ErrorIs(t, err, types.ErrConflict)
		assert.Zero(t, runID2)
	})

	t.Run("not found parent", func(t *testing.T) {
		_, err := eng.ExecuteManual(context.Background(), "missing", types.DataEvent{EventID: "x"})
		require.ErrorIs(t, err, types.ErrNotFound)
	})

	t.Run("refuses paused definition", func(t *testing.T) {
		paused := ExpandDefinitions([]EditorDefinition{{
			Name: "paused-manual", Enabled: false,
			Triggers: []TriggerEntry{{Type: "event", Enabled: true, Event: "demo.created"}},
		}})
		require.Len(t, paused, 1)
		pausedEng := NewEngine(paused, newMockPipelineStore(), nil, nil, nil)
		t.Cleanup(func() { go pausedEng.Stop() })
		_, err := pausedEng.ExecuteManual(context.Background(), "paused-manual", types.DataEvent{EventID: "x"})
		require.ErrorIs(t, err, types.ErrInvalidArgument)
	})
}
