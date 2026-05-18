package workflow

import (
	"fmt"

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
