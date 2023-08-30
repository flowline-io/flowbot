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
	PageForm     PageType = "form"
	PageChart    PageType = "chart"
	PageTable    PageType = "table"
	PageShare    PageType = "share"
	PageJson     PageType = "json"
	PageHtml     PageType = "html"
	PageMarkdown PageType = "markdown"
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
	InstructObjectLinkit InstructObject = "linkit"
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

type JobState int

const (
	JobStateUnknown JobState = iota
	JobReady
	JobStart
	JobFinished
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
	StepRunning
	StepFinished
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

type Node struct {
	Id        string `json:"id,omitempty"`
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
	Parameters  map[string]interface{} `json:"parameters"`
	Variables   []string               `json:"variables"`
	Connections []string               `json:"connections"`
	Status      NodeStatus             `json:"status,omitempty"`
}

type Edge struct {
	Id                string `json:"id,omitempty"`
	Source            string `json:"source,omitempty"`
	Target            string `json:"target,omitempty"`
	SourcePortId      string `json:"sourcePortId,omitempty"`
	TargetPortId      string `json:"targetPortId,omitempty"`
	Label             string `json:"label,omitempty"`
	EdgeContentWidth  int    `json:"edgeContentWidth,omitempty"`
	EdgeContentHeight int    `json:"edgeContentHeight,omitempty"`
	Connector         struct {
		Name string `json:"name,omitempty"`
	} `json:"connector"`
	Router struct {
		Name string `json:"name,omitempty"`
	} `json:"router"`
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
