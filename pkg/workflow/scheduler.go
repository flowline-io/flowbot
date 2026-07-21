package workflow

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/executor"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// dagNode represents a node in the execution DAG.
type dagNode struct {
	task     types.WorkflowTask
	inDegree int      // number of unfinished dependencies
	deps     []string // tasks that depend on this node (reverse edges)
}

// buildDAG constructs a DAG from workflow tasks using the Conn dependency field.
// Returns a map from task ID to dagNode, and a list of task IDs with zero in-degree (ready to run).
// The Conn field on each task lists its dependencies: task.Conn = [dep1, dep2] means
// "this task depends on dep1 and dep2", i.e., edges dep1->task and dep2->task exist.
func buildDAG(tasks []types.WorkflowTask) (map[string]*dagNode, []string, error) {
	nodes := make(map[string]*dagNode, len(tasks))
	for _, t := range tasks {
		nodes[t.ID] = &dagNode{task: t}
	}

	for _, t := range tasks {
		for _, dep := range t.Conn {
			depNode, ok := nodes[dep]
			if !ok {
				return nil, nil, fmt.Errorf("task %s references unknown dependency %s", t.ID, dep)
			}
			nodes[t.ID].inDegree++
			depNode.deps = append(depNode.deps, t.ID)
		}
	}

	ready := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if nodes[t.ID].inDegree == 0 {
			ready = append(ready, t.ID)
		}
	}

	return nodes, ready, nil
}

// parallelTaskFn is the signature used by dispatchReadyTasks to run a single task
// inside a dispatch goroutine.
type parallelTaskFn func(
	ctx context.Context,
	taskID string,
	wt types.WorkflowTask,
	nodes map[string]*dagNode,
	input types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	run *gen.WorkflowRun,
	ready *[]string,
	wf *types.WorkflowMetadata,
	firstErr *error,
	errOnce *sync.Once,
	cancel context.CancelFunc,
)

// runParallel executes workflow tasks in parallel based on the DAG defined by Conn dependencies.
func (r *Runner) runParallel(ctx context.Context, wf types.WorkflowMetadata, input types.KV, taskMap map[string]types.WorkflowTask, run *gen.WorkflowRun, cancelHeartbeat context.CancelFunc) error {
	nodes, ready, err := buildDAG(wf.Tasks)
	if err != nil {
		return fmt.Errorf("build dag: %w", err)
	}

	parallelStart := time.Now()
	var finalErr error
	defer func() {
		if r.metrics != nil {
			status := "done"
			if finalErr != nil {
				status = "failed"
			}
			r.metrics.IncRunTotal(wf.Name, status)
			r.metrics.ObserveRunDuration(wf.Name, status, time.Since(parallelStart).Seconds())
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, wf.MaxConcurrency)
	results := make(map[string]string)
	var mu sync.RWMutex
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	done := make(chan struct{}, len(wf.Tasks))
	activeCount := 0
	totalRemaining := len(wf.Tasks)

	for totalRemaining > 0 {
		r.dispatchReadyTasks(ctx, wf, &ready, &activeCount, taskMap, nodes, input, &results, &mu, run, sem, &wg, done, &firstErr, &errOnce, cancel, r.runParallelTaskHandler)

		select {
		case <-ctx.Done():
			goto drain
		case <-done:
			mu.Lock()
			activeCount--
			if r.metrics != nil {
				r.metrics.SetConcurrency(wf.Name, activeCount)
			}
			totalRemaining--
			mu.Unlock()
		}
	}

drain:
	finalErr = firstErr
	// Persist terminal status before waiting on cancelled siblings so a hung
	// branch cannot leave the run stuck in Running.
	statusErr := r.finalizeParallelStatus(ctx, run, 0, finalErr)
	wg.Wait()
	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}
	return statusErr
}

// dispatchReadyTasks pops ready tasks from the queue and spawns dispatch goroutines
// until no more tasks are dispatchable (concurrency limit reached or queue empty).
func (r *Runner) dispatchReadyTasks(
	ctx context.Context,
	wf types.WorkflowMetadata,
	ready *[]string,
	activeCount *int,
	taskMap map[string]types.WorkflowTask,
	nodes map[string]*dagNode,
	input types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	run *gen.WorkflowRun,
	sem chan struct{},
	wg *sync.WaitGroup,
	done chan struct{},
	firstErr *error,
	errOnce *sync.Once,
	cancel context.CancelFunc,
	taskFn parallelTaskFn,
) {
	for {
		mu.Lock()
		hasReady := len(*ready) > 0 && *activeCount < wf.MaxConcurrency
		if hasReady {
			id := (*ready)[0]
			*ready = (*ready)[1:]
			*activeCount++
			if r.metrics != nil {
				r.metrics.SetConcurrency(wf.Name, *activeCount)
			}
			mu.Unlock()

			sem <- struct{}{}
			wg.Add(1)

			go func(taskID string) {
				defer wg.Done()
				defer func() {
					<-sem
					done <- struct{}{}
				}()

				wt := taskMap[taskID]
				taskFn(ctx, taskID, wt, nodes, input, results, mu, run, ready, &wf, firstErr, errOnce, cancel)
			}(id)
		} else {
			mu.Unlock()
			break
		}
	}
}

