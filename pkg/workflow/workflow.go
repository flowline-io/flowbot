package workflow

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/executor"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	capabilityruntime "github.com/flowline-io/flowbot/pkg/executor/runtime/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline/template"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

var pooledSonic = sonic.Config{}.Froze()

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
		ID:  wt.ID,
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
	paramsJSON, err := pooledSonic.Marshal(params)
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
	auditor      audit.Auditor
	metrics      *metrics.WorkflowCollector
	workflowFile string
	triggerType  string
}

// NewRunner creates a Runner without persistence. Use NewRunnerWithStore to enable run records.
func NewRunner() *Runner {
	return NewRunnerWithStore(nil, nil, nil, "", "")
}

// NewRunnerWithStore creates a Runner that persists run and step records to the given store.
// workflowFile and triggerType are recorded in the run for audit and potential resume.
func NewRunnerWithStore(store WorkflowRunStore, auditor audit.Auditor, wc *metrics.WorkflowCollector, workflowFile, triggerType string) *Runner {
	return &Runner{
		engines: map[string]*executor.Engine{
			runtime.Capability: executor.New(runtime.Capability),
			runtime.Shell:      executor.New(runtime.Shell),
			runtime.Docker:     executor.New(runtime.Docker),
			runtime.Machine:    executor.New(runtime.Machine),
		},
		store:        store,
		auditor:      auditor,
		metrics:      wc,
		workflowFile: workflowFile,
		triggerType:  triggerType,
	}
}

func (r *Runner) auditWorkflowEvent(ctx context.Context, wfName, action string) {
	if r.auditor == nil {
		return
	}
	_ = r.auditor.Record(ctx, audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "workflow",
			SubjectID:   "system:workflow",
		},
		Action: action,
		Target: audit.Target{Type: "workflow", ID: wfName},
	})
}

// Close releases all executor engine resources (Docker clients, SSH connections, capability runtimes).
func (r *Runner) Close() error {
	for _, eng := range r.engines {
		if cerr := eng.Close(); cerr != nil {
			flog.Error(fmt.Errorf("[workflow] close engine: %w", cerr))
		}
	}
	return nil
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

	if wf.MaxConcurrency <= 0 {
		wf.MaxConcurrency = 1
	}

	if wf.MaxConcurrency > 1 {
		return r.executeWithRunRecord(ctx, wf, input, file, taskMap, true)
	}
	return r.executeWithRunRecord(ctx, wf, input, file, taskMap, false)
}

