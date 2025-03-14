package model

import (
	"database/sql/driver"
)

type FormState int

const (
	FormStateUnknown FormState = iota
	FormStateCreated
	FormStateSubmitSuccess
	FormStateSubmitFailed
)

func (j FormState) Value() (driver.Value, error) {
	return int64(j), nil
}

type ActionState int

const (
	ActionStateUnknown ActionState = iota
	ActionStateLongTerm
	ActionStateSubmitSuccess
	ActionStateSubmitFailed
)

func (j ActionState) Value() (driver.Value, error) {
	return int64(j), nil
}

type SessionState int

const (
	SessionStateUnknown SessionState = iota
	SessionStart
	SessionDone
	SessionCancel
)

func (j SessionState) Value() (driver.Value, error) {
	return int64(j), nil
}

type PageType string

const (
	PageForm  PageType = "form"
	PageTable PageType = "table"
	PageHtml  PageType = "html"
	PageChart PageType = "chart"
)

func (j PageType) Value() (driver.Value, error) {
	return string(j), nil
}

type PageState int

const (
	PageStateUnknown PageState = iota
	PageStateCreated
	PageStateProcessedSuccess
	PageStateProcessedFailed
)

func (j PageState) Value() (driver.Value, error) {
	return int64(j), nil
}

type UrlState int

const (
	UrlStateUnknown UrlState = iota
	UrlStateEnable
	UrlStateDisable
)

func (j UrlState) Value() (driver.Value, error) {
	return int64(j), nil
}

type InstructState int

const (
	InstructStateUnknown InstructState = iota
	InstructCreate
	InstructDone
	InstructCancel
)

func (j InstructState) Value() (driver.Value, error) {
	return int64(j), nil
}

type InstructObject string

const (
	InstructObjectAgent InstructObject = "agent"
)

func (j InstructObject) Value() (driver.Value, error) {
	return string(j), nil
}

type InstructPriority int

const (
	InstructPriorityHigh    InstructPriority = 3
	InstructPriorityDefault InstructPriority = 2
	InstructPriorityLow     InstructPriority = 1
)

func (j InstructPriority) Value() (driver.Value, error) {
	return int64(j), nil
}

type PipelineState int

const (
	PipelineStateUnknown PipelineState = iota
	PipelineStart
	PipelineDone
	PipelineCancel
)

func (j PipelineState) Value() (driver.Value, error) {
	return int64(j), nil
}

type ValueModeType string

const (
	ValueSumMode  ValueModeType = "sum"
	ValueLastMode ValueModeType = "last"
	ValueAvgMode  ValueModeType = "avg"
	ValueMaxMode  ValueModeType = "max"
)

func (j ValueModeType) Value() (driver.Value, error) {
	return string(j), nil
}

type CycleState int

const (
	CycleStateUnknown CycleState = iota
	CycleStart
	CycleDone
	CycleCancel
)

func (j CycleState) Value() (driver.Value, error) {
	return int64(j), nil
}

type ReviewType int

const (
	ReviewTypeUnknown ReviewType = iota
	ReviewMid
	ReviewEnd
)

func (j ReviewType) Value() (driver.Value, error) {
	return int64(j), nil
}

type WorkflowState int

const (
	WorkflowStateUnknown WorkflowState = iota
	WorkflowEnable
	WorkflowDisable
)

func (j WorkflowState) Value() (driver.Value, error) {
	return int64(j), nil
}

type WorkflowTriggerState int

const (
	WorkflowTriggerStateUnknown WorkflowTriggerState = iota
	WorkflowTriggerEnable
	WorkflowTriggerDisable
)

func (j WorkflowTriggerState) Value() (driver.Value, error) {
	return int64(j), nil
}

type WorkflowScriptLang string

const (
	WorkflowScriptYaml WorkflowScriptLang = "yaml"
)

func (j WorkflowScriptLang) Value() (driver.Value, error) {
	return string(j), nil
}

type JobState int

const (
	JobStateUnknown JobState = iota
	JobReady
	JobStart
	JobRunning
	JobSucceeded
	JobCanceled
	JobFailed
)

