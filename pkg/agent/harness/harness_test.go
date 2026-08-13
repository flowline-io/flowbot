package harness_test

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/tools/echo"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestHarnessOverflowRetryUsesFinalResult(t *testing.T) {
	tests := []struct {
		name       string
		scripts    []agentllm.ResponseScript
		wantErr    bool
		wantSubstr string
	}{
		{
			name: "overflow then success",
			scripts: []agentllm.ResponseScript{
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
				{Content: "## Goal\nCompact summary"},
				{Content: "recovered reply"},
			},
			wantSubstr: "recovered reply",
		},
		{
			name:       "single success",
			scripts:    []agentllm.ResponseScript{{Content: "ok"}},
			wantSubstr: "ok",
		},
		{
			name: "overflow without recovery script",
			scripts: []agentllm.ResponseScript{
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
				{Content: "## Goal\nCompact summary"},
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
				{Content: "## Goal\nForce compact summary"},
				{Err: fmt.Errorf("Your input exceeds the context window of this model")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := session.NewMemoryStorage()
			sess := session.New(store)
			long := strings.Repeat("word ", 5000)
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage(long),
			}))
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("recent"),
			}))

			fakeModel := agentllm.NewFakeModel(tt.scripts...)
			ctxMgr := ctxmgr.New(ctxmgr.Options{
				Model:         fakeModel,
				ModelName:     "fake",
				ContextWindow: 128000,
				Settings:      ctxmgr.Settings{Enabled: true, ReserveTokens: 16384, KeepRecentTokens: 2},
				SystemPrompt:  "system",
			})
			h := harness.New(harness.Options{
				AgentOptions:   loop.Options{Model: fakeModel},
				Session:        sess,
				ContextManager: ctxMgr,
				SystemPrompt:   "system",
				ModelName:      "fake",
			})

			_, err := h.Prompt(ctx, msg.NewUserMessage("hello"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))

			result := h.LastRunResult()
			if tt.wantErr {
				assert.Error(t, result.Err)
				return
			}
			require.NoError(t, result.Err)
			reply := ""
			for i := len(result.Messages) - 1; i >= 0; i-- {
				assistant, ok := result.Messages[i].(msg.AssistantMessage)
				if !ok {
					continue
				}
				reply = assistant.TextContent()
				if reply != "" {
					break
				}
			}
			assert.Contains(t, reply, tt.wantSubstr)
		})
	}
}

func TestHarnessPersistsUserBeforeToolApproval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "user visible while tool call blocked"},
		{name: "user not duplicated after turn completes"},
		{name: "user remains when run aborted mid wait"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := session.NewMemoryStorage()
			sess := session.New(store)

			blocked := make(chan struct{})
			release := make(chan struct{})
			regHooks := hooks.NewRegistry()
			hooks.OnToolCall(regHooks, func(context.Context, hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
				close(blocked)
				<-release
				return nil, nil
			})

			fakeModel := agentllm.NewFakeModel(
				agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
				}}},
				agentllm.ResponseScript{Content: "done"},
			)
			toolReg := tool.NewRegistry()
			require.NoError(t, toolReg.Register(echo.Tool{}))

			h := harness.New(harness.Options{
				AgentOptions: loop.Options{
					Model:    fakeModel,
					Registry: toolReg,
					Config:   msg.Config{MaxSteps: 10},
				},
				Session:   sess,
				Hooks:     regHooks,
				ModelName: "fake",
			})

			_, err := h.Prompt(ctx, msg.NewUserMessage("please echo"))
			require.NoError(t, err)

			select {
			case <-blocked:
			case <-time.After(3 * time.Second):
				t.Fatal("timeout waiting for tool approval gate")
			}

			branch, err := sess.GetBranch(ctx, "")
			require.NoError(t, err)
			require.NotEmpty(t, branch, "user must be persisted before tool approval")
			userCount := 0
			for _, entry := range branch {
				um, ok := entry.Message.(msg.UserMessage)
				if !ok {
					continue
				}
				userCount++
				var got strings.Builder
				for _, part := range um.Parts {
					if tp, ok := part.(msg.TextPart); ok {
						got.WriteString(tp.Text)
					}
				}
				assert.Contains(t, got.String(), "please echo")
			}
			require.Equal(t, 1, userCount)

			if tt.name == "user remains when run aborted mid wait" {
				h.Agent().Abort()
				close(release)
				require.NoError(t, h.WaitIdle(ctx))
				branch, err = sess.GetBranch(ctx, "")
				require.NoError(t, err)
				userCount = 0
				for _, entry := range branch {
					if _, ok := entry.Message.(msg.UserMessage); ok {
						userCount++
					}
				}
				assert.Equal(t, 1, userCount)
				return
			}

			close(release)
			require.NoError(t, h.WaitIdle(ctx))
			require.NoError(t, h.LastRunResult().Err)

			branch, err = sess.GetBranch(ctx, "")
			require.NoError(t, err)
			userCount = 0
			for _, entry := range branch {
				if _, ok := entry.Message.(msg.UserMessage); ok {
					userCount++
				}
			}
			assert.Equal(t, 1, userCount, "finishStream must not duplicate early-persisted user")
		})
	}
}

