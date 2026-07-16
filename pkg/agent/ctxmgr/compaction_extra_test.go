package ctxmgr_test

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareCompactionExtraOnly(t *testing.T) {
	t.Parallel()

	entries := []session.TreeEntry{
		{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("kept")},
		{ID: "2", Type: session.EntryCompaction, Summary: "prior", FirstKeptEntryID: "1"},
	}
	extra := []agent.AgentMessage{agent.NewUserMessage("in-flight turn")}

	tests := []struct {
		name    string
		entries []session.TreeEntry
		extra   []agent.AgentMessage
		force   bool
		wantNil bool
	}{
		{name: "extra messages on compaction leaf", entries: entries, extra: extra, force: true, wantNil: false},
		{name: "no extra messages still recompacts with force", entries: entries, extra: nil, force: true, wantNil: false},
		{name: "without force skips compacted leaf", entries: entries, extra: extra, force: false, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.PrepareCompaction(tt.entries, ctxmgr.Settings{Enabled: true, KeepRecentTokens: 100000}, ctxmgr.PrepareOptions{
				Force:         tt.force,
				ExtraMessages: tt.extra,
			})
			require.True(t, got.IsOk())
			if tt.wantNil {
				assert.Nil(t, got.Value())
				return
			}
			require.NotNil(t, got.Value())
			assert.Equal(t, "1", got.Value().FirstKeptEntryID)
			assert.Len(t, got.Value().MessagesToSummarize, 1)
		})
	}
}

func TestFindCutPointSplitTurn(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("token ", 300)
	entries := []session.TreeEntry{
		{ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("question")},
		{ID: "2", Type: session.EntryMessage, Message: agent.AssistantMessage{
			Parts: []agent.ContentPart{
				agent.TextPart{Text: long},
				agent.ToolCallPart{Name: "read_file", Arguments: `{}`},
			},
		}},
		{ID: "3", Type: session.EntryMessage, Message: agent.ToolResultMessage{
			Parts: []agent.ContentPart{agent.TextPart{Text: "file contents"}},
		}},
		{ID: "4", Type: session.EntryMessage, Message: agent.AssistantMessage{
			Parts: []agent.ContentPart{agent.TextPart{Text: "final answer"}},
		}},
	}

	tests := []struct {
		name          string
		keepRecent    int
		wantSplitTurn bool
	}{
		{name: "tight budget can split turn", keepRecent: 5, wantSplitTurn: true},
		{name: "large budget keeps whole turn", keepRecent: 100000, wantSplitTurn: false},
		{name: "moderate budget may keep assistant tail", keepRecent: 50, wantSplitTurn: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.FindCutPoint(entries, 0, len(entries), tt.keepRecent)
			assert.Equal(t, tt.wantSplitTurn, got.IsSplitTurn)
			if tt.wantSplitTurn {
				assert.GreaterOrEqual(t, got.TurnStartIndex, 0)
			}
		})
	}
}

func TestSerializeConversationTruncatesToolOutput(t *testing.T) {
	t.Parallel()

	longText := strings.Repeat("x", 2500)
	got := ctxmgr.SerializeConversation([]agent.AgentMessage{
		agent.ToolResultMessage{Parts: []agent.ContentPart{agent.TextPart{Text: longText}}},
	})

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "includes prefix", text: got, want: "[Tool result]:"},
		{name: "truncates long output", text: got, want: "more characters truncated"},
		{name: "starts with tool marker", text: got, want: "[Tool result]: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, tt.text, tt.want)
		})
	}
}
