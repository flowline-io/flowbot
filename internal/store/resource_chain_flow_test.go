package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestAggregateFlowGraph_SeededEdges(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "e1", tgtEvt: "e2", srcApp: "miniflux", tgtApp: "karakeep",
		via: "reader_star", status: resourcelink.StatusOk, at: now.Add(-2 * time.Hour),
	})
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "e3", tgtEvt: "e4", srcApp: "miniflux", tgtApp: "karakeep",
		via: "reader_star", status: resourcelink.StatusOk, at: now.Add(-time.Hour),
	})
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "e5", tgtEvt: "e6", srcApp: "miniflux", tgtApp: "karakeep",
		via: "reader_star", status: resourcelink.StatusFailed, at: now.Add(-30 * time.Minute),
	})
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "e7", tgtEvt: "e8", srcApp: "agent:7", tgtApp: "kanboard",
		via: "kanboard.create_task", kind: resourcelink.RunKindMcp, status: resourcelink.StatusOk, at: now.Add(-10 * time.Minute),
	})

	rc := NewResourceChainStore(client)
	edges, err := rc.AggregateFlowGraph(ctx, now.Add(-24*time.Hour), now, "")
	require.NoError(t, err)
	require.Len(t, edges, 2)

	var star, mcp *types.FlowGraphEdge
	for i := range edges {
		e := &edges[i]
		if e.Source == "miniflux" && e.Target == "karakeep" {
			star = e
		}
		if e.Source == "agent:7" {
			mcp = e
		}
	}
	require.NotNil(t, star)
	assert.Equal(t, int64(2), star.CountOK)
	assert.Equal(t, int64(1), star.CountFailed)
	assert.Equal(t, "pipeline", star.Via.Kind)
	assert.Equal(t, "reader_star", star.Via.Name)
	assert.False(t, star.Static)
	require.NotNil(t, star.LastAt)
	assert.True(t, star.LastAt.Equal(now.Add(-30*time.Minute)) || star.LastAt.Equal(now.Add(-30*time.Minute).UTC()))

	require.NotNil(t, mcp)
	assert.Equal(t, int64(1), mcp.CountOK)
	assert.Equal(t, "mcp", mcp.Via.Kind)

	filtered, err := rc.AggregateFlowGraph(ctx, now.Add(-24*time.Hour), now, "kanboard")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, "agent:7", filtered[0].Source)
}

func TestTrace_FiveHopChainAndDepthTruncation(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ctx := context.Background()
	now := time.Now().UTC()

	// Chain: e0 -> e1 -> e2 -> e3 -> e4 -> e5 (5 hops)
	for i := 0; i < 5; i++ {
		seedLink(ctx, t, client, linkSeed{
			srcEvt: fmt.Sprintf("e%d", i),
			tgtEvt: fmt.Sprintf("e%d", i+1),
			srcApp: "app",
			tgtApp: "app",
			srcEnt: fmt.Sprintf("n%d", i),
			tgtEnt: fmt.Sprintf("n%d", i+1),
			via:    "chain",
			at:     now.Add(time.Duration(i) * time.Minute),
		})
	}

	rc := NewResourceChainStore(client)
	full, err := rc.Trace(ctx, types.LineageTraceQuery{
		EventID:   "e5",
		Direction: types.LineageTraceDirectionUp,
		Depth:     10,
	})
	require.NoError(t, err)
	assert.False(t, full.Truncated)
	assert.Len(t, full.Edges, 5)
	assert.GreaterOrEqual(t, len(full.Nodes), 6)

	limited, err := rc.Trace(ctx, types.LineageTraceQuery{
		EventID:   "e5",
		Direction: types.LineageTraceDirectionUp,
		Depth:     2,
	})
	require.NoError(t, err)
	assert.True(t, limited.Truncated)
	assert.LessOrEqual(t, len(limited.Edges), 2)
}

func TestTrace_EntitySeedAndFailLeaf(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ctx := context.Background()
	now := time.Now().UTC()
	failID := types.ResourceLinkFailTargetEventID(9, "step-a")
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "src-ok", tgtEvt: failID, srcApp: "miniflux", tgtApp: "karakeep",
		srcEnt: "42", tgtEnt: "", via: "pipe", status: resourcelink.StatusFailed, at: now,
	})

	rc := NewResourceChainStore(client)
	out, err := rc.Trace(ctx, types.LineageTraceQuery{
		App:       "miniflux",
		EntityID:  "42",
		Direction: types.LineageTraceDirectionDown,
	})
	require.NoError(t, err)
	require.Len(t, out.Edges, 1)
	assert.Equal(t, failID, out.Edges[0].TargetEventID)
	assert.Equal(t, string(types.ResourceLinkStatusFailed), out.Edges[0].Status)
}

func TestTrace_CorrelationSeed(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ctx := context.Background()
	now := time.Now().UTC()
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "c1", tgtEvt: "c2", srcApp: "a", tgtApp: "b", via: "p",
		corr: "cor_shared", at: now,
	})
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "c2", tgtEvt: "c3", srcApp: "b", tgtApp: "c", via: "p",
		corr: "cor_shared", at: now.Add(time.Minute),
	})
	seedLink(ctx, t, client, linkSeed{
		srcEvt: "other1", tgtEvt: "other2", srcApp: "x", tgtApp: "y", via: "p",
		corr: "cor_other", at: now,
	})

	rc := NewResourceChainStore(client)
	out, err := rc.Trace(ctx, types.LineageTraceQuery{
		CorrelationID: "cor_shared",
		Direction:     types.LineageTraceDirectionBoth,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(out.Edges), 2)
	for _, e := range out.Edges {
		assert.Equal(t, "cor_shared", e.CorrelationID)
	}
}

type linkSeed struct {
	srcEvt, tgtEvt, srcApp, tgtApp, srcEnt, tgtEnt, via, corr string
	kind                                                      resourcelink.RunKind
	status                                                    resourcelink.Status
	at                                                        time.Time
}

func seedLink(ctx context.Context, t *testing.T, client *gen.Client, s linkSeed) {
	t.Helper()
	if s.kind == "" {
		s.kind = resourcelink.RunKindPipeline
	}
	if s.status == "" {
		s.status = resourcelink.StatusOk
	}
	b := client.ResourceLink.Create().
		SetSourceEventID(s.srcEvt).
		SetTargetEventID(s.tgtEvt).
		SetSourceApp(s.srcApp).
		SetTargetApp(s.tgtApp).
		SetSourceEntityID(s.srcEnt).
		SetTargetEntityID(s.tgtEnt).
		SetPipelineName(s.via).
		SetRunKind(s.kind).
		SetStatus(s.status).
		SetCorrelationID(s.corr).
		SetCreatedAt(s.at)
	_, err := b.Save(ctx)
	require.NoError(t, err)
}
