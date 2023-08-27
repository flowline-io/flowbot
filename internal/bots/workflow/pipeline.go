package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/pipeline"
	"github.com/flowline-io/flowbot/internal/types/schema"
)

const (
	examplePipelineId = "example_pipeline"
)

var pipelineRules = []pipeline.Rule{
	{
		Id:      examplePipelineId,
		Version: 1,
		Help:    "example pipeline",
		Trigger: schema.CommandTrigger("example [string]"),
		Step: schema.Stage(
			schema.Form("dev_form"),
			schema.Action("dev_action"),
			schema.Command(schema.Bot("dev"), "rand", "1", "100"),
			//schema.Instruct("dev_example"),
			//schema.Session("guess_session", "100"),
		),
	},
}
