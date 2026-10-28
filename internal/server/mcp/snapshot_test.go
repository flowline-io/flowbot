package mcp

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

const toolsSnapshotPath = "testdata/tools.snapshot.json"

var updateToolsSnapshot = flag.Bool("update", false, "refresh "+toolsSnapshotPath)

type toolsSnapshotEntry struct {
	Name           string                   `json:"name"`
	Tier           string                   `json:"tier"`
	IdempotentHint bool                     `json:"idempotentHint"`
	Annotations    toolsSnapshotAnnotations `json:"annotations"`
	InputSchema    toolsSnapshotInput       `json:"inputSchema"`
}

type toolsSnapshotAnnotations struct {
	ReadOnlyHint    bool  `json:"readOnlyHint"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  bool  `json:"idempotentHint"`
	OpenWorldHint   bool  `json:"openWorldHint"`
}

type toolsSnapshotInput struct {
	Required []string `json:"required"`
}

// idempotentWriteTools is the explicit allowlist for write and destructive
// tools that may advertise idempotentHint=true. Everything else must default
// to false.
var idempotentWriteTools = map[string]struct{}{
	"confluence.update_page":        {},
	"core.kv_set":                   {},
	"email.mark_read":               {},
	"email.mark_unread":             {},
	"fireflyiii.update_transaction": {},
	"kanboard.update_task":          {},
	"karakeep.attach_tags":          {},
	"karakeep.detach_tags":          {},
	"memos.update":                  {},
	"miniflux.mark_entry_read":      {},
	"miniflux.mark_entry_unread":    {},
	"miniflux.star_entry":           {},
	"miniflux.unstar_entry":         {},
	"nocodb.update_record":          {},
	"trello.update_card":            {},
	"trilium.set_content":           {},
	"trilium.update":                {},
}

func TestToolSnapshot(t *testing.T) {
	got := advertisedToolSnapshots()
	if *updateToolsSnapshot {
		require.NoError(t, writeToolsSnapshot(got))
		return
	}

	raw, err := os.ReadFile(toolsSnapshotPath)
	require.NoError(t, err, "run go test ./internal/server/mcp -run TestToolSnapshot -update")
	var want []toolsSnapshotEntry
	require.NoError(t, sonic.Unmarshal(raw, &want))
	require.Equal(t, want, got)

	names := make([]string, 0, len(want))
	byName := make(map[string]toolsSnapshotEntry, len(want))
	for _, tool := range want {
		names = append(names, tool.Name)
		byName[tool.Name] = tool
	}
	assert.True(t, slices.IsSorted(names), "snapshot must be sorted by tool name")
	deploy, ok := byName["drone.deploy"]
	require.True(t, ok)
	assert.Equal(t, "write", deploy.Tier)
	assert.False(t, deploy.IdempotentHint)
	assert.False(t, deploy.Annotations.ReadOnlyHint)
	require.NotNil(t, deploy.Annotations.DestructiveHint)
	assert.False(t, *deploy.Annotations.DestructiveHint)
	assert.False(t, deploy.Annotations.IdempotentHint)
	assert.True(t, deploy.Annotations.OpenWorldHint)

	health, ok := byName["drone.health"]
	require.True(t, ok)
	assert.Equal(t, "read", health.Tier)
	assert.True(t, health.Annotations.ReadOnlyHint)
	assert.Nil(t, health.Annotations.DestructiveHint)
	assert.True(t, health.Annotations.OpenWorldHint)

	assert.NotContains(t, string(raw), "server-card")
	assert.NotContains(t, string(raw), "serverCard")
}

func TestWriteToolIdempotencyContract(t *testing.T) {
	tools := advertisedToolSnapshots()
	seen := make(map[string]bool, len(tools))
	for _, tool := range tools {
		seen[tool.Name] = true
		if tool.Tier == "read" {
			continue
		}
		_, allowed := idempotentWriteTools[tool.Name]
		if tool.IdempotentHint {
			assert.True(t, allowed, "%s is tier %s but idempotentHint=true; add it to idempotentWriteTools", tool.Name, tool.Tier)
			continue
		}
		assert.False(t, allowed, "%s is in idempotentWriteTools but idempotentHint=false; remove the stale entry", tool.Name)
	}
	for name := range idempotentWriteTools {
		assert.True(t, seen[name], "%s is in idempotentWriteTools but is not in the snapshot", name)
	}
}

func advertisedToolSnapshots() []toolsSnapshotEntry {
	specs := advertisedCatalogSpecs()
	declaredMutations := make(map[string]bool)
	descs := make([]hub.Descriptor, 0, len(specs))
	for _, spec := range specs {
		descs = append(descs, capability.DescriptorFromSpec(spec))
		for _, op := range spec.Ops {
			if op.Mutation {
				declaredMutations[string(spec.Type)+"."+op.Name] = true
			}
		}
	}

	tools := BuildCatalog(descs, nil, nil)
	out := make([]toolsSnapshotEntry, 0, len(tools))
	for _, tool := range tools {
		// CatalogSpec is the test baseline; Register normally fills the
		// mutation table as a side effect, which a test process never runs.
		if declaredMutations[tool.Name] {
			tool.Mutation = true
		}
		tier, _ := toolTier(tool)
		ann := toolAnnotations(tool)
		out = append(out, toolsSnapshotEntry{
			Name:           tool.Name,
			Tier:           tierName(tier),
			IdempotentHint: ann.IdempotentHint,
			Annotations: toolsSnapshotAnnotations{
				ReadOnlyHint:    ann.ReadOnlyHint,
				DestructiveHint: ann.DestructiveHint,
				IdempotentHint:  ann.IdempotentHint,
				OpenWorldHint:   ann.OpenWorldHint != nil && *ann.OpenWorldHint,
			},
			InputSchema: toolsSnapshotInput{Required: requiredParams(toolInputSchema(tool))},
		})
	}
	slices.SortFunc(out, func(a, b toolsSnapshotEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

func tierName(tier capability.Tier) string {
	switch tier {
	case capability.TierWrite:
		return "write"
	case capability.TierDestructive:
		return "destructive"
	default:
		return "read"
	}
}

func requiredParams(schema map[string]any) []string {
	out := make([]string, 0)
	switch required := schema["required"].(type) {
	case []string:
		out = append(out, required...)
	case []any:
		for _, item := range required {
			if name, ok := item.(string); ok {
				out = append(out, name)
			}
		}
	}
	return out
}

func writeToolsSnapshot(tools []toolsSnapshotEntry) error {
	if err := os.MkdirAll(filepath.Dir(toolsSnapshotPath), 0o750); err != nil {
		return err
	}
	raw, err := sonic.MarshalIndent(tools, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(toolsSnapshotPath, raw, 0o644)
}