func TestHarnessPersistsToolStepsBetweenApprovals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "no fake completed tool before first approval"},
		{name: "tool result persisted before second approval"},
		{name: "no duplicate messages after final turn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runPersistsToolStepsBetweenApprovals(t)
		})
	}
}

func runPersistsToolStepsBetweenApprovals(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	store := session.NewMemoryStorage()
	sess := session.New(store)

	gate := make(chan struct{})
	release := make(chan struct{})
	approvals := 0
	regHooks := hooks.NewRegistry()
	hooks.OnToolCall(regHooks, func(context.Context, hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
		approvals++
		if approvals == 1 {
			close(gate)
		} else if approvals == 2 {
			assert.True(t, branchHasToolResult(t, sess), "tool result must be persisted before the next approval wait")
		}
		<-release
		return nil, nil
	})

	fakeModel := agentllm.NewFakeModel(
		agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
			ID: "call-1", Type: "function",
			FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"one"}`},
		}}},
		agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
			ID: "call-2", Type: "function",
			FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"two"}`},
		}}},
		agentllm.ResponseScript{Content: "all done"},
	)
	toolReg := tool.NewRegistry()
	require.NoError(t, toolReg.Register(echo.Tool{}))

	h := harness.New(harness.Options{
		AgentOptions: loop.Options{
			Model:    fakeModel,
			Registry: toolReg,
			Config:   msg.Config{MaxSteps: 10},
		},
		Session:   sess,
		Hooks:     regHooks,
		ModelName: "fake",
	})

	_, err := h.Prompt(ctx, msg.NewUserMessage("echo twice"))
	require.NoError(t, err)

	select {
	case <-gate:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for first approval")
	}

	assertNoMidTurnToolPersist(t, sess)

	go func() {
		release <- struct{}{}
		release <- struct{}{}
	}()
	require.NoError(t, h.WaitIdle(ctx))
	require.NoError(t, h.LastRunResult().Err)

	userCount, toolCount, textAssistant := countBranchMessageKinds(t, sess)
	assert.Equal(t, 1, userCount)
	assert.Equal(t, 2, toolCount)
	assert.Equal(t, 1, textAssistant)
	assertBranchToolOrderValidForLLM(t, sess)
}

func branchHasToolResult(t *testing.T, sess *session.Session) bool {
	t.Helper()
	branch, err := sess.GetBranch(context.Background(), "")
	require.NoError(t, err)
	for _, entry := range branch {
		if _, ok := entry.Message.(msg.ToolResultMessage); ok {
			return true
		}
	}
	return false
}

func assertNoMidTurnToolPersist(t *testing.T, sess *session.Session) {
	t.Helper()
	branch, err := sess.GetBranch(context.Background(), "")
	require.NoError(t, err)
	for _, entry := range branch {
		if _, ok := entry.Message.(msg.ToolResultMessage); ok {
			t.Fatal("tool result must not exist before first approval")
		}
		if as, ok := entry.Message.(msg.AssistantMessage); ok && len(as.ToolCalls()) > 0 {
			t.Fatal("tool-call assistant must not be mid-persisted before approval")
		}
	}
}

func countBranchMessageKinds(t *testing.T, sess *session.Session) (userCount, toolCount, textAssistant int) {
	t.Helper()
	branch, err := sess.GetBranch(context.Background(), "")
	require.NoError(t, err)
	for _, entry := range branch {
		switch m := entry.Message.(type) {
		case msg.UserMessage:
			userCount++
		case msg.ToolResultMessage:
			toolCount++
		case msg.AssistantMessage:
			if len(m.ToolCalls()) == 0 && m.TextContent() != "" {
				textAssistant++
			}
		}
	}
	return userCount, toolCount, textAssistant
}

