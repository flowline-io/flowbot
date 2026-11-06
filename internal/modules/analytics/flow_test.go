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
	t.Parallel()
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
