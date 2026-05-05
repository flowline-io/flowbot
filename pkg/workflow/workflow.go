package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/flowline-io/flowbot/internal/store/model"
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
	engines      map[string]*executor.Engine
	store        WorkflowRunStore
	workflowFile string
	triggerType  string
}

// NewRunner creates a Runner without persistence. Use NewRunnerWithStore to enable run records.
func NewRunner() *Runner {
	return NewRunnerWithStore(nil, "", "")
}

// NewRunnerWithStore creates a Runner that persists run and step records to the given store.
// workflowFile and triggerType are recorded in the run for audit and potential resume.
func NewRunnerWithStore(store WorkflowRunStore, workflowFile, triggerType string) *Runner {
	return &Runner{
		engines: map[string]*executor.Engine{
			runtime.Capability: executor.New(runtime.Capability),
			runtime.Shell:      executor.New(runtime.Shell),
			runtime.Docker:     executor.New(runtime.Docker),
			runtime.Machine:    executor.New(runtime.Machine),
		},
		store:        store,
		workflowFile: workflowFile,
		triggerType:  triggerType,
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

func (r *Runner) Execute(ctx context.Context, wf types.WorkflowMetadata, input types.KV, file string) error {
	taskMap := make(map[string]types.WorkflowTask)
	for _, wt := range wf.Tasks {
		taskMap[wt.ID] = wt
	}

	results := make(map[string]string)

	// Persist run record if store is available.
	var run *model.WorkflowRun
	if r.store != nil {
		workflowFile := file
		if workflowFile == "" {
			workflowFile = r.workflowFile
		}
		triggerType := r.triggerType
		if triggerType == "" {
			triggerType = "manual"
		}
		inputJSON := model.JSON{}
		if len(input) > 0 {
			raw, _ := json.Marshal(input)
			_ = inputJSON.Scan(raw)
		}
		var err error
		run, err = r.store.CreateRun(wf.Name, workflowFile, triggerType, nil, inputJSON)
		if err != nil {
			flog.Error(fmt.Errorf("[workflow] create run record: %w", err))
		}
	}

	// Start heartbeat goroutine if resumable and store is available.
	var cancelHeartbeat context.CancelFunc
	if wf.Resumable && r.store != nil && run != nil {
		var hbCtx context.Context
		hbCtx, cancelHeartbeat = context.WithCancel(ctx)
		go r.heartbeat(hbCtx, run.ID)
	}

	stepIndex := 0
	for _, stepID := range wf.Pipeline {
		wt, ok := taskMap[stepID]
		if !ok {
			err := fmt.Errorf("task %s not found in workflow", stepID)
			r.failRun(run, cancelHeartbeat, err)
			return err
		}

		// Save checkpoint before step execution.
		if wf.Resumable && r.store != nil && run != nil {
			cp := CheckpointData{
				StepIndex:   stepIndex,
				StepResults: resultCopy(results),
				Input:       input,
				HeartbeatAt: time.Now(),
			}
			if cerr := r.store.SaveCheckpoint(run.ID, &cp); cerr != nil {
				flog.Error(fmt.Errorf("[workflow] save checkpoint step %s: %w", stepID, cerr))
			}
		}

		params, err := resolveParams(wt.Params, results, input)
		if err != nil {
			err = fmt.Errorf("resolve params step %s: %w", stepID, err)
			r.failRun(run, cancelHeartbeat, err)
			return err
		}

		info := ParseAction(wt.Action)

		// Create step run record.
		var stepRun *model.WorkflowStepRun
		if r.store != nil && run != nil {
			stepRun, err = r.store.CreateStepRun(run.ID, stepID, wt.Describe, wt.Action, info.Type, model.JSON(params), 1)
			if err != nil {
				flog.Error(fmt.Errorf("[workflow] create step run record %s: %w", stepID, err))
			}
		}

		// Mapper steps are handled inline.
		if info.Type == "mapper" {
			mappedJSON, merr := json.Marshal(map[string]any(params))
			if merr != nil {
				merr = fmt.Errorf("mapper step %s: %w", stepID, merr)
				r.failStep(stepRun, merr, 1)
				r.failRun(run, cancelHeartbeat, merr)
				return merr
			}
			results[stepID] = string(mappedJSON)
			if r.store != nil && stepRun != nil {
				resultJSON := model.JSON{}
				_ = resultJSON.Scan(mappedJSON)
				_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", 1)
			}
			flog.Info("[workflow] mapper step %s completed", stepID)
			stepIndex++
			continue
		}

		wtWithParams := wt
		wtWithParams.Params = params

		task, err := WorkflowTaskToTask(wtWithParams)
		if err != nil {
			err = fmt.Errorf("convert task %s: %w", stepID, err)
			r.failStep(stepRun, err, 1)
			r.failRun(run, cancelHeartbeat, err)
			return err
		}

		flog.Info("[workflow] running step %s: %s", stepID, wt.Action)

		attempt, rerr := r.runWithRetry(ctx, task, wt.Retry, stepID, stepRun)
		if rerr != nil {
			r.failStep(stepRun, rerr, attempt)
			r.failRun(run, cancelHeartbeat, rerr)
			return fmt.Errorf("step %s failed: %w", stepID, rerr)
		}

		if task.Result != "" {
			results[stepID] = task.Result
		}

		// Update step run to done.
		if r.store != nil && stepRun != nil {
			resultJSON := model.JSON{}
			if task.Result != "" {
				resultRaw, _ := json.Marshal(map[string]any{"result": task.Result})
				_ = resultJSON.Scan(resultRaw)
			}
			_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", attempt)
		}

		flog.Info("[workflow] step %s completed", stepID)
		stepIndex++
	}

	// Mark run as done.
	if r.store != nil && run != nil {
		if cancelHeartbeat != nil {
			cancelHeartbeat()
		}
		_ = r.store.UpdateRunStatus(run.ID, model.WorkflowRunDone, "")
	}

	return nil
}

// ResumeWorkflow resumes a previously failed or incomplete workflow run from its checkpoint.
func (r *Runner) ResumeWorkflow(runID int64) error {
	if r.store == nil {
		return fmt.Errorf("cannot resume workflow without a store")
	}

	run, err := r.store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("get run %d: %w", runID, err)
	}

	if run.Status != model.WorkflowRunRunning && run.Status != model.WorkflowRunFailed {
		return fmt.Errorf("workflow run %d status is %d, not resumable", runID, run.Status)
	}

	wf, err := LoadFile(run.WorkflowFile)
	if err != nil {
		return fmt.Errorf("load workflow file %s: %w", run.WorkflowFile, err)
	}

	var cp CheckpointData
	if err := r.store.GetCheckpoint(runID, &cp); err != nil {
		return fmt.Errorf("get checkpoint for run %d: %w", runID, err)
	}

	taskMap := make(map[string]types.WorkflowTask)
	for _, wt := range wf.Tasks {
		taskMap[wt.ID] = wt
	}

	results := resultCopy(cp.StepResults)
	input := cp.Input

	// Start heartbeat goroutine if resumable.
	var cancelHeartbeat context.CancelFunc
	if wf.Resumable {
		var hbCtx context.Context
		hbCtx, cancelHeartbeat = context.WithCancel(context.Background())
		go r.heartbeat(hbCtx, runID)
	}

	// Skip already-completed steps and resume from checkpoint.
	for i := cp.StepIndex + 1; i < len(wf.Pipeline); i++ {
		stepID := wf.Pipeline[i]
		wt, ok := taskMap[stepID]
		if !ok {
			err := fmt.Errorf("task %s not found in workflow", stepID)
			r.failRun(run, cancelHeartbeat, err)
			return err
		}

		if wf.Resumable {
			cpData := CheckpointData{
				StepIndex:   i,
				StepResults: resultCopy(results),
				Input:       input,
				HeartbeatAt: time.Now(),
			}
			if cerr := r.store.SaveCheckpoint(runID, &cpData); cerr != nil {
				flog.Error(fmt.Errorf("[workflow] resume save checkpoint step %s: %w", stepID, cerr))
			}
		}

		params, err := resolveParams(wt.Params, results, input)
		if err != nil {
			err = fmt.Errorf("resolve params step %s: %w", stepID, err)
			r.failRun(run, cancelHeartbeat, err)
			return err
		}

		info := ParseAction(wt.Action)

		var stepRun *model.WorkflowStepRun
		stepRun, err = r.store.CreateStepRun(runID, stepID, wt.Describe, wt.Action, info.Type, model.JSON(params), 1)
		if err != nil {
			flog.Error(fmt.Errorf("[workflow] resume create step run %s: %w", stepID, err))
		}

		if info.Type == "mapper" {
			mappedJSON, merr := json.Marshal(map[string]any(params))
			if merr != nil {
				merr = fmt.Errorf("resume mapper step %s: %w", stepID, merr)
				r.failStep(stepRun, merr, 1)
				r.failRun(run, cancelHeartbeat, merr)
				return merr
			}
			results[stepID] = string(mappedJSON)
			if stepRun != nil {
				resultJSON := model.JSON{}
				_ = resultJSON.Scan(mappedJSON)
				_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", 1)
			}
			continue
		}

		wtWithParams := wt
		wtWithParams.Params = params

		task, err := WorkflowTaskToTask(wtWithParams)
		if err != nil {
			err = fmt.Errorf("convert task %s: %w", stepID, err)
			r.failStep(stepRun, err, 1)
			r.failRun(run, cancelHeartbeat, err)
			return err
		}

		ctx := context.Background()
		attempt, rerr := r.runWithRetry(ctx, task, wt.Retry, stepID, stepRun)
		if rerr != nil {
			r.failStep(stepRun, rerr, attempt)
			r.failRun(run, cancelHeartbeat, rerr)
			return fmt.Errorf("step %s failed: %w", stepID, rerr)
		}

		if task.Result != "" {
			results[stepID] = task.Result
		}

		if stepRun != nil {
			resultJSON := model.JSON{}
			if task.Result != "" {
				resultRaw, _ := json.Marshal(map[string]any{"result": task.Result})
				_ = resultJSON.Scan(resultRaw)
			}
			_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", attempt)
		}
	}

	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}
	_ = r.store.UpdateRunStatus(runID, model.WorkflowRunDone, "")
	return nil
}