func assertBranchToolOrderValidForLLM(t *testing.T, sess *session.Session) {
	t.Helper()
	branch, err := sess.GetBranch(context.Background(), "")
	require.NoError(t, err)
	built := session.BuildContext(branch)
	open := make([]string, 0)
	for i, message := range built.Messages {
		switch m := message.(type) {
		case msg.AssistantMessage:
			if len(open) > 0 {
				t.Fatalf("message[%d]: assistant while tool_calls still open %v (insufficient tool messages)", i, open)
			}
			for _, call := range m.ToolCalls() {
				if call.ID != "" {
					open = append(open, call.ID)
				}
			}
		case msg.ToolResultMessage:
			if m.ToolCallID == "" {
				continue
			}
			if len(open) == 0 {
				t.Fatalf("message[%d]: tool result %q has no preceding assistant tool_calls", i, m.ToolCallID)
			}
			found := -1
			for j, id := range open {
				if id == m.ToolCallID {
					found = j
					break
				}
			}
			if found < 0 {
				t.Fatalf("message[%d]: tool result %q does not match open tool_calls %v", i, m.ToolCallID, open)
			}
			open = append(open[:found], open[found+1:]...)
		default:
			if len(open) > 0 {
				t.Fatalf("message[%d]: non-tool message while tool_calls still open %v", i, open)
			}
		}
	}
	if len(open) > 0 {
		t.Fatalf("unclosed tool_calls at end of branch: %v", open)
	}
}

func TestHarnessPersistedToolOrderValidAfterReload(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := session.NewMemoryStorage()
	sess := session.New(store)

	fakeModel := agentllm.NewFakeModel(
		agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
			ID: "call-1", Type: "function",
			FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
		}}},
		agentllm.ResponseScript{Content: "done"},
	)
	toolReg := tool.NewRegistry()
	require.NoError(t, toolReg.Register(echo.Tool{}))

	h := harness.New(harness.Options{
		AgentOptions: loop.Options{
			Model:    fakeModel,
			Registry: toolReg,
			Config:   msg.Config{MaxSteps: 10},
		},
		Session:   sess,
		ModelName: "fake",
	})

	_, err := h.Prompt(ctx, msg.NewUserMessage("please echo"))
	require.NoError(t, err)
	require.NoError(t, h.WaitIdle(ctx))
	require.NoError(t, h.LastRunResult().Err)

	assertBranchToolOrderValidForLLM(t, sess)

	// Simulate EvictHarnessPool: rebuild agent context from persisted branch.
	branch, err := sess.GetBranch(ctx, "")
	require.NoError(t, err)
	agentCtx := session.ToAgentContext(session.BuildContext(branch), "system")
	llmMessages, err := transform.DefaultConvertToLLM(agentCtx.Messages)
	require.NoError(t, err)
	openToolCalls := make(map[string]struct{})
	for i, m := range llmMessages {
		switch m.Role {
		case llms.ChatMessageTypeAI:
			for _, part := range m.Parts {
				if tc, ok := part.(llms.ToolCall); ok && tc.ID != "" {
					openToolCalls[tc.ID] = struct{}{}
				}
			}
		case llms.ChatMessageTypeTool:
			for _, part := range m.Parts {
				tr, ok := part.(llms.ToolCallResponse)
				if !ok {
					continue
				}
				if _, ok := openToolCalls[tr.ToolCallID]; !ok {
					t.Fatalf("llm message[%d]: tool response %q missing preceding tool_calls", i, tr.ToolCallID)
				}
				delete(openToolCalls, tr.ToolCallID)
			}
		}
	}
}

func TestHarnessRespectsCompactionDisabledOnOverflow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fakeModel := agentllm.NewFakeModel(agentllm.ResponseScript{
		Err: fmt.Errorf("Your input exceeds the context window of this model"),
	})
	ctxMgr := ctxmgr.New(ctxmgr.Options{
		Model:         fakeModel,
		ModelName:     "fake",
		ContextWindow: 128000,
		Settings:      ctxmgr.Settings{Enabled: false},
	})
	h := harness.New(harness.Options{
		AgentOptions:   loop.Options{Model: fakeModel},
		ContextManager: ctxMgr,
		ModelName:      "fake",
	})
	_, err := h.Prompt(ctx, msg.NewUserMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, h.WaitIdle(ctx))
	require.Error(t, h.LastRunResult().Err)
	assert.Equal(t, 1, fakeModel.Calls())
}

