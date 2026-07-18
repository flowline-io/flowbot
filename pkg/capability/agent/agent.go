// Package agent provides the agent capability for pipeline steps to run chat agent prompts.
package agent

import (
	"context"
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RunParams carries one pipeline-rendered agent run request.
type RunParams struct {
	Prompt      string
	UID         types.Uid
	Tools       []string
	Skills      []string
	MemoryScope string
}

// RunResult holds the outcome of one agent.run invocation.
type RunResult struct {
	Reply     string
	SessionID string
}

// Runner executes one agent prompt from a pipeline step.
type Runner interface {
	Run(ctx context.Context, params RunParams) (*RunResult, error)
}

var (
	runnerMu sync.RWMutex
	runner   Runner
)

// SetRunner wires the product-layer agent runner used by pipeline steps.
func SetRunner(r Runner) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	runner = r
}

func getRunner() Runner {
	runnerMu.RLock()
	defer runnerMu.RUnlock()
	return runner
}

// serviceMarker is a non-nil instance used for hub registration when the agent is enabled.
type serviceMarker struct{}

// Register registers the agent capability with hub and invoker registry.
// Registration runs when chat agent is enabled or a Runner has been wired
// (including test stubs via SetRunner).
func Register() error {
	var instance any
	if config.ChatAgentEnabled() || getRunner() != nil {
		instance = serviceMarker{}
	}
	return capability.Register(capability.Spec{
		Type:        hub.CapAgent,
		Description: "Run the chat agent with a rendered prompt",
		Instance:    instance,
		Ops: []capability.OpDef{
			{
				Name: OpRun, Description: "Execute one autonomous agent turn with a prompt", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "prompt", Type: "string", Required: true, Description: "User prompt (supports pipeline template rendering)"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID for permissions; use {{.Event.uid}} in YAML when available"},
					{Name: "tools", Type: "[]string", Required: false, Description: "Tool allowlist; omit or leave empty for pipeline defaults"},
					{Name: "skills", Type: "[]string", Required: false, Description: "Skill allowlist by name; omit or leave empty for all enabled skills"},
					{Name: "memory_scope", Type: "string", Required: false, Description: "Memory scope for update_memory; defaults to pipeline name when omitted"},
				},
				Handler: runInvoker,
			},
			{
				Name: OpHealth, Description: "Health check",
				Handler: healthInvoker,
			},
		},
	})
}

func runInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	prompt, err := capability.RequiredString(params, "prompt")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "prompt is required")
	}

	var uid types.Uid
	if raw, ok := params["uid"]; ok {
		switch v := raw.(type) {
		case string:
			uid = types.Uid(v)
		case types.Uid:
			uid = v
		}
	}

	tools, err := optionalStringListParam(params, "tools")
	if err != nil {
		return nil, err
	}
	skills, err := optionalStringListParam(params, "skills")
	if err != nil {
		return nil, err
	}
	memoryScope, _ := capability.StringParam(params, "memory_scope")

	r := getRunner()
	if r == nil {
		return nil, types.Errorf(types.ErrUnavailable, "agent pipeline runner is not configured")
	}

	result, err := r.Run(ctx, RunParams{
		Prompt:      prompt,
		UID:         uid,
		Tools:       tools,
		Skills:      skills,
		MemoryScope: memoryScope,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &capability.InvokeResult{Data: map[string]any{"reply": ""}}, nil
	}
	return &capability.InvokeResult{
		Data: map[string]any{
			"reply":      result.Reply,
			"session_id": result.SessionID,
		},
		Text: result.Reply,
	}, nil
}

func healthInvoker(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	if config.ChatAgentEnabled() || getRunner() != nil {
		return &capability.InvokeResult{Data: true}, nil
	}
	return nil, types.Errorf(types.ErrUnavailable, "agent is not configured")
}