// runParallelTaskHandler executes a single parallel task with metrics recording and error handling.
//
//nolint:revive
func (r *Runner) runParallelTaskHandler(
	ctx context.Context,
	taskID string,
	wt types.WorkflowTask,
	nodes map[string]*dagNode,
	input types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	run *gen.WorkflowRun,
	ready *[]string,
	wf *types.WorkflowMetadata,
	firstErr *error,
	errOnce *sync.Once,
	cancel context.CancelFunc,
) {
	stepStart := time.Now()
	info := ParseAction(wt.Action)
	if r.metrics != nil {
		r.metrics.IncStepTotal(wf.Name, taskID, "running")
	}

	rerr := r.executeParallelTask(ctx, taskID, wt, nodes, input, results, mu, run, ready, wf)
	if rerr != nil {
		errOnce.Do(func() {
			*firstErr = rerr
			cancel()
		})
		if r.metrics != nil {
			r.metrics.IncStepTotal(wf.Name, taskID, "failed")
			r.metrics.ObserveStepDuration(wf.Name, taskID, info.Type, "failed", time.Since(stepStart).Seconds())
		}
	} else if r.metrics != nil {
		r.metrics.IncStepTotal(wf.Name, taskID, "done")
		r.metrics.ObserveStepDuration(wf.Name, taskID, info.Type, "done", time.Since(stepStart).Seconds())
	}
}

// runParallelResumeTaskHandler executes a single parallel task during resume
// with error handling but without metrics recording (resume has no metrics context).
//
//nolint:revive
func (r *Runner) runParallelResumeTaskHandler(
	ctx context.Context,
	taskID string,
	wt types.WorkflowTask,
	nodes map[string]*dagNode,
	input types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	run *gen.WorkflowRun,
	ready *[]string,
	wf *types.WorkflowMetadata,
	firstErr *error,
	errOnce *sync.Once,
	cancel context.CancelFunc,
) {
	rerr := r.executeParallelTask(ctx, taskID, wt, nodes, input, results, mu, run, ready, wf)
	if rerr != nil {
		errOnce.Do(func() {
			*firstErr = rerr
			cancel()
		})
	}
}

// finalizeParallelStatus updates the run status in the store and returns the error.
// If run is non-nil its ID takes precedence; otherwise runID is used.
// A zero or negative effective ID skips store updates.
func (r *Runner) finalizeParallelStatus(ctx context.Context, run *gen.WorkflowRun, runID int64, err error) error {
	id := runID
	if run != nil {
		id = run.ID
	}
	if id <= 0 || r.store == nil {
		return err
	}
	storeCtx := workflowStoreCtx(ctx)
	if err != nil {
		_ = r.store.UpdateRunStatus(storeCtx, id, int(schema.WorkflowRunFailed), err.Error())
		return err
	}
	_ = r.store.UpdateRunStatus(storeCtx, id, int(schema.WorkflowRunDone), "")
	return nil
}

// executeParallelTask runs a single task and enqueues newly-ready dependents.
func (r *Runner) executeParallelTask(
	ctx context.Context,
	taskID string,
	wt types.WorkflowTask,
	nodes map[string]*dagNode,
	input types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	run *gen.WorkflowRun,
	ready *[]string,
	wf *types.WorkflowMetadata,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	mu.RLock()
	params, err := resolveParams(wt.Params, *results, input)
	mu.RUnlock()
	if err != nil {
		r.failRun(ctx, run, nil, fmt.Errorf("resolve params step %s: %w", taskID, err))
		return fmt.Errorf("resolve params step %s: %w", taskID, err)
	}

	info := ParseAction(wt.Action)

	var stepRun *gen.WorkflowStepRun
	if r.store != nil && run != nil {
		stepRun, err = r.store.CreateStepRun(workflowStoreCtx(ctx), run.ID, taskID, wt.Describe, wt.Action, info.Type, schema.JSON(params), 1)
		if err != nil {
			flog.Error(fmt.Errorf("[workflow] create step run record %s: %w", taskID, err))
		}
	}

	if rerr := r.executeStepResult(ctx, taskID, wt, params, info, results, mu, stepRun, wf.Name); rerr != nil {
		return rerr
	}

	r.enqueueDependentsAndSaveCheckpoint(ctx, nodes, taskID, results, mu, ready, wf, input, run)
	return nil
}

