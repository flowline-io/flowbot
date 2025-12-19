package flows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/flows"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/lithammer/shortuuid/v4"
	"gorm.io/gorm"
)

// Engine executes IF THEN flows
type Engine struct {
	store store.Adapter
	reg   RuleRegistry
	rnd   TemplateRenderer

	maxDepth int
}

// NewEngine creates a new flow engine
func NewEngine(storeAdapter store.Adapter, reg RuleRegistry, renderer TemplateRenderer) *Engine {
	if reg == nil {
		reg = NewChatbotRuleRegistry()
	}
	if renderer == nil {
		renderer = NewSimpleTemplateRenderer()
	}
	return &Engine{store: storeAdapter, reg: reg, rnd: renderer, maxDepth: 64}
}

func (e *Engine) SetMaxDepth(n int) {
	if e == nil {
		return
	}
	if n <= 0 {
		e.maxDepth = 64
		return
	}
	e.maxDepth = n
}

// ExecuteFlow executes a flow with the given trigger
func (e *Engine) ExecuteFlow(ctx context.Context, flowID int64, triggerType string, triggerID string, payload types.KV) error {
	_, err := e.ExecuteFlowWithExecutionID(ctx, flowID, "", triggerType, triggerID, payload)
	return err
}

// ExecuteFlowWithExecutionID executes a flow using a caller-provided executionID.
// If executionID is provided and an execution already exists, it will be reused and updated.
// This makes queue retries idempotent (one queued job => one execution row).
func (e *Engine) ExecuteFlowWithExecutionID(ctx context.Context, flowID int64, executionID string, triggerType string, triggerID string, payload types.KV) (_ string, err error) {
	var execution *model.Execution

	defer func() {
		r := recover()
		if r == nil {
			return
		}

		stack := debug.Stack()
		flog.Error(fmt.Errorf("flow execution panicked: %v\n%s", r, string(stack)))

		if execution != nil {
			execution.State = model.ExecutionFailed
			execution.Error = fmt.Sprintf("panic: %v", r)
			finishedAt := time.Now()
			execution.FinishedAt = &finishedAt
			_ = e.store.UpdateExecution(execution)
		}

		err = fmt.Errorf("flow execution panicked: %v", r)
	}()

	flow, err := e.store.GetFlow(flowID)
	if err != nil {
		return "", fmt.Errorf("failed to get flow: %w", err)
	}

	if !flow.Enabled {
		return "", fmt.Errorf("flow is disabled")
	}

	if executionID == "" {
		executionID = shortuuid.New()
	}

	// Create (or reuse) execution record.
	execution, err = e.store.GetExecution(executionID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return executionID, fmt.Errorf("failed to get execution: %w", err)
		}

		execution = &model.Execution{
			FlowID:      flowID,
			ExecutionID: executionID,
			TriggerType: triggerType,
			TriggerID:   triggerID,
			State:       model.ExecutionPending,
			Payload:     model.JSON(payload),
			Variables:   model.JSON(make(types.KV)),
		}

		now := time.Now()
		execution.StartedAt = &now
		execution.State = model.ExecutionRunning

		_, err = e.store.CreateExecution(execution)
		if err != nil {
			return executionID, fmt.Errorf("failed to create execution: %w", err)
		}
	} else {
		// Reuse existing execution row (e.g., queue retry) instead of inserting a new one.
		if execution.FlowID != flowID {
			return executionID, fmt.Errorf("execution_id is bound to a different flow")
		}
		if execution.StartedAt == nil {
			now := time.Now()
			execution.StartedAt = &now
		}
		execution.TriggerType = triggerType
		execution.TriggerID = triggerID
		execution.Payload = model.JSON(payload)
		execution.State = model.ExecutionRunning
		execution.Error = ""
		execution.FinishedAt = nil
		_ = e.store.UpdateExecution(execution)
	}

	// Execute flow nodes
	err = e.executeNodes(ctx, flow, execution, payload)
	if err != nil {
		execution.State = model.ExecutionFailed
		execution.Error = err.Error()
		finishedAt := time.Now()
		execution.FinishedAt = &finishedAt
		_ = e.store.UpdateExecution(execution)
		return executionID, fmt.Errorf("failed to execute flow: %w", err)
	}

	execution.State = model.ExecutionSucceeded
	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt
	return executionID, e.store.UpdateExecution(execution)
}

