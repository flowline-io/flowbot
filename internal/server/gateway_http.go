package server

import (
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// RegisterGatewayRoutes mounts local-CLI gateway worker APIs (not notification gateway).
func RegisterGatewayRoutes(a *fiber.App) {
	a.Post("/gateway/v1/claim", route.Authorize(route.RequireScope(auth.ScopeGatewayWorker, gatewayClaim)))
	a.Post("/gateway/v1/jobs/:id/result", route.Authorize(route.RequireScope(auth.ScopeGatewayWorker, gatewayComplete)))
	a.Post("/gateway/v1/heartbeat", route.Authorize(route.RequireScope(auth.ScopeGatewayWorker, gatewayHeartbeat)))
	a.Get("/gateway/v1/jobs/:id", route.Authorize(route.RequireScope(auth.ScopeGatewayWorker, gatewayGetJob)))
}

func gatewayLeaseTTL() time.Duration {
	ttl := config.App.Gateway.LeaseTTL
	if ttl <= 0 {
		return 90 * time.Second
	}
	return ttl
}

func gatewayMaxOutputBytes() int {
	if n := config.App.Gateway.MaxOutputBytes; n > 0 {
		return n
	}
	if n := config.App.ChatAgent.MaxToolOutput; n > 0 {
		return n
	}
	return 8192
}

func gatewayClaim(ctx fiber.Ctx) error {
	var req types.GatewayClaimRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid json body")
	}
	job, err := storepkg.GatewayStoreFromDB().Claim(ctx.Context(), req.WorkerID, gatewayLeaseTTL())
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.GatewayClaimResponse{Job: job}))
}

func gatewayComplete(ctx fiber.Ctx) error {
	jobID := strings.TrimSpace(ctx.Params("id"))
	var req types.GatewayCompleteRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid json body")
	}
	job, err := storepkg.GatewayStoreFromDB().Complete(ctx.Context(), jobID, req, gatewayMaxOutputBytes())
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(job))
}

func gatewayHeartbeat(ctx fiber.Ctx) error {
	var req types.GatewayHeartbeatRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid json body")
	}
	if err := storepkg.GatewayStoreFromDB().TouchWorkerLease(ctx.Context(), req.WorkerID, req.JobID, gatewayLeaseTTL()); err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]any{"ok": true}))
}

func gatewayGetJob(ctx fiber.Ctx) error {
	jobID := strings.TrimSpace(ctx.Params("id"))
	gs := storepkg.GatewayStoreFromDB()
	if err := gs.ReclaimExpired(ctx.Context()); err != nil {
		return err
	}
	job, err := gs.Get(ctx.Context(), jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return types.Errorf(types.ErrNotFound, "job not found")
	}
	return ctx.JSON(protocol.NewSuccessResponse(job))
}
