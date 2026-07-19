package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEventStore_NilSafe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, s *EventStore)
	}{
		{
			name: "AppendEventOutbox nil store",
			run: func(t *testing.T, s *EventStore) {
				t.Helper()
				assert.NoError(t, s.AppendEventOutbox(context.Background(), types.DataEvent{EventID: "e1"}))
			},
		},
		{
			name: "MarkOutboxPublished nil store",
			run: func(t *testing.T, s *EventStore) {
				t.Helper()
				assert.NoError(t, s.MarkOutboxPublished(context.Background(), "e1"))
			},
		},
		{
			name: "GetDataEventByEventID nil store",
			run: func(t *testing.T, s *EventStore) {
				t.Helper()
				ev, err := s.GetDataEventByEventID(context.Background(), "e1")
				require.NoError(t, err)
				assert.Nil(t, ev)
			},
		},
		{
			name: "GetPipelineRunsForEvents nil store",
			run: func(t *testing.T, s *EventStore) {
				t.Helper()
				runs, err := s.GetPipelineRunsForEvents(context.Background(), []string{"e1"})
				require.NoError(t, err)
				assert.Nil(t, runs)
			},
		},
		{
			name: "DeleteDataEventsOlderThan nil store",
			run: func(t *testing.T, s *EventStore) {
				t.Helper()
				n, err := s.DeleteDataEventsOlderThan(context.Background(), time.Now())
				require.NoError(t, err)
				assert.Equal(t, 0, n)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, nil)
			tt.run(t, &EventStore{})
		})
	}
}

func TestEventStore_OutboxLifecycle(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewEventStore(client)
	ctx := context.Background()

	event := types.DataEvent{
		EventID:   "outbox-evt-1",
		EventType: "bookmark.created",
		Source:    "karakeep",
		Tags:      map[string]any{"project": "alpha"},
	}
	require.NoError(t, store.AppendEventOutbox(ctx, event))

	rows, err := client.EventOutbox.Query().
		Where(eventoutbox.EventID("outbox-evt-1")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Published)

	require.NoError(t, store.MarkOutboxPublished(ctx, "outbox-evt-1"))
	updated, err := client.EventOutbox.Get(ctx, rows[0].ID)
	require.NoError(t, err)
	assert.True(t, updated.Published)
}

func TestEventStore_GetDataEventByEventID(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewEventStore(client)
	ctx := context.Background()

	require.NoError(t, store.AppendDataEvent(ctx, types.DataEvent{
		EventID: "lookup-1", EventType: "issue.created", Source: "github",
	}))

	tests := []struct {
		name    string
		eventID string
		wantID  string
	}{
		{name: "existing event", eventID: "lookup-1", wantID: "lookup-1"},
		{name: "missing event", eventID: "missing", wantID: ""},
		{name: "empty id", eventID: "", wantID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := store.GetDataEventByEventID(ctx, tt.eventID)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, ev)
				return
			}
			require.NotNil(t, ev)
			assert.Equal(t, tt.wantID, ev.EventID)
		})
	}
}

func TestEventStore_GetPipelineRunsForEvents(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	eventStore := NewEventStore(client)
	pipeStore := NewPipelineStore(client)
	ctx := context.Background()

	require.NoError(t, eventStore.AppendDataEvent(ctx, types.DataEvent{
		EventID: "evt-run-1", EventType: "issue.created", Source: "github",
	}))
	run, err := pipeStore.CreateRun(ctx, "sync-pipeline", "evt-run-1", "issue.created", "event")
	require.NoError(t, err)
	require.NotNil(t, run)

	tests := []struct {
		name     string
		eventIDs []string
		wantLen  int
	}{
		{name: "matches pipeline run", eventIDs: []string{"evt-run-1"}, wantLen: 1},
		{name: "empty ids returns nil", eventIDs: nil, wantLen: 0},
		{name: "unknown event returns empty map entry", eventIDs: []string{"missing"}, wantLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eventStore.GetPipelineRunsForEvents(ctx, tt.eventIDs)
			require.NoError(t, err)
			if tt.eventIDs == nil {
				assert.Nil(t, result)
				return
			}
			total := 0
			for _, infos := range result {
				total += len(infos)
			}
			assert.Equal(t, tt.wantLen, total)
		})
	}
}

func TestEventStore_ListDistinctSourcesAndTypes(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewEventStore(client)
	ctx := context.Background()

	events := []types.DataEvent{
		{EventID: "d-1", EventType: "bookmark.created", Source: "karakeep"},
		{EventID: "d-2", EventType: "entry.new", Source: "miniflux"},
		{EventID: "d-3", EventType: "bookmark.created", Source: "karakeep"},
	}
	for _, e := range events {
		require.NoError(t, store.AppendDataEvent(ctx, e))
	}

	sources, err := store.ListDistinctEventSources(ctx, 24*time.Hour)
	require.NoError(t, err)
	assert.Contains(t, sources, "karakeep")
	assert.Contains(t, sources, "miniflux")

	typesList, err := store.ListDistinctEventTypes(ctx, 24*time.Hour)
	require.NoError(t, err)
	assert.Contains(t, typesList, "bookmark.created")
	assert.Contains(t, typesList, "entry.new")
}