func (j JobState) Value() (driver.Value, error) {
	return int64(j), nil
}

type StepState int

const (
	StepStateUnknown StepState = iota
	StepCreated
	StepReady
	StepStart
	StepRunning
	StepSucceeded
	StepFailed
	StepCanceled
	StepSkipped
)

func (j StepState) Value() (driver.Value, error) {
	return int64(j), nil
}

type TriggerType string

const (
	TriggerCron    TriggerType = "cron"
	TriggerManual  TriggerType = "manual"
	TriggerWebhook TriggerType = "webhook"
)

func (j TriggerType) Value() (driver.Value, error) {
	return string(j), nil
}

type TriggerCronRule struct {
	Spec string `json:"spec"`
}

type Node struct {
	Id        string `json:"id"`
	Describe  string `json:"describe"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Label     string `json:"label,omitempty"`
	RenderKey string `json:"renderKey,omitempty"`
	IsGroup   bool   `json:"isGroup,omitempty"`
	Group     string `json:"group,omitempty"`
	ParentId  string `json:"parentId,omitempty"`
	Ports     []struct {
		Id        string `json:"id,omitempty"`
		Group     string `json:"group,omitempty"`
		Type      string `json:"type,omitempty"`
		Tooltip   string `json:"tooltip,omitempty"`
		Connected bool   `json:"connected,omitempty"`
	} `json:"ports,omitempty"`
	Order       int                    `json:"_order,omitempty"`
	Bot         string                 `json:"bot"`
	RuleId      string                 `json:"rule_id"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Variables   []string               `json:"variables,omitempty"`
	Connections []string               `json:"connections,omitempty"`
	Status      NodeStatus             `json:"status,omitempty"`
}

type Edge struct {
	Id                string `json:"id"`
	Source            string `json:"source"`
	Target            string `json:"target"`
	SourcePortId      string `json:"sourcePortId,omitempty"`
	TargetPortId      string `json:"targetPortId,omitempty"`
	Label             string `json:"label,omitempty"`
	EdgeContentWidth  int    `json:"edgeContentWidth,omitempty"`
	EdgeContentHeight int    `json:"edgeContentHeight,omitempty"`
	Connector         struct {
		Name string `json:"name,omitempty"`
	} `json:"connector,omitempty"`
	Router struct {
		Name string `json:"name,omitempty"`
	} `json:"router,omitempty"`
	SourcePort string `json:"sourcePort,omitempty"`
	TargetPort string `json:"targetPort,omitempty"`
}

type NodeStatus string

const (
	NodeDefault    NodeStatus = "default"
	NodeSuccess    NodeStatus = "success"
	NodeProcessing NodeStatus = "processing"
	NodeError      NodeStatus = "error"
	NodeWarning    NodeStatus = "warning"
)

func (j NodeStatus) Value() (driver.Value, error) {
	return string(j), nil
}

type UserState int

const (
	UserStateUnknown UserState = iota
	UserActive
	UserInactive
)

func (j UserState) Value() (driver.Value, error) {
	return int64(j), nil
}

type TopicState int

func (j TopicState) Value() (driver.Value, error) {
	return int64(j), nil
}

type MessageState int

const (
	MessageStateUnknown MessageState = iota
	MessageCreated
)

func (j MessageState) Value() (driver.Value, error) {
	return int64(j), nil
}

type FileState int

const (
	FileStateUnknown FileState = iota
	FileStart
	FileFinish
)

func (j FileState) Value() (driver.Value, error) {
	return int64(j), nil
}

type BotState int

const (
	BotStateUnknown BotState = iota
	BotActive
	BotInactive
)

func (j BotState) Value() (driver.Value, error) {
	return int64(j), nil
}

type ChannelState int

const (
	ChannelStateUnknown ChannelState = iota
	ChannelActive
	ChannelInactive
)

func (j ChannelState) Value() (driver.Value, error) {
	return int64(j), nil
}

type WebhookState int

const (
	WebhookStateUnknown WebhookState = iota
	WebhookActive
	WebhookInactive
)

func (j WebhookState) Value() (driver.Value, error) {
	return int64(j), nil
}
