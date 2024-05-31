package dag

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	dagLib "github.com/heimdalr/dag"
	"github.com/pkg/errors"
)

type nodeId string

func (n nodeId) ID() string {
	return string(n)
}

func TopologySort(item *model.Dag) ([]model.Step, error) {
	d := dagLib.NewDAG()
	nodeMap := make(map[string]*model.Node)
	for i, node := range item.Nodes {
		_, err := d.AddVertex(nodeId(node.Id))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add vertex %s", node.Id)
		}
		nodeMap[node.Id] = item.Nodes[i]
	}
	for _, edge := range item.Edges {
		err := d.AddEdge(edge.Source, edge.Target)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add edge %s -> %s", edge.Source, edge.Target)
		}
	}

	baseRoots := d.GetRoots()
	roots := baseRoots
	have := make(map[string]struct{}, len(item.Nodes))
	var result []model.Step
	for {
		if len(roots) == 0 {
			break
		}
		for id := range roots {
			if _, ok := have[id]; ok {
				continue
			}
			parents, err := d.GetParents(id)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get parents of %s", id)
			}
			dependNodeId := make([]string, 0)
			for pid := range parents {
				dependNodeId = append(dependNodeId, pid)
			}
			state := model.StepCreated
			_, ok := baseRoots[id]
			if ok {
				state = model.StepReady
			}

			n := nodeMap[id]
			action := model.JSON{
				"bot":        n.Bot,
				"rule_id":    n.RuleId,
				"parameters": n.Parameters,
			}
			result = append(result, model.Step{
				NodeID: id,
				Depend: dependNodeId,
				Action: action,
				State:  state,
			})
			have[id] = struct{}{}
		}

		children := make(map[string]interface{})
		for id := range roots {
			items, err := d.GetChildren(id)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get children of %s", id)
			}
			for k, v := range items {
				children[k] = v
			}
		}
		roots = children
	}

	return result, nil
}
