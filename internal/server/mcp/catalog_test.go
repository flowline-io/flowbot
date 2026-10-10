package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestBuildCatalogDenyAndInclude(t *testing.T) {
	t.Parallel()
	descs := []hub.Descriptor{
		{
			Type: hub.CapCore,
			Operations: []hub.Operation{
				{Name: "notify_send", Description: "send"},
				{Name: "run_terminal", Description: "shell"},
			},
		},
		{
			Type: hub.CapKarakeep,
			Operations: []hub.Operation{
				{Name: "create", Description: "create bookmark"},
				{Name: "list", Description: "list"},
			},
		},
	}
	all := BuildCatalog(descs, nil, nil)
	names := specNames(all)
	assert.Contains(t, names, "core.notify_send")
	assert.NotContains(t, names, "core.run_terminal")
	assert.Contains(t, names, "karakeep.create")
	assert.Contains(t, names, "pipeline.list")
	assert.Contains(t, names, "pipeline.run")
	assert.NotContains(t, names, "pipeline.apply")
	assert.Contains(t, names, "hub.apps")

	onlyKeep := BuildCatalog(descs, []string{"karakeep"}, nil)
	keepNames := specNames(onlyKeep)
	assert.Contains(t, keepNames, "karakeep.create")
	assert.NotContains(t, keepNames, "core.notify_send")
	assert.NotContains(t, keepNames, "pipeline.list")

	noCreate := BuildCatalog(descs, []string{"karakeep"}, []string{"karakeep.create"})
	assert.NotContains(t, specNames(noCreate), "karakeep.create")
	assert.Contains(t, specNames(noCreate), "karakeep.list")
}

func TestFilterByScopes(t *testing.T) {
	t.Parallel()
	descs := []hub.Descriptor{{
		Type:       hub.CapKarakeep,
		Operations: []hub.Operation{{Name: "list"}, {Name: "create"}},
	}}
	specs := BuildCatalog(descs, []string{"karakeep"}, nil)
	read := FilterByScopes(specs, []string{auth.ScopeServiceKarakeepRead})
	require.Len(t, read, 1)
	assert.Equal(t, "karakeep.list", read[0].Name)

	write := FilterByScopes(specs, []string{auth.ScopeServiceKarakeepWrite})
	assert.Len(t, write, 2)

	admin := FilterByScopes(specs, []string{auth.ScopeAdmin})
	assert.Len(t, admin, 2)
}

func specNames(specs []ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}
