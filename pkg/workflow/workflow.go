package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/flowline-io/flowbot/pkg/executor"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	capabilityruntime "github.com/flowline-io/flowbot/pkg/executor/runtime/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/pipeline/template"
	"github.com/flowline-io/flowbot/pkg/types"
)

type ActionInfo struct {
	Type         string
	Details      string
	IsCapability bool
	CapType      string
	Operation    string
}

func ParseAction(action string) ActionInfo {
	info := ActionInfo{}

	if strings.HasPrefix(action, capabilityruntime.Prefix) {
		info.IsCapability = true
		rest := strings.TrimPrefix(action, capabilityruntime.Prefix)
		dot := strings.LastIndex(rest, ".")
		if dot < 0 {
			info.Type = "capability"
			info.Details = rest
			return info
		}
		info.Type = "capability"
		info.CapType = rest[:dot]
		info.Operation = rest[dot+1:]
		info.Details = rest
		return info
	}

	parts := strings.SplitN(action, ":", 2)
	info.Type = parts[0]
	if len(parts) > 1 {
		info.Details = parts[1]
	}
	return info
}

func WorkflowTaskToTask(wt types.WorkflowTask) (*types.Task, error) {
	info := ParseAction(wt.Action)

	task := &types.Task{
		Run: wt.Action,
		Env: make(map[string]string),
	}

	if info.IsCapability {
		if err := marshalCapabilityParams(task, wt.Params); err != nil {
			return nil, err
		}
		return task, nil
	}

	applyActionParams(task, info, wt.Params)
	return task, nil
}

func marshalCapabilityParams(task *types.Task, params types.KV) error {
	if len(params) == 0 {
		return nil
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	task.Env["CAPABILITY_PARAMS"] = string(paramsJSON)
	return nil
}

func applyActionParams(task *types.Task, info ActionInfo, params types.KV) {
	switch info.Type {
	case "docker":
		task.Image = info.Details
		if len(params) > 0 {
			if cmd, ok := params["cmd"]; ok {
				task.CMD = extractCMDSlice(cmd)
			}
		}
	case "shell":
		task.Run = info.Details
		if len(params) > 0 {
			if cmd, ok := params["cmd"]; ok {
				if s, ok := cmd.(string); ok {
					task.Run = s
				}
			}
		}
	case "machine":
		task.Run = info.Details
	default:
		if info.Details != "" {
			task.Run = info.Details
		}
	}
}

func extractCMDSlice(cmd any) []string {
	switch v := cmd.(type) {
	case string:
		return []string{v}
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func DetermineRuntimeType(t *types.Task) string {
	if strings.HasPrefix(t.Run, capabilityruntime.Prefix) {
		return runtime.Capability
	}
	if t.Image != "" {
		return runtime.Docker
	}
	return runtime.Shell
}

type Runner struct {
	engines map[string]*executor.Engine
}

func NewRunner() *Runner {
	return &Runner{
		engines: map[string]*executor.Engine{
			runtime.Capability: executor.New(runtime.Capability),
			runtime.Shell:      executor.New(runtime.Shell),
			runtime.Docker:     executor.New(runtime.Docker),
			runtime.Machine:    executor.New(runtime.Machine),
		},
	}
}

func (r *Runner) Run(ctx context.Context, t *types.Task) error {
	rt := DetermineRuntimeType(t)
	eng, ok := r.engines[rt]
	if !ok {
		return fmt.Errorf("unknown runtime type for task: %s", t.Run)
	}
	return eng.Run(ctx, t)
}

func (r *Runner) Execute(ctx context.Context, wf types.WorkflowMetadata) error {
	taskMap := make(map[string]types.WorkflowTask)
	for _, wt := range wf.Tasks {
		taskMap[wt.ID] = wt
	}

	results := make(map[string]string)

	for _, stepID := range wf.Pipeline {
		wt, ok := taskMap[stepID]
		if !ok {
			return fmt.Errorf("task %s not found in workflow", stepID)
		}

		params, err := resolveParams(wt.Params, results)
		if err != nil {
			return fmt.Errorf("resolve params step %s: %w", stepID, err)
		}

		// Mapper steps are handled inline: they serialize resolved
		// params to JSON and store as the step result for downstream
		// steps to consume. No external runtime is needed.
		info := ParseAction(wt.Action)
		if info.Type == "mapper" {
			mappedJSON, err := json.Marshal(map[string]any(params))
			if err != nil {
				return fmt.Errorf("mapper step %s: %w", stepID, err)
			}
			results[stepID] = string(mappedJSON)
			flog.Info("[workflow] mapper step %s completed", stepID)
			continue
		}

		wtWithParams := wt
		wtWithParams.Params = params

		task, err := WorkflowTaskToTask(wtWithParams)
		if err != nil {
			return fmt.Errorf("convert task %s: %w", stepID, err)
		}

		flog.Info("[workflow] running step %s: %s", stepID, wt.Action)

		if err := r.runWithRetry(ctx, task, wt.Retry, stepID); err != nil {
			return fmt.Errorf("step %s failed: %w", stepID, err)
		}

		if task.Result != "" {
			results[stepID] = task.Result
		}

		flog.Info("[workflow] step %s completed", stepID)
	}

	return nil
}

func (r *Runner) runWithRetry(ctx context.Context, task *types.Task, retryCfg *types.RetryConfig, stepID string) error {
	bo := retryCfg.BuildBackOff()

	attempt := 0
	for {
		attempt++
		err := r.Run(ctx, task)
		if err == nil {
			return nil
		}

		if !retryCfg.RetryEnabled() {
			return err
		}

		nextDelay := bo.NextBackOff()
		if nextDelay == backoff.Stop {
			return fmt.Errorf("step %s (retries exhausted, attempt %d): %w", stepID, attempt, err)
		}

		flog.Info("[workflow] step %s attempt %d failed, retrying in %v: %v", stepID, attempt, nextDelay, err)

		select {
		case <-ctx.Done():
			return fmt.Errorf("step %s cancelled: %w", stepID, ctx.Err())
		case <-time.After(nextDelay):
		}
	}
}

func ValidateDAG(tasks []types.WorkflowTask) error {
	adj := make(map[string][]string)
	for _, t := range tasks {
		adj[t.ID] = t.Conn
	}

	taskIDs := make(map[string]bool)
	for _, t := range tasks {
		taskIDs[t.ID] = true
	}

	for _, t := range tasks {
		for _, dep := range t.Conn {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s references unknown dependency %s", t.ID, dep)
			}
		}
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(id string) error
	dfs = func(id string) error {
		if inStack[id] {
			return fmt.Errorf("cycle detected: task %s", id)
		}
		if visited[id] {
			return nil
		}
		inStack[id] = true
		for _, dep := range adj[id] {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		inStack[id] = false
		visited[id] = true
		return nil
	}

	for _, t := range tasks {
		if !visited[t.ID] {
			if err := dfs(t.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

var workflowEngine = template.New()

func resolveParams(params types.KV, results map[string]string) (types.KV, error) {
	steps := make(map[string]map[string]any, len(results))
	for stepID, result := range results {
		steps[stepID] = map[string]any{
			"id":     result,
			"result": result,
		}
	}

	data := &template.TemplateData{
		Steps: steps,
	}

	rendered, err := workflowEngine.Render(map[string]any(params), data)
	if err != nil {
		return nil, err
	}

	return types.KV(rendered), nil
}
