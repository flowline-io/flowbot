package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerInstructionsFits(t *testing.T) {
	t.Parallel()
	assert.LessOrEqual(t, len(ServerInstructions), 512)
	assert.Contains(t, ServerInstructions, "structuredContent.error")
	assert.Contains(t, ServerInstructions, "trace.request_id")
	assert.NotContains(t, ServerInstructions, "job_id")
}

func TestOutputSchemaContract(t *testing.T) {
	t.Parallel()
	trace := map[string]any{"request_id": "req-test"}
	errObj := ErrorEnvelope(map[string]any{
		"code": "internal", "message": "internal error", "retryable": false, "hint": "",
	}, trace)

	cases := []struct {
		group string
		op    string
		ok    any
	}{
		{group: "pipeline", op: "list", ok: map[string]any{"items": []any{}, "trace": trace}},
		{group: "workflow", op: "list", ok: map[string]any{"items": []any{}, "trace": trace}},
		{group: "function", op: "list", ok: map[string]any{"items": []any{}, "trace": trace}},
		{group: "hub", op: "apps", ok: map[string]any{"items": []any{}, "trace": trace}},
		{group: "pipeline", op: "get", ok: map[string]any{"name": "daily", "trace": trace}},
		{group: "pipeline", op: "run", ok: map[string]any{"run_id": 7, "trace": trace}},
		{group: "workflow", op: "get_run", ok: map[string]any{"runs": []any{}, "trace": trace}},
		{group: "function", op: "run", ok: map[string]any{"ok": true, "trace": trace}},
		{group: "hub", op: "health", ok: map[string]any{"status": "healthy", "trace": trace}},
		{group: "karakeep", op: "list", ok: map[string]any{
			"capability": "karakeep",
			"operation":  "list",
			"data":       []any{},
			"page":       map[string]any{"limit": 20, "has_more": false, "next_cursor": "abc"},
			"trace":      trace,
		}},
		{group: "devops", op: "beszel_list_systems", ok: map[string]any{"trace": trace, "data": []any{}}},
	}
	for _, tt := range cases {
		t.Run(tt.group+"."+tt.op, func(t *testing.T) {
			t.Parallel()
			schema := OutputSchema(tt.group, tt.op)
			require.NotNil(t, schema)
			require.NoError(t, ValidateOutput(schema, tt.ok))
			require.NoError(t, ValidateOutput(schema, errObj))
			require.Error(t, ValidateOutput(schema, map[string]any{"items": []any{}}))
		})
	}

	assert.Nil(t, OutputSchema("karakeep", "create"))
	assert.Nil(t, OutputSchema("pipeline", "apply"))
	assert.Nil(t, OutputSchema("hub", "restart"))

	fnRun := OutputSchema("function", "run")
	require.NotNil(t, fnRun)
	require.NoError(t, ValidateOutput(fnRun, map[string]any{
		"name": "f", "error": "boom", "trace": trace,
	}))
	require.NoError(t, ValidateOutput(fnRun, errObj))
	require.Error(t, ValidateOutput(fnRun, map[string]any{"name": "f"}))

	confirm := ErrorEnvelope(map[string]any{
		"code": "internal", "message": "internal error", "retryable": false, "hint": "",
		"confirmation_required": map[string]any{"token": "abc"},
	}, trace)
	require.NoError(t, ValidateOutput(OutputSchema("pipeline", "list"), confirm))
}

func TestAttachTraceShapes(t *testing.T) {
	t.Parallel()
	trace := map[string]any{"request_id": "req-test"}
	items, err := AttachTrace([]string{"a"}, trace, true)
	require.NoError(t, err)
	got, ok := items.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"a"}, got["items"])
	assert.Equal(t, trace, got["trace"])

	_, err = AttachTrace([]string{"a"}, trace, false)
	require.Error(t, err)
	_, err = AttachTrace("plain", trace, false)
	require.Error(t, err)

	obj, err := AttachTrace(map[string]any{"name": "daily"}, trace, false)
	require.NoError(t, err)
	body, ok := obj.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "daily", body["name"])
	assert.Equal(t, trace, body["trace"])

	empty, err := AttachTrace(nil, trace, false)
	require.NoError(t, err)
	assert.Equal(t, trace, empty.(map[string]any)["trace"])
}

func TestOpenWorldAndTitle(t *testing.T) {
	t.Parallel()
	for _, group := range []string{"hub", "pipeline", "workflow", "example"} {
		assert.False(t, OpenWorld(group), group)
	}
	assert.True(t, OpenWorld("karakeep"))
	assert.True(t, OpenWorld("function"))
	assert.True(t, OpenWorld("core"))
	assert.Equal(t, "miniflux: Mark a message as read", Title("miniflux", "Mark a message as read"))
	assert.Positive(t, strings.Count(Title("pipeline", "List published pipelines"), ":"))
}
