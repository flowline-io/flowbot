package web

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var blueprintWebserviceRules = []webservice.Rule{
	webservice.Get("/blueprints", blueprintListPage),
	webservice.Get("/blueprints/list", blueprintListTable),
	webservice.Post("/blueprints/import", importBlueprint),
	webservice.Get("/blueprints/:source/:id/download", downloadBlueprint),
	webservice.Get("/blueprints/:source/:id/instantiate", blueprintInstantiatePage),
	webservice.Post("/blueprints/:source/:id/instantiate", instantiateBlueprint),
	webservice.Post("/blueprints/:source/:id/replace", replaceBlueprint),
	webservice.Delete("/blueprints/:source/:id", deleteBlueprint),
	webservice.Get("/blueprints/:source/:id", blueprintDetailPage),
	webservice.Post("/pipelines/:name/blueprint/inputs", updateBlueprintInputs),
	webservice.Post("/pipelines/:name/blueprint/update", confirmBlueprintUpdate),
	webservice.Post("/pipelines/:name/blueprint/take-control", takeBlueprintControl),
}

func getBlueprintService() *pipeline.BlueprintService {
	return pipeline.ActiveBlueprintService()
}

func blueprintListPage(c fiber.Ctx) error {
	entries, err := loadBlueprintEntries(c.Context())
	if err != nil {
		return err
	}
	c.Type("html")
	return pages.BlueprintListPage(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func blueprintListTable(c fiber.Ctx) error {
	entries, err := loadBlueprintEntries(c.Context())
	if err != nil {
		return err
	}
	c.Type("html")
	return partials.BlueprintListTable(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func loadBlueprintEntries(ctx context.Context) ([]partials.BlueprintListEntry, error) {
	svc := getBlueprintService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	items, err := svc.ListCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return catalogEntriesToList(items), nil
}

func importBlueprint(c fiber.Ctx) error {
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	yamlText := c.FormValue("yaml")
	rec, err := svc.ImportYAML(c.Context(), []byte(yamlText), getUID(c))
	if err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderBlueprintFormError(c, err)
	}
	c.Response().Header.Set("HX-Redirect", "/service/web/blueprints/"+url.PathEscape(pipeline.BlueprintSourceUpload)+"/"+url.PathEscape(rec.BlueprintID))
	return c.SendStatus(fiber.StatusOK)
}

func blueprintDetailPage(c fiber.Ctx) error {
	source, id, err := blueprintPathParams(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	entry, err := svc.ResolveTemplate(c.Context(), source, id)
	if err != nil {
		return err
	}
	c.Type("html")
	data := partials.BlueprintDetailData{
		Entry:  catalogEntryToList(entry),
		Inputs: blueprintInputViews(entry.Document.Inputs),
		YAML:   entry.YAML,
	}
	return pages.BlueprintDetailPage(c.Context(), data).Render(c.Context(), c.Response().BodyWriter())
}

func downloadBlueprint(c fiber.Ctx) error {
	source, id, err := blueprintPathParams(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	entry, err := svc.ResolveTemplate(c.Context(), source, id)
	if err != nil {
		return err
	}
	c.Set("Content-Type", "application/yaml")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.yaml"`, id))
	return c.SendString(entry.YAML)
}

func blueprintInstantiatePage(c fiber.Ctx) error {
	source, id, err := blueprintPathParams(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	entry, err := svc.ResolveTemplate(c.Context(), source, id)
	if err != nil {
		return err
	}
	channels, err := listEnabledNotifyChannels(c.Context())
	if err != nil {
		return err
	}
	c.Type("html")
	data := partials.BlueprintInstantiateData{
		Entry:    catalogEntryToList(entry),
		Inputs:   blueprintInputViews(entry.Document.Inputs),
		Channels: channels,
	}
	return pages.BlueprintInstantiatePage(c.Context(), data).Render(c.Context(), c.Response().BodyWriter())
}

func instantiateBlueprint(c fiber.Ctx) error {
	source, id, err := blueprintPathParams(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	entry, err := svc.ResolveTemplate(c.Context(), source, id)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(c.FormValue("name"))
	inputs := collectBlueprintFormInputs(c, entry.Document.Inputs)
	_, err = svc.Instantiate(c.Context(), pipeline.InstantiateRequest{
		Source:       source,
		BlueprintID:  id,
		PipelineName: name,
		Inputs:       inputs,
		Enable:       formCheckboxOn(c, "enable"),
		CreatedBy:    getUID(c),
	})
	if err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderBlueprintFormError(c, err)
	}
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+url.PathEscape(name))
	return c.SendStatus(fiber.StatusOK)
}

func replaceBlueprint(c fiber.Ctx) error {
	source, id, err := blueprintPathParams(c)
	if err != nil {
		return err
	}
	if source != pipeline.BlueprintSourceUpload {
		return types.Errorf(types.ErrForbidden, "official blueprints cannot be replaced")
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	_, err = svc.ReplaceYAML(c.Context(), id, []byte(c.FormValue("yaml")))
	if err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderBlueprintFormError(c, err)
	}
	setShowToastKey(c, "success", "toast.blueprint.replaced")
	c.Response().Header.Set("HX-Redirect", partials.BlueprintWebPath(source, id))
	return c.SendStatus(fiber.StatusOK)
}

func deleteBlueprint(c fiber.Ctx) error {
	source, id, err := blueprintPathParams(c)
	if err != nil {
		return err
	}
	if source != pipeline.BlueprintSourceUpload {
		return toastErrorKey(c, "error.blueprint.official_delete")
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	if err := svc.DeleteUserTemplate(c.Context(), id); err != nil {
		if errors.Is(err, types.ErrConflict) {
			return toastErrorKey(c, "error.blueprint.in_use")
		}
		return toastBlueprintError(c, err)
	}
	entries, err := loadBlueprintEntries(c.Context())
	if err != nil {
		return err
	}
	setShowToastKey(c, "success", "toast.blueprint.deleted")
	c.Type("html")
	return partials.BlueprintListTable(c.Context(), entries).Render(c.Context(), c.Response().BodyWriter())
}

func updateBlueprintInputs(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	def, origin, err := linkedPipeline(c.Context(), name)
	if err != nil {
		return err
	}
	doc, err := pipeline.ParseBlueprintYAML([]byte(origin.YAML))
	if err != nil {
		return err
	}
	inputs := collectBlueprintFormInputs(c, doc.Inputs)
	enable := formCheckboxOn(c, "enable")
	if _, err := svc.UpdateInstanceInputs(c.Context(), def.Name, inputs, &enable); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return renderBlueprintFormError(c, err)
	}
	setShowToastKey(c, "success", "toast.blueprint.inputs_saved")
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+url.PathEscape(name))
	return c.SendStatus(fiber.StatusOK)
}

func confirmBlueprintUpdate(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	if _, err := svc.ConfirmInstanceUpdate(c.Context(), name); err != nil {
		return toastBlueprintError(c, err)
	}
	setShowToastKey(c, "success", "toast.blueprint.updated")
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+url.PathEscape(name))
	return c.SendStatus(fiber.StatusOK)
}

func takeBlueprintControl(c fiber.Ctx) error {
	name, err := pipelineNameParam(c)
	if err != nil {
		return err
	}
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	if err := svc.TakeControl(c.Context(), name); err != nil {
		return toastBlueprintError(c, err)
	}
	setShowToastKey(c, "success", "toast.blueprint.took_control")
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+url.PathEscape(name))
	return c.SendStatus(fiber.StatusOK)
}

func blueprintPathParams(c fiber.Ctx) (source, id string, err error) {
	source = strings.TrimSpace(c.Params("source"))
	id = strings.TrimSpace(c.Params("id"))
	if source != pipeline.BlueprintSourceBuiltin && source != pipeline.BlueprintSourceUpload {
		return "", "", types.Errorf(types.ErrInvalidArgument, "unknown blueprint source")
	}
	if err = pipeline.ValidateBlueprintID(id); err != nil {
		return "", "", err
	}
	return source, id, nil
}

func catalogEntriesToList(items []pipeline.BlueprintCatalogEntry) []partials.BlueprintListEntry {
	out := make([]partials.BlueprintListEntry, 0, len(items))
	for i := range items {
		out = append(out, catalogEntryToList(&items[i]))
	}
	return out
}

func catalogEntryToList(entry *pipeline.BlueprintCatalogEntry) partials.BlueprintListEntry {
	if entry == nil {
		return partials.BlueprintListEntry{}
	}
	return partials.BlueprintListEntry{
		Source:        entry.Source,
		ID:            entry.Document.ID,
		Title:         entry.Document.Title,
		Description:   entry.Document.Description,
		Missing:       entry.Missing,
		InstanceCount: entry.InstanceCount,
		Official:      entry.Source == pipeline.BlueprintSourceBuiltin,
	}
}

func blueprintInputViews(defs []pipeline.BlueprintInputDef) []partials.BlueprintInputView {
	out := make([]partials.BlueprintInputView, 0, len(defs))
	for _, def := range defs {
		out = append(out, partials.BlueprintInputView{
			Name:        def.Name,
			Type:        def.Type,
			Required:    def.Required,
			Default:     def.Default,
			Description: def.Description,
		})
	}
	return out
}

func listEnabledNotifyChannels(ctx context.Context) ([]model.NotifyChannel, error) {
	enabled := true
	return store.NotifyConfigStoreFromDB().ListNotifyChannels(ctx, store.ListNotifyChannelOptions{Enabled: &enabled})
}

func renderBlueprintFormError(c fiber.Ctx, err error) error {
	return renderFormErrorKey(c, "#form-error", blueprintErrorKey(err))
}

func toastBlueprintError(c fiber.Ctx, err error) error {
	return toastErrorKey(c, blueprintErrorKey(err))
}

func blueprintErrorKey(err error) string {
	switch {
	case errors.Is(err, types.ErrUnavailable):
		return "error.blueprint.unavailable"
	case errors.Is(err, types.ErrNotFound):
		return "error.not_found"
	case errors.Is(err, types.ErrAlreadyExists):
		return "error.blueprint.already_exists"
	case errors.Is(err, types.ErrForbidden):
		if strings.Contains(err.Error(), "replaced") {
			return "error.blueprint.official_replace"
		}
		return "error.blueprint.official_delete"
	case errors.Is(err, types.ErrConflict):
		if strings.Contains(err.Error(), "used by") {
			return "error.blueprint.in_use"
		}
		return "error.blueprint.linked_readonly"
	case errors.Is(err, types.ErrInvalidArgument):
		msg := err.Error()
		if strings.Contains(msg, "missing capabilities") {
			return "error.blueprint.missing_caps"
		}
		if strings.Contains(msg, "not a blueprint instance") {
			return "error.blueprint.not_instance"
		}
		return "error.blueprint.invalid"
	default:
		return "error.server"
	}
}

func collectBlueprintFormInputs(c fiber.Ctx, defs []pipeline.BlueprintInputDef) map[string]any {
	out := make(map[string]any, len(defs))
	for _, def := range defs {
		key := "input_" + def.Name
		switch def.Type {
		case pipeline.BlueprintInputNotifyChannel:
			out[def.Name] = formMultiValues(c, key)
		case pipeline.BlueprintInputBoolean:
			out[def.Name] = formCheckboxOn(c, key)
		default:
			out[def.Name] = c.FormValue(key)
		}
	}
	return out
}

func formCheckboxOn(c fiber.Ctx, key string) bool {
	v := strings.ToLower(strings.TrimSpace(c.FormValue(key)))
	return v == "on" || v == "true" || v == "1"
}

func formMultiValues(c fiber.Ctx, key string) []string {
	req := c.Request()
	if req == nil {
		return nil
	}
	raw := req.PostArgs().PeekMulti(key)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s := strings.TrimSpace(string(v))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func renderLinkedPipelinePage(c fiber.Ctx, def model.PipelineDefinition) error {
	svc := getBlueprintService()
	if svc == nil {
		return types.Errorf(types.ErrUnavailable, "blueprint service not ready")
	}
	doc, err := pipeline.ParseBlueprintYAML([]byte(def.BlueprintYAML))
	if err != nil {
		return err
	}
	updateAvail, err := svc.InstanceUpdateAvailable(c.Context(), &def)
	if err != nil {
		return err
	}
	channels, err := listEnabledNotifyChannels(c.Context())
	if err != nil {
		return err
	}
	yamlText := def.YamlDraft
	if def.YamlPublished != nil && *def.YamlPublished != "" {
		yamlText = *def.YamlPublished
	}
	c.Type("html")
	return pages.BlueprintInstancePage(c.Context(), partials.BlueprintInstanceData{
		Pipeline:        def,
		Title:           doc.Title,
		Inputs:          blueprintInputViews(doc.Inputs),
		Channels:        channels,
		Enabled:         pipeline.IsEnabledInYAML(yamlText),
		UpdateAvailable: updateAvail,
		Materialized:    yamlText,
	}).Render(c.Context(), c.Response().BodyWriter())
}

func linkedPipeline(ctx context.Context, name string) (*model.PipelineDefinition, pipeline.BlueprintOrigin, error) {
	s := getPipelineDefStore()
	row, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, pipeline.BlueprintOrigin{}, err
	}
	def := mapPipelineDefinition(row)
	origin, ok := pipeline.OriginFromDefinition(&def)
	if !ok {
		return nil, pipeline.BlueprintOrigin{}, types.Errorf(types.ErrInvalidArgument, "pipeline %s is not a blueprint instance", name)
	}
	return &def, origin, nil
}
