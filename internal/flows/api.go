package flows

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/trigger"
	"github.com/gofiber/fiber/v3"
)

// API provides HTTP handlers for flow management
type API struct {
	engine      *Engine
	rateLimiter *RateLimiter
	store       store.Adapter
	queue       *QueueManager
}

// FlowWebhook triggers a flow using a webhook token.
// This endpoint is intended for external systems.
func (a *API) FlowWebhook(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid flow id"})
	}
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid token"})
	}

	flow, err := a.store.GetFlow(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "flow not found"})
	}
	if !flow.Enabled {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "flow is disabled"})
	}

	nodes, err := a.store.GetFlowNodes(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get flow nodes"})
	}

	var matched *model.FlowNode
	for i := range nodes {
		n := nodes[i]
		if n.Type != model.NodeTypeTrigger {
			continue
		}
		r, err := findTriggerRule(n.Bot, n.RuleID)
		if err != nil {
			continue
		}
		if r.Mode != trigger.ModeWebhook {
			continue
		}
		if n.Parameters == nil {
			continue
		}
		// token is stored in node.Parameters["token"].
		if v, ok := n.Parameters["token"].(string); ok && v == token {
			matched = n
			break
		}
	}
	if matched == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "webhook trigger not found"})
	}

	// Parse payload.
	payload := make(types.KV)
	switch c.Method() {
	case http.MethodGet:
		args := c.Request().URI().QueryArgs()
		args.VisitAll(func(k, v []byte) {
			payload[string(k)] = string(v)
		})
	default:
		body := c.Body()
		if len(body) > 0 {
			var obj map[string]any
			if err := sonic.Unmarshal(body, &obj); err == nil {
				payload = obj
			} else {
				payload["raw"] = string(body)
			}
		}
	}

	// Rate limit.
	flowID := &id
	allowed, _ := a.rateLimiter.CheckRateLimit(c.Context(), flowID, "")
	if !allowed {
		return c.Status(http.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
	}

	triggerType := fmt.Sprintf("%s|%s", matched.Bot, matched.RuleID)

	if a.queue != nil {
		executionID, err := a.queue.EnqueueFlowExecution(c.Context(), id, triggerType, token, payload)
		if err == nil {
			return c.JSON(fiber.Map{"message": "flow execution queued successfully", "execution_id": executionID})
		}
		flog.Error(err)
	}

	executionID, err := a.engine.ExecuteFlowWithExecutionID(c.Context(), id, "", triggerType, token, payload)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "flow executed successfully", "execution_id": executionID})
}

// NewAPI creates a new flow API
func NewAPI(engine *Engine, rateLimiter *RateLimiter, storeAdapter store.Adapter, queue *QueueManager) *API {
	return &API{
		engine:      engine,
		rateLimiter: rateLimiter,
		store:       storeAdapter,
		queue:       queue,
	}
}

// ListFlows lists all flows
func (a *API) ListFlows(c fiber.Ctx) error {
	uid, topic := resolveUIDTopicFromRequest(c, a.store)

	flows, err := a.store.GetFlows(uid, topic)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(flows)
}

func resolveUIDTopicFromRequest(c fiber.Ctx, storeAdapter store.Adapter) (types.Uid, string) {
	uidStr := c.Query("uid", "")
	topic := c.Query("topic", "")
	if uidStr != "" {
		return types.Uid(uidStr), topic
	}

	flag := c.Query("p", "")
	if flag == "" {
		return "", ""
	}

	p, err := storeAdapter.ParameterGet(flag)
	if err != nil || p.IsExpired() {
		return "", ""
	}
	uid, _ := types.KV(p.Params).String("uid")
	topic, _ = types.KV(p.Params).String("topic")
	return types.Uid(uid), topic
}

// GetFlow gets a flow by ID
func (a *API) GetFlow(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid flow id",
		})
	}

	flow, err := a.store.GetFlow(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "flow not found",
		})
	}

	// Get nodes and edges
	nodes, _ := a.store.GetFlowNodes(id)
	edges, _ := a.store.GetFlowEdges(id)

	return c.JSON(fiber.Map{
		"flow":  flow,
		"nodes": nodes,
		"edges": edges,
	})
}

// CreateFlow creates a new flow
func (a *API) CreateFlow(c fiber.Ctx) error {
	var flow model.Flow
	if err := c.Bind().Body(&flow); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	uid, topic := resolveUIDTopicFromRequest(c, a.store)
	if flow.UID == "" {
		flow.UID = uid.String()
	}
	if flow.Topic == "" {
		flow.Topic = topic
	}

	id, err := a.store.CreateFlow(&flow)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create flow",
		})
	}

	flow.ID = id
	return c.Status(http.StatusCreated).JSON(flow)
}

// UpdateFlow updates a flow
func (a *API) UpdateFlow(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid flow id",
		})
	}

	var flow model.Flow
	if err := c.Bind().Body(&flow); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	uid, topic := resolveUIDTopicFromRequest(c, a.store)
	if flow.UID == "" {
		flow.UID = uid.String()
	}
	if flow.Topic == "" {
		flow.Topic = topic
	}

	flow.ID = id
	if err := a.store.UpdateFlow(&flow); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update flow",
		})
	}

	return c.JSON(flow)
}