func TestPipelineStore_NilSafe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, s *PipelineStore)
	}{
		{
			name: "SaveCheckpoint nil store",
			run: func(t *testing.T, s *PipelineStore) {
				t.Helper()
				assert.NoError(t, s.SaveCheckpoint(context.Background(), 1, map[string]any{"step": 1}))
			},
		},
		{
			name: "GetCheckpoint nil store",
			run: func(t *testing.T, s *PipelineStore) {
				t.Helper()
				var target map[string]any
				assert.NoError(t, s.GetCheckpoint(context.Background(), 1, &target))
			},
		},
		{
			name: "HasConsumed nil store",
			run: func(t *testing.T, s *PipelineStore) {
				t.Helper()
				ok, err := s.HasConsumed(context.Background(), "consumer", "evt")
				require.NoError(t, err)
				assert.False(t, ok)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, nil)
			tt.run(t, &PipelineStore{})
		})
	}
}

func TestPipelineStore_CheckpointAndConsumption(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewPipelineStore(client)
	ctx := context.Background()

	run, err := store.CreateRun(ctx, "checkpoint-pipe", "evt-cp-1", "test.event", "event")
	require.NoError(t, err)

	checkpoint := map[string]any{"step_index": 2, "vars": map[string]any{"k": "v"}}
	require.NoError(t, store.SaveCheckpoint(ctx, run.ID, checkpoint))

	var loaded map[string]any
	require.NoError(t, store.GetCheckpoint(ctx, run.ID, &loaded))
	assert.InDelta(t, 2, loaded["step_index"], 0.001)

	require.NoError(t, store.RecordConsumption(ctx, "pipeline-engine", "evt-cp-1"))
	consumed, err := store.HasConsumed(ctx, "pipeline-engine", "evt-cp-1")
	require.NoError(t, err)
	assert.True(t, consumed)

	missing, err := store.HasConsumed(ctx, "pipeline-engine", "evt-missing")
	require.NoError(t, err)
	assert.False(t, missing)
}

func TestPipelineStore_RecordResourceLink(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewPipelineStore(client)
	ctx := context.Background()

	run, err := store.CreateRun(ctx, "link-pipe", "evt-link-1", "test.event", "event")
	require.NoError(t, err)

	link := &gen.ResourceLink{
		SourceEventID:    "evt-src",
		TargetEventID:    "evt-tgt",
		SourceApp:        "github",
		TargetApp:        "forge",
		SourceCapability: "issue",
		TargetCapability: "issue",
		SourceEntityID:   "42",
		TargetEntityID:   "99",
		PipelineRunID:    run.ID,
		PipelineName:     "link-pipe",
	}
	require.NoError(t, store.RecordResourceLink(ctx, link))

	rc := NewResourceChainStore(client)
	links, err := rc.FindResourceLinks(ctx, []string{"evt-src"})
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, "forge", links[0].TargetApp)
}

func TestResourceChainStore_FindRelations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		seed     func(context.Context, *gen.Client) error
		app      string
		entityID string
		wantUp   int
		wantDown int
	}{
		{
			name: "source node has downstream",
			seed: func(ctx context.Context, client *gen.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-rel-a").
					SetTargetEventID("tgt-rel-a").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			app: "github", entityID: "42", wantDown: 1,
		},
		{
			name: "target node has upstream",
			seed: func(ctx context.Context, client *gen.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-rel-b").
					SetTargetEventID("tgt-rel-b").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			app: "forge", entityID: "99", wantUp: 1,
		},
		{
			name: "unknown node empty relations",
			app:  "github", entityID: "missing", wantUp: 0, wantDown: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, tt.name)
			rc := NewResourceChainStore(client)
			ctx := context.Background()
			if tt.seed != nil {
				require.NoError(t, tt.seed(ctx, client))
			}
			rel, err := rc.FindRelations(ctx, tt.app, tt.entityID)
			require.NoError(t, err)
			require.NotNil(t, rel)
			assert.Len(t, rel.Upstream, tt.wantUp)
			assert.Len(t, rel.Downstream, tt.wantDown)
		})
	}
}

func TestResourceChainStore_FindResourceLinks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		eventIDs []string
		wantLen  int
	}{
		{name: "nil store returns nil", eventIDs: []string{"a"}, wantLen: 0},
		{name: "empty ids returns nil", eventIDs: nil, wantLen: 0},
		{name: "zero-value store returns nil", eventIDs: []string{"x"}, wantLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *ResourceChainStore
			if tt.name == "zero-value store returns nil" {
				s = &ResourceChainStore{}
			}
			links, err := s.FindResourceLinks(context.Background(), tt.eventIDs)
			require.NoError(t, err)
			assert.Nil(t, links)
		})
	}
}