// executeStepResult runs either a mapper or an executor step and stores the result.
func (r *Runner) executeStepResult(
	ctx context.Context,
	taskID string,
	wt types.WorkflowTask,
	params types.KV,
	info ActionInfo,
	results *map[string]string,
	mu *sync.RWMutex,
	stepRun *gen.WorkflowStepRun,
	wfName string,
) error {
	if info.Type == "mapper" {
		return r.executeMapperStep(ctx, taskID, params, results, mu, stepRun)
	}
	return r.executeExecutorStep(ctx, taskID, wt, params, results, mu, stepRun, wfName)
}

// executeMapperStep marshals params into the results map for a mapper-type step.
func (r *Runner) executeMapperStep(
	ctx context.Context,
	taskID string,
	params types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	stepRun *gen.WorkflowStepRun,
) error {
	mappedJSON, merr := pooledSonic.Marshal(map[string]any(params))
	if merr != nil {
		merr = fmt.Errorf("mapper step %s: %w", taskID, merr)
		r.failStep(ctx, stepRun, merr, 1)
		return merr
	}
	mu.Lock()
	(*results)[taskID] = string(mappedJSON)
	mu.Unlock()
	if r.store != nil && stepRun != nil {
		resultJSON := schema.JSON{}
		_ = resultJSON.Scan(mappedJSON)
		_ = r.store.UpdateStepRun(workflowStoreCtx(ctx), stepRun.ID, int(schema.WorkflowRunDone), resultJSON, "", 1)
	}
	flog.Info("[workflow] mapper step %s completed (parallel)", taskID)
	return nil
}

// executeExecutorStep converts, runs, and records the result for a non-mapper step.
func (r *Runner) executeExecutorStep(
	ctx context.Context,
	taskID string,
	wt types.WorkflowTask,
	params types.KV,
	results *map[string]string,
	mu *sync.RWMutex,
	stepRun *gen.WorkflowStepRun,
	wfName string,
) error {
	wtWithParams := wt
	wtWithParams.Params = params
	task, err := WorkflowTaskToTask(wtWithParams)
	if err != nil {
		err = fmt.Errorf("convert task %s: %w", taskID, err)
		r.failStep(ctx, stepRun, err, 1)
		return err
	}

	rt := DetermineRuntimeType(task)
	engine := executor.New(rt)
	defer engine.Close()

	flog.Info("[workflow] running step %s: %s (parallel)", taskID, wt.Action)

	backoffCfg := wt.Retry.ToBackoffConfig()
	backoffCfg.OnRetry = func(a int, d time.Duration, err error) {
		if r.store != nil && stepRun != nil {
			_ = r.store.UpdateStepRun(workflowStoreCtx(ctx), stepRun.ID, int(schema.WorkflowRunRunning), nil, err.Error(), a)
		}
		flog.Info("[workflow] step %s attempt %d failed, retrying in %v: %v", taskID, a, d, err)
	}
	attempt, rerr := backoff.Do(ctx, backoffCfg, func(ctx context.Context) error {
		return engine.Run(ctx, task)
	})
	if r.metrics != nil && attempt > 1 {
		r.metrics.IncStepRetry(wfName, taskID)
	}
	if rerr != nil {
		r.failStep(ctx, stepRun, rerr, attempt)
		return fmt.Errorf("step %s failed: %w", taskID, rerr)
	}

	if task.Result != "" {
		mu.Lock()
		(*results)[taskID] = task.Result
		mu.Unlock()
	}

	if r.store != nil && stepRun != nil {
		resultJSON := schema.JSON{}
		if task.Result != "" {
			resultRaw, _ := pooledSonic.Marshal(map[string]any{"result": task.Result})
			_ = resultJSON.Scan(resultRaw)
		}
		_ = r.store.UpdateStepRun(workflowStoreCtx(ctx), stepRun.ID, int(schema.WorkflowRunDone), resultJSON, "", attempt)
	}

	flog.Info("[workflow] step %s completed (parallel)", taskID)
	return nil
}

