package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSummarySession(t *testing.T, db store.Adapter, text string) string {
	t.Helper()
	ctx := context.Background()
	sessionID := types.Id()
	require.NoError(t, db.CreateChatSession(ctx, &gen.ChatSession{
		Flag:  sessionID,
		UID:   "user-1",
		State: int(schema.ChatSessionActive),
		Title: "Widgets",
	}))
	storage := NewDBStorage(sessionID, types.Uid("user-1"), "")
	e1 := sessionID + "-e1"
	e2 := sessionID + "-e2"
	require.NoError(t, storage.Append(ctx, session.TreeEntry{
		ID:   e1,
		Type: session.EntryMessage,
		Message: msg.UserMessage{
			Parts: []msg.ContentPart{msg.TextPart{Text: text}},
		},
	}))
	require.NoError(t, storage.Append(ctx, session.TreeEntry{
		ID:       e2,
		ParentID: e1,
		Type:     session.EntryMessage,
		Message: msg.AssistantMessage{
			Parts: []msg.ContentPart{msg.TextPart{Text: "widgets are useful"}},
		},
	}))
	return sessionID
}

func TestSetSessionArchivedEnqueuesSummary(t *testing.T) {
	orig := store.Database
	db := postgres.NewSQLiteTestAdapter(t)
	store.Database = db
	t.Cleanup(func() {
		WaitForSessionSummaryGenerationForTest()
		store.Database = orig
	})
	restore := SetSessionSummaryLLMForTest(func(context.Context, string, string, sessionTitleModelFunc) (string, error) {
		return "Summary about widgets", nil
	})
	t.Cleanup(restore)

	ctx := context.Background()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "archive triggers ready summary",
			run: func(t *testing.T) {
				sessionID := seedSummarySession(t, db, "tell me about widgets")
				require.NoError(t, SetSessionArchived(ctx, sessionID, true))
				require.Eventually(t, func() bool {
					row, err := db.GetAgentSessionSummaryBySession(ctx, sessionID)
					return err == nil && row.Status == schema.AgentSessionSummaryReady
				}, 3*time.Second, 20*time.Millisecond)
				row, err := db.GetAgentSessionSummaryBySession(ctx, sessionID)
				require.NoError(t, err)
				assert.Contains(t, row.Summary, "widgets")
			},
		},
		{
			name: "unarchive keeps summary",
			run: func(t *testing.T) {
				sessionID := seedSummarySession(t, db, "tell me about widgets")
				require.NoError(t, SetSessionArchived(ctx, sessionID, true))
				require.Eventually(t, func() bool {
					row, err := db.GetAgentSessionSummaryBySession(ctx, sessionID)
					return err == nil && row.Status == schema.AgentSessionSummaryReady
				}, 3*time.Second, 20*time.Millisecond)
				require.NoError(t, SetSessionArchived(ctx, sessionID, false))
				row, err := db.GetAgentSessionSummaryBySession(ctx, sessionID)
				require.NoError(t, err)
				assert.NotEmpty(t, row.Summary)
			},
		},
		{
			name: "retry requeues failed summary",
			run: func(t *testing.T) {
				sessionID := seedSummarySession(t, db, "tell me about widgets")
				_, err := db.UpsertAgentSessionSummaryPending(ctx, sessionID, "default", "Widgets")
				require.NoError(t, err)
				_, err = db.ClaimAgentSessionSummaryPending(ctx, "fail-tok")
				require.NoError(t, err)
				require.NoError(t, db.MarkAgentSessionSummaryFailed(ctx, sessionID, "fail-tok", "boom"))
				require.NoError(t, RetrySessionSummary(ctx, sessionID))
				require.Eventually(t, func() bool {
					row, err := db.GetAgentSessionSummaryBySession(ctx, sessionID)
					return err == nil && row.Status == schema.AgentSessionSummaryReady
				}, 3*time.Second, 20*time.Millisecond)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestBuildSessionSummaryInputBudget(t *testing.T) {
	orig := store.Database
	db := postgres.NewSQLiteTestAdapter(t)
	store.Database = db
	t.Cleanup(func() { store.Database = orig })

	ctx := context.Background()
	sessionID := types.Id()
	require.NoError(t, db.CreateChatSession(ctx, &gen.ChatSession{
		Flag:  sessionID,
		UID:   "user-1",
		State: int(schema.ChatSessionActive),
	}))
	storage := NewDBStorage(sessionID, types.Uid("user-1"), "")
	long := stringsRepeat("x", 3000)
	require.NoError(t, storage.Append(ctx, session.TreeEntry{
		ID:      sessionID + "-e1",
		Type:    session.EntryMessage,
		Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: long}}},
	}))

	tests := []struct {
		name string
	}{
		{name: "truncates long messages"},
		{name: "returns non-empty input"},
		{name: "includes role prefix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := buildSessionSummaryInput(ctx, sessionID)
			require.NoError(t, err)
			assert.Contains(t, text, "user:")
			assert.LessOrEqual(t, len(text), sessionSummaryMaxInputChars)
		})
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
