package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	appcfg "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type analyticsTestDB struct {
	client *gen.Client
}

func (*analyticsTestDB) Open(appcfg.StoreType) error { return nil }
func (*analyticsTestDB) Close() error               { return nil }
func (*analyticsTestDB) IsOpen() bool               { return true }
func (*analyticsTestDB) GetName() string            { return "analytics-test" }
func (*analyticsTestDB) Stats() any                 { return nil }
func (*analyticsTestDB) Ping(context.Context) (time.Duration, error) {
	return 0, nil
}
func (d *analyticsTestDB) GetDB() any             { return d.client }
func (d *analyticsTestDB) GetClient() *gen.Client { return d.client }

func TestService_FlowGraph_WindowCap(t *testing.T) {
	client := sqlitetest.OpenClient(t, t.Name())
	prev := store.Database
	store.Database = &analyticsTestDB{client: client}
	t.Cleanup(func() { store.Database = prev })

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	_, err := client.ResourceLink.Create().
		SetSourceEventID("fg-src").
		SetTargetEventID("fg-tgt").
		SetSourceApp("miniflux").
		SetTargetApp("karakeep").
		SetPipelineName("reader_star").
		SetRunKind(resourcelink.RunKindPipeline).
		SetStatus(resourcelink.StatusOk).
		SetCreatedAt(now.Add(-2 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	svc := NewService()
	graph, err := svc.FlowGraph(ctx, types.FlowGraphQuery{
		DataEventsRetentionDays: 30,
	})
	require.NoError(t, err)
	require.Len(t, graph.Edges, 1)
	assert.Equal(t, "miniflux", graph.Edges[0].Source)
	assert.Equal(t, "karakeep", graph.Edges[0].Target)
	assert.Contains(t, []string{graph.Nodes[0].ID, graph.Nodes[1].ID}, "miniflux")
	span := graph.Window.Until.Sub(graph.Window.Since)
	assert.LessOrEqual(t, span, 30*24*time.Hour)
	assert.GreaterOrEqual(t, span, 7*24*time.Hour-time.Second)

	ui, err := svc.FlowGraphUI(ctx, types.FlowGraphQuery{DataEventsRetentionDays: 30})
	require.NoError(t, err)
	require.Len(t, ui.Edges, 1)
	assert.LessOrEqual(t, len(ui.Nodes), 20)
}

func TestService_Trace_RequiresOneSeed(t *testing.T) {
	t.Parallel()
	svc := NewService()
	_, err := svc.Trace(context.Background(), types.LineageTraceQuery{})
	require.Error(t, err)
	_, err = svc.Trace(context.Background(), types.LineageTraceQuery{
		EventID:       "e1",
		CorrelationID: "c1",
	})
	require.Error(t, err)
}

func TestService_Search_RequiresQuery(t *testing.T) {
	t.Parallel()
	svc := NewService()
	_, err := svc.Search(context.Background(), types.LineageSearchQuery{})
	require.Error(t, err)
}

func TestService_Timeline_PayloadRules(t *testing.T) {
	client := sqlitetest.OpenClient(t, t.Name())
	prev := store.Database
	store.Database = &analyticsTestDB{client: client}
	t.Cleanup(func() { store.Database = prev })

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	_, err := client.DataEvent.Create().
		SetEventID("ev-timeline-1").
		SetEventType("karakeep.bookmark.created").
		SetApp("karakeep").
		SetCapability("karakeep").
		SetUID("user-a").
		SetData(map[string]any{"secret": "x"}).
		SetCreatedAt(now.Add(-time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	svc := NewService()
	denied, err := svc.Timeline(ctx, types.EventsTimelineQuery{
		App:            "karakeep",
		IncludePayload: true,
		CallerUID:      "user-a",
		Scopes:         []string{"lineage:read"},
	})
	require.Error(t, err)
	assert.Nil(t, denied)

	out, err := svc.Timeline(ctx, types.EventsTimelineQuery{
		IncludePayload: true,
		CallerUID:      "user-a",
		Scopes:         []string{"lineage:read"},
	})
	require.NoError(t, err)
	require.Len(t, out.Items, 1)
	assert.True(t, out.Items[0].PayloadOmitted)
	assert.Nil(t, out.Items[0].Data)

	defaultOmit, err := svc.Timeline(ctx, types.EventsTimelineQuery{
		CallerUID: "user-a",
		Scopes:    []string{"lineage:read", "service:karakeep:read"},
	})
	require.NoError(t, err)
	require.Len(t, defaultOmit.Items, 1)
	assert.True(t, defaultOmit.Items[0].PayloadOmitted)
	assert.Nil(t, defaultOmit.Items[0].Data)

	withRead, err := svc.Timeline(ctx, types.EventsTimelineQuery{
		IncludePayload: true,
		CallerUID:      "user-a",
		Scopes:         []string{"lineage:read", "service:karakeep:read"},
	})
	require.NoError(t, err)
	require.Len(t, withRead.Items, 1)
	assert.False(t, withRead.Items[0].PayloadOmitted)
	assert.Equal(t, "x", withRead.Items[0].Data["secret"])
}

func TestService_Trace_RequiresOneSeed_Typed(t *testing.T) {
	t.Parallel()
	svc := NewService()
	_, err := svc.Trace(context.Background(), types.LineageTraceQuery{})
	require.Error(t, err)
	var te *types.Error
	require.ErrorAs(t, err, &te)
	require.ErrorIs(t, te, types.ErrInvalidArgument)
	assert.Equal(t, "seed", te.Field)
}