// DeleteFlow deletes a flow
func (a *API) DeleteFlow(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid flow id",
		})
	}

	if err := a.store.DeleteFlow(id); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete flow",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

// ExecuteFlow executes a flow
func (a *API) ExecuteFlow(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid flow id",
		})
	}

	// If flow is disabled, don't enqueue or create execution records.
	flow, err := a.store.GetFlow(id)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "flow not found",
		})
	}
	if !flow.Enabled {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "flow is disabled",
		})
	}

	var req struct {
		TriggerType string   `json:"trigger_type"`
		TriggerID   string   `json:"trigger_id"`
		Payload     types.KV `json:"payload"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Check rate limit
	flowID := &id
	allowed, err := a.rateLimiter.CheckRateLimit(c.Context(), flowID, "")
	if err != nil {
		flog.Error(err)
	}
	if !allowed {
		return c.Status(http.StatusTooManyRequests).JSON(fiber.Map{
			"error": "rate limit exceeded",
		})
	}

	// Execute flow asynchronously via queue
	if a.queue != nil {
		executionID, err := a.queue.EnqueueFlowExecution(c.Context(), id, req.TriggerType, req.TriggerID, req.Payload)
		if err == nil {
			return c.JSON(fiber.Map{
				"message":      "flow execution queued successfully",
				"execution_id": executionID,
			})
		}
		// If queue isn't configured or enqueue fails, fall back to synchronous
		// execution to keep UX working.
		flog.Error(err)
	}

	// Fallback to synchronous execution
	executionID, err := a.engine.ExecuteFlowWithExecutionID(c.Context(), id, "", req.TriggerType, req.TriggerID, req.Payload)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":      "flow executed successfully",
		"execution_id": executionID,
	})
}

// ListExecutions lists executions for a flow
func (a *API) ListExecutions(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid flow id",
		})
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	executions, err := a.store.GetExecutions(id, limit)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get executions",
		})
	}

	return c.JSON(executions)
}

// GetExecution gets an execution by ID
func (a *API) GetExecution(c fiber.Ctx) error {
	executionID := c.Params("execution_id")
	if executionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid execution id",
		})
	}

	execution, err := a.store.GetExecution(executionID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "execution not found",
		})
	}

	return c.JSON(execution)
}

// ListExecutionJobs lists per-node jobs for an execution
func (a *API) ListExecutionJobs(c fiber.Ctx) error {
	executionID := c.Params("execution_id")
	if executionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid execution id",
		})
	}

	jobs, err := a.store.GetFlowJobsByExecution(executionID)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get execution jobs",
		})
	}

	return c.JSON(jobs)
}

// UpdateFlowNodes updates flow nodes
func (a *API) UpdateFlowNodes(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid flow id",
		})
	}

	var req struct {
		Nodes []model.FlowNode `json:"nodes"`
		Edges []model.FlowEdge `json:"edges"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Preserve existing node variables/state by NodeID.
	oldNodes, _ := a.store.GetFlowNodes(id)
	oldByNodeID := make(map[string]*model.FlowNode, len(oldNodes))
	for _, n := range oldNodes {
		oldByNodeID[n.NodeID] = n
	}

	// Delete existing nodes and edges.
	if err := a.store.DeleteFlowNodesByFlowID(id); err != nil {
		flog.Error(fmt.Errorf("failed to delete flow nodes: %w", err))
	}
	if err := a.store.DeleteFlowEdgesByFlowID(id); err != nil {
		flog.Error(fmt.Errorf("failed to delete flow edges: %w", err))
	}

	// Create new nodes
	for _, node := range req.Nodes {
		node.FlowID = id
		if node.Bot == "" {
			node.Bot = "dev"
		}
		if old := oldByNodeID[node.NodeID]; old != nil {
			if node.Variables == nil {
				node.Variables = old.Variables
			}
		}

		// Ensure referenced trigger/action rules exist.
		switch node.Type {
		case model.NodeTypeTrigger:
			r, err := findTriggerRule(node.Bot, node.RuleID)
			if err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			// Generate token for webhook triggers if missing.
			if r.Mode == trigger.ModeWebhook {
				if node.Parameters == nil {
					node.Parameters = model.JSON{}
				}
				tok, _ := node.Parameters["token"].(string)
				if strings.TrimSpace(tok) == "" {
					node.Parameters["token"] = types.Id()
				}
			}
		case model.NodeTypeAction:
			if _, err := findActionRule(node.Bot, node.RuleID); err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
		}

		if _, err := a.store.CreateFlowNode(&node); err != nil {
			flog.Error(fmt.Errorf("failed to create flow node: %w", err))
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to create flow node: %v", err),
			})
		}
	}

	// Create new edges
	for _, edge := range req.Edges {
		edge.FlowID = id
		if _, err := a.store.CreateFlowEdge(&edge); err != nil {
			flog.Error(fmt.Errorf("failed to create flow edge: %w", err))
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to create flow edge: %v", err),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "flow nodes updated successfully",
	})
}