// executeNodes executes flow nodes in order
func (e *Engine) executeNodes(ctx context.Context, flow *model.Flow, execution *model.Execution, initialPayload types.KV) error {
	// Get all nodes
	nodes, err := e.store.GetFlowNodes(flow.ID)
	if err != nil {
		return fmt.Errorf("failed to get flow nodes: %w", err)
	}

	// Get all edges
	edges, err := e.store.GetFlowEdges(flow.ID)
	if err != nil {
		return fmt.Errorf("failed to get flow edges: %w", err)
	}

	// Build node graph
	nodeMap := make(map[string]*model.FlowNode)
	for _, node := range nodes {
		nodeMap[node.NodeID] = node
	}

	edgeMap := make(map[string][]*model.FlowEdge)
	for _, edge := range edges {
		edgeMap[edge.SourceNode] = append(edgeMap[edge.SourceNode], edge)
	}

	// Find trigger nodes
	var triggerNodes []*model.FlowNode
	for _, node := range nodes {
		if node.Type == model.NodeTypeTrigger {
			triggerNodes = append(triggerNodes, node)
		}
	}

	if len(triggerNodes) == 0 {
		return fmt.Errorf("no trigger nodes found")
	}

	// Execute from trigger nodes
	if initialPayload == nil {
		initialPayload = types.KV{}
	}

	variables := make(types.KV)
	payloadData, err := json.Marshal(initialPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal initial payload: %w", err)
	}
	if err := json.Unmarshal(payloadData, &variables); err != nil {
		return fmt.Errorf("failed to unmarshal initial payload: %w", err)
	}
	if variables == nil {
		variables = make(types.KV)
	}
	variables["payload"] = initialPayload

	// Add trigger metadata for templates and conditions.
	variables["__trigger_type"] = execution.TriggerType
	variables["__trigger_id"] = execution.TriggerID

	// Only execute the trigger node that matches the incoming trigger.
	triggerBot, triggerRule := normalizeFlowTriggerType(execution.TriggerType)
	triggerKey := ""
	if triggerBot != "" && triggerRule != "" {
		triggerKey = fmt.Sprintf("%s|%s", triggerBot, triggerRule)
	}

	for _, triggerNode := range triggerNodes {
		if triggerKey != "" {
			k := fmt.Sprintf("%s|%s", triggerNode.Bot, triggerNode.RuleID)
			if k != triggerKey {
				continue
			}
		}
		if err := e.executeNodeChain(ctx, flow, triggerNode, nodeMap, edgeMap, variables, execution, 0, make(map[string]bool)); err != nil {
			return err
		}
	}

	// Update execution variables
	execution.Variables = model.JSON(variables)
	return nil
}

// executeNodeChain executes a chain of nodes starting from a trigger node
// maxDepth limits the chain depth to 2-3 nodes
func (e *Engine) executeNodeChain(ctx context.Context, flow *model.Flow, node *model.FlowNode, nodeMap map[string]*model.FlowNode, edgeMap map[string][]*model.FlowEdge, variables types.KV, execution *model.Execution, depth int, path map[string]bool) error {
	maxDepth := e.maxDepth
	if maxDepth <= 0 {
		maxDepth = 64
	}
	if depth > maxDepth {
		return fmt.Errorf("max chain depth exceeded (%d)", maxDepth)
	}
	if node == nil {
		return nil
	}
	if path != nil {
		if path[node.NodeID] {
			return fmt.Errorf("cycle detected at node '%s'", node.NodeID)
		}
		path[node.NodeID] = true
		defer delete(path, node.NodeID)
	}

	// Execute current node with job logging
	job := &model.FlowJob{
		FlowID:      node.FlowID,
		ExecutionID: execution.ExecutionID,
		NodeID:      node.NodeID,
		NodeType:    node.Type,
		Bot:         node.Bot,
		RuleID:      node.RuleID,
		Attempt:     1,
		State:       model.JobStart,
		Params:      node.Parameters,
	}
	now := time.Now()
	job.StartedAt = &now
	job.CreatedAt = now
	job.UpdatedAt = now
	_, _ = e.store.CreateFlowJob(job)

	result, err := e.executeNode(ctx, flow, node, variables)
	if err != nil {
		job.State = model.JobFailed
		job.Error = err.Error()
		finishedAt := time.Now()
		job.FinishedAt = &finishedAt
		job.UpdatedAt = finishedAt
		_ = e.store.UpdateFlowJob(job)
		return fmt.Errorf("failed to execute node %s: %w", node.NodeID, err)
	}

	job.State = model.JobSucceeded
	if result != nil {
		res := make(map[string]interface{}, len(result))
		for k, v := range result {
			res[k] = v
		}
		job.Result = model.JSON(res)
	}
	finishedAt := time.Now()
	job.FinishedAt = &finishedAt
	job.UpdatedAt = finishedAt
	_ = e.store.UpdateFlowJob(job)

	// Merge result into variables
	for k, v := range result {
		variables[k] = v
	}

	// Execute connected nodes
	edges := edgeMap[node.NodeID]
	for _, edge := range edges {
		targetNode, ok := nodeMap[edge.TargetNode]
		if !ok {
			continue
		}

		// Check conditions if this is a filter/condition node
		if targetNode.Type == model.NodeTypeFilter || targetNode.Type == model.NodeTypeCondition {
			shouldContinue, err := e.evaluateConditions(targetNode, variables)
			if err != nil {
				return fmt.Errorf("failed to evaluate conditions: %w", err)
			}
			if !shouldContinue {
				continue
			}
		}

		// Execute next node in chain
		if err := e.executeNodeChain(ctx, flow, targetNode, nodeMap, edgeMap, variables, execution, depth+1, path); err != nil {
			return err
		}
	}

	return nil
}

