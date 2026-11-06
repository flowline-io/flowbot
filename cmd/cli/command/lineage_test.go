package command

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineageCommandHelpAndLeaves(t *testing.T) {
	t.Parallel()
	cmd := LineageCommand()
	require.Equal(t, "lineage", cmd.Use)
	require.Contains(t, subcommandNames(cmd), "trace")
	require.Contains(t, subcommandNames(cmd), "search")
}

func TestFlowGraphCLISuccess(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/service/flows/graph", r.URL.Path)
		assert.Equal(t, "miniflux", r.URL.Query().Get("app"))
		_, _ = w.Write([]byte(okJSON(`{
			"nodes":[{"id":"miniflux"},{"id":"karakeep"}],
			"edges":[{"source":"miniflux","target":"karakeep","via":{"kind":"pipeline","name":"reader_star"},"count_ok":1,"count_failed":0,"static":false}],
			"window":{"since":"2027-01-01T00:00:00Z","until":"2027-01-08T00:00:00Z"}
		}`)))
	})

	root := NewRoot("test")
	out := runCommand(t, root, "flow", "graph", "--app", "miniflux", "-o", "json")
	assert.Contains(t, out, `"data"`)
	assert.Contains(t, out, "miniflux")
	assert.Contains(t, out, "karakeep")
	assert.Contains(t, out, "reader_star")
}

func TestLineageSearchCLISuccess(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/service/lineage/search", r.URL.Path)
		assert.Equal(t, "bm:991", r.URL.Query().Get("query"))
		_, _ = w.Write([]byte(okJSON(`{"items":[{"app":"karakeep","entity_id":"bm:991"}],"page":{}}`)))
	})

	root := NewRoot("test")
	out := runCommand(t, root, "lineage", "search", "--query", "bm:991", "-o", "json")
	assert.Contains(t, out, `"data"`)
	assert.Contains(t, out, "bm:991")
	assert.Contains(t, out, "karakeep")
}
