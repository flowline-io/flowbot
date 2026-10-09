package automate

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	pkgpipeline "github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func activeBlueprintService() (*pkgpipeline.BlueprintService, error) {
	return requireActive(pkgpipeline.ActiveBlueprintService(), "blueprint service not ready")
}

func listBlueprints(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	items, err := svc.ListCatalog(ctx.Context())
	if err != nil {
		return err
	}
	out := make([]types.KV, 0, len(items))
	for _, item := range items {
		out = append(out, types.KV{
			"source":         item.Source,
			"id":             item.Document.ID,
			"title":          item.Document.Title,
			"description":    item.Document.Description,
			"missing":        item.Missing,
			"instance_count": item.InstanceCount,
			"hash":           item.Hash,
		})
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"blueprints": out}))
}

func importBlueprintJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	var body struct {
		YAML string `json:"yaml"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	rec, err := svc.ImportYAML(ctx.Context(), []byte(body.YAML), requestUID(ctx))
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"id":     rec.BlueprintID,
		"title":  rec.Title,
		"source": pkgpipeline.BlueprintSourceUpload,
		"hash":   rec.Hash,
	}))
}

func getBlueprintJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	source, id, err := blueprintJSONParams(ctx)
	if err != nil {
		return err
	}
	entry, err := svc.ResolveTemplate(ctx.Context(), source, id)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"source":      entry.Source,
		"id":          entry.Document.ID,
		"title":       entry.Document.Title,
		"description": entry.Document.Description,
		"missing":     entry.Missing,
		"hash":        entry.Hash,
		"inputs":      entry.Document.Inputs,
		"requires":    entry.Document.Requires,
	}))
}

func exportBlueprintJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	source, id, err := blueprintJSONParams(ctx)
	if err != nil {
		return err
	}
	entry, err := svc.ResolveTemplate(ctx.Context(), source, id)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"yaml": entry.YAML}))
}

func replaceBlueprintJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	source, id, err := blueprintJSONParams(ctx)
	if err != nil {
		return err
	}
	if source != pkgpipeline.BlueprintSourceUpload {
		return types.Errorf(types.ErrForbidden, "official blueprints cannot be replaced")
	}
	var body struct {
		YAML string `json:"yaml"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	rec, err := svc.ReplaceYAML(ctx.Context(), id, []byte(body.YAML))
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"id": rec.BlueprintID, "hash": rec.Hash}))
}

func deleteBlueprintJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	source, id, err := blueprintJSONParams(ctx)
	if err != nil {
		return err
	}
	if source != pkgpipeline.BlueprintSourceUpload {
		return types.Errorf(types.ErrForbidden, "official blueprints cannot be deleted")
	}
	if err := svc.DeleteUserTemplate(ctx.Context(), id); err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"deleted": id}))
}

func instantiateBlueprintJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	source, id, err := blueprintJSONParams(ctx)
	if err != nil {
		return err
	}
	var body struct {
		Name   string         `json:"name"`
		Inputs map[string]any `json:"inputs"`
		Enable bool           `json:"enable"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	def, err := svc.Instantiate(ctx.Context(), pkgpipeline.InstantiateRequest{
		Source:       source,
		BlueprintID:  id,
		PipelineName: body.Name,
		Inputs:       body.Inputs,
		Enable:       body.Enable,
		CreatedBy:    requestUID(ctx),
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"name":    def.Name,
		"id":      def.ID,
		"version": def.Version,
	}))
}

func updateBlueprintInputsJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	name := strings.TrimSpace(ctx.Params("name"))
	var body struct {
		Inputs map[string]any `json:"inputs"`
		Enable *bool          `json:"enable"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid request body", err)
	}
	def, err := svc.UpdateInstanceInputs(ctx.Context(), name, body.Inputs, body.Enable)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"name": def.Name, "version": def.Version}))
}

func confirmBlueprintUpdateJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	name := strings.TrimSpace(ctx.Params("name"))
	def, err := svc.ConfirmInstanceUpdate(ctx.Context(), name)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"name": def.Name, "version": def.Version}))
}

func takeBlueprintControlJSON(ctx fiber.Ctx) error {
	svc, err := activeBlueprintService()
	if err != nil {
		return err
	}
	name := strings.TrimSpace(ctx.Params("name"))
	if err := svc.TakeControl(ctx.Context(), name); err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{"name": name, "detached": true}))
}

func blueprintJSONParams(ctx fiber.Ctx) (source, id string, err error) {
	source = strings.TrimSpace(ctx.Params("source"))
	id = strings.TrimSpace(ctx.Params("id"))
	if source != pkgpipeline.BlueprintSourceBuiltin && source != pkgpipeline.BlueprintSourceUpload {
		return "", "", types.Errorf(types.ErrInvalidArgument, "unknown blueprint source")
	}
	if err = pkgpipeline.ValidateBlueprintID(id); err != nil {
		return "", "", err
	}
	return source, id, nil
}
