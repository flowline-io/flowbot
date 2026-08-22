package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var configWebserviceRules = []webservice.Rule{
	webservice.Get("/configs", configsPage),
	webservice.Get("/configs/list", listConfigs),
	webservice.Get("/configs/new", newConfigForm),
	webservice.Post("/configs", createConfig),
	webservice.Get("/configs/:uid/:topic/:key", getConfig),
	webservice.Get("/configs/:uid/:topic/:key/edit", editConfigForm),
	webservice.Put("/configs/:uid/:topic/:key", updateConfig),
	webservice.Delete("/configs/:uid/:topic/:key", deleteConfig),
}

func configsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.ModuleDataStoreFromDB().ListConfigs(context.Background(), store.ListConfigOptions{Limit: 100})
	if err != nil {
		return types.Errorf(types.ErrInternal, "list configs: %v", err)
	}
	ctx.Type("html")
	return pages.ConfigsPage(ctx.Context(), items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func listConfigs(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.ModuleDataStoreFromDB().ListConfigs(context.Background(), store.ListConfigOptions{Limit: 100})
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderErrorKey(ctx, "error.load.configs")
	}
	ctx.Type("html")
	return partials.ConfigTable(items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func getConfig(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	value, err := store.ModuleDataStoreFromDB().ConfigGet(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderErrorKey(ctx, "toast.config.not_found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderErrorKey(ctx, "error.load.config")
	}
	ctx.Type("html")
	return partials.ConfigRow(ctx.Context(), model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func newConfigForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	// Remove any existing new-config form row and empty state row
	ctx.Response().BodyWriter().Write([]byte(`<tr id="config-form-new" hx-swap-oob="delete"></tr><tr id="configs-empty" hx-swap-oob="delete"></tr>`))
	return partials.ConfigForm(model.ConfigItem{}, true, nil).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func createConfig(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := ctx.FormValue("uid")
	topic := ctx.FormValue("topic")
	key := ctx.FormValue("key")
	valueRaw := ctx.FormValue("value")
	errorsMsg := make(map[string]string)
	if uid == "" {
		errorsMsg["uid"] = webMsg(ctx, "error.validation.uid_required")
	}
	if topic == "" {
		errorsMsg["topic"] = webMsg(ctx, "error.validation.topic_required")
	}
	if key == "" {
		errorsMsg["key"] = webMsg(ctx, "error.validation.key_required")
	}
	value := parseConfigValue(valueRaw)
	if valueRaw != "" && value == nil {
		errorsMsg["value"] = webMsg(ctx, "error.validation.invalid_json")
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.ConfigForm(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}, true, errorsMsg).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	err := store.ModuleDataStoreFromDB().ConfigSet(context.Background(), types.Uid(uid), topic, key, value)
	if err != nil {
		return toastErrorKey(ctx, "toast.config.save_failed")
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.config.saved")
	// Remove empty-state row now that a config exists
	ctx.Response().BodyWriter().Write([]byte(`<tr id="configs-empty" hx-swap-oob="delete"></tr>`))
	return partials.ConfigRow(ctx.Context(), model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func editConfigForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	value, err := store.ModuleDataStoreFromDB().ConfigGet(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderErrorKey(ctx, "toast.config.not_found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderErrorKey(ctx, "error.load.config")
	}
	ctx.Type("html")
	return partials.ConfigForm(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}, false, nil).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func updateConfig(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	urlUID, urlTopic, urlKey, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	valueRaw := ctx.FormValue("value")
	value := parseConfigValue(valueRaw)
	if valueRaw != "" && value == nil {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.ConfigForm(model.ConfigItem{UID: urlUID, Topic: urlTopic, Key: urlKey, Value: value}, false, map[string]string{"value": webMsg(ctx, "error.validation.invalid_json")}).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	err = store.ModuleDataStoreFromDB().ConfigSet(context.Background(), types.Uid(urlUID), urlTopic, urlKey, value)
	if err != nil {
		return toastErrorKey(ctx, "toast.config.save_failed")
	}
	ctx.Type("html")
	setShowToastKey(ctx, "success", "toast.config.saved")
	return partials.ConfigRow(ctx.Context(), model.ConfigItem{UID: urlUID, Topic: urlTopic, Key: urlKey, Value: value}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func deleteConfig(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	err = store.ModuleDataStoreFromDB().ConfigDelete(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastErrorKey(ctx, "toast.config.not_found")
		}
		return toastErrorKey(ctx, "toast.config.delete_failed")
	}
	// After deletion, show empty state if no configs remain
	items, err := store.ModuleDataStoreFromDB().ListConfigs(context.Background(), store.ListConfigOptions{Limit: 1})
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		_ = partials.WriteTableEmptyOOB(
			context.Background(),
			ctx.Response().BodyWriter(),
			"configs-empty",
			"#configs-rows",
			"7",
			partials.EmptyStateHXCTA(
				webMsg(ctx, "table.empty.configs"),
				webMsg(ctx, "table.empty.configs_detail"),
				"/service/web/configs/new",
				"#configs-rows",
				"afterbegin",
				webMsg(ctx, "table.empty.configs_cta"),
			),
		)
	}
	return nil
}

func decodeConfigParams(ctx fiber.Ctx) (uid, topic, key string, err error) {
	uid, e1 := url.PathUnescape(ctx.Params("uid"))
	topic, e2 := url.PathUnescape(ctx.Params("topic"))
	key, e3 := url.PathUnescape(ctx.Params("key"))
	if e1 != nil || e2 != nil || e3 != nil {
		return "", "", "", types.Errorf(types.ErrInvalidArgument, "invalid config params")
	}
	if uid == "" || topic == "" || key == "" {
		return "", "", "", types.Errorf(types.ErrInvalidArgument, "uid, topic, and key are required")
	}
	return uid, topic, key, nil
}

// parseConfigValue parses the raw value string into types.KV.
// Valid JSON objects are used as-is. Valid non-object JSON values are
// auto-wrapped into {"value": <input>}. Returns nil if the input is empty
// or contains invalid JSON.
func parseConfigValue(raw string) types.KV {
	if raw == "" {
		return types.KV{}
	}
	var value types.KV
	if sonic.Unmarshal([]byte(raw), &value) == nil && value != nil {
		return value
	}
	if !sonic.Valid([]byte(raw)) {
		return nil
	}
	var wrapped any
	if sonic.Unmarshal([]byte(raw), &wrapped) == nil {
		return types.KV{"value": wrapped}
	}
	return nil
}
