// Package gateway provides the chatagent tool that delegates to CapGateway (local CLI workers).
package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/capability"
	capgw "github.com/flowline-io/flowbot/pkg/capability/gateway"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RunCursorToolName is the agent tool name for local Cursor CLI delegation.
const RunCursorToolName = "run_cursor"

// RunCursorTool submits a CapGateway run job and returns the terminal result.
type RunCursorTool struct {
	// UID is recorded on the gateway job for audit.
	UID string
}

// Name returns the tool identifier.
func (RunCursorTool) Name() string { return RunCursorToolName }

// Description explains the tool to the model.
func (RunCursorTool) Description() string {
	return "Delegate a coding task to the local Cursor CLI via flowbot-gateway (coarse prompt in, result out). Omit cwd to use the worker default workspace."
}

// Parameters returns the JSON schema for tool arguments.
func (RunCursorTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Task prompt for the local Cursor agent",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Optional absolute workspace path on the worker machine (must be allowlisted)",
			},
		},
		"required": []string{"prompt"},
	}
}

// Execute invokes CapGateway run and formats the job result for the model.
func (t RunCursorTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	if !config.App.Gateway.Enabled {
		return tool.ErrorResult(id, t.Name(), "unavailable", "gateway capability is disabled", "set gateway.enabled=true and run cmd/gateway"), nil
	}
	prompt := strings.TrimSpace(fmt.Sprint(args["prompt"]))
	if prompt == "" || prompt == "<nil>" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "prompt is required", "pass the task prompt"), nil
	}
	params := map[string]any{
		"prompt": prompt,
		"cli":    string(types.GatewayCLICursor),
	}
	if t.UID != "" {
		params["uid"] = t.UID
	}
	if raw, ok := args["cwd"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(raw)); s != "" && s != "<nil>" {
			params["cwd"] = s
		}
	}

	res, err := capability.Invoke(ctx, hub.CapGateway, capgw.OpRun, params)
	if err != nil {
		return invokeErrorResult(id, t.Name(), err), nil
	}
	job, ok := res.Data.(*types.GatewayJob)
	if !ok || job == nil {
		return tool.ErrorResult(id, t.Name(), "tool_error", "unexpected gateway response", "retry later"), nil
	}
	text := formatJob(job)
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    job.Status != types.GatewayJobSucceeded,
	}, nil
}

// Register registers run_cursor when CapGateway is enabled.
func Register(registry *tool.Registry, uid string) error {
	if registry == nil {
		return fmt.Errorf("gateway tools: registry is nil")
	}
	if !config.App.Gateway.Enabled {
		return nil
	}
	return registry.Register(RunCursorTool{UID: uid})
}

// ActiveToolNames returns gateway tool names when enabled.
func ActiveToolNames() []string {
	if !config.App.Gateway.Enabled {
		return nil
	}
	return []string{RunCursorToolName}
}

func formatJob(job *types.GatewayJob) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "status: %s\njob_id: %s\ncli: %s\ncwd: %s\nduration_ms: %d\n",
		job.Status, job.JobID, job.CLI, job.Cwd, job.DurationMs)
	if job.ExitCode != nil {
		_, _ = fmt.Fprintf(&b, "exit_code: %d\n", *job.ExitCode)
	}
	if job.Error != "" {
		_, _ = fmt.Fprintf(&b, "error: %s\n", job.Error)
	}
	if job.Output != "" {
		_, _ = fmt.Fprintf(&b, "\n%s", job.Output)
	}
	return b.String()
}

func invokeErrorResult(callID, name string, err error) msg.ToolResultMessage {
	code := "tool_error"
	hint := "ensure cmd/gateway is online with a gateway:worker token"
	switch {
	case errors.Is(err, types.ErrUnavailable):
		code = "unavailable"
		hint = "start flowbot-gateway and confirm heartbeat"
	case errors.Is(err, types.ErrTimeout):
		code = "timeout"
		hint = "increase gateway.run_timeout or simplify the prompt"
	case errors.Is(err, types.ErrInvalidArgument):
		code = "invalid_args"
		hint = "fix the tool arguments"
	case errors.Is(err, context.Canceled):
		code = "canceled"
		hint = "the run was canceled"
	}
	return tool.ErrorResult(callID, name, code, err.Error(), hint)
}
