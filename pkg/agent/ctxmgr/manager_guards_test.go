package ctxmgr_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestManagerNilSessionGuards(t *testing.T) {
	t.Parallel()

	mgr := ctxmgr.New(ctxmgr.Options{
		Model:         agentllm.NewFakeModel(agentllm.ResponseScript{Content: "summary"}),
		ModelName:     "fake",
		ContextWindow: 4096,
		Settings:      ctxmgr.Settings{Enabled: true},
	})

	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{
			name: "ensure within budget nil session",
			run: func() error {
				return mgr.EnsureWithinBudget(context.Background(), nil, nil)
			},
		},
		{
			name: "compact and reload nil session",
			run: func() error {
				_, err := mgr.CompactAndReload(context.Background(), nil, nil, ctxmgr.CompactOpts{})
				return err
			},
			wantErr: true,
		},
		{
			name:    "move to nil session",
			run:     func() error { return mgr.MoveTo(context.Background(), nil, "x", "") },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.run()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRunCompaction(t *testing.T) {
	t.Parallel()

	model := agentllm.NewFakeModel(
		agentllm.ResponseScript{Content: "## Goal\nCompacted"},
		agentllm.ResponseScript{Content: "prefix chunk"},
	)

	tests := []struct {
		name     string
		prep     *ctxmgr.CompactionPreparation
		wantOK   bool
		wantText string
	}{
		{
			name: "split turn compacts prefix and history",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID:    "keep",
				MessagesToSummarize: []msg.AgentMessage{msg.NewUserMessage("history")},
				TurnPrefixMessages:  []msg.AgentMessage{msg.NewUserMessage("prefix")},
				IsSplitTurn:         true,
				FileOps:             ctxmgr.NewFileOperations(),
				Settings:            ctxmgr.Settings{},
			},
			wantOK:   true,
			wantText: "Compacted",
		},
		{
			name: "empty messages returns nothing to compact",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID: "keep",
				FileOps:          ctxmgr.NewFileOperations(),
				Settings:         ctxmgr.Settings{},
			},
			wantOK: false,
		},
		{
			name:   "nil preparation returns error result",
			prep:   nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.prep == nil {
				got := ctxmgr.RunCompaction(context.Background(), model, "fake", nil)
				assert.False(t, got.IsOk())
				return
			}
			got := ctxmgr.RunCompaction(context.Background(), model, "fake", tt.prep)
			require.Equal(t, tt.wantOK, got.IsOk())
			if tt.wantText != "" {
				assert.Contains(t, got.Value().Summary, tt.wantText)
			}
		})
	}
}

