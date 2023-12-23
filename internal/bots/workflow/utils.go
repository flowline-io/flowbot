package workflow

import (
	"errors"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
	"strings"
)

func ParseYamlWorkflow(code string) (workflow *model.Workflow, trigger *model.WorkflowTrigger, dag *model.Dag, err error) {
	var meta types.WorkflowMetadata
	err = yaml.Unmarshal(utils.StringToBytes(code), &meta)
	if err != nil {
		return
	}

	if err = MetaDataValidate(meta); err != nil {
		return
	}

	// workflow
	workflow = new(model.Workflow)
	workflow.Name = meta.Name
	workflow.Describe = meta.Describe

	// trigger
	trigger = new(model.WorkflowTrigger)
	trigger.Type = model.TriggerType(meta.Trigger.Type)
	trigger.Rule = model.JSON(meta.Trigger.Rule)
	trigger.State = model.WorkflowTriggerEnable

	// dag
	dag = new(model.Dag)

	return
}

func MetaDataValidate(meta types.WorkflowMetadata) (err error) {
	if strings.TrimSpace(meta.Name) == "" {
		err = errors.New("name empty")
		return
	}

	if !utils.InStringSlice([]string{
		string(model.TriggerManual),
		string(model.TriggerCron),
		string(model.TriggerWebhook),
	}, meta.Trigger.Type) {
		err = errors.New("trigger type error")
		return
	}

	switch model.TriggerType(meta.Trigger.Type) {
	case model.TriggerCron:
		spec, ok := meta.Trigger.Rule.String("spec")
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
			return
		}
	case model.TriggerWebhook:
		// todo webhook
	}

	if len(meta.Pipeline) == 0 {
		err = errors.New("pipeline empty")
		return
	}

	if len(meta.Tasks) == 0 {
		err = errors.New("tasks empty")
		return
	}

	return
}
