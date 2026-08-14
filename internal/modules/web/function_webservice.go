package web

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability"
	capfunctions "github.com/flowline-io/flowbot/pkg/capability/functions"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var functionWebserviceRules = []webservice.Rule{
	webservice.Get("/functions", functionListPage),
	webservice.Get("/functions/list", functionListTable),
	webservice.Get("/functions/stats", functionStats),
	webservice.Post("/functions", createFunction),
	webservice.Get("/functions/:name", functionEditorPage),
	webservice.Put("/functions/:name", saveFunctionDraft),
	webservice.Post("/functions/:name/publish", publishFunction),
	webservice.Delete("/functions/:name", deleteFunction),
	webservice.Get("/functions/:name/runs", functionRunsPartial),
	webservice.Get("/functions/:name/stats", functionStats),
	webservice.Post("/functions/:name/try", tryFunction),
}

func getFunctionService() *pkgfunctions.Service {
	return pkgfunctions.ActiveService()
}

func getFunctionStore() *store.FunctionStore {
	if store.Database == nil {
		return nil
	}
	return store.FunctionStoreFromDB()
}

func functionNameParam(c fiber.Ctx) (string, error) {
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	if err := pkgfunctions.ValidateName(name); err != nil {
		return "", types.WrapError(types.ErrInvalidArgument, "invalid function name", err)
	}
	return name, nil
}

func functionListPage(c fiber.Ctx) error {
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	items, err := svc.ListAll(context.Background())
	if err != nil {
		return types.WrapError(types.ErrInternal, "list functions", err)
	}
	c.Type("html")
	return pages.FunctionListPage(partials.BuildFunctionListEntries(items)).Render(context.Background(), c.Response().BodyWriter())
}

func functionListTable(c fiber.Ctx) error {
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	items, err := svc.ListAll(context.Background())
	if err != nil {
		return types.WrapError(types.ErrInternal, "list functions", err)
	}
	c.Type("html")
	return partials.FunctionListTable(partials.BuildFunctionListEntries(items)).Render(context.Background(), c.Response().BodyWriter())
}

func functionStats(c fiber.Ctx) error {
	name, err := decodePathParam(c.Params("name"))
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid function name")
	}
	if name != "" {
		if err := pkgfunctions.ValidateName(name); err != nil {
			return types.WrapError(types.ErrInvalidArgument, "invalid function name", err)
		}
	}
	since, tabs, err := parseStatsTabQuery(c)
	if err != nil {
		return err
	}

	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	if name != "" {
		if _, err = svc.GetDraft(context.Background(), name); err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return types.Errorf(types.ErrNotFound, "function %s not found", name)
			}
			return types.WrapError(types.ErrInternal, "get function", err)
		}
	}

	s := getFunctionStore()
	if s == nil {
		return types.Errorf(types.ErrInternal, "function store not available")
	}
	stats, err := s.FunctionStats(context.Background(), name, since, tabs.GroupBy)
	if err != nil {
		return types.WrapError(types.ErrInternal, "function stats", err)
	}

	accept := c.Get("Accept", "")
	if accept == "application/json" {
		return c.JSON(stats)
	}
	c.Type("html")
	return partials.FunctionStats(name, stats, tabs).Render(context.Background(), c.Response().BodyWriter())
}

func createFunction(c fiber.Ctx) error {
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name := strings.TrimSpace(c.FormValue("name"))
	entrypoint := strings.TrimSpace(c.FormValue("entrypoint"))
	if entrypoint == "" {
		entrypoint = "main.py"
	}
	if err := pkgfunctions.ValidateName(name); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderFormError(c, "#form-error", err.Error())
	}
	if _, err := svc.Create(context.Background(), name, entrypoint, getUID(c)); err != nil {
		if errors.Is(err, types.ErrAlreadyExists) {
			c.Status(fiber.StatusUnprocessableEntity)
			return renderFormError(c, "#form-error", fmt.Sprintf("Function %q already exists.", name))
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			c.Status(fiber.StatusUnprocessableEntity)
			return renderFormError(c, "#form-error", err.Error())
		}
		return types.WrapError(types.ErrInternal, "create function", err)
	}
	c.Response().Header.Set("HX-Redirect", "/service/web/functions/"+url.PathEscape(name))
	return c.SendStatus(200)
}

