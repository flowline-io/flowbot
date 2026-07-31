package notify

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresenceTouchAndIsPresent(t *testing.T) {
	ClearPresenceForTest()
	t.Cleanup(ClearPresenceForTest)

	SetPresenceWindowForTest(50 * time.Millisecond)
	t.Cleanup(func() { SetPresenceWindowForTest(5 * time.Minute) })

	assert.False(t, IsPresent("u1"))
	TouchPresence("u1")
	assert.True(t, IsPresent("u1"))
	time.Sleep(60 * time.Millisecond)
	assert.False(t, IsPresent("u1"))
}

func TestComputeEscalateAt(t *testing.T) {
	ClearPresenceForTest()
	t.Cleanup(ClearPresenceForTest)
	SetEscalateAfterForTest(10 * time.Minute)
	t.Cleanup(func() { SetEscalateAfterForTest(10 * time.Minute) })

	offline := computeEscalateAt("u-offline", nil)
	require.Less(t, time.Since(offline), time.Second)

	TouchPresence("u-online")
	online := computeEscalateAt("u-online", nil)
	require.True(t, online.After(time.Now().Add(9*time.Minute)))

	custom := computeEscalateAt("u-online", map[string]any{PayloadKeyEscalateAfter: "2m"})
	require.True(t, custom.Before(time.Now().Add(3*time.Minute)))
	require.True(t, custom.After(time.Now().Add(time.Minute)))
}

func TestEnsureCorrelationPayload(t *testing.T) {
	p := ensureCorrelationPayload(nil)
	require.NotEmpty(t, p[PayloadKeyCorrelationID])

	p2 := ensureCorrelationPayload(map[string]any{PayloadKeyCorrelationID: "fixed-id"})
	assert.Equal(t, "fixed-id", p2[PayloadKeyCorrelationID])
}

func TestIsSystemNotifyChannel(t *testing.T) {
	assert.True(t, IsSystemNotifyChannel(ChannelInapp))
	assert.False(t, IsSystemNotifyChannel("slack"))
}

func TestDefaultInboxChannelsWithoutDB(t *testing.T) {
	channels := DefaultInboxChannels(t.Context())
	require.Equal(t, []string{ChannelInapp}, channels)
}
