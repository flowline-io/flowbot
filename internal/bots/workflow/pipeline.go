package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/pipeline"
	"github.com/flowline-io/flowbot/internal/types/step"
)

const (
	examplePipelineId = "example_pipeline"
)

var pipelineRules = []pipeline.Rule{
	{
		Id:      examplePipelineId,
		Version: 1,
		Help:    "example pipeline",
		Trigger: step.CommandTrigger("example [string]"),
		Step: step.Stage(
			step.Form("dev_form"),
			step.Action("dev_action"),
			step.Command(step.Bot("dev"), "rand", "1", "100"),
			//schema.Instruct("dev_example"),
			//schema.Session("guess_session", "100"),
		),
	},
}
