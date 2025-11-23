package flows

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
)

// API provides HTTP handlers for flow management
type API struct {
	engine      *Engine
	rateLimiter *RateLimiter
	store       store.Adapter
	queue       *QueueManager
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
	uid := types.Uid(c.Query("uid", ""))
	topic := c.Query("topic", "")

	flows, err := a.store.GetFlows(uid, topic)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(flows)
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
		err = a.queue.EnqueueFlowExecution(c.Context(), id, req.TriggerType, req.TriggerID, req.Payload)
		if err != nil {
			flog.Error(err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(fiber.Map{
			"message": "flow execution queued successfully",
		})
	}

	// Fallback to synchronous execution
	err = a.engine.ExecuteFlow(c.Context(), id, req.TriggerType, req.TriggerID, req.Payload)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "flow executed successfully",
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

	// Delete existing nodes and edges
	if err := a.store.DeleteFlowNodesByFlowID(id); err != nil {
		flog.Error(fmt.Errorf("failed to delete flow nodes: %w", err))
	}
	if err := a.store.DeleteFlowEdgesByFlowID(id); err != nil {
		flog.Error(fmt.Errorf("failed to delete flow edges: %w", err))
	}

	// Create new nodes
	for _, node := range req.Nodes {
		node.FlowID = id
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
