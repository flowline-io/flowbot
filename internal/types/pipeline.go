package types

type TriggerType string

const (
	TriggerCommandType TriggerType = "command"
)

type StageType string

const (
	ActionStage   StageType = "action"
	CommandStage  StageType = "command"
	FormStage     StageType = "form"
	InstructStage StageType = "instruct"
	SessionStage  StageType = "session"
)

type Stage struct {
	Type StageType
	Bot  Bot
	Flag string
	Args []string
}

type Trigger struct {
	Type   TriggerType
	Define string
}

type PipelineOperate string

const (
	PipelineCommandTriggerOperate PipelineOperate = "command_trigger"
	PipelineProcessOperate        PipelineOperate = "pipeline_process"
	PipelineNextOperate           PipelineOperate = "pipeline_next"
)

type Bot string
