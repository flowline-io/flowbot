package store

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

func TestAuditStore_ListFiltersAndCursor(t *testing.T) {
	client := getTestClient(t)
	s := NewAuditStore(client)
	ctx := context.Background()
	rows := []struct {
		action string
		uid    string
		result string
		code   string
	}{
		{action: "mcp.tool.call", uid: "user-a", result: "success"},
		{action: "mcp.tool.call", uid: "user-a", result: "failure", code: "permission_denied"},
		{action: "mcp.tool.call", uid: "user-b", result: "success"},
	}
	for _, row := range rows {
		err := s.writeEntry(ctx, audit.Entry{
			Subject:       &audit.Subject{SubjectType: "token", SubjectID: row.uid, UID: row.uid},
			Action:        row.action,
			Target:        audit.Target{Type: "mcp_tool", ID: "karakeep.create"},
			Request:       map[string]any{"tool": "karakeep.create", "token": "secret"},
			CorrelationID: "cor_test",
		}, row.result, row.code)
		require.NoError(t, err)
		n, err := client.AuditLog.Query().Count(ctx)
		require.NoError(t, err)
		require.Positive(t, n)
	}

	got, cursor, err := s.List(ctx, ListAuditOptions{Action: "mcp.tool.call", SubjectID: "user-a", Limit: 1})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "user-a", got[0].SubjectID)
	assert.Equal(t, "failure", got[0].Result)
	assert.Equal(t, "permission_denied", got[0].ErrorCode)
	assert.Equal(t, "cor_test", got[0].CorrelationID)
	req, ok := got[0].Request.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[redacted]", req["token"])
	assert.NotEmpty(t, cursor)

	next, nextCursor, err := s.List(ctx, ListAuditOptions{SubjectID: "user-a", Limit: 1, Cursor: cursor})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, "success", next[0].Result)
	assert.Empty(t, nextCursor)

	failed, _, err := s.List(ctx, ListAuditOptions{SubjectID: "user-a", Result: "failure"})
	require.NoError(t, err)
	require.Len(t, failed, 1)

	plain := types.Errorf(types.ErrInvalidArgument, "name is required")
	require.NoError(t, s.RecordFailure(ctx, audit.Entry{
		Subject: &audit.Subject{SubjectType: "token", SubjectID: "user-a", UID: "user-a"},
		Action:  "mcp.tool.call",
		Target:  audit.Target{Type: "mcp_tool", ID: "x"},
	}, plain))
	coded, _, err := s.List(ctx, ListAuditOptions{SubjectID: "user-a", Result: "failure", TargetType: "mcp_tool"})
	require.NoError(t, err)
	var sawCode bool
	for _, item := range coded {
		if item.ErrorCode == "invalid_argument" {
			sawCode = true
		}
		assert.NotContains(t, item.ErrorCode, "name is required")
	}
	assert.True(t, sawCode)

	require.NoError(t, s.writeEntry(ctx, audit.Entry{
		Subject: &audit.Subject{SubjectType: "token", SubjectID: "user-a", UID: "user-b"},
		Action:  "mcp.tool.call",
		Target:  audit.Target{Type: "mcp_tool", ID: "other"},
	}, "success", ""))
	own, _, err := s.List(ctx, ListAuditOptions{SubjectID: "user-a"})
	require.NoError(t, err)
	for _, item := range own {
		assert.NotEqual(t, "other", item.TargetID)
	}
	foreign, _, err := s.List(ctx, ListAuditOptions{SubjectID: "user-b"})
	require.NoError(t, err)
	var sawForeign bool
	for _, item := range foreign {
		if item.TargetID == "other" {
			sawForeign = true
		}
	}
	assert.True(t, sawForeign)

	big := map[string]any{"note": strings.Repeat("x", 5000)}
	require.NoError(t, s.writeEntry(ctx, audit.Entry{
		Subject: &audit.Subject{SubjectType: "token", SubjectID: "user-c", UID: "user-c"},
		Action:  "mcp.tool.call",
		Target:  audit.Target{Type: "mcp_tool", ID: "wide"},
		Request: big,
	}, "success", ""))
	wide, _, err := s.List(ctx, ListAuditOptions{SubjectID: "user-c"})
	require.NoError(t, err)
	require.Len(t, wide, 1)
	assert.True(t, wide[0].Truncated)
	_, isMap := wide[0].Request.(map[string]any)
	assert.True(t, isMap)
}

func TestAuditPageLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 20, auditPageLimit(0))
	assert.Equal(t, 20, auditPageLimit(-1))
	assert.Equal(t, 50, auditPageLimit(50))
	assert.Equal(t, 100, auditPageLimit(101))
}