// failRun marks a workflow run as failed if run is non-nil and cancels the heartbeat.
func (r *Runner) failRun(run *model.WorkflowRun, cancelHeartbeat context.CancelFunc, err error) {
	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}
	if r.store != nil && run != nil {
		_ = r.store.UpdateRunStatus(run.ID, model.WorkflowRunFailed, err.Error())
	}
}

// failStep marks a step run as failed if stepRun is non-nil.
func (r *Runner) failStep(stepRun *model.WorkflowStepRun, err error, attempt int) {
	if r.store != nil && stepRun != nil {
		_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunFailed, nil, err.Error(), attempt)
	}
}

// heartbeat periodically updates last_heartbeat for a resumable workflow run.
func (r *Runner) heartbeat(ctx context.Context, runID int64) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if r.store != nil {
				_ = r.store.UpdateRunHeartbeat(runID)
			}
		}
	}
}

// resultCopy returns a shallow copy of the results map.
func resultCopy(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (r *Runner) runWithRetry(ctx context.Context, task *types.Task, retryCfg *types.RetryConfig, stepID string, stepRun *model.WorkflowStepRun) (int, error) {
	bo := retryCfg.BuildBackOff()

	attempt := 0
	for {
		attempt++
		err := r.Run(ctx, task)
		if err == nil {
			return attempt, nil
		}

		// Update step run attempt on each retry.
		if r.store != nil && stepRun != nil {
			_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunRunning, nil, err.Error(), attempt)
		}

		if !retryCfg.RetryEnabled() {
			return attempt, err
		}

		nextDelay := bo.NextBackOff()
		if nextDelay == backoff.Stop {
			return attempt, fmt.Errorf("step %s (retries exhausted, attempt %d): %w", stepID, attempt, err)
		}

		flog.Info("[workflow] step %s attempt %d failed, retrying in %v: %v", stepID, attempt, nextDelay, err)

		select {
		case <-ctx.Done():
			return attempt, fmt.Errorf("step %s cancelled: %w", stepID, ctx.Err())
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

func resolveParams(params types.KV, results map[string]string, input types.KV) (types.KV, error) {
	steps := make(map[string]map[string]any, len(results))
	for stepID, result := range results {
		steps[stepID] = map[string]any{
			"id":     result,
			"result": result,
		}
	}

	data := &template.TemplateData{
		Steps: steps,
		Input: map[string]any(input),
	}

	rendered, err := workflowEngine.Render(map[string]any(params), data)
	if err != nil {
		return nil, err
	}

	return types.KV(rendered), nil
}
