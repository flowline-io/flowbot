package chatagent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryStore struct {
	leaf    string
	entries []session.TreeEntry
}

func (m *memoryStore) Append(_ context.Context, entry session.TreeEntry) error {
	m.entries = append(m.entries, entry)
	m.leaf = entry.ID
	return nil
}

func (m *memoryStore) GetBranch(_ context.Context, leafID string) ([]session.TreeEntry, error) {
	if leafID == "" {
		leafID = m.leaf
	}
	if leafID == "" {
		return nil, nil
	}
	byID := make(map[string]session.TreeEntry, len(m.entries))
	for _, entry := range m.entries {
		byID[entry.ID] = entry
	}
	leaf, ok := byID[leafID]
	if !ok {
		return nil, assert.AnError
	}
	path := []session.TreeEntry{leaf}
	current := leaf
	for current.ParentID != "" {
		parent, exists := byID[current.ParentID]
		if !exists {
			break
		}
		path = append([]session.TreeEntry{parent}, path...)
		current = parent
	}
	return path, nil
}

func (m *memoryStore) GetLeafID(_ context.Context) (string, error) {
	return m.leaf, nil
}

func (m *memoryStore) SetLeafID(_ context.Context, id string) error {
	m.leaf = id
	return nil
}

func TestMemoryStoreBranch(t *testing.T) {
	tests := []struct {
		name    string
		append  []session.TreeEntry
		wantLen int
	}{
		{
			name: "single entry branch",
			append: []session.TreeEntry{
				{ID: "1", Type: session.EntryMessage, Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}}},
			},
			wantLen: 1,
		},
		{
			name: "linked branch",
			append: []session.TreeEntry{
				{ID: "1", Type: session.EntryMessage, Message: msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}}},
				{ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}}}},
			},
			wantLen: 2,
		},
		{
			name:    "empty branch",
			append:  nil,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &memoryStore{}
			for _, entry := range tt.append {
				require.NoError(t, store.Append(context.Background(), entry))
			}
			branch, err := store.GetBranch(context.Background(), "")
			require.NoError(t, err)
			assert.Len(t, branch, tt.wantLen)
		})
	}
}

func TestIsChatControlCommand(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "chat command", text: "chat", want: true},
		{name: "end command", text: "END", want: true},
		{name: "help command", text: "help", want: true},
		{name: "plan command", text: "plan", want: true},
		{name: "proceed command", text: "PROCEED", want: true},
		{name: "normal message", text: "write a function", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, chatagent.IsChatControlCommand(tt.text))
		})
	}
}
