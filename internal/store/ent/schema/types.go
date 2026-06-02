// Package schema provides Ent schema definitions and domain types for the store layer.
package schema

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
)

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

// JSON is a map[string]any with database Scan/Value support for JSONB columns.
type JSON map[string]any

func (j *JSON) Scan(value any) error {
	if bytes, ok := value.([]byte); ok {
		result := make(map[string]any)
		err := sonic.Unmarshal(bytes, &result)
		if err != nil {
			return err
		}
		*j = result
		return nil
	}
	if result, ok := value.(map[string]any); ok {
		*j = result
		return nil
	}
	return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
}

func (j *JSON) Value() (driver.Value, error) {
	if len(*j) == 0 {
		return nil, nil
	}
	return sonic.Marshal(*j)
}

// IDList is a []int64 with database Scan/Value support for JSONB array columns.
type IDList []int64

func (j *IDList) Scan(value any) error {
	if bytes, ok := value.([]byte); ok {
		result := make([]int64, 0)
		err := sonic.Unmarshal(bytes, &result)
		if err != nil {
			return err
		}
		*j = result
		return nil
	}
	if result, ok := value.([]int64); ok {
		*j = result
		return nil
	}
	return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
}

func (j *IDList) Value() (driver.Value, error) {
	if len(*j) == 0 {
		return nil, nil
	}
	return sonic.Marshal(*j)
}

// ---------------------------------------------------------------------------
// State and enum types
// ---------------------------------------------------------------------------

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
	PipelineFailed
)

func (j PipelineState) Value() (driver.Value, error) {
	return int64(j), nil
}

// WorkflowRunState represents the execution state of a local workflow engine run.
type WorkflowRunState int

const (
	WorkflowRunStateUnknown WorkflowRunState = iota
	WorkflowRunRunning
	WorkflowRunDone
	WorkflowRunFailed
)

func (j WorkflowRunState) Value() (driver.Value, error) {
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
	TriggerCron   TriggerType = "cron"
	TriggerManual TriggerType = "manual"
)

func (j TriggerType) Value() (driver.Value, error) {
	return string(j), nil
}

type TriggerCronRule struct {
	Spec string `json:"spec"`
}

// NodePort describes a port on a flow node.
type NodePort struct {
	Id        string `json:"id,omitempty"`
	Group     string `json:"group,omitempty"`
	Type      string `json:"type,omitempty"`
	Tooltip   string `json:"tooltip,omitempty"`
	Connected bool   `json:"connected,omitempty"`
}

type Node struct {
	Id          string         `json:"id"`
	Describe    string         `json:"describe"`
	X           int            `json:"x,omitempty"`
	Y           int            `json:"y,omitempty"`
	Width       int            `json:"width,omitempty"`
	Height      int            `json:"height,omitempty"`
	Label       string         `json:"label,omitempty"`
	RenderKey   string         `json:"renderKey,omitempty"`
	IsGroup     bool           `json:"isGroup,omitempty"`
	Group       string         `json:"group,omitempty"`
	ParentId    string         `json:"parentId,omitempty"`
	Ports       []NodePort     `json:"ports,omitempty"`
	Order       int            `json:"_order,omitempty"`
	Bot         string         `json:"bot"`
	RuleId      string         `json:"rule_id"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Variables   []string       `json:"variables,omitempty"`
	Connections []string       `json:"connections,omitempty"`
	Status      NodeStatus     `json:"status,omitempty"`
}

// EdgeConnector describes the connector for an edge.
type EdgeConnector struct {
	Name string `json:"name,omitempty"`
}

// EdgeRouter describes the router for an edge.
type EdgeRouter struct {
	Name string `json:"name,omitempty"`
}

type Edge struct {
	Id                string        `json:"id"`
	Source            string        `json:"source"`
	Target            string        `json:"target"`
	SourcePortId      string        `json:"sourcePortId,omitempty"`
	TargetPortId      string        `json:"targetPortId,omitempty"`
	Label             string        `json:"label,omitempty"`
	EdgeContentWidth  int           `json:"edgeContentWidth,omitempty"`
	EdgeContentHeight int           `json:"edgeContentHeight,omitempty"`
	Connector         EdgeConnector `json:"connector"`
	Router            EdgeRouter    `json:"router"`
	SourcePort        string        `json:"sourcePort,omitempty"`
	TargetPort        string        `json:"targetPort,omitempty"`
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
	FileFailed
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

type AppStatus string

const (
	AppStatusUnknown    AppStatus = "unknown"
	AppStatusRunning    AppStatus = "running"
	AppStatusStopped    AppStatus = "stopped"
	AppStatusPaused     AppStatus = "paused"
	AppStatusRestarting AppStatus = "restarting"
	AppStatusRemoving   AppStatus = "removing"
)

func (j AppStatus) Value() (driver.Value, error) {
	return string(j), nil
}

type FlowState int

const (
	FlowStateUnknown FlowState = iota
	FlowActive
	FlowInactive
)

func (j FlowState) Value() (driver.Value, error) {
	return int64(j), nil
}

type ExecutionState int

const (
	ExecutionStateUnknown ExecutionState = iota
	ExecutionPending
	ExecutionRunning
	ExecutionSucceeded
	ExecutionFailed
	ExecutionCancelled
)

func (j ExecutionState) Value() (driver.Value, error) {
	return int64(j), nil
}

type NodeType string

const (
	NodeTypeTrigger   NodeType = "trigger"
	NodeTypeAction    NodeType = "action"
	NodeTypeFilter    NodeType = "filter"
	NodeTypeCondition NodeType = "condition"
)

func (j NodeType) Value() (driver.Value, error) {
	return string(j), nil
}

type RateLimitType string

const (
	RateLimitTypeFlow RateLimitType = "flow"
	RateLimitTypeNode RateLimitType = "node"
)

func (j RateLimitType) Value() (driver.Value, error) {
	return string(j), nil
}

// ---------------------------------------------------------------------------
// Resource chain types
// ---------------------------------------------------------------------------

// ResourceRelations holds upstream and downstream resource references
// for a specific resource identified by app and entity_id.
type ResourceRelations struct {
	App        string        `json:"app"`
	EntityID   string        `json:"entity_id"`
	Upstream   []ResourceRef `json:"upstream"`
	Downstream []ResourceRef `json:"downstream"`
}

// ResourceRef identifies a resource by app, entity_id, and optional metadata.
type ResourceRef struct {
	App          string `json:"app"`
	EntityID     string `json:"entity_id"`
	Capability   string `json:"capability,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
}

// ResourceEdge represents a directed resource link with full source and target
// details plus pipeline metadata and creation time.
type ResourceEdge struct {
	SourceApp        string    `json:"source_app"`
	SourceCapability string    `json:"source_capability"`
	SourceEntityID   string    `json:"source_entity_id"`
	TargetApp        string    `json:"target_app"`
	TargetCapability string    `json:"target_capability"`
	TargetEntityID   string    `json:"target_entity_id"`
	PipelineName     string    `json:"pipeline_name"`
	CreatedAt        time.Time `json:"created_at"`
}
