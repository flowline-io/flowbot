package automate

import (
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var functionsRules = []webservice.Rule{
	webservice.Post("/apply", applyFunction),
	webservice.Get("/list", listFunctions),
	webservice.Get("/get/:name", getFunction),
	webservice.Get("/export/:name", exportFunction),
	webservice.Delete("/delete/:name", deleteFunction),
	webservice.Get("/runs/:name", listFunctionRuns),
	webservice.Post("/call/:name", callFunction, route.WithNotAuth()),
	webservice.Post("/call/:name/v/:version", callFunctionVersion, route.WithNotAuth()),
}

func applyFunction(ctx fiber.Ctx) error {
	var body struct {
		Metadata   string `json:"metadata"`
		Entrypoint string `json:"entrypoint"`
		Source     string `json:"source"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	if strings.TrimSpace(body.Metadata) == "" || strings.TrimSpace(body.Entrypoint) == "" || strings.TrimSpace(body.Source) == "" {
		return types.Errorf(types.ErrInvalidArgument, "metadata, entrypoint, and source are required")
	}
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	result, err := svc.ApplyBundle(ctx.Context(), body.Metadata, body.Entrypoint, body.Source, requestUID(ctx))
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"name":    result.Name,
		"id":      result.ID,
		"version": result.Version,
		"status":  result.Status,
	}))
}

func listFunctions(ctx fiber.Ctx) error {
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	items, err := svc.List(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"functions": items}))
}

func getFunction(ctx fiber.Ctx) error {
	name := functionNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	meta, err := svc.Get(ctx.Context(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(meta))
}

func exportFunction(ctx fiber.Ctx) error {
	name := functionNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	bundle, err := svc.Export(ctx.Context(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"name":       bundle.Name,
		"version":    bundle.Version,
		"metadata":   bundle.Metadata,
		"entrypoint": bundle.Entrypoint,
		"source":     bundle.Source,
	}))
}

func deleteFunction(ctx fiber.Ctx) error {
	name := functionNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	if err := svc.Delete(ctx.Context(), name); err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"deleted": name}))
}

func listFunctionRuns(ctx fiber.Ctx) error {
	name := functionNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	runs, err := svc.ListRuns(ctx.Context(), name)
	if err != nil {
		return err
	}
	items := make([]types.KV, 0, len(runs))
	for _, r := range runs {
		if r == nil {
			continue
		}
		item := types.KV{
			"id":            r.ID,
			"function_name": r.FunctionName,
			"version":       r.Version,
			"status":        r.Status,
			"duration_ms":   r.DurationMs,
			"created_at":    r.CreatedAt,
		}
		if r.ExitCode != nil {
			item["exit_code"] = *r.ExitCode
		}
		if r.Error != "" {
			item["error"] = r.Error
		}
		if r.ResultJSON != nil {
			item["result_json"] = *r.ResultJSON
		}
		items = append(items, item)
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"runs": items}))
}

func callFunction(ctx fiber.Ctx) error {
	return invokeCall(ctx, nil)
}

func callFunctionVersion(ctx fiber.Ctx) error {
	raw := strings.TrimSpace(ctx.Params("version"))
	if raw == "" {
		return types.Errorf(types.ErrInvalidArgument, "version is required")
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid version")
	}
	return invokeCall(ctx, &v)
}

func invokeCall(ctx fiber.Ctx, version *int) error {
	name := functionNameParam(ctx)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	svc, err := activeFunctionService()
	if err != nil {
		return err
	}
	meta, err := svc.PublishedMetadata(ctx.Context(), name, version)
	if err != nil {
		return err
	}
	body := ctx.Body()
	if !pkgfunctions.AuthenticateCall(meta, ctx.Get("X-Webhook-Token"), ctx.Query("token"), ctx.Get("X-Hub-Signature-256"), body) {
		return types.Errorf(types.ErrUnauthorized, "function call authentication failed")
	}
	var event any
	if len(body) > 0 {
		if err := sonic.Unmarshal(body, &event); err != nil {
			return types.WrapError(types.ErrInvalidArgument, "invalid event JSON", err)
		}
	}
	idem := strings.TrimSpace(ctx.Get("Idempotency-Key"))
	result, err := svc.Invoke(ctx.Context(), pkgfunctions.InvokeRequest{
		Name:           name,
		Version:        version,
		Event:          event,
		IdempotencyKey: idem,
		RequireVersion: false,
	})
	if result != nil && result.Replayed {
		return ctx.JSON(protocol.NewSuccessResponse(result))
	}
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func functionNameParam(ctx fiber.Ctx) string {
	name := strings.TrimSpace(ctx.Params("name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Query("name"))
	}
	return name
}
