package harness_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
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
				ID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage(long),
			}))
			require.NoError(t, sess.Append(ctx, session.TreeEntry{
				ID: "2", ParentID: "1", Type: session.EntryMessage, Message: agent.NewUserMessage("recent"),
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
				AgentOptions:   agent.Options{Model: fakeModel},
				Session:        sess,
				ContextManager: ctxMgr,
				SystemPrompt:   "system",
				ModelName:      "fake",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
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
				assistant, ok := result.Messages[i].(agent.AssistantMessage)
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
				AgentOptions: agent.Options{
					Model:    fakeModel,
					Registry: toolReg,
					Config:   agent.Config{MaxSteps: 10},
				},
				Session:   sess,
				Hooks:     regHooks,
				ModelName: "fake",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("please echo"))
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
				um, ok := entry.Message.(agent.UserMessage)
				if !ok {
					continue
				}
				userCount++
				got := ""
				for _, part := range um.Parts {
					if tp, ok := part.(agent.TextPart); ok {
						got += tp.Text
					}
				}
				assert.Contains(t, got, "please echo")
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
					if _, ok := entry.Message.(agent.UserMessage); ok {
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
				if _, ok := entry.Message.(agent.UserMessage); ok {
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
		AgentOptions: agent.Options{
			Model:    fakeModel,
			Registry: toolReg,
			Config:   agent.Config{MaxSteps: 10},
		},
		Session:   sess,
		Hooks:     regHooks,
		ModelName: "fake",
	})

	_, err := h.Prompt(ctx, agent.NewUserMessage("echo twice"))
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
}

func branchHasToolResult(t *testing.T, sess *session.Session) bool {
	t.Helper()
	branch, err := sess.GetBranch(context.Background(), "")
	require.NoError(t, err)
	for _, entry := range branch {
		if _, ok := entry.Message.(agent.ToolResultMessage); ok {
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
		if _, ok := entry.Message.(agent.ToolResultMessage); ok {
			t.Fatal("tool result must not exist before first approval")
		}
		if as, ok := entry.Message.(agent.AssistantMessage); ok && len(as.ToolCalls()) > 0 {
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
		case agent.UserMessage:
			userCount++
		case agent.ToolResultMessage:
			toolCount++
		case agent.AssistantMessage:
			if len(m.ToolCalls()) == 0 && m.TextContent() != "" {
				textAssistant++
			}
		}
	}
	return userCount, toolCount, textAssistant
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
		AgentOptions:   agent.Options{Model: fakeModel},
		ContextManager: ctxMgr,
		ModelName:      "fake",
	})
	_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, h.WaitIdle(ctx))
	require.Error(t, h.LastRunResult().Err)
	assert.Equal(t, 1, fakeModel.Calls())
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
				AgentOptions: agent.Options{
					Model:    fakeModel,
					Registry: reg,
					Config:   agent.Config{MaxSteps: 10},
				},
				Router:    model.NewRouter("chat-model", "tool-model"),
				ModelName: "chat-model",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("run"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(ctx))
			require.NoError(t, h.LastRunResult().Err)

			got := make([]string, 0)
			for _, item := range h.LastRunResult().Messages {
				assistant, ok := item.(agent.AssistantMessage)
				if !ok {
					continue
				}
				got = append(got, assistant.Model)
			}
			assert.Equal(t, tt.wantModel, got)
		})
	}
}
