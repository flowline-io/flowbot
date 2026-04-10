package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test FormState
func TestFormState(t *testing.T) {
	tests := []struct {
		state    FormState
		expected int64
	}{
		{FormStateUnknown, 0},
		{FormStateCreated, 1},
		{FormStateSubmitSuccess, 2},
		{FormStateSubmitFailed, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (f FormState) String() string {
	switch f {
	case FormStateUnknown:
		return "FormStateUnknown"
	case FormStateCreated:
		return "FormStateCreated"
	case FormStateSubmitSuccess:
		return "FormStateSubmitSuccess"
	case FormStateSubmitFailed:
		return "FormStateSubmitFailed"
	default:
		return "Unknown"
	}
}

// Test ActionState
func TestActionState(t *testing.T) {
	tests := []struct {
		state    ActionState
		expected int64
	}{
		{ActionStateUnknown, 0},
		{ActionStateLongTerm, 1},
		{ActionStateSubmitSuccess, 2},
		{ActionStateSubmitFailed, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (a ActionState) String() string {
	switch a {
	case ActionStateUnknown:
		return "ActionStateUnknown"
	case ActionStateLongTerm:
		return "ActionStateLongTerm"
	case ActionStateSubmitSuccess:
		return "ActionStateSubmitSuccess"
	case ActionStateSubmitFailed:
		return "ActionStateSubmitFailed"
	default:
		return "Unknown"
	}
}

// Test SessionState
func TestSessionState(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected int64
	}{
		{SessionStateUnknown, 0},
		{SessionStart, 1},
		{SessionDone, 2},
		{SessionCancel, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (s SessionState) String() string {
	switch s {
	case SessionStateUnknown:
		return "SessionStateUnknown"
	case SessionStart:
		return "SessionStart"
	case SessionDone:
		return "SessionDone"
	case SessionCancel:
		return "SessionCancel"
	default:
		return "Unknown"
	}
}

// Test PageType
func TestPageType(t *testing.T) {
	tests := []struct {
		pageType PageType
		expected string
	}{
		{PageForm, "form"},
		{PageTable, "table"},
		{PageHtml, "html"},
	}

	for _, tt := range tests {
		t.Run(string(tt.pageType), func(t *testing.T) {
			val, err := tt.pageType.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test PageState
func TestPageState(t *testing.T) {
	tests := []struct {
		state    PageState
		expected int64
	}{
		{PageStateUnknown, 0},
		{PageStateCreated, 1},
		{PageStateProcessedSuccess, 2},
		{PageStateProcessedFailed, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (p PageState) String() string {
	switch p {
	case PageStateUnknown:
		return "PageStateUnknown"
	case PageStateCreated:
		return "PageStateCreated"
	case PageStateProcessedSuccess:
		return "PageStateProcessedSuccess"
	case PageStateProcessedFailed:
		return "PageStateProcessedFailed"
	default:
		return "Unknown"
	}
}

// Test UrlState
func TestUrlState(t *testing.T) {
	tests := []struct {
		state    UrlState
		expected int64
	}{
		{UrlStateUnknown, 0},
		{UrlStateEnable, 1},
		{UrlStateDisable, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (u UrlState) String() string {
	switch u {
	case UrlStateUnknown:
		return "UrlStateUnknown"
	case UrlStateEnable:
		return "UrlStateEnable"
	case UrlStateDisable:
		return "UrlStateDisable"
	default:
		return "Unknown"
	}
}

// Test InstructState
func TestInstructState(t *testing.T) {
	tests := []struct {
		state    InstructState
		expected int64
	}{
		{InstructStateUnknown, 0},
		{InstructCreate, 1},
		{InstructDone, 2},
		{InstructCancel, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (i InstructState) String() string {
	switch i {
	case InstructStateUnknown:
		return "InstructStateUnknown"
	case InstructCreate:
		return "InstructCreate"
	case InstructDone:
		return "InstructDone"
	case InstructCancel:
		return "InstructCancel"
	default:
		return "Unknown"
	}
}

// Test InstructObject
func TestInstructObject(t *testing.T) {
	tests := []struct {
		obj      InstructObject
		expected string
	}{
		{InstructObjectAgent, "agent"},
	}

	for _, tt := range tests {
		t.Run(string(tt.obj), func(t *testing.T) {
			val, err := tt.obj.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test InstructPriority
func TestInstructPriority(t *testing.T) {
	tests := []struct {
		priority InstructPriority
		expected int64
	}{
		{InstructPriorityLow, 1},
		{InstructPriorityDefault, 2},
		{InstructPriorityHigh, 3},
	}

	for _, tt := range tests {
		t.Run(tt.priority.String(), func(t *testing.T) {
			val, err := tt.priority.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (i InstructPriority) String() string {
	switch i {
	case InstructPriorityLow:
		return "InstructPriorityLow"
	case InstructPriorityDefault:
		return "InstructPriorityDefault"
	case InstructPriorityHigh:
		return "InstructPriorityHigh"
	default:
		return "Unknown"
	}
}

// Test PipelineState
func TestPipelineState(t *testing.T) {
	tests := []struct {
		state    PipelineState
		expected int64
	}{
		{PipelineStateUnknown, 0},
		{PipelineStart, 1},
		{PipelineDone, 2},
		{PipelineCancel, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (p PipelineState) String() string {
	switch p {
	case PipelineStateUnknown:
		return "PipelineStateUnknown"
	case PipelineStart:
		return "PipelineStart"
	case PipelineDone:
		return "PipelineDone"
	case PipelineCancel:
		return "PipelineCancel"
	default:
		return "Unknown"
	}
}

// Test ValueModeType
func TestValueModeType(t *testing.T) {
	tests := []struct {
		mode     ValueModeType
		expected string
	}{
		{ValueSumMode, "sum"},
		{ValueLastMode, "last"},
		{ValueAvgMode, "avg"},
		{ValueMaxMode, "max"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			val, err := tt.mode.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test CycleState
func TestCycleState(t *testing.T) {
	tests := []struct {
		state    CycleState
		expected int64
	}{
		{CycleStateUnknown, 0},
		{CycleStart, 1},
		{CycleDone, 2},
		{CycleCancel, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (c CycleState) String() string {
	switch c {
	case CycleStateUnknown:
		return "CycleStateUnknown"
	case CycleStart:
		return "CycleStart"
	case CycleDone:
		return "CycleDone"
	case CycleCancel:
		return "CycleCancel"
	default:
		return "Unknown"
	}
}

// Test ReviewType
func TestReviewType(t *testing.T) {
	tests := []struct {
		reviewType ReviewType
		expected   int64
	}{
		{ReviewTypeUnknown, 0},
		{ReviewMid, 1},
		{ReviewEnd, 2},
	}

	for _, tt := range tests {
		t.Run(tt.reviewType.String(), func(t *testing.T) {
			val, err := tt.reviewType.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (r ReviewType) String() string {
	switch r {
	case ReviewTypeUnknown:
		return "ReviewTypeUnknown"
	case ReviewMid:
		return "ReviewMid"
	case ReviewEnd:
		return "ReviewEnd"
	default:
		return "Unknown"
	}
}

// Test WorkflowState
func TestWorkflowState(t *testing.T) {
	tests := []struct {
		state    WorkflowState
		expected int64
	}{
		{WorkflowStateUnknown, 0},
		{WorkflowEnable, 1},
		{WorkflowDisable, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (w WorkflowState) String() string {
	switch w {
	case WorkflowStateUnknown:
		return "WorkflowStateUnknown"
	case WorkflowEnable:
		return "WorkflowEnable"
	case WorkflowDisable:
		return "WorkflowDisable"
	default:
		return "Unknown"
	}
}

// Test WorkflowTriggerState
func TestWorkflowTriggerState(t *testing.T) {
	tests := []struct {
		state    WorkflowTriggerState
		expected int64
	}{
		{WorkflowTriggerStateUnknown, 0},
		{WorkflowTriggerEnable, 1},
		{WorkflowTriggerDisable, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (w WorkflowTriggerState) String() string {
	switch w {
	case WorkflowTriggerStateUnknown:
		return "WorkflowTriggerStateUnknown"
	case WorkflowTriggerEnable:
		return "WorkflowTriggerEnable"
	case WorkflowTriggerDisable:
		return "WorkflowTriggerDisable"
	default:
		return "Unknown"
	}
}

// Test WorkflowScriptLang
func TestWorkflowScriptLang(t *testing.T) {
	tests := []struct {
		lang     WorkflowScriptLang
		expected string
	}{
		{WorkflowScriptYaml, "yaml"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			val, err := tt.lang.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test JobState
func TestJobState(t *testing.T) {
	tests := []struct {
		state    JobState
		expected int64
	}{
		{JobStateUnknown, 0},
		{JobReady, 1},
		{JobStart, 2},
		{JobRunning, 3},
		{JobSucceeded, 4},
		{JobCanceled, 5},
		{JobFailed, 6},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (j JobState) String() string {
	switch j {
	case JobStateUnknown:
		return "JobStateUnknown"
	case JobReady:
		return "JobReady"
	case JobStart:
		return "JobStart"
	case JobRunning:
		return "JobRunning"
	case JobSucceeded:
		return "JobSucceeded"
	case JobCanceled:
		return "JobCanceled"
	case JobFailed:
		return "JobFailed"
	default:
		return "Unknown"
	}
}

// Test StepState
func TestStepState(t *testing.T) {
	tests := []struct {
		state    StepState
		expected int64
	}{
		{StepStateUnknown, 0},
		{StepCreated, 1},
		{StepReady, 2},
		{StepStart, 3},
		{StepRunning, 4},
		{StepSucceeded, 5},
		{StepFailed, 6},
		{StepCanceled, 7},
		{StepSkipped, 8},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (s StepState) String() string {
	switch s {
	case StepStateUnknown:
		return "StepStateUnknown"
	case StepCreated:
		return "StepCreated"
	case StepReady:
		return "StepReady"
	case StepStart:
		return "StepStart"
	case StepRunning:
		return "StepRunning"
	case StepSucceeded:
		return "StepSucceeded"
	case StepFailed:
		return "StepFailed"
	case StepCanceled:
		return "StepCanceled"
	case StepSkipped:
		return "StepSkipped"
	default:
		return "Unknown"
	}
}

// Test TriggerType
func TestTriggerType(t *testing.T) {
	tests := []struct {
		trigger  TriggerType
		expected string
	}{
		{TriggerCron, "cron"},
		{TriggerManual, "manual"},
		{TriggerWebhook, "webhook"},
	}

	for _, tt := range tests {
		t.Run(string(tt.trigger), func(t *testing.T) {
			val, err := tt.trigger.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test TriggerCronRule
func TestTriggerCronRule(t *testing.T) {
	rule := TriggerCronRule{
		Spec: "0 0 * * *",
	}
	assert.Equal(t, "0 0 * * *", rule.Spec)
}

// Test NodeStatus
func TestNodeStatus(t *testing.T) {
	tests := []struct {
		status   NodeStatus
		expected string
	}{
		{NodeDefault, "default"},
		{NodeSuccess, "success"},
		{NodeProcessing, "processing"},
		{NodeError, "error"},
		{NodeWarning, "warning"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			val, err := tt.status.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test UserState
func TestUserState(t *testing.T) {
	tests := []struct {
		state    UserState
		expected int64
	}{
		{UserStateUnknown, 0},
		{UserActive, 1},
		{UserInactive, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (u UserState) String() string {
	switch u {
	case UserStateUnknown:
		return "UserStateUnknown"
	case UserActive:
		return "UserActive"
	case UserInactive:
		return "UserInactive"
	default:
		return "Unknown"
	}
}

// Test TopicState
func TestTopicState(t *testing.T) {
	state := TopicState(1)
	val, err := state.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

// Test MessageState
func TestMessageState(t *testing.T) {
	tests := []struct {
		state    MessageState
		expected int64
	}{
		{MessageStateUnknown, 0},
		{MessageCreated, 1},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (m MessageState) String() string {
	switch m {
	case MessageStateUnknown:
		return "MessageStateUnknown"
	case MessageCreated:
		return "MessageCreated"
	default:
		return "Unknown"
	}
}

// Test FileState
func TestFileState(t *testing.T) {
	tests := []struct {
		state    FileState
		expected int64
	}{
		{FileStateUnknown, 0},
		{FileStart, 1},
		{FileFinish, 2},
		{FileFailed, 3},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (f FileState) String() string {
	switch f {
	case FileStateUnknown:
		return "FileStateUnknown"
	case FileStart:
		return "FileStart"
	case FileFinish:
		return "FileFinish"
	case FileFailed:
		return "FileFailed"
	default:
		return "Unknown"
	}
}

// Test BotState
func TestBotState(t *testing.T) {
	tests := []struct {
		state    BotState
		expected int64
	}{
		{BotStateUnknown, 0},
		{BotActive, 1},
		{BotInactive, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (b BotState) String() string {
	switch b {
	case BotStateUnknown:
		return "BotStateUnknown"
	case BotActive:
		return "BotActive"
	case BotInactive:
		return "BotInactive"
	default:
		return "Unknown"
	}
}

// Test ChannelState
func TestChannelState(t *testing.T) {
	tests := []struct {
		state    ChannelState
		expected int64
	}{
		{ChannelStateUnknown, 0},
		{ChannelActive, 1},
		{ChannelInactive, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (c ChannelState) String() string {
	switch c {
	case ChannelStateUnknown:
		return "ChannelStateUnknown"
	case ChannelActive:
		return "ChannelActive"
	case ChannelInactive:
		return "ChannelInactive"
	default:
		return "Unknown"
	}
}

// Test WebhookState
func TestWebhookState(t *testing.T) {
	tests := []struct {
		state    WebhookState
		expected int64
	}{
		{WebhookStateUnknown, 0},
		{WebhookActive, 1},
		{WebhookInactive, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (w WebhookState) String() string {
	switch w {
	case WebhookStateUnknown:
		return "WebhookStateUnknown"
	case WebhookActive:
		return "WebhookActive"
	case WebhookInactive:
		return "WebhookInactive"
	default:
		return "Unknown"
	}
}

// Test AppStatus
func TestAppStatus(t *testing.T) {
	tests := []struct {
		status   AppStatus
		expected string
	}{
		{AppStatusUnknown, "unknown"},
		{AppStatusRunning, "running"},
		{AppStatusStopped, "stopped"},
		{AppStatusPaused, "paused"},
		{AppStatusRestarting, "restarting"},
		{AppStatusRemoving, "removing"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			val, err := tt.status.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test FlowState
func TestFlowState(t *testing.T) {
	tests := []struct {
		state    FlowState
		expected int64
	}{
		{FlowStateUnknown, 0},
		{FlowActive, 1},
		{FlowInactive, 2},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (f FlowState) String() string {
	switch f {
	case FlowStateUnknown:
		return "FlowStateUnknown"
	case FlowActive:
		return "FlowActive"
	case FlowInactive:
		return "FlowInactive"
	default:
		return "Unknown"
	}
}

// Test ExecutionState
func TestExecutionState(t *testing.T) {
	tests := []struct {
		state    ExecutionState
		expected int64
	}{
		{ExecutionStateUnknown, 0},
		{ExecutionPending, 1},
		{ExecutionRunning, 2},
		{ExecutionSucceeded, 3},
		{ExecutionFailed, 4},
		{ExecutionCancelled, 5},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			val, err := tt.state.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func (e ExecutionState) String() string {
	switch e {
	case ExecutionStateUnknown:
		return "ExecutionStateUnknown"
	case ExecutionPending:
		return "ExecutionPending"
	case ExecutionRunning:
		return "ExecutionRunning"
	case ExecutionSucceeded:
		return "ExecutionSucceeded"
	case ExecutionFailed:
		return "ExecutionFailed"
	case ExecutionCancelled:
		return "ExecutionCancelled"
	default:
		return "Unknown"
	}
}

// Test NodeType
func TestNodeType(t *testing.T) {
	tests := []struct {
		nodeType NodeType
		expected string
	}{
		{NodeTypeTrigger, "trigger"},
		{NodeTypeAction, "action"},
		{NodeTypeFilter, "filter"},
		{NodeTypeCondition, "condition"},
	}

	for _, tt := range tests {
		t.Run(string(tt.nodeType), func(t *testing.T) {
			val, err := tt.nodeType.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test RateLimitType
func TestRateLimitType(t *testing.T) {
	tests := []struct {
		rlType   RateLimitType
		expected string
	}{
		{RateLimitTypeFlow, "flow"},
		{RateLimitTypeNode, "node"},
	}

	for _, tt := range tests {
		t.Run(string(tt.rlType), func(t *testing.T) {
			val, err := tt.rlType.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// Test Node struct
func TestNode(t *testing.T) {
	node := Node{
		Id:        "node-1",
		Describe:  "Test Node",
		X:         100,
		Y:         200,
		Width:     300,
		Height:    400,
		Label:     "Test",
		RenderKey: "render-1",
		IsGroup:   false,
		Group:     "group-1",
		ParentId:  "parent-1",
		Ports: []struct {
			Id        string `json:"id,omitempty"`
			Group     string `json:"group,omitempty"`
			Type      string `json:"type,omitempty"`
			Tooltip   string `json:"tooltip,omitempty"`
			Connected bool   `json:"connected,omitempty"`
		}{
			{Id: "port-1", Group: "group", Type: "type", Tooltip: "tooltip", Connected: true},
		},
		Order:       1,
		Bot:         "test-bot",
		RuleId:      "rule-1",
		Parameters:  map[string]any{"key": "value"},
		Variables:   []string{"var1", "var2"},
		Connections: []string{"conn1", "conn2"},
		Status:      NodeSuccess,
	}

	assert.Equal(t, "node-1", node.Id)
	assert.Equal(t, "Test Node", node.Describe)
	assert.Equal(t, 100, node.X)
	assert.Equal(t, 200, node.Y)
	assert.Equal(t, NodeSuccess, node.Status)
}

// Test Edge struct
func TestEdge(t *testing.T) {
	edge := Edge{
		Id:                "edge-1",
		Source:            "source-node",
		Target:            "target-node",
		SourcePortId:      "source-port",
		TargetPortId:      "target-port",
		Label:             "Test Edge",
		EdgeContentWidth:  100,
		EdgeContentHeight: 50,
	}

	assert.Equal(t, "edge-1", edge.Id)
	assert.Equal(t, "source-node", edge.Source)
	assert.Equal(t, "target-node", edge.Target)
	assert.Equal(t, "Test Edge", edge.Label)
}

// Test Parameter IsExpired
func TestParameter_IsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		expiredAt time.Time
		expected  bool
	}{
		{
			name:      "expired",
			expiredAt: past,
			expected:  true,
		},
		{
			name:      "not expired",
			expiredAt: future,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parameter{
				ExpiredAt: tt.expiredAt,
			}
			assert.Equal(t, tt.expected, p.IsExpired())
		})
	}
}

// Test Job MarshalBinary
func TestJob_MarshalBinary(t *testing.T) {
	job := &Job{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := job.MarshalBinary()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Greater(t, len(data), 0)
}