// executeNode executes a single node

func (e *Engine) executeNode(ctx context.Context, flow *model.Flow, node *model.FlowNode, variables types.KV) (types.KV, error) {
	// Prepare parameters with variable substitution
	params := e.prepareParameters(node.Parameters, variables)

	// Create context
	ctxWithVars := types.Context{
		AsUser: types.Uid(flow.UID),
		Topic:  flow.Topic,
	}
	ctxWithVars.SetTimeout(2 * time.Minute)

	// Execute node based on type
	switch node.Type {
	case model.NodeTypeTrigger:
		r, err := e.reg.FindTrigger(node.Bot, node.RuleID)
		if err != nil {
			return nil, err
		}
		if r.Config != nil {
			if err := r.Config(params); err != nil {
				return nil, err
			}
		}
		payload := make(types.KV)
		// Variables include initial payload; expose it for triggers.
		if v, ok := variables["payload"].(types.KV); ok {
			payload = v
		} else if v, ok := variables["payload"].(map[string]any); ok {
			payload = v
		}
		out, err := r.Extract(ctxWithVars, params, payload)
		if err != nil {
			return nil, err
		}
		return out, nil
	case model.NodeTypeAction:
		r, err := e.reg.FindAction(node.Bot, node.RuleID)
		if err != nil {
			return nil, err
		}
		if r.Validate != nil {
			if err := r.Validate(params); err != nil {
				return nil, err
			}
		} else {
			if err := flows.ValidateParams(params, r.Inputs); err != nil {
				return nil, err
			}
		}
		return r.Run(ctxWithVars, params, variables)
	case model.NodeTypeFilter, model.NodeTypeCondition:
		// Conditions are evaluated separately
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// evaluateConditions evaluates filter/condition nodes
func (e *Engine) evaluateConditions(node *model.FlowNode, variables types.KV) (bool, error) {
	if node.Conditions == nil {
		return true, nil
	}

	// Parse conditions
	var conditions []Condition
	condData, _ := json.Marshal(node.Conditions)
	if err := json.Unmarshal(condData, &conditions); err != nil {
		return false, fmt.Errorf("failed to parse conditions: %w", err)
	}

	// Evaluate all conditions (AND logic)
	for _, cond := range conditions {
		value, ok := variables[cond.Variable]
		if !ok {
			return false, nil
		}

		if !evaluateCondition(value, cond.Operator, cond.Value) {
			return false, nil
		}
	}

	return true, nil
}

// evaluateCondition evaluates a single condition
func evaluateCondition(value interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq", "==":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case "ne", "!=":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case "gt", ">":
		return compareNumbers(value, expected) > 0
	case "gte", ">=":
		return compareNumbers(value, expected) >= 0
	case "lt", "<":
		return compareNumbers(value, expected) < 0
	case "lte", "<=":
		return compareNumbers(value, expected) <= 0
	case "contains":
		return containsString(value, expected)
	default:
		return false
	}
}

// prepareParameters prepares node parameters with variable substitution
func (e *Engine) prepareParameters(params model.JSON, variables types.KV) types.KV {
	if params == nil {
		return make(types.KV)
	}

	var paramMap types.KV
	paramData, err := json.Marshal(params)
	if err != nil {
		return make(types.KV)
	}
	if err := json.Unmarshal(paramData, &paramMap); err != nil {
		return make(types.KV)
	}

	// Substitute variables
	result := make(types.KV)
	for k, v := range paramMap {
		result[k] = e.renderValue(v, variables)
	}

	return result
}

func (e *Engine) renderValue(value any, variables types.KV) any {
	switch v := value.(type) {
	case string:
		return e.rnd.Render(v, variables)
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, vv := range v {
			out[k] = e.renderValue(vv, variables)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = e.renderValue(v[i], variables)
		}
		return out
	default:
		return value
	}
}

// Condition represents a filter condition
type Condition struct {
	Variable string      `json:"variable"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// Helper functions
func compareNumbers(a, b interface{}) int {
	// Convert both values to float64 for comparison
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)

	if aFloat > bFloat {
		return 1
	} else if aFloat < bFloat {
		return -1
	}
	return 0
}

// toFloat64 converts an interface{} to float64
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	case string:
		// Try to parse as number
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f
		}
		return 0
	default:
		// Try to convert via reflection
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(rv.Uint())
		case reflect.Float32, reflect.Float64:
			return rv.Float()
		}
		return 0
	}
}

func containsString(value interface{}, expected interface{}) bool {
	str := fmt.Sprintf("%v", value)
	substr := fmt.Sprintf("%v", expected)
	return len(str) > 0 && len(substr) > 0 && contains(str, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