// executeWithRunRecord creates the run record and delegates to sequential or parallel execution.
func (r *Runner) executeWithRunRecord(ctx context.Context, wf types.WorkflowMetadata, input types.KV, file string, taskMap map[string]types.WorkflowTask, parallel bool) error {
	// Persist run record if store is available.
	var run *gen.WorkflowRun
	if r.store != nil {
		workflowFile := file
		if workflowFile == "" {
			workflowFile = r.workflowFile
		}
		triggerType := r.triggerType
		if triggerType == "" {
			triggerType = "manual"
		}
		inputJSON := schema.JSON{}
		if len(input) > 0 {
			raw, _ := pooledSonic.Marshal(input)
			_ = inputJSON.Scan(raw)
		}
		var err error
		run, err = r.store.CreateRun(ctx, wf.Name, workflowFile, triggerType, nil, inputJSON)
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

	if parallel {
		return r.runParallel(ctx, wf, input, taskMap, run, cancelHeartbeat)
	}
	return r.runSequential(ctx, wf, input, taskMap, run, cancelHeartbeat)
}

// runSequential executes workflow tasks one at a time in pipeline order.
func (r *Runner) runSequential(ctx context.Context, wf types.WorkflowMetadata, input types.KV, taskMap map[string]types.WorkflowTask, run *gen.WorkflowRun, cancelHeartbeat context.CancelFunc) error {
	start := time.Now()
	r.auditWorkflowEvent(ctx, wf.Name, "workflow.start")
	var runErr error
	defer func() {
		if r.metrics != nil {
			status := "done"
			if runErr != nil {
				status = "failed"
			}
			r.metrics.IncRunTotal(wf.Name, status)
			r.metrics.ObserveRunDuration(wf.Name, status, time.Since(start).Seconds())
		}
	}()

	results := make(map[string]string)

	for stepIndex, stepID := range wf.Pipeline {
		saveCheckpoint(ctx, stepIndex, r, wf, results, input, run)

		if err := r.executeSequentialStep(ctx, stepID, taskMap, wf, results, input, run); err != nil {
			r.failRun(ctx, run, cancelHeartbeat, err)
			r.auditWorkflowEvent(ctx, wf.Name, "workflow.fail")
			runErr = err
			return runErr
		}
	}

	if r.store != nil && run != nil {
		if cancelHeartbeat != nil {
			cancelHeartbeat()
		}
		_ = r.store.UpdateRunStatus(ctx, run.ID, int(schema.WorkflowRunDone), "")
	}

	r.auditWorkflowEvent(ctx, wf.Name, "workflow.complete")
	return nil
}

// executeSequentialStep executes a single step of a sequential workflow.
func (r *Runner) executeSequentialStep(
	ctx context.Context,
	stepID string,
	taskMap map[string]types.WorkflowTask,
	wf types.WorkflowMetadata,
	results map[string]string,
	input types.KV,
	run *gen.WorkflowRun,
) error {
	wt, ok := taskMap[stepID]
	if !ok {
		return fmt.Errorf("task %s not found in workflow", stepID)
	}

	params, err := resolveParams(wt.Params, results, input)
	if err != nil {
		return fmt.Errorf("resolve params step %s: %w", stepID, err)
	}

	info := ParseAction(wt.Action)

	var stepRun *gen.WorkflowStepRun
	if r.store != nil && run != nil {
		stepRun, err = r.store.CreateStepRun(ctx, run.ID, stepID, wt.Describe, wt.Action, info.Type, schema.JSON(params), 1)
		if err != nil {
			flog.Error(fmt.Errorf("[workflow] create step run record %s: %w", stepID, err))
		}
	}

	if info.Type == "mapper" {
		return r.executeSequentialMapperStep(ctx, stepID, params, info, wf.Name, results, stepRun)
	}
	return r.executeSequentialExecutorStep(ctx, stepID, wt, params, info, wf.Name, results, stepRun)
}

// executeSequentialMapperStep marshals params into the results map for a sequential mapper step.
func (r *Runner) executeSequentialMapperStep(
	ctx context.Context,
	stepID string,
	params types.KV,
	info ActionInfo,
	wfName string,
	results map[string]string,
	stepRun *gen.WorkflowStepRun,
) error {
	stepStart := time.Now()
	if r.metrics != nil {
		r.metrics.IncStepTotal(wfName, stepID, "running")
	}
	mappedJSON, merr := pooledSonic.Marshal(map[string]any(params))
	if merr != nil {
		merr = fmt.Errorf("mapper step %s: %w", stepID, merr)
		if r.metrics != nil {
			r.metrics.IncStepTotal(wfName, stepID, "failed")
			r.metrics.ObserveStepDuration(wfName, stepID, info.Type, "failed", time.Since(stepStart).Seconds())
		}
		r.failStep(ctx, stepRun, merr, 1)
		return merr
	}
	results[stepID] = string(mappedJSON)
	if r.store != nil && stepRun != nil {
		resultJSON := schema.JSON{}
		_ = resultJSON.Scan(mappedJSON)
		_ = r.store.UpdateStepRun(ctx, stepRun.ID, int(schema.WorkflowRunDone), resultJSON, "", 1)
	}
	if r.metrics != nil {
		r.metrics.IncStepTotal(wfName, stepID, "done")
		r.metrics.ObserveStepDuration(wfName, stepID, info.Type, "done", time.Since(stepStart).Seconds())
	}
	flog.Info("[workflow] mapper step %s completed", stepID)
	return nil
}

// executeSequentialExecutorStep converts, runs, and records the result for a sequential executor step.
func (r *Runner) executeSequentialExecutorStep(
	ctx context.Context,
	stepID string,
	wt types.WorkflowTask,
	params types.KV,
	info ActionInfo,
	wfName string,
	results map[string]string,
	stepRun *gen.WorkflowStepRun,
) error {
	wtWithParams := wt
	wtWithParams.Params = params

	task, err := WorkflowTaskToTask(wtWithParams)
	if err != nil {
		err = fmt.Errorf("convert task %s: %w", stepID, err)
		r.failStep(ctx, stepRun, err, 1)
		return err
	}

	flog.Info("[workflow] running step %s: %s", stepID, wt.Action)
	stepStart := time.Now()
	if r.metrics != nil {
		r.metrics.IncStepTotal(wfName, stepID, "running")
	}

	attempt, rerr := r.runWithRetry(ctx, task, wt.Retry, stepID, stepRun)
	if rerr != nil {
		if r.metrics != nil {
			r.metrics.IncStepTotal(wfName, stepID, "failed")
			r.metrics.ObserveStepDuration(wfName, stepID, info.Type, "failed", time.Since(stepStart).Seconds())
			if attempt > 1 {
				r.metrics.IncStepRetry(wfName, stepID)
			}
		}
		r.failStep(ctx, stepRun, rerr, attempt)
		return fmt.Errorf("step %s failed: %w", stepID, rerr)
	}

	if r.metrics != nil {
		r.metrics.IncStepTotal(wfName, stepID, "done")
		r.metrics.ObserveStepDuration(wfName, stepID, info.Type, "done", time.Since(stepStart).Seconds())
		if attempt > 1 {
			r.metrics.IncStepRetry(wfName, stepID)
		}
	}

	if task.Result != "" {
		results[stepID] = task.Result
	}

	if r.store != nil && stepRun != nil {
		resultJSON := schema.JSON{}
		if task.Result != "" {
			resultRaw, _ := pooledSonic.Marshal(map[string]any{"result": task.Result})
			_ = resultJSON.Scan(resultRaw)
		}
		_ = r.store.UpdateStepRun(ctx, stepRun.ID, int(schema.WorkflowRunDone), resultJSON, "", attempt)
	}

	flog.Info("[workflow] step %s completed", stepID)
	return nil
}

// saveCheckpoint persists a checkpoint for a sequential workflow step.
func saveCheckpoint(ctx context.Context, stepIndex int, r *Runner, wf types.WorkflowMetadata, results map[string]string, input types.KV, run *gen.WorkflowRun) {
	if wf.Resumable && r.store != nil && run != nil {
		cp := CheckpointData{
			StepIndex:   stepIndex,
			StepResults: resultCopy(results),
			Input:       input,
			HeartbeatAt: time.Now(),
		}
		if cerr := r.store.SaveCheckpoint(ctx, run.ID, &cp); cerr != nil {
			flog.Error(fmt.Errorf("[workflow] save checkpoint step %d: %w", stepIndex, cerr))
		}
	}
}

// ResumeWorkflow resumes a previously failed or incomplete workflow run from its checkpoint.
func (r *Runner) ResumeWorkflow(ctx context.Context, runID int64) error {
	defer r.Close()

	if r.store == nil {
		return fmt.Errorf("cannot resume workflow without a store")
	}

	run, err := r.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run %d: %w", runID, err)
	}

	if run.Status != int(schema.WorkflowRunRunning) && run.Status != int(schema.WorkflowRunFailed) {
		return fmt.Errorf("workflow run %d status is %d, not resumable", runID, run.Status)
	}

	wf, err := LoadFile(run.WorkflowFile)
	if err != nil {
		return fmt.Errorf("load workflow file %s: %w", run.WorkflowFile, err)
	}

	if r.metrics != nil {
		r.metrics.IncResume(wf.Name)
	}

	var cp CheckpointData
	if err := r.store.GetCheckpoint(ctx, runID, &cp); err != nil {
		return fmt.Errorf("get checkpoint for run %d: %w", runID, err)
	}

	// Parallel resume path.
	if wf.MaxConcurrency > 1 {
		return r.runParallelResume(ctx, runID, *wf, cp)
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
		hbCtx, cancelHeartbeat = context.WithCancel(ctx)
		go r.heartbeat(hbCtx, runID)
	}

	// Skip already-completed steps and resume from checkpoint.
	for i := cp.StepIndex + 1; i < len(wf.Pipeline); i++ {
		stepID := wf.Pipeline[i]
		if err := r.executeResumeStep(ctx, stepID, i, wf, taskMap, results, input, runID, run, cancelHeartbeat); err != nil {
			return err
		}
	}

	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}
	_ = r.store.UpdateRunStatus(ctx, runID, int(schema.WorkflowRunDone), "")
	return nil
}

