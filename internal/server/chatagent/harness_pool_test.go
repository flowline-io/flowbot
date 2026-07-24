package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateHarness(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "harness reply")

	svc := NewService()
	t.Cleanup(func() { svc.ResetHarnessPoolForTest() })

	ctx := context.Background()
	sessionID := "sess-harness-pool"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))

	tests := []struct {
		name string
		run  func(*testing.T, *Service)
	}{
		{
			name: "creates harness on first call",
			run: func(t *testing.T, svc *Service) {
				h, err := svc.getOrCreateHarness(ctx, RunRequest{SessionID: sessionID, Text: "hi"}, 2)
				require.NoError(t, err)
				require.NotNil(t, h)
				_, ok := svc.harnessPoolMap().Load(sessionID)
				assert.True(t, ok)
			},
		},
		{
			name: "reuses pooled harness on second call",
			run: func(t *testing.T, svc *Service) {
				req := RunRequest{SessionID: sessionID, Text: "hello"}
				h1, err := svc.getOrCreateHarness(ctx, req, len(req.Text))
				require.NoError(t, err)
				h2, err := svc.getOrCreateHarness(ctx, req, len(req.Text))
				require.NoError(t, err)
				assert.Same(t, h1, h2)
			},
		},
		{
			name: "evict removes stale pooled entry",
			run: func(t *testing.T, svc *Service) {
				req := RunRequest{SessionID: sessionID, Text: "ping"}
				first, err := svc.getOrCreateHarness(ctx, req, len(req.Text))
				require.NoError(t, err)
				svc.EvictHarnessPool(sessionID)
				second, err := svc.getOrCreateHarness(ctx, req, len(req.Text))
				require.NoError(t, err)
				assert.NotSame(t, first, second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.ResetHarnessPoolForTest()
			tt.run(t, svc)
		})
	}
}

func TestApplySessionModeUpdatesTools(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "ok")

	svc := NewService()
	t.Cleanup(func() { svc.ResetHarnessPoolForTest() })

	ctx := context.Background()
	sessionID := "sess-mode"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive), Mode: ModePlan,
	}))

	h, err := svc.getOrCreateHarness(ctx, RunRequest{SessionID: sessionID, Text: "plan task"}, 9)
	require.NoError(t, err)
	require.NotNil(t, h)

	err = applySessionMode(ctx, h, RunRequest{
		SessionID: sessionID,
		Text:      "plan task",
		Kind:      RunKindInteractive,
	})
	require.NoError(t, err)
	require.NotNil(t, h.Agent())
}
