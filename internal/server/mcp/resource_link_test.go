package mcp

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
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type mcpTestDB struct {
	client *gen.Client
}

func (*mcpTestDB) Open(config.StoreType) error { return nil }
func (*mcpTestDB) Close() error               { return nil }
func (*mcpTestDB) IsOpen() bool               { return true }
func (*mcpTestDB) GetName() string            { return "mcp-test" }
func (*mcpTestDB) Stats() any                 { return nil }
func (*mcpTestDB) Ping(context.Context) (time.Duration, error) {
	return 0, nil
}
func (d *mcpTestDB) GetDB() any             { return d.client }
func (d *mcpTestDB) GetClient() *gen.Client { return d.client }

func TestMCPExternalInputLink_Fields(t *testing.T) {
	t.Parallel()
	link := mcpExternalInputLink(
		Identity{TokenID: 42},
		"req-abc",
		"cor_01TEST",
		ToolSpec{Name: "kanboard.create_task", Group: "kanboard"},
		&capability.ResourceMeta{EventID: "evt-tgt", EntityID: "99", App: "kanboard-app"},
	)
	require.NotNil(t, link)
	assert.Equal(t, "mcp:req-abc", link.SourceEventID)
	assert.Equal(t, "agent:42", link.SourceApp)
	assert.Equal(t, "req-abc", link.SourceEntityID)
	assert.Equal(t, "kanboard.create_task", link.TargetCapability)
	assert.Equal(t, resourcelink.RunKindMcp, link.RunKind)
	assert.Equal(t, resourcelink.StatusOk, link.Status)
	assert.Equal(t, "cor_01TEST", link.CorrelationID)
}

func TestRecordMCPExternalInputLink_Queryable(t *testing.T) {
	client := sqlitetest.OpenClient(t, t.Name())
	prev := store.Database
	store.Database = &mcpTestDB{client: client}
	t.Cleanup(func() { store.Database = prev })

	ctx := withRequestID(context.Background(), "req-mcp-1")
	ctx = types.WithLineage(ctx, "cor_mcp_write", "")
	ident := Identity{TokenID: 7, UID: types.Uid("user-1")}
	spec := ToolSpec{Name: "example.create", Group: "example", Kind: kindCapability, Mutation: true}
	result := &capability.InvokeResult{
		Resource: &capability.ResourceMeta{
			EventID:  "evt-created-1",
			EntityID: "ent-1",
			App:      "example-app",
		},
	}

	recordMCPExternalInputLink(ctx, ident, spec, result)

	links, err := store.NewResourceChainStore(client).FindResourceLinks(ctx, []string{
		types.ResourceLinkMCPSourceEventID("req-mcp-1"),
	})
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, "agent:7", links[0].SourceApp)
	assert.Equal(t, "evt-created-1", links[0].TargetEventID)
	assert.Equal(t, resourcelink.RunKindMcp, links[0].RunKind)
	assert.Equal(t, "cor_mcp_write", links[0].CorrelationID)
}