func TestPipelineStore_UpdateRunAndStepStatus(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewPipelineStore(client)
	ctx := context.Background()

	run, err := store.CreateRun(ctx, "status-pipe", "evt-status", "test.event", "event")
	require.NoError(t, err)

	step, err := store.CreateStepRun(ctx, run.ID, "step-a", "example", "echo", map[string]any{"x": 1}, 1)
	require.NoError(t, err)

	require.NoError(t, store.UpdateStepRun(ctx, step.ID, int(schema.PipelineDone), map[string]any{"ok": true}, "", 1))
	require.NoError(t, store.UpdateRunStatus(ctx, run.ID, int(schema.PipelineDone), ""))

	loaded, err := store.GetRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, int(schema.PipelineDone), loaded.Status)
}

func TestPipelineStore_GetIncompleteRuns(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	store := NewPipelineStore(client)
	ctx := context.Background()

	_, err := store.CreateRun(ctx, "incomplete-pipe", "evt-inc", "test.event", "event")
	require.NoError(t, err)

	runs, err := store.GetIncompleteRuns(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, runs)
}

func TestEventStore_DeleteDataEventsOlderThan(t *testing.T) {
	t.Parallel()

	cutoff := time.Now().Add(-24 * time.Hour)
	oldAt := cutoff.Add(-time.Hour)
	freshAt := time.Now()

	tests := []struct {
		name    string
		setup   func(t *testing.T, client *gen.Client, events *EventStore, pipes *PipelineStore)
		wantDel int
		assert  func(t *testing.T, client *gen.Client)
	}{
		{
			name: "cascades related history for old events",
			setup: func(t *testing.T, client *gen.Client, events *EventStore, pipes *PipelineStore) {
				t.Helper()
				ctx := context.Background()
				_, err := client.DataEvent.Create().
					SetEventID("old-evt").
					SetEventType("bookmark.created").
					SetSource("karakeep").
					SetCreatedAt(oldAt).
					Save(ctx)
				require.NoError(t, err)
				require.NoError(t, events.AppendEventOutbox(ctx, types.DataEvent{EventID: "old-evt"}))
				require.NoError(t, pipes.RecordConsumption(ctx, "engine", "old-evt"))
				run, err := pipes.CreateRun(ctx, "pipe", "old-evt", "bookmark.created", "event")
				require.NoError(t, err)
				_, err = pipes.CreateStepRun(ctx, run.ID, "step1", "bookmark", "create", nil, 1)
				require.NoError(t, err)
				require.NoError(t, pipes.RecordResourceLink(ctx, &gen.ResourceLink{
					SourceEventID: "old-evt",
					TargetEventID: "old-evt-out",
					PipelineRunID: run.ID,
					PipelineName:  "pipe",
				}))
				_, err = client.DataEvent.Create().
					SetEventID("fresh-evt").
					SetEventType("bookmark.created").
					SetSource("karakeep").
					SetCreatedAt(freshAt).
					Save(ctx)
				require.NoError(t, err)
			},
			wantDel: 1,
			assert: func(t *testing.T, client *gen.Client) {
				t.Helper()
				ctx := context.Background()
				assert.Equal(t, 0, client.DataEvent.Query().Where(dataevent.EventID("old-evt")).CountX(ctx))
				assert.Equal(t, 1, client.DataEvent.Query().Where(dataevent.EventID("fresh-evt")).CountX(ctx))
				assert.Equal(t, 0, client.PipelineRun.Query().CountX(ctx))
				assert.Equal(t, 0, client.PipelineStepRun.Query().CountX(ctx))
				assert.Equal(t, 0, client.EventOutbox.Query().CountX(ctx))
				assert.Equal(t, 0, client.EventConsumption.Query().CountX(ctx))
				assert.Equal(t, 0, client.ResourceLink.Query().CountX(ctx))
			},
		},
		{
			name: "noop when nothing older than cutoff",
			setup: func(t *testing.T, client *gen.Client, _ *EventStore, _ *PipelineStore) {
				t.Helper()
				_, err := client.DataEvent.Create().
					SetEventID("only-fresh").
					SetEventType("issue.created").
					SetSource("gitea").
					SetCreatedAt(freshAt).
					Save(context.Background())
				require.NoError(t, err)
			},
			wantDel: 0,
			assert: func(t *testing.T, client *gen.Client) {
				t.Helper()
				assert.Equal(t, 1, client.DataEvent.Query().CountX(context.Background()))
			},
		},
		{
			name: "deletes old event without dependents",
			setup: func(t *testing.T, client *gen.Client, _ *EventStore, _ *PipelineStore) {
				t.Helper()
				_, err := client.DataEvent.Create().
					SetEventID("orphan-old").
					SetEventType("bookmark.created").
					SetSource("karakeep").
					SetCreatedAt(oldAt).
					Save(context.Background())
				require.NoError(t, err)
			},
			wantDel: 1,
			assert: func(t *testing.T, client *gen.Client) {
				t.Helper()
				assert.Equal(t, 0, client.DataEvent.Query().CountX(context.Background()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			events := NewEventStore(client)
			pipes := NewPipelineStore(client)
			tt.setup(t, client, events, pipes)

			n, err := events.DeleteDataEventsOlderThan(context.Background(), cutoff)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDel, n)
			tt.assert(t, client)
		})
	}
}
