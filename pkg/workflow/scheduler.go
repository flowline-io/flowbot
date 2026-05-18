package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/internal/store/model"
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

// runParallel executes workflow tasks in parallel based on the DAG defined by Conn dependencies.
func (r *Runner) runParallel(ctx context.Context, wf types.WorkflowMetadata, input types.KV, taskMap map[string]types.WorkflowTask, run *model.WorkflowRun, cancelHeartbeat context.CancelFunc) error {
	nodes, ready, err := buildDAG(wf.Tasks)
	if err != nil {
		return fmt.Errorf("build dag: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, wf.MaxConcurrency)
	results := make(map[string]string)
	var mu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	done := make(chan struct{}, len(wf.Tasks))
	activeCount := 0
	totalRemaining := len(wf.Tasks)

	for totalRemaining > 0 {
		for len(ready) > 0 && activeCount < wf.MaxConcurrency {
			select {
			case <-ctx.Done():
				goto drain
			default:
			}

			id := ready[0]
			ready = ready[1:]

			sem <- struct{}{}
			activeCount++
			wg.Add(1)

			go func(taskID string) {
				defer wg.Done()
				defer func() {
					<-sem
					done <- struct{}{}
				}()

				wt := taskMap[taskID]

				rerr := r.executeParallelTask(ctx, taskID, wt, nodes, input, &results, &mu, run, &ready, taskMap, &wf)
				if rerr != nil {
					errOnce.Do(func() {
						firstErr = rerr
						cancel()
					})
				}
			}(id)
		}

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
	wg.Wait()

	if cancelHeartbeat != nil {
		cancelHeartbeat()
	}

	if firstErr != nil {
		if r.store != nil && run != nil {
			_ = r.store.UpdateRunStatus(run.ID, model.WorkflowRunFailed, firstErr.Error())
		}
		return firstErr
	}

	if r.store != nil && run != nil {
		_ = r.store.UpdateRunStatus(run.ID, model.WorkflowRunDone, "")
	}

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
	mu *sync.Mutex,
	run *model.WorkflowRun,
	ready *[]string,
	taskMap map[string]types.WorkflowTask,
	wf *types.WorkflowMetadata,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	mu.Lock()
	currentResults := make(map[string]string, len(*results))
	for k, v := range *results {
		currentResults[k] = v
	}
	mu.Unlock()

	params, err := resolveParams(wt.Params, currentResults, input)
	if err != nil {
		r.failRun(run, nil, fmt.Errorf("resolve params step %s: %w", taskID, err))
		return fmt.Errorf("resolve params step %s: %w", taskID, err)
	}

	info := ParseAction(wt.Action)

	var stepRun *model.WorkflowStepRun
	if r.store != nil && run != nil {
		stepRun, err = r.store.CreateStepRun(run.ID, taskID, wt.Describe, wt.Action, info.Type, model.JSON(params), 1)
		if err != nil {
			flog.Error(fmt.Errorf("[workflow] create step run record %s: %w", taskID, err))
		}
	}

	if info.Type == "mapper" {
		mappedJSON, merr := pooledSonic.Marshal(map[string]any(params))
		if merr != nil {
			merr = fmt.Errorf("mapper step %s: %w", taskID, merr)
			r.failStep(stepRun, merr, 1)
			return merr
		}
		mu.Lock()
		(*results)[taskID] = string(mappedJSON)
		mu.Unlock()
		if r.store != nil && stepRun != nil {
			resultJSON := model.JSON{}
			_ = resultJSON.Scan(mappedJSON)
			_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", 1)
		}
		flog.Info("[workflow] mapper step %s completed (parallel)", taskID)
	} else {
		wtWithParams := wt
		wtWithParams.Params = params
		task, err := WorkflowTaskToTask(wtWithParams)
		if err != nil {
			err = fmt.Errorf("convert task %s: %w", taskID, err)
			r.failStep(stepRun, err, 1)
			return err
		}

		rt := DetermineRuntimeType(task)
		engine := executor.New(rt)
		defer engine.Close()

		flog.Info("[workflow] running step %s: %s (parallel)", taskID, wt.Action)

		attempt, rerr := r.runEngineWithRetry(ctx, engine, task, wt.Retry, taskID, stepRun)
		if rerr != nil {
			r.failStep(stepRun, rerr, attempt)
			return fmt.Errorf("step %s failed: %w", taskID, rerr)
		}

		if task.Result != "" {
			mu.Lock()
			(*results)[taskID] = task.Result
			mu.Unlock()
		}

		if r.store != nil && stepRun != nil {
			resultJSON := model.JSON{}
			if task.Result != "" {
				resultRaw, _ := pooledSonic.Marshal(map[string]any{"result": task.Result})
				_ = resultJSON.Scan(resultRaw)
			}
			_ = r.store.UpdateStepRun(stepRun.ID, model.WorkflowRunDone, resultJSON, "", attempt)
		}

		flog.Info("[workflow] step %s completed (parallel)", taskID)
	}

	mu.Lock()
	node := nodes[taskID]
	for _, depID := range node.deps {
		depNode := nodes[depID]
		depNode.inDegree--
		if depNode.inDegree == 0 {
			*ready = append(*ready, depID)
		}
	}
	mu.Unlock()

	return nil
}
