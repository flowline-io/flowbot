package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var viewWebserviceRules = []webservice.Rule{
	webservice.Get("/view/:token", viewPage, route.WithNotAuth()),
	webservice.Post("/view", createView, route.WithNotAuth()),
	webservice.Delete("/view/:token", deleteView, route.WithNotAuth()),
}

// viewPage renders a shareable view page by token.
func viewPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	token := ctx.Params("token")
	if token == "" {
		return ctx.Status(http.StatusBadRequest).SendString("missing token")
	}

	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return ctx.Status(http.StatusInternalServerError).SendString("store not available")
	}
	pageDataStore := store.NewPageDataStore(client)

	pageData, err := pageDataStore.GetPageDataByToken(context.Background(), token)
	if err != nil {
		flog.Error(fmt.Errorf("viewPage: get page_data: %w", err))
		return ctx.Status(http.StatusInternalServerError).SendString("failed to load page")
	}
	if pageData == nil {
		ctx.Type("html")
		return pages.ViewPage("Not Found", partials.ViewExpiredPage(), false).Render(context.Background(), ctx.Response().BodyWriter())
	}

	expired := pageData.ExpiresAt != nil && pageData.ExpiresAt.Before(time.Now())
	if expired {
		ctx.Type("html")
		return pages.ViewPage(pageData.Title, partials.ViewExpiredPage(), true).Render(context.Background(), ctx.Response().BodyWriter())
	}

	dataKV := types.KV(pageData.Data)

	if pageData.Type == "pipeline_run" {
		dataKV = preFetchPipelineData(context.Background(), store.NewPipelineStore(client), dataKV)
	}

	fn, ok := viewTemplates[pageData.Type]
	if !ok {
		flog.Error(fmt.Errorf("viewPage: unknown type %q", pageData.Type))
		ctx.Type("html")
		return pages.ViewPage(pageData.Title, partials.ViewExpiredPage(), false).Render(context.Background(), ctx.Response().BodyWriter())
	}

	body := fn(dataKV)
	ctx.Type("html")
	return pages.ViewPage(pageData.Title, body, expired).Render(context.Background(), ctx.Response().BodyWriter())
}

// preFetchPipelineData fetches step runs for a pipeline_run view and injects them into data.
func preFetchPipelineData(ctx context.Context, pipeStore *store.PipelineStore, data types.KV) types.KV {
	runID, ok := data.Int64("run_id")
	if !ok {
		return data
	}
	steps, err := pipeStore.GetStepRunsByRunID(ctx, runID)
	if err != nil {
		flog.Error(fmt.Errorf("preFetchPipelineData: GetStepRunsByRunID: %w", err))
		return data
	}
	data["steps"] = steps
	return data
}

// createView saves a new view page and returns the token and URL.
func createView(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	type createRequest struct {
		Type      string     `json:"type"`
		Title     string     `json:"title"`
		Data      types.KV   `json:"data"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}

	var req createRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(types.KV{"error": "invalid JSON: " + err.Error()})
	}
	if req.Type == "" {
		return ctx.Status(http.StatusBadRequest).JSON(types.KV{"error": "type is required"})
	}
	if req.Data == nil {
		req.Data = types.KV{}
	}

	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "store not available"})
	}
	pageDataStore := store.NewPageDataStore(client)

	token := types.Id()

	rc := route.GetRequestContext(ctx)
	createdBy := ""
	if rc != nil {
		createdBy = string(rc.UID)
	}

	if err := pageDataStore.CreatePageData(context.Background(), token, req.Type, req.Title, req.Data, createdBy, req.ExpiresAt); err != nil {
		flog.Error(fmt.Errorf("createView: CreatePageData: %w", err))
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "failed to create page"})
	}

	return ctx.Status(http.StatusCreated).JSON(types.KV{
		"token": token,
		"url":   "/service/web/view/" + token,
	})
}

// deleteView removes a view page by token.
func deleteView(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	token := ctx.Params("token")
	if token == "" {
		return ctx.Status(http.StatusBadRequest).JSON(types.KV{"error": "missing token"})
	}

	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "store not available"})
	}
	pageDataStore := store.NewPageDataStore(client)

	affected, err := pageDataStore.DeletePageData(context.Background(), token)
	if err != nil {
		flog.Error(fmt.Errorf("deleteView: DeletePageData: %w", err))
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "failed to delete page"})
	}
	if affected == 0 {
		return ctx.Status(http.StatusNotFound).JSON(types.KV{"error": "page not found"})
	}

	return ctx.SendStatus(http.StatusNoContent)
}
