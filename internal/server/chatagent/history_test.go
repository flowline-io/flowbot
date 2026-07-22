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

func TestHistoryMessagesFromMessage(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name           string
		in             msg.AgentMessage
		want           int
		kind           string
		ms             int64
		rejectToolRows bool
	}{
		{
			name: "tool result row",
			in: msg.ToolResultMessage{
				Name:       "echo",
				Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
				DurationMs: 88,
			},
			want: 1,
			kind: "tool",
			ms:   88,
		},
		{
			name: "thinking and assistant rows",
			in: msg.AssistantMessage{
				Parts:              []msg.ContentPart{msg.TextPart{Text: "answer"}},
				ThinkingText:       "plan",
				ThinkingDurationMs: 200,
				TurnDurationMs:     900,
				RunDurationMs:      4000,
			},
			want: 2,
			kind: "thinking",
			ms:   200,
		},
		{
			name: "user row",
			in:   msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}},
			want: 1,
			kind: "user",
		},
		{
			name: "tool-call assistant emits thinking only not completed tool",
			in: msg.AssistantMessage{
				Parts: []msg.ContentPart{
					msg.ToolCallPart{ID: "c1", Name: "run_terminal", Arguments: `{"command":"ls"}`},
				},
				ThinkingText:       "need ls",
				ThinkingDurationMs: 50,
			},
			want:           1,
			kind:           "thinking",
			rejectToolRows: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := historyMessagesFromMessage(tt.in, createdAt)
			assert.Len(t, got, tt.want)
			if tt.want == 0 {
				return
			}
			assert.Equal(t, tt.kind, got[0].Kind)
			if tt.rejectToolRows {
				for _, row := range got {
					assert.NotEqual(t, "tool", row.Kind, "unexecuted tool calls must not become tool rows")
					assert.NotEqual(t, "completed", row.ToolStatus)
				}
			}
			if tt.kind == "tool" {
				assert.Equal(t, tt.ms, got[0].DurationMs)
				assert.Equal(t, "completed", got[0].ToolStatus)
			}
			if tt.kind == "thinking" && tt.want == 2 {
				assert.Equal(t, tt.ms, got[0].ThinkingDurationMs)
				assert.Equal(t, "assistant", got[1].Kind)
				assert.Equal(t, int64(900), got[1].TurnDurationMs)
				assert.Equal(t, int64(4000), got[1].RunDurationMs)
			}
		})
	}
}

func TestListSessionMessages(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func(sessionID string) error
		wantLen    int
		wantKinds  []string
		wantErr    bool
		skipCreate bool
	}{
		{
			name: "returns user assistant and compaction rows",
			setup: func(sessionID string) error {
				storage := NewDBStorage(sessionID, types.Uid("user-1"), "")
				if err := storage.Append(ctx, session.TreeEntry{
					ID: "m1", Type: session.EntryMessage,
					Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}},
				}); err != nil {
					return err
				}
				if err := storage.Append(ctx, session.TreeEntry{
					ID: "m2", ParentID: "m1", Type: session.EntryMessage,
					Message: msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi there"}}},
				}); err != nil {
					return err
				}
				return storage.Append(ctx, session.TreeEntry{
					ID: "c1", ParentID: "m2", Type: session.EntryCompaction, Summary: "summary text",
				})
			},
			wantLen:   3,
			wantKinds: []string{"user", "assistant", "assistant"},
		},
		{
			name:    "empty session returns no rows",
			setup:   func(string) error { return nil },
			wantLen: 0,
		},
		{
			name:       "missing session errors",
			setup:      func(string) error { return nil },
			wantErr:    true,
			skipCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentSessionID := "missing-history-session"
			if !tt.skipCreate {
				currentSessionID = types.Id()
				require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
					Flag: currentSessionID, UID: "user-1", State: int(schema.ChatSessionActive),
				}))
			}
			require.NoError(t, tt.setup(currentSessionID))

			messages, err := ListSessionMessages(ctx, currentSessionID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, messages, tt.wantLen)
			for i, kind := range tt.wantKinds {
				assert.Equal(t, kind, messages[i].Kind)
			}
		})
	}
}

func TestHasPersistedToolResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []HistoryMessage
		want bool
	}{
		{name: "has tool", in: []HistoryMessage{{Kind: "tool"}}, want: true},
		{name: "assistant only", in: []HistoryMessage{{Kind: "assistant"}}, want: false},
		{name: "empty", in: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasPersistedToolResults(tt.in))
		})
	}
}
