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

func TestExportSession(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setup      func(string) error
		state      int
		wantCount  int
		wantState  string
		wantErr    bool
		sessionID  string
		skipCreate bool
	}{
		{
			name: "exports message and compaction entries",
			setup: func(sessionID string) error {
				storage := NewDBStorage(sessionID, types.Uid("user-1"), "")
				if err := storage.Append(ctx, session.TreeEntry{
					ID: "e1", Type: session.EntryMessage,
					Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}},
				}); err != nil {
					return err
				}
				return storage.Append(ctx, session.TreeEntry{
					ID: "e2", ParentID: "e1", Type: session.EntryCompaction, Summary: "compact summary",
				})
			},
			wantCount: 2,
			wantState: "active",
		},
		{
			name:      "empty session exports metadata only",
			setup:     func(string) error { return nil },
			wantCount: 0,
			wantState: "active",
		},
		{
			name:       "missing session returns error",
			setup:      func(string) error { return nil },
			wantErr:    true,
			sessionID:  "missing-session",
			skipCreate: true,
		},
		{
			name: "closed session state label",
			setup: func(string) error {
				return nil
			},
			state:     int(schema.ChatSessionClosed),
			wantCount: 0,
			wantState: "closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := tt.sessionID
			if sessionID == "" {
				sessionID = types.Id()
			}
			if !tt.skipCreate {
				state := tt.state
				if state == 0 {
					state = int(schema.ChatSessionActive)
				}
				require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
					Flag: sessionID, UID: "user-1", State: state, CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, tt.setup(sessionID))
			}

			export, err := ExportSession(ctx, sessionID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, export)
			assert.Equal(t, sessionID, export.SessionID)
			assert.Equal(t, "user-1", export.UID)
			assert.Equal(t, tt.wantState, export.State)
			assert.Equal(t, tt.wantCount, export.EntryCount)
			assert.Len(t, export.Entries, tt.wantCount)
			assert.False(t, export.ExportedAt.IsZero())
		})
	}
}

func TestSessionStateLabel(t *testing.T) {
	tests := []struct {
		name  string
		state int
		want  string
	}{
		{name: "active", state: int(schema.ChatSessionActive), want: "active"},
		{name: "closed", state: int(schema.ChatSessionClosed), want: "closed"},
		{name: "unknown", state: 99, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sessionStateLabel(tt.state))
		})
	}
}
