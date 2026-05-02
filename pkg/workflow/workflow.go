package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/executor"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	capabilityruntime "github.com/flowline-io/flowbot/pkg/executor/runtime/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
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
		task.Run = wt.Action
		if len(wt.Params) > 0 {
			paramsJSON, err := json.Marshal(wt.Params)
			if err != nil {
				return nil, fmt.Errorf("marshal params: %w", err)
			}
			task.Env["CAPABILITY_PARAMS"] = string(paramsJSON)
		}
		return task, nil
	}

	switch info.Type {
	case "docker":
		task.Image = info.Details
		if len(wt.Params) > 0 {
			if cmd, ok := wt.Params["cmd"]; ok {
				switch v := cmd.(type) {
				case string:
					task.CMD = []string{v}
				case []any:
					for _, item := range v {
						if s, ok := item.(string); ok {
							task.CMD = append(task.CMD, s)
						}
					}
				}
			}
		}
	case "shell":
		task.Run = info.Details
		if len(wt.Params) > 0 {
			if cmd, ok := wt.Params["cmd"]; ok {
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

	return task, nil
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

		params := resolveParams(wt.Params, results)

		wtWithParams := wt
		wtWithParams.Params = params

		task, err := WorkflowTaskToTask(wtWithParams)
		if err != nil {
			return fmt.Errorf("convert task %s: %w", stepID, err)
		}

		flog.Info("[workflow] running step %s: %s", stepID, wt.Action)

		if err := r.Run(ctx, task); err != nil {
			return fmt.Errorf("step %s failed: %w", stepID, err)
		}

		if task.Result != "" {
			results[stepID] = task.Result
		}

		flog.Info("[workflow] step %s completed", stepID)
	}

	return nil
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

func resolveParams(params types.KV, results map[string]string) types.KV {
	resolved := make(types.KV, len(params))
	for k, v := range params {
		if s, ok := v.(string); ok && strings.Contains(s, "{{") {
			for stepID, result := range results {
				s = strings.ReplaceAll(s, "{{"+stepID+".id}}", result)
				if result != "" {
					s = strings.ReplaceAll(s, "{{"+stepID+".result}}", result)
				}
			}
			resolved[k] = s
		} else {
			resolved[k] = v
		}
	}
	return resolved
}
