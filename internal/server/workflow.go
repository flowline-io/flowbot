package server

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/workflow"
)

func initWorkflow(
	lc fx.Lifecycle,
	auditor audit.Auditor,
	wc *metrics.WorkflowCollector,
) error {
	if store.Database == nil || store.Database.GetDB() == nil {
		flog.Warn("workflow service skipped: store.Database not ready")
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok || client == nil {
		flog.Warn("workflow service skipped: store client unavailable")
		return nil
	}

	catalog := store.NewWorkflowStore(client)
	runs := store.NewWorkflowRunStore(client)
	svc := workflow.NewService(catalog, runs, auditor, wc)
	if err := svc.ReloadTriggers(context.Background()); err != nil {
		return fmt.Errorf("reload workflow triggers: %w", err)
	}
	workflow.SetReloadService(svc)
	registerWorkflowWebhookRoutes(svc)

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			svc.Stop()
			workflow.SetReloadService(nil)
			return nil
		},
	})

	flog.Info("workflow service initialized")
	return nil
}

// registerWorkflowWebhookRoutes mounts a catch-all handler under /webhook/workflow/*
// so ReloadTriggers can update endpoints without re-registering Fiber routes.
func registerWorkflowWebhookRoutes(svc *workflow.Service) {
	handler := makeWorkflowWebhookHandler(svc)
	sharedAppPtr().All("/webhook/workflow/*", handler)
	flog.Info("workflow webhook route registered: ALL /webhook/workflow/*")
}

func makeWorkflowWebhookHandler(svc *workflow.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		if svc == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		path := c.Params("*")
		path = strings.TrimPrefix(path, "/")
		ep, ok := svc.LookupWebhook(path)
		if !ok || ep == nil || ep.Config == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		method := string(c.Request().Header.Method())
		if !strings.EqualFold(method, ep.Config.Method) {
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}

		status, ok := authenticateWebhook(c, ep.Config)
		if !ok {
			return c.Status(status).SendString(http.StatusText(status))
		}

		input, err := workflowWebhookInput(c, ep.Config)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		asyncCtx, asyncSpan := fbtrace.StartSpan(c.Context(), "workflow.webhook.async")
		go func() {
			defer asyncSpan.End()
			ctx, cancel := fbtrace.DetachWithTimeout(asyncCtx, 10*time.Minute)
			defer cancel()
			runID, err := svc.StartRunAsync(ctx, ep.WorkflowName, "webhook", input)
			if err != nil {
				flog.Error(fmt.Errorf("webhook workflow %s: %w", ep.WorkflowName, err))
				return
			}
			flog.Info("webhook workflow %s: started run %d", ep.WorkflowName, runID)
		}()

		return c.SendStatus(fiber.StatusAccepted)
	}
}

func workflowWebhookInput(c fiber.Ctx, wcfg *pipeline.WebhookConfig) (types.KV, error) {
	body := c.Body()
	input := make(types.KV)

	if wcfg.Payload == config.WebhookPayloadMapped {
		var parsed map[string]any
		if err := sonic.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON body")
		}
		maps.Copy(input, parsed)
	} else {
		input["_webhook_body"] = truncateWebhookBody(body)
		if len(body) > maxWebhookBodySize {
			input["_webhook_body_truncated"] = true
		}
	}

	headers := sanitizeWebhookHeaders(c, wcfg)
	input["_webhook_headers"] = headers
	input["_webhook_method"] = string(c.Request().Header.Method())
	input["_webhook_path"] = string(c.Request().URI().Path())
	return input, nil
}