// executeResumeStep executes a single step during workflow resume.
func (r *Runner) executeResumeStep(
	ctx context.Context,
	stepID string,
	index int,
	wf *types.WorkflowMetadata,
	taskMap map[string]types.WorkflowTask,
	results map[string]string,
	input types.KV,
	runID int64,
	run *gen.WorkflowRun,
	cancelHeartbeat context.CancelFunc,
) error {
	wt, ok := taskMap[stepID]
	if !ok {
		err := fmt.Errorf("task %s not found in workflow", stepID)
		r.failRun(ctx, run, cancelHeartbeat, err)
		return err
	}

	if wf.Resumable {
		cpData := CheckpointData{
			StepIndex:   index,
			StepResults: resultCopy(results),
			Input:       input,
			HeartbeatAt: time.Now(),
		}
		if cerr := r.store.SaveCheckpoint(ctx, runID, &cpData); cerr != nil {
			flog.Error(fmt.Errorf("[workflow] resume save checkpoint step %s: %w", stepID, cerr))
		}
	}

	params, err := resolveParams(wt.Params, results, input)
	if err != nil {
		err = fmt.Errorf("resolve params step %s: %w", stepID, err)
		r.failRun(ctx, run, cancelHeartbeat, err)
		return err
	}

	info := ParseAction(wt.Action)

	var stepRun *gen.WorkflowStepRun
	stepRun, err = r.store.CreateStepRun(ctx, runID, stepID, wt.Describe, wt.Action, info.Type, schema.JSON(params), 1)
	if err != nil {
		flog.Error(fmt.Errorf("[workflow] resume create step run %s: %w", stepID, err))
	}

	if info.Type == "mapper" {
		return r.executeResumeMapperStep(ctx, stepID, params, stepRun, results, run, cancelHeartbeat)
	}
	return r.executeResumeExecutorStep(ctx, stepID, wt, params, stepRun, results, run, cancelHeartbeat)
}