func TestHarnessOverflowRetryAfterPruneOnly(t *testing.T) {
	t.Parallel()

	appendToolTurn := func(ctx context.Context, t *testing.T, sess *session.Session, toolText string) {
		t.Helper()
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "1", Type: session.EntryMessage, Message: msg.NewUserMessage("read the file"),
		}))
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "2", ParentID: "1", Type: session.EntryMessage, Message: msg.AssistantMessage{
				Parts: []msg.ContentPart{msg.ToolCallPart{ID: "c1", Name: "echo", Arguments: `{}`}},
			},
		}))
		require.NoError(t, sess.Append(ctx, session.TreeEntry{
			ID: "3", ParentID: "2", Type: session.EntryMessage, Message: msg.ToolResultMessage{
				ToolCallID: "c1",
				Name:       "echo",
				Parts:      []msg.ContentPart{msg.TextPart{Text: toolText}},
			},
		}))
	}

	tests := []struct {
		name           string
		prune          bool
		toolText       string
		wantCalls      int
		wantCompaction bool
	}{
		{
			name:           "recovers without summarization",
			prune:          true,
			toolText:       strings.Repeat("a", 20000),
			wantCalls:      2,
			wantCompaction: false,
		},
		{
			name:           "prune disabled still summarizes",
			prune:          false,
			toolText:       strings.Repeat("a", 20000),
			wantCalls:      3,
			wantCompaction: true,
		},
		{
			name:           "small tool results still summarize after overflow",
			prune:          true,
			toolText:       "small",
			wantCalls:      3,
			wantCompaction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			store := session.NewMemoryStorage()
			sess := session.New(store)
			appendToolTurn(ctx, t, sess, tt.toolText)

			fakeModel := agentllm.NewFakeModel(
				agentllm.ResponseScript{Err: fmt.Errorf("Your input exceeds the context window of this model")},
				agentllm.ResponseScript{Content: "## Goal\ncompacted"},
				agentllm.ResponseScript{Content: "recovered after overflow"},
			)
			ctxMgr := ctxmgr.New(ctxmgr.Options{
				Model:         fakeModel,
				ModelName:     "fake",
				ContextWindow: 128000,
				Settings: ctxmgr.Settings{
					Enabled:          true,
					PruneToolOutputs: tt.prune,
					ReserveTokens:    16384,
					KeepRecentTokens: 2,
				},
				SystemPrompt: "system",
			})
			h := harness.New(harness.Options{
				AgentOptions:   loop.Options{Model: fakeModel},
				Session:        sess,
				ContextManager: ctxMgr,
				SystemPrompt:   "system",
				ModelName:      "fake",
			})

			_, err := h.Prompt(ctx, msg.NewUserMessage("hello"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			require.NoError(t, h.LastRunResult().Err)
			assert.Equal(t, tt.wantCalls, fakeModel.Calls())
			if !tt.wantCompaction {
				assert.NotContains(t, joinedHarnessText(fakeModel), "compaction engine")
			}

			entries, err := store.ListEntries(ctx)
			require.NoError(t, err)
			hasCompaction := false
			for _, entry := range entries {
				if entry.Type == session.EntryCompaction {
					hasCompaction = true
					break
				}
			}
			assert.Equal(t, tt.wantCompaction, hasCompaction)
		})
	}
}

func joinedHarnessText(fake *agentllm.FakeModel) string {
	var b strings.Builder
	for _, message := range fake.LastMessages() {
		for _, part := range message.Parts {
			if text, ok := part.(llms.TextContent); ok {
				_, _ = b.WriteString(text.Text)
			}
		}
	}
	return b.String()
}

func TestHarnessRouterDualModelRouting(t *testing.T) {
	tests := []struct {
		name      string
		scripts   []agentllm.ResponseScript
		wantModel []string
	}{
		{
			name: "router sync routes after tool execution",
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"hi"}`},
				}}},
				{Content: "done"},
			},
			wantModel: []string{"chat-model", "tool-model"},
		},
		{
			name: "router keeps tool model on chained tool rounds",
			scripts: []agentllm.ResponseScript{
				{ToolCalls: []llms.ToolCall{{
					ID: "call-1", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"one"}`},
				}}},
				{ToolCalls: []llms.ToolCall{{
					ID: "call-2", Type: "function",
					FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"two"}`},
				}}},
				{Content: "done"},
			},
			wantModel: []string{"chat-model", "tool-model", "tool-model"},
		},
		{
			name:      "router without tools stays on chat model",
			scripts:   []agentllm.ResponseScript{{Content: "ok"}},
			wantModel: []string{"chat-model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fakeModel := agentllm.NewFakeModel(tt.scripts...)
			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(echo.Tool{}))

			h := harness.New(harness.Options{
				AgentOptions: loop.Options{
					Model:    fakeModel,
					Registry: reg,
					Config:   msg.Config{MaxSteps: 10},
				},
				Router:    model.NewRouter("chat-model", "tool-model"),
				ModelName: "chat-model",
			})

			_, err := h.Prompt(ctx, msg.NewUserMessage("run"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			require.NoError(t, h.LastRunResult().Err)

			got := make([]string, 0)
			for _, item := range h.LastRunResult().Messages {
				assistant, ok := item.(msg.AssistantMessage)
				if !ok {
					continue
				}
				got = append(got, assistant.Model)
			}
			assert.Equal(t, tt.wantModel, got)
		})
	}
}
