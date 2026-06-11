package session_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_BuildContextAndMoveTo(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(context.Context, *session.Session) error
		wantMessages int
		wantModel    string
	}{
		{
			name: "linear branch",
			setup: func(ctx context.Context, s *session.Session) error {
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("a")}))
				return s.Append(ctx, session.TreeEntry{ID: "leaf", ParentID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("b")})
			},
			wantMessages: 2,
		},
		{
			name: "branch summary",
			setup: func(ctx context.Context, s *session.Session) error {
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "root", Type: session.EntryMessage, Message: agent.NewUserMessage("a")}))
				return s.MoveTo(ctx, "root", "rolled back summary")
			},
			wantMessages: 2,
		},
		{
			name: "compaction boundary",
			setup: func(ctx context.Context, s *session.Session) error {
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "old", Type: session.EntryMessage, Message: agent.NewUserMessage("old")}))
				require.NoError(t, s.Append(ctx, session.TreeEntry{ID: "keep", ParentID: "old", Type: session.EntryMessage, Message: agent.NewUserMessage("kept")}))
				return s.Append(ctx, session.TreeEntry{
					ID: "compact", ParentID: "keep", Type: session.EntryCompaction,
					Summary: "summary", FirstKeptEntryID: "keep", TokensBefore: 100,
				})
			},
			wantMessages: 2,
		},
		{
			name: "model change entry",
			setup: func(ctx context.Context, s *session.Session) error {
				return s.Append(ctx, session.TreeEntry{ID: "root", Type: session.EntryModelChange, ModelName: "gpt"})
			},
			wantModel: "gpt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := session.NewMemoryStorage()
			s := session.New(store)
			ctx := context.Background()
			require.NoError(t, tt.setup(ctx, s))

			branch, err := s.GetBranch(ctx, "")
			require.NoError(t, err)
			built := session.BuildContext(branch)
			if tt.wantModel != "" {
				assert.Equal(t, tt.wantModel, built.ModelName)
				return
			}
			assert.Len(t, built.Messages, tt.wantMessages)
		})
	}
}

func TestJSONL_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name    string
		entries []session.TreeEntry
	}{
		{name: "single message", entries: []session.TreeEntry{{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("hi")}}},
		{name: "model change", entries: []session.TreeEntry{{ID: "1", Type: session.EntryModelChange, ModelName: "gpt"}}},
		{name: "multiple entries", entries: []session.TreeEntry{
			{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("a")},
			{ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("b")},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := session.SerializeSession(tt.entries)
			require.NoError(t, err)
			loaded, err := session.DeserializeSession(data)
			require.NoError(t, err)
			assert.Len(t, loaded, len(tt.entries))
		})
	}
}