// enqueueDependentsAndSaveCheckpoint decrements in-degrees of dependents,
// enqueues newly-ready tasks, and persists a checkpoint if resumable.
func (r *Runner) enqueueDependentsAndSaveCheckpoint(
	ctx context.Context,
	nodes map[string]*dagNode,
	taskID string,
	results *map[string]string,
	mu *sync.RWMutex,
	ready *[]string,
	wf *types.WorkflowMetadata,
	input types.KV,
	run *gen.WorkflowRun,
) {
	mu.Lock()
	defer mu.Unlock()

	node := nodes[taskID]
	for _, depID := range node.deps {
		depNode := nodes[depID]
		depNode.inDegree--
		if depNode.inDegree == 0 {
			*ready = append(*ready, depID)
		}
	}

	if wf.Resumable && r.store != nil && run != nil {
		completedTasks := make(map[string]bool)
		for taskID := range *results {
			completedTasks[taskID] = true
		}
		resultCopy := make(map[string]string, len(*results))
		maps.Copy(resultCopy, *results)
		cp := CheckpointData{
			CompletedTasks: completedTasks,
			StepResults:    resultCopy,
			Input:          input,
			HeartbeatAt:    time.Now(),
		}
		if cerr := r.store.SaveCheckpoint(workflowStoreCtx(ctx), run.ID, &cp); cerr != nil {
			flog.Error(fmt.Errorf("[workflow] save checkpoint step %s: %w", taskID, cerr))
		}
	}
}

// premarkCompletedTasksForResume decrements the in-degree of dependents
// for all tasks that have already completed (recorded in the checkpoint).
func (*Runner) premarkCompletedTasksForResume(cp CheckpointData, nodes map[string]*dagNode) {
	for taskID := range cp.CompletedTasks {
		node, ok := nodes[taskID]
		if !ok {
			continue
		}
		for _, depID := range node.deps {
			depNode := nodes[depID]
			depNode.inDegree--
		}
	}
}

// recomputeReadyList rebuilds the ready list from scratch, including only
// tasks with inDegree==0 that have not yet completed.
func (*Runner) recomputeReadyList(wf types.WorkflowMetadata, cp CheckpointData, nodes map[string]*dagNode, ready []string) []string {
	ready = ready[:0]
	for _, t := range wf.Tasks {
		if cp.CompletedTasks[t.ID] {
			continue
		}
		if nodes[t.ID].inDegree == 0 {
			ready = append(ready, t.ID)
		}
	}
	return ready
}

// countRemainingTasksOnResume counts tasks in the workflow that have not been completed
// according to the checkpoint.
func (*Runner) countRemainingTasksOnResume(wf types.WorkflowMetadata, cp CheckpointData) int {
	totalRemaining := 0
	for _, t := range wf.Tasks {
		if !cp.CompletedTasks[t.ID] {
			totalRemaining++
		}
	}
	return totalRemaining
}

// runParallelResume resumes a parallel workflow from its checkpoint.
func (r *Runner) runParallelResume(ctx context.Context, runID int64, wf types.WorkflowMetadata, cp CheckpointData) error {
	run, err := r.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run %d: %w", runID, err)
	}

	taskMap := make(map[string]types.WorkflowTask)
	for _, wt := range wf.Tasks {
		taskMap[wt.ID] = wt
	}

	nodes, ready, err := buildDAG(wf.Tasks)
	if err != nil {
		return fmt.Errorf("build dag: %w", err)
	}

	results := resultCopy(cp.StepResults)

	r.premarkCompletedTasksForResume(cp, nodes)

	ready = r.recomputeReadyList(wf, cp, nodes, ready)

	input := cp.Input

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, wf.MaxConcurrency)
	var mu sync.RWMutex
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	done := make(chan struct{}, len(wf.Tasks))
	activeCount := 0
	totalRemaining := r.countRemainingTasksOnResume(wf, cp)

	if totalRemaining == 0 {
		_ = r.store.UpdateRunStatus(workflowStoreCtx(ctx), runID, int(schema.WorkflowRunDone), "")
		return nil
	}

	for totalRemaining > 0 {
		r.dispatchReadyTasks(ctx, wf, &ready, &activeCount, taskMap, nodes, input, &results, &mu, run, sem, &wg, done, &firstErr, &errOnce, cancel, r.runParallelResumeTaskHandler)

		select {
		case <-ctx.Done():
			goto drain
		case <-done:
			mu.Lock()
			activeCount--
			totalRemaining--
			mu.Unlock()
		}
	}

drain:
	statusErr := r.finalizeParallelStatus(ctx, run, 0, firstErr)
	wg.Wait()
	return statusErr
}
