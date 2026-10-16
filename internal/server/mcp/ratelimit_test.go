package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

func useMiniredis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	SetRateRedis(client)
	t.Cleanup(func() {
		SetRateRedis(nil)
		_ = client.Close()
	})
	return client
}

func TestAllowToolCallSlidingWindow(t *testing.T) {
	useMiniredis(t)
	prev := config.App.MCP.RateLimit
	config.App.MCP.RateLimit = config.MCPRateLimitConfig{PerTokenRPM: 2, WritePerTokenRPM: 1, LineagePerTokenRPM: 1}
	t.Cleanup(func() { config.App.MCP.RateLimit = prev })

	read := ToolSpec{Name: "hub.health", Group: "hub", Operation: "health"}
	write := ToolSpec{Name: "pipeline.run", Group: "pipeline", Operation: "run"}
	lineage := ToolSpec{Name: "lineage.trace", Group: "lineage", Operation: "trace"}
	now := time.Now()

	allowed, _, err := allowToolCall(context.Background(), bucketsForTool(7, read, true), now)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, retry, err := allowToolCall(context.Background(), bucketsForTool(7, write, true), now)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Zero(t, retry)

	allowed, retry, err = allowToolCall(context.Background(), bucketsForTool(7, write, true), now)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Positive(t, retry)

	tier, _ := capability.Classify(write.Group, write.Operation)
	assert.NotEqual(t, capability.TierRead, tier)

	allowed, _, err = allowToolCall(context.Background(), bucketsForTool(8, lineage, true), now)
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, _, err = allowToolCall(context.Background(), bucketsForTool(8, lineage, true), now.Add(time.Millisecond))
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, _, err = allowToolCall(context.Background(), bucketsForTool(7, read, true), now.Add(rateWindow+time.Second))
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAllowToolCallFirstFullBucket(t *testing.T) {
	client := useMiniredis(t)
	prev := config.App.MCP.RateLimit
	config.App.MCP.RateLimit = config.MCPRateLimitConfig{PerTokenRPM: 1, WritePerTokenRPM: 1}
	t.Cleanup(func() { config.App.MCP.RateLimit = prev })

	now := time.Now()
	ctx := context.Background()
	require.NoError(t, client.ZAdd(ctx, "mcp:rl:9:all", redis.Z{
		Score:  float64(now.Add(-10 * time.Second).UnixMilli()),
		Member: "older-total",
	}).Err())
	require.NoError(t, client.ZAdd(ctx, "mcp:rl:9:write", redis.Z{
		Score:  float64(now.Add(-50 * time.Second).UnixMilli()),
		Member: "newer-write",
	}).Err())

	write := ToolSpec{Name: "pipeline.run", Group: "pipeline", Operation: "run"}
	allowed, retry, err := allowToolCall(ctx, bucketsForTool(9, write, true), now)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Greater(t, retry, 40*time.Second)
}

func TestAllowToolCallRedisDown(t *testing.T) {
	client := useMiniredis(t)
	require.NoError(t, client.Close())
	_, _, err := allowToolCall(context.Background(), bucketsForTool(1, ToolSpec{Name: "hub.health", Operation: "health"}, true), time.Now())
	require.ErrorIs(t, err, types.ErrUnavailable)
}

func TestArgumentByteCount(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, argumentByteCount(nil))
	assert.Equal(t, 0, argumentByteCount([]byte("null")))
	assert.Equal(t, 2, argumentByteCount([]byte("{}")))
}

func TestPreflightRejectsOversizedArguments(t *testing.T) {
	prev := config.App.MCP
	config.App.MCP.Enabled = true
	config.App.MCP.MaxArgumentBytes = 8
	t.Cleanup(func() { config.App.MCP = prev })

	app := fiber.New()
	Register(app, nil)
	body := []byte(`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"hub.health","arguments":{"q":"too-long"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+seedAPIToken(t))
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(raw), `"code":"invalid_argument"`)
	assert.Contains(t, string(raw), `"field":"arguments"`)
}

func seedAPIToken(t *testing.T) string {
	t.Helper()
	store := &memoryTokenStore{}
	route.SetAccessTokenStore(store)
	t.Cleanup(func() { route.SetAccessTokenStore(nil) })
	token := "rate-argument-token"
	require.NoError(t, store.Set(context.Background(), auth.HashToken(token), types.KV{
		"uid": "user-1", "kind": "api", "scopes": []string{"hub:health:read"},
	}, time.Now().Add(time.Hour)))
	return token
}