func TestRunCompactionReplaysConversationPrefix(t *testing.T) {
	t.Parallel()

	echoTool := llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "echo",
			Description: "echo",
			Parameters:  map[string]any{"type": "object"},
		},
	}

	tests := []struct {
		name            string
		prep            *ctxmgr.CompactionPreparation
		wantSystem      string
		wantTool        string
		wantInstruction bool
		wantCheckpoint  bool
		wantSanitized   bool
		wantThinking    string
	}{
		{
			name: "replays system tools and trailing instruction",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID:    "keep",
				MessagesToSummarize: []msg.AgentMessage{msg.NewUserMessage("history")},
				FileOps:             ctxmgr.NewFileOperations(),
				Settings:            ctxmgr.Settings{},
				SystemPrompt:        "you are the coding assistant",
				Tools:               []llms.Tool{echoTool},
				ThinkingLevel:       agentllm.ThinkingLevelHigh,
			},
			wantSystem:      "you are the coding assistant",
			wantTool:        "echo",
			wantInstruction: true,
			wantThinking:    agentllm.ThinkingLevelHigh,
		},
		{
			name: "does not flatten history into a transcript",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID:    "keep",
				MessagesToSummarize: []msg.AgentMessage{msg.NewUserMessage("history")},
				FileOps:             ctxmgr.NewFileOperations(),
				Settings:            ctxmgr.Settings{},
				SystemPrompt:        "system",
			},
			wantSystem:      "system",
			wantInstruction: true,
		},
		{
			name: "prepends prior checkpoint as a converted message",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID:    "keep",
				MessagesToSummarize: []msg.AgentMessage{msg.NewUserMessage("newer work")},
				PreviousSummary:     "old checkpoint",
				FileOps:             ctxmgr.NewFileOperations(),
				Settings:            ctxmgr.Settings{},
				SystemPrompt:        "system",
			},
			wantSystem:      "system",
			wantCheckpoint:  true,
			wantInstruction: true,
		},
		{
			name: "sanitizes tool results that precede their assistant",
			prep: &ctxmgr.CompactionPreparation{
				FirstKeptEntryID: "keep",
				MessagesToSummarize: []msg.AgentMessage{
					msg.ToolResultMessage{
						ToolCallID: "c1",
						Name:       "echo",
						Parts:      []msg.ContentPart{msg.TextPart{Text: "tool-out"}},
					},
					msg.AssistantMessage{
						Parts: []msg.ContentPart{
							msg.ToolCallPart{ID: "c1", Name: "echo", Arguments: `{}`},
						},
					},
				},
				FileOps:      ctxmgr.NewFileOperations(),
				Settings:     ctxmgr.Settings{},
				SystemPrompt: "system",
			},
			wantSystem:      "system",
			wantInstruction: true,
			wantSanitized:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "## Goal\nCompacted"})
			got := ctxmgr.RunCompaction(context.Background(), model, "fake", tt.prep)
			require.True(t, got.IsOk())
			messages := model.LastMessages()
			require.NotEmpty(t, messages)
			assert.Equal(t, llms.ChatMessageTypeSystem, messages[0].Role)
			require.NotEmpty(t, messages[0].Parts)
			sys, ok := messages[0].Parts[0].(llms.TextContent)
			require.True(t, ok)
			assert.Equal(t, tt.wantSystem, sys.Text)
			assert.NotContains(t, joinedMessageText(messages), "<conversation>")
			last := messages[len(messages)-1]
			assert.Equal(t, llms.ChatMessageTypeHuman, last.Role)
			if tt.wantInstruction {
				assert.Contains(t, joinedMessageText([]llms.MessageContent{last}), "compaction engine")
			}
			if tt.wantTool != "" {
				require.NotEmpty(t, model.LastTools())
				assert.Equal(t, tt.wantTool, model.LastTools()[0].Function.Name)
			} else {
				assert.Empty(t, model.LastTools())
			}
			if tt.wantCheckpoint {
				assert.Contains(t, joinedMessageText(messages), "old checkpoint")
			}
			if tt.wantThinking != "" {
				assert.Equal(t, tt.wantThinking, agentllm.ThinkingLevelFromContext(model.LastContext()))
			}
			if tt.wantSanitized {
				require.GreaterOrEqual(t, len(messages), 4)
				assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)
				assert.Equal(t, llms.ChatMessageTypeTool, messages[2].Role)
			}
		})
	}
}

func joinedMessageText(messages []llms.MessageContent) string {
	var b strings.Builder
	for _, message := range messages {
		for _, part := range message.Parts {
			if text, ok := part.(llms.TextContent); ok {
				_, _ = b.WriteString(text.Text)
			}
		}
	}
	return b.String()
}

func TestManagerCompactAndReloadDisabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := session.NewMemoryStorage()
	sess := session.New(store)
	require.NoError(t, sess.Append(ctx, session.TreeEntry{
		ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("hello"),
	}))

	mgr := ctxmgr.New(ctxmgr.Options{
		Model:         agentllm.NewFakeModel(),
		ModelName:     "fake",
		ContextWindow: 4096,
		Settings:      ctxmgr.Settings{Enabled: false},
	})

	_, err := mgr.CompactAndReload(ctx, sess, nil, ctxmgr.CompactOpts{Force: false})
	require.NoError(t, err)
}
