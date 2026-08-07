// Package run executes one headless agent turn.
package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/subagent"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// Options configures a headless run.
type Options struct {
	Config    *config.Config
	Workspace string
	Prompt    string
	Force     bool
	Timeout   time.Duration
}

// Result is the outcome of a headless run.
type Result struct {
	Text string
}

// Execute runs one print-mode agent turn and returns final assistant text.
func Execute(ctx context.Context, opts Options) (Result, error) {
	if opts.Config == nil {
		return Result{}, fmt.Errorf("config is required")
	}
	if err := opts.Config.Validate(); err != nil {
		return Result{}, err
	}
	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		return Result{}, fmt.Errorf("prompt is required")
	}
	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		return Result{}, fmt.Errorf("workspace is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Minute
	}

	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	model, err := newProxyModel(opts.Config)
	if err != nil {
		return Result{}, err
	}

	ws := coding.Workspace{
		Root:      workspace,
		Timeout:   5 * time.Minute,
		MaxOutput: 8192,
	}
	registry := tool.NewRegistry()
	active, err := coding.RegisterHeadless(registry, ws, env.Default(), coding.HeadlessOptions{Force: opts.Force})
	if err != nil {
		return Result{}, fmt.Errorf("register tools: %w", err)
	}
	registry.SetActive(active)

	hookReg := hooks.NewRegistry()
	registerDCGHook(hookReg)

	h := harness.New(harness.Options{
		AgentOptions: loop.Options{
			Model:    model,
			Registry: registry,
			Config: msg.Config{
				MaxSteps: 50,
			},
		},
		Hooks:        hookReg,
		SystemPrompt: headlessSystemPrompt(workspace, opts.Force),
		ModelName:    "flowbot",
	})

	flog.Info("flowbot-agent run start workspace=%s force=%v prompt_len=%d timeout=%s",
		workspace, opts.Force, len(prompt), opts.Timeout)
	start := time.Now()

	_, err = h.Prompt(runCtx, msg.NewUserMessage(prompt))
	if err != nil {
		return Result{}, fmt.Errorf("prompt: %w", err)
	}
	if err := h.WaitIdle(runCtx); err != nil {
		return Result{}, fmt.Errorf("wait idle: %w", err)
	}
	last := h.LastRunResult()
	text := finalText(last)
	if last.Err != nil {
		flog.Warn("flowbot-agent run failed duration=%s text_len=%d err=%v",
			time.Since(start).Round(time.Millisecond), len(text), last.Err)
		return Result{Text: text}, last.Err
	}
	flog.Info("flowbot-agent run complete duration=%s text_len=%d",
		time.Since(start).Round(time.Millisecond), len(text))
	return Result{Text: text}, nil
}

func newProxyModel(cfg *config.Config) (llms.Model, error) {
	model, err := openai.New(
		openai.WithToken(cfg.AccessToken),
		openai.WithModel("flowbot"),
		openai.WithBaseURL(cfg.LLMBaseURL()),
	)
	if err != nil {
		return nil, fmt.Errorf("openai client: %w", err)
	}
	return model, nil
}

func headlessSystemPrompt(workspace string, force bool) string {
	mode := "read-only (do not modify files)"
	if force {
		mode = "you may modify files and run terminal commands inside the workspace"
	}
	return fmt.Sprintf(`You are flowbot-agent, a local headless coding agent.
Workspace root: %s
Mode: %s
Prefer using tools to inspect and change the workspace. Keep the final answer concise.`, workspace, mode)
}

func registerDCGHook(reg *hooks.Registry) {
	hooks.OnToolCall(reg, func(ctx context.Context, event hooks.ToolCallEvent) (*hooks.ToolCallResult, error) {
		verdict, err := dcg.EvaluateToolCall(ctx, event.ToolCall.Name, event.Args, nil)
		if err != nil {
			flog.Warn("dcg evaluate failed tool=%s: %v", event.ToolCall.Name, err)
			return &hooks.ToolCallResult{Block: true, Reason: err.Error()}, nil
		}
		if verdict.Skip {
			return nil, nil
		}
		if verdict.Block {
			flog.Info("dcg blocked tool=%s reason=%s", event.ToolCall.Name, verdict.Reason)
			return &hooks.ToolCallResult{Block: true, Reason: verdict.Reason}, nil
		}
		return nil, nil
	})
}

func finalText(result agentevent.Result) string {
	msgs := make([]msg.AgentMessage, 0, len(result.Messages))
	for _, item := range result.Messages {
		if m, ok := item.(msg.AgentMessage); ok {
			msgs = append(msgs, m)
		}
	}
	return subagent.FinalText(msgs)
}
