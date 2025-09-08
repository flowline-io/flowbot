package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/samber/lo"

	"github.com/flc1125/go-cron/v4"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/goccy/go-yaml"
)

const arrowSymbol = "->"

func parseWorkflowMetadata(code string) (types.WorkflowMetadata, error) {
	var meta types.WorkflowMetadata
	err := yaml.Unmarshal(utils.StringToBytes(code), &meta)
	if err != nil {
		return types.WorkflowMetadata{}, fmt.Errorf("failed to unmarshal workflow metadata: %w", err)
	}
	return meta, nil
}

func ParseYamlWorkflow(code string) (workflow *model.Workflow, triggers []*model.WorkflowTrigger, dag *model.Dag, err error) {
	meta, err := parseWorkflowMetadata(code)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse workflow metadata: %w", err)
	}

	if err = MetaDataValidate(meta); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to validate workflow metadata: %w", err)
	}

	// workflow
	workflow = new(model.Workflow)
	workflow.Name = meta.Name
	workflow.Describe = meta.Describe
	workflow.State = model.WorkflowEnable
	workflow.Flag = types.Id()

	// trigger
	triggers = make([]*model.WorkflowTrigger, 0, len(meta.Triggers))
	for _, trigger := range meta.Triggers {
		triggers = append(triggers, &model.WorkflowTrigger{
			Type:  model.TriggerType(trigger.Type),
			Rule:  model.JSON(trigger.Rule),
			State: model.WorkflowTriggerEnable,
		})
	}

	// dag
	dag = new(model.Dag)

	nodes := make([]*model.Node, 0, len(meta.Tasks))
	for _, task := range meta.Tasks {
		botName, ruleId, ok := parseTaskAction(task.Action)
		if !ok {
			err = fmt.Errorf("task action %s parse error", task.Action)
			return
		}
		nodes = append(nodes, &model.Node{
			Id:          task.ID,
			Describe:    task.Describe,
			Bot:         botName,
			RuleId:      ruleId,
			Parameters:  task.Params,
			Variables:   task.Vars,
			Connections: task.Conn,
			Status:      model.NodeDefault,
		})
	}

	edges := make([]*model.Edge, 0, len(meta.Tasks))

	index := 1
	for _, pipeline := range meta.Pipeline {
		pipelineEdges := parsePipelineEdges(pipeline)
		for _, edge := range pipelineEdges {
			edges = append(edges, &model.Edge{
				Id:     fmt.Sprintf("edge-%d", index),
				Source: edge[0],
				Target: edge[1],
			})
			index++
		}
	}

	dag.Nodes = nodes
	dag.Edges = edges

	return
}

func MetaDataValidate(meta types.WorkflowMetadata) (err error) {
	if strings.TrimSpace(meta.Name) == "" {
		err = errors.New("name empty")
		return
	}

	if len(meta.Triggers) == 0 {
		err = errors.New("triggers empty")
		return
	}
	for _, trigger := range meta.Triggers {
		if !lo.Contains([]string{
			string(model.TriggerManual),
			string(model.TriggerCron),
			string(model.TriggerWebhook),
		}, trigger.Type) {
			err = errors.New("trigger type error")
			return
		}

		switch model.TriggerType(trigger.Type) {
		case model.TriggerCron:
			spec, ok := trigger.Rule.String("spec")
			if !ok {
				err = errors.New("trigger cron: rule error")
				return
			}
			if strings.TrimSpace(spec) == "" {
				err = errors.New("trigger cron: spec empty")
				return
			}

			specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			_, err = specParser.Parse(spec)
			if err != nil {
				return fmt.Errorf("failed to parse trigger cron: %w", err)
			}
		case model.TriggerWebhook:
			// todo webhook params validate
		}
	}

	if len(meta.Pipeline) == 0 {
		err = errors.New("pipeline empty")
		return
	}

	for index, pipeline := range meta.Pipeline {
		if !pipelineFormatValidate(pipeline) {
			err = fmt.Errorf("line %d pipeline %s format error", index+1, pipeline)
			return
		}
	}

	if len(meta.Tasks) == 0 {
		err = errors.New("tasks empty")
		return
	}

	re, err := regexp.Compile(`^[a-z_]+@[a-z_]+$`)
	if err != nil {
		return
	}

	taskIdSet := make(map[string]struct{})
	for _, task := range meta.Tasks {
		if strings.TrimSpace(task.ID) == "" {
			err = errors.New("task id empty")
			return
		}
		if _, ok := taskIdSet[task.ID]; ok {
			err = fmt.Errorf("task id %s duplicated", task.ID)
			return
		}
		taskIdSet[task.ID] = struct{}{}

		if strings.TrimSpace(task.Action) == "" {
			err = errors.New("task action empty")
			return
		}
		if !re.Match(utils.StringToBytes(task.Action)) {
			err = errors.New("task action format error, e.g. action@bot")
			return
		}
	}

	spaceRe, err := regexp.Compile(`\s+`)
	if err != nil {
		return
	}
	for _, pipeline := range meta.Pipeline {
		tokens := spaceRe.Split(pipeline, -1)
		for _, token := range tokens {
			if token == arrowSymbol {
				continue
			}
			if _, ok := taskIdSet[token]; !ok {
				err = fmt.Errorf("pipeline node id %s not exist in tasks", token)
				return
			}
		}
	}

	return
}

func pipelineFormatValidate(s string) bool {
	if len(s) == 0 {
		return false
	}

	s = strings.TrimSpace(s)

	re, err := regexp.Compile(`\s+`)
	if err != nil {
		return false
	}

	tokens := re.Split(s, -1)
	l := len(tokens)

	if len(tokens) == 1 || len(tokens) == 2 {
		return false
	}

	if tokens[l-1] == arrowSymbol {
		return false
	}

	i := 2
	for i < l {
		if !(tokens[i-2] != arrowSymbol && tokens[i-1] == arrowSymbol && tokens[i] != arrowSymbol) {
			return false
		}
		i += 2
	}

	return true
}

func parseTaskAction(action string) (string, string, bool) {
	tokens := strings.Split(action, "@")

	if len(tokens) != 2 {
		return "", "", false
	}

	return tokens[1], tokens[0], true
}

func parsePipelineEdges(pipeline string) [][2]string {
	if pipeline == "" {
		return nil
	}

	pipeline = strings.TrimSpace(pipeline)

	re, err := regexp.Compile(`\s+`)
	if err != nil {
		return nil
	}

	tokens := re.Split(pipeline, -1)
	l := len(tokens)

	if l < 3 {
		return nil
	}

	i := 2

	edges := make([][2]string, 0)
	for i < l {
		edges = append(edges, [2]string{tokens[i-2], tokens[i]})
		i += 2
	}

	return edges
}

func kvGrep(pattern string, input types.KV) (types.KV, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var data = make(map[string][]types.KV)

	for k, v := range input {
		list, ok := v.([]any)
		if !ok {
			continue
		}

		for _, item := range list {
			kv, ok := item.(map[string]any)
			if !ok {
				continue
			}

			for _, value := range kv {
				valueStr, ok := value.(string)
				if !ok {
					continue
				}

				if re.MatchString(valueStr) {
					data[k] = append(data[k], kv)
				}
			}
		}
	}

	result := make(types.KV)
	for k := range data {
		result[k] = data[k]
	}

	return result, nil
}