// executeResumeMapperStep marshals params for a mapper step during resume.
func (r *Runner) executeResumeMapperStep(
	ctx context.Context,
	stepID string,
	params types.KV,
	stepRun *gen.WorkflowStepRun,
	results map[string]string,
	run *gen.WorkflowRun,
	cancelHeartbeat context.CancelFunc,
) error {
	mappedJSON, merr := pooledSonic.Marshal(map[string]any(params))
	if merr != nil {
		merr = fmt.Errorf("resume mapper step %s: %w", stepID, merr)
		r.failStep(ctx, stepRun, merr, 1)
		r.failRun(ctx, run, cancelHeartbeat, merr)
		return merr
	}
	results[stepID] = string(mappedJSON)
	if stepRun != nil {
		resultJSON := schema.JSON{}
		_ = resultJSON.Scan(mappedJSON)
		_ = r.store.UpdateStepRun(ctx, stepRun.ID, int(schema.WorkflowRunDone), resultJSON, "", 1)
	}
	return nil
}

// executeResumeExecutorStep converts, runs, and records the result for an executor step during resume.
func (r *Runner) executeResumeExecutorStep(
	ctx context.Context,
	stepID string,
	wt types.WorkflowTask,
	params types.KV,
	stepRun *gen.WorkflowStepRun,
	results map[string]string,
	run *gen.WorkflowRun,
	cancelHeartbeat context.CancelFunc,
) error {
	wtWithParams := wt
	wtWithParams.Params = params

	task, err := WorkflowTaskToTask(wtWithParams)
	if err != nil {
		err = fmt.Errorf("convert task %s: %w", stepID, err)
		r.failStep(ctx, stepRun, err, 1)
		r.failRun(ctx, run, cancelHeartbeat, err)
		return err
	}

	attempt, rerr := r.runWithRetry(ctx, task, wt.Retry, stepID, stepRun)
	if rerr != nil {
		r.failStep(ctx, stepRun, rerr, attempt)
		r.failRun(ctx, run, cancelHeartbeat, rerr)
		return fmt.Errorf("step %s failed: %w", stepID, rerr)
	}

	if task.Result != "" {
		results[stepID] = task.Result
	}

	if stepRun != nil {
		resultJSON := schema.JSON{}
		if task.Result != "" {
			resultRaw, _ := pooledSonic.Marshal(map[string]any{"result": task.Result})
			_ = resultJSON.Scan(resultRaw)
		}
		_ = r.store.UpdateStepRun(ctx, stepRun.ID, int(schema.WorkflowRunDone), resultJSON, "", attempt)
	}
	return nil
}

// failRun marks a workflow run as failed if run is non-nil and cancels the heartbeat.
func (r *Runner) failRun(ctx context.Context, run *gen.WorkflowRun, cancelHeartbeat context.CancelFunc, err error) {
	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}
	if r.store != nil && run != nil {
		_ = r.store.UpdateRunStatus(ctx, run.ID, int(schema.WorkflowRunFailed), err.Error())
	}
}

// failStep marks a step run as failed if stepRun is non-nil.
func (r *Runner) failStep(ctx context.Context, stepRun *gen.WorkflowStepRun, err error, attempt int) {
	if r.store != nil && stepRun != nil {
		_ = r.store.UpdateStepRun(ctx, stepRun.ID, int(schema.WorkflowRunFailed), nil, err.Error(), attempt)
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
				_ = r.store.UpdateRunHeartbeat(ctx, runID)
			}
		}
	}
}

// resultCopy returns a shallow copy of the results map.
func resultCopy(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}

func (r *Runner) runWithRetry(ctx context.Context, task *types.Task, retryCfg *types.RetryConfig, stepID string, stepRun *gen.WorkflowStepRun) (int, error) {
	backoffCfg := retryCfg.ToBackoffConfig()
	backoffCfg.OnRetry = func(attempt int, delay time.Duration, err error) {
		if r.store != nil && stepRun != nil {
			_ = r.store.UpdateStepRun(ctx, stepRun.ID, int(schema.WorkflowRunRunning), nil, err.Error(), attempt)
		}
		flog.Info("[workflow] step %s attempt %d failed, retrying in %v: %v", stepID, attempt, delay, err)
	}
	return backoff.Do(ctx, backoffCfg, func(ctx context.Context) error {
		return r.Run(ctx, task)
	})
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
