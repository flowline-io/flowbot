package chatagent

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSumRunDurationFromBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		branch []session.TreeEntry
		want   int64
	}{
		{
			name: "sums assistant run durations",
			branch: []session.TreeEntry{
				{
					Type: session.EntryMessage,
					Message: msg.AssistantMessage{
						Parts:         []msg.ContentPart{msg.TextPart{Text: "first"}},
						RunDurationMs: 1200,
					},
				},
				{
					Type: session.EntryMessage,
					Message: msg.AssistantMessage{
						Parts:         []msg.ContentPart{msg.TextPart{Text: "second"}},
						RunDurationMs: 3400,
					},
				},
			},
			want: 4600,
		},
		{
			name: "ignores user and assistant without run duration",
			branch: []session.TreeEntry{
				{
					Type:    session.EntryMessage,
					Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}},
				},
				{
					Type: session.EntryMessage,
					Message: msg.AssistantMessage{
						Parts: []msg.ContentPart{msg.TextPart{Text: "reply"}},
					},
				},
				{
					Type: session.EntryMessage,
					Message: msg.ToolResultMessage{
						Name:       "echo",
						Parts:      []msg.ContentPart{msg.TextPart{Text: "ok"}},
						DurationMs: 50,
					},
				},
			},
			want: 0,
		},
		{
			name:   "empty branch",
			branch: nil,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, sumRunDurationFromBranch(tt.branch))
		})
	}
}

func TestSumSessionsRunDurationMs(t *testing.T) {
	e1Payload, err := session.MarshalEntry(session.TreeEntry{
		ID:       "e1",
		Type:     session.EntryMessage,
		ParentID: "",
		Message: msg.AssistantMessage{
			Parts:         []msg.ContentPart{msg.TextPart{Text: "first"}},
			RunDurationMs: 1200,
		},
	})
	require.NoError(t, err)
	e2Payload, err := session.MarshalEntry(session.TreeEntry{
		ID:       "e2",
		Type:     session.EntryMessage,
		ParentID: "e1",
		Message: msg.AssistantMessage{
			Parts:         []msg.ContentPart{msg.TextPart{Text: "second"}},
			RunDurationMs: 3400,
		},
	})
	require.NoError(t, err)
	otherPayload, err := session.MarshalEntry(session.TreeEntry{
		ID:       "o1",
		Type:     session.EntryMessage,
		ParentID: "",
		Message: msg.AssistantMessage{
			Parts:         []msg.ContentPart{msg.TextPart{Text: "other"}},
			RunDurationMs: 500,
		},
	})
	require.NoError(t, err)

	var e1Map, e2Map, otherMap map[string]any
	require.NoError(t, sonic.Unmarshal(e1Payload, &e1Map))
	require.NoError(t, sonic.Unmarshal(e2Payload, &e2Map))
	require.NoError(t, sonic.Unmarshal(otherPayload, &otherMap))

	tests := []struct {
		name          string
		leafBySession map[string]string
		seed          func(t *testing.T)
		want          map[string]int64
		wantErr       bool
	}{
		{
			name:          "sums multiple sessions",
			leafBySession: map[string]string{"sess-a": "e2", "sess-b": "o1"},
			seed: func(t *testing.T) {
				require.NoError(t, store.ChatStoreFromDB().CreateChatSessionEntry(context.Background(), &gen.ChatSessionEntry{
					Flag: "e1", SessionID: "sess-a", ParentID: "", EntryType: "message", Payload: e1Map,
				}))
				require.NoError(t, store.ChatStoreFromDB().CreateChatSessionEntry(context.Background(), &gen.ChatSessionEntry{
					Flag: "e2", SessionID: "sess-a", ParentID: "e1", EntryType: "message", Payload: e2Map,
				}))
				require.NoError(t, store.ChatStoreFromDB().CreateChatSessionEntry(context.Background(), &gen.ChatSessionEntry{
					Flag: "o1", SessionID: "sess-b", ParentID: "", EntryType: "message", Payload: otherMap,
				}))
			},
			want: map[string]int64{"sess-a": 4600, "sess-b": 500},
		},
		{
			name:          "skips empty leaf and broken branch",
			leafBySession: map[string]string{"empty": "", "broken": "missing", "ok": "o1"},
			seed: func(t *testing.T) {
				require.NoError(t, store.ChatStoreFromDB().CreateChatSessionEntry(context.Background(), &gen.ChatSessionEntry{
					Flag: "o1", SessionID: "ok", ParentID: "", EntryType: "message", Payload: otherMap,
				}))
			},
			want: map[string]int64{"ok": 500},
		},
		{
			name:          "returns list error when database unavailable",
			leafBySession: map[string]string{"sess-a": "e1"},
			seed:          func(_ *testing.T) { store.Database = nil },
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			t.Cleanup(func() { store.Database = orig })
			if tt.seed != nil {
				tt.seed(t)
			}

			got, err := SumSessionsRunDurationMs(context.Background(), tt.leafBySession)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
