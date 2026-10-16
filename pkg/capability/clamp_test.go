package capability

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyFullListPageTwoPages(t *testing.T) {
	t.Parallel()
	type row struct {
		ID string `json:"id"`
	}
	items := make([]*row, 0, 25)
	for i := range 25 {
		items = append(items, &row{ID: string(rune('a'+i/26)) + string(rune('a'+i%26))})
	}
	ctx := context.Background()
	first, page := ApplyFullListPage(ctx, "example", map[string]any{"limit": 20}, items, nil)
	require.NotNil(t, page)
	assert.True(t, page.HasMore)
	assert.NotEmpty(t, page.NextCursor)
	firstIDs := idsOf(t, first)

	second, page2 := ApplyFullListPage(ctx, "example", map[string]any{"limit": 20, "cursor": page.NextCursor}, items, nil)
	require.NotNil(t, page2)
	assert.False(t, page2.HasMore)
	secondIDs := idsOf(t, second)
	seen := map[string]bool{}
	for _, id := range append(firstIDs, secondIDs...) {
		assert.False(t, seen[id], "duplicate %s", id)
		seen[id] = true
	}
	assert.Len(t, seen, 25)
}

func TestClampListedDropsTailAndResumes(t *testing.T) {
	t.Parallel()
	type row struct {
		ID   string `json:"id"`
		Body string `json:"body"`
	}
	items := make([]*row, 0, 40)
	for i := range 40 {
		items = append(items, &row{ID: string(rune('A'+(i%26))) + string(rune('0'+i/26)), Body: strings.Repeat("x", 20000)})
	}
	ctx := WithCursorSecretBox(context.Background())
	ctx = WithPageBudget(ctx, PageBudget{Max: 32768, Measure: measureDataPage})
	paged, page := ApplyFullListPage(ctx, "example", map[string]any{"limit": 20}, items, nil)
	require.NotNil(t, page)
	out, sealed := ClampListed(ctx, "example", map[string]any{"limit": 20}, paged, page, fullListSecret)
	require.NotNil(t, sealed)
	raw, err := sonic.Marshal(map[string]any{"data": out, "page": sealed, "trace": map[string]any{"request_id": "req"}})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(raw), 32768)
	assert.True(t, sealed.Truncated)
	assert.NotEmpty(t, sealed.NextCursor)
	payload, err := DecodeCursor(fullListSecret, sealed.NextCursor, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "offset", payload.Strategy)
	assert.Positive(t, payload.Offset)
}

func TestClampListedStubsOversizedString(t *testing.T) {
	t.Parallel()
	item := map[string]any{"id": "1", "body": strings.Repeat("y", 100000)}
	ctx := WithPageBudget(context.Background(), PageBudget{Max: 4096, Measure: measureDataPage})
	out, page := ClampListed(ctx, "example", nil, []any{item}, &PageInfo{Limit: 1}, nil)
	raw, err := sonic.Marshal(map[string]any{"data": out, "page": page, "trace": map[string]any{"request_id": "req"}})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(raw), 4096)
	require.NotNil(t, page)
	assert.True(t, page.Truncated)
	blob := string(raw)
	assert.True(t, strings.Contains(blob, "…[truncated ") || strings.Contains(blob, StubItemMessage))
}

func TestApplyFullListPageKeepsCallerLimit(t *testing.T) {
	t.Parallel()
	items := make([]map[string]any, 0, 3)
	for i := range 3 {
		items = append(items, map[string]any{"id": string(rune('a' + i))})
	}
	out, page := ApplyFullListPage(context.Background(), "devops", map[string]any{"limit": 1000}, items, nil)
	require.NotNil(t, page)
	assert.Equal(t, 1000, page.Limit)
	assert.False(t, page.HasMore)
	assert.Len(t, idsOf(t, out), 3)
}

func TestClampListedKeepsProviderCursor(t *testing.T) {
	t.Parallel()
	secret := []byte("provider-cursor-secret")
	next, err := EncodeCursor(secret, CursorPayload{
		Capability:     "github",
		Strategy:       "offset",
		ProviderCursor: "2",
		Limit:          2,
	})
	require.NoError(t, err)
	items := []any{
		map[string]any{"id": "1", "body": strings.Repeat("x", 4000)},
		map[string]any{"id": "2", "body": strings.Repeat("y", 4000)},
		map[string]any{"id": "3", "body": strings.Repeat("z", 4000)},
	}
	ctx := WithPageBudget(context.Background(), PageBudget{Max: 500, Measure: measureDataPage})
	out, page := ClampListed(ctx, "github", nil, items, &PageInfo{Limit: 2, HasMore: true, NextCursor: next}, secret)
	require.NotNil(t, page)
	assert.True(t, page.Truncated)
	assert.Equal(t, next, page.NextCursor)
	assert.Len(t, idsOf(t, out), 3)
}

func measureDataPage(items []any, page map[string]any) (int, error) {
	raw, err := sonic.Marshal(map[string]any{
		"data":  items,
		"page":  page,
		"trace": map[string]any{"request_id": "req"},
	})
	if err != nil {
		return 0, err
	}
	return len(raw), nil
}

func idsOf(t *testing.T, data any) []string {
	t.Helper()
	raw, err := sonic.Marshal(data)
	require.NoError(t, err)
	var rows []struct {
		ID string `json:"id"`
	}
	require.NoError(t, sonic.Unmarshal(raw, &rows))
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