func functionEditorPage(c fiber.Ctx) error {
	name, err := functionNameParam(c)
	if err != nil {
		return err
	}
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	draft, err := svc.GetDraft(context.Background(), name)
	if err != nil {
		return err
	}
	c.Type("html")
	data := partials.FunctionDraftFromView(draft)
	origin := requestPublicOrigin(c)
	data.CallURL = partials.FunctionCallURL(draft.Name, origin)
	if draft.PublishedVersion != nil {
		data.CallVersionURL = partials.FunctionCallVersionURL(draft.Name, *draft.PublishedVersion, origin)
	}
	return pages.FunctionEditorPage(data).Render(context.Background(), c.Response().BodyWriter())
}

func saveFunctionDraft(c fiber.Ctx) error {
	name, err := functionNameParam(c)
	if err != nil {
		return err
	}
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	var body struct {
		Metadata   string `json:"metadata"`
		Entrypoint string `json:"entrypoint"`
		Source     string `json:"source"`
		Version    int    `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid body", err)
	}
	view, err := svc.SaveDraft(context.Background(), name, body.Metadata, body.Entrypoint, body.Source, body.Version)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return c.Status(409).JSON(fiber.Map{
				"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere. Please refresh the page."},
			})
		}
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Function not found"},
			})
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "VALIDATION_ERROR", "message": err.Error()},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL", "message": "Save draft failed"},
		})
	}
	return c.JSON(fiber.Map{
		"version":                 view.Version,
		"status":                  view.Status,
		"has_unpublished_changes": view.HasUnpublishedChanges,
		"published_version":       view.PublishedVersion,
	})
}

func publishFunction(c fiber.Ctx) error {
	name, err := functionNameParam(c)
	if err != nil {
		return err
	}
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	var body struct {
		Version int `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid body", err)
	}
	res, err := svc.Publish(context.Background(), name, body.Version)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return c.Status(409).JSON(fiber.Map{
				"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere. Please refresh the page."},
			})
		}
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Function not found"},
			})
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "VALIDATION_ERROR", "message": err.Error()},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL", "message": "Publish failed"},
		})
	}
	draft, err := svc.GetDraft(context.Background(), name)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"name":               res.Name,
		"version":            res.Version,
		"status":             res.Status,
		"definition_version": draft.Version,
	})
}

func deleteFunction(c fiber.Ctx) error {
	name, err := functionNameParam(c)
	if err != nil {
		return err
	}
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	if err := svc.Delete(context.Background(), name); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(c, "Function not found")
		}
		return err
	}
	c.Response().Header.Set("HX-Redirect", "/service/web/functions")
	return c.SendStatus(200)
}

func functionRunsPartial(c fiber.Ctx) error {
	name, err := functionNameParam(c)
	if err != nil {
		return err
	}
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	runs, err := svc.ListRuns(context.Background(), name)
	if err != nil {
		return err
	}
	c.Type("html")
	return partials.FunctionRunsTable(runs).Render(context.Background(), c.Response().BodyWriter())
}

func tryFunction(c fiber.Ctx) error {
	name, err := functionNameParam(c)
	if err != nil {
		return err
	}
	svc := getFunctionService()
	if svc == nil || !svc.Ready() {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	var body struct {
		Event any `json:"event"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid body", err)
	}
	publishedVersion, err := svc.LatestPublishedVersion(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "VALIDATION_ERROR", "message": "Publish a version before trying"},
			})
		}
		return functionTryJSONError(c, err)
	}
	result, err := capability.Invoke(context.Background(), hub.CapFunctions, capfunctions.OpInvoke, map[string]any{
		"name":    name,
		"version": publishedVersion,
		"event":   body.Event,
	})
	if err != nil {
		return functionTryJSONError(c, err)
	}
	if result == nil {
		return c.JSON(fiber.Map{"ok": true})
	}
	return c.JSON(result.Data)
}

func functionTryJSONError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, types.ErrUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": fiber.Map{"code": "UNAVAILABLE", "message": err.Error()},
		})
	case errors.Is(err, types.ErrInvalidArgument):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": fiber.Map{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
	case errors.Is(err, types.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": "FORBIDDEN", "message": err.Error()},
		})
	case errors.Is(err, types.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{"code": "NOT_FOUND", "message": "Function not found"},
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "INTERNAL", "message": "Function invoke failed"},
		})
	}
}
