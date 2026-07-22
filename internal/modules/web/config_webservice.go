package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var configWebserviceRules = []webservice.Rule{
	webservice.Get("/configs", configsPage, route.WithNotAuth()),
	webservice.Get("/configs/list", listConfigs, route.WithNotAuth()),
	webservice.Get("/configs/new", newConfigForm, route.WithNotAuth()),
	webservice.Post("/configs", createConfig, route.WithNotAuth()),
	webservice.Get("/configs/:uid/:topic/:key", getConfig, route.WithNotAuth()),
	webservice.Get("/configs/:uid/:topic/:key/edit", editConfigForm, route.WithNotAuth()),
	webservice.Put("/configs/:uid/:topic/:key", updateConfig, route.WithNotAuth()),
	webservice.Delete("/configs/:uid/:topic/:key", deleteConfig, route.WithNotAuth()),
}

func configsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListConfigs(context.Background(), store.ListConfigOptions{Limit: 100})
	if err != nil {
		return types.Errorf(types.ErrInternal, "list configs: %v", err)
	}
	ctx.Type("html")
	return pages.ConfigsPage(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func listConfigs(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListConfigs(context.Background(), store.ListConfigOptions{Limit: 100})
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load configs")
	}
	ctx.Type("html")
	return partials.ConfigTable(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func getConfig(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	value, err := store.Database.ConfigGet(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Config not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load config")
	}
	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}).Render(context.Background(), ctx.Response().BodyWriter())
}

func newConfigForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	// Remove any existing new-config form row and empty state row
	ctx.Response().BodyWriter().Write([]byte(`<tr id="config-form-new" hx-swap-oob="delete"></tr><tr id="configs-empty" hx-swap-oob="delete"></tr>`))
	return partials.ConfigForm(model.ConfigItem{}, true, nil).Render(context.Background(), ctx.Response().BodyWriter())
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
		errorsMsg["uid"] = "UID is required"
	}
	if topic == "" {
		errorsMsg["topic"] = "Topic is required"
	}
	if key == "" {
		errorsMsg["key"] = "Key is required"
	}
	value := parseConfigValue(valueRaw)
	if valueRaw != "" && value == nil {
		errorsMsg["value"] = "Invalid JSON"
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.ConfigForm(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}, true, errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}
	err := store.Database.ConfigSet(context.Background(), types.Uid(uid), topic, key, value)
	if err != nil {
		return toastError(ctx, "Failed to create config")
	}
	ctx.Type("html")
	// Remove empty-state row now that a config exists
	ctx.Response().BodyWriter().Write([]byte(`<tr id="configs-empty" hx-swap-oob="delete"></tr>`))
	return partials.ConfigRow(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}).Render(context.Background(), ctx.Response().BodyWriter())
}

func editConfigForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	value, err := store.Database.ConfigGet(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Config not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load config")
	}
	ctx.Type("html")
	return partials.ConfigForm(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}, false, nil).Render(context.Background(), ctx.Response().BodyWriter())
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
		return partials.ConfigForm(model.ConfigItem{UID: urlUID, Topic: urlTopic, Key: urlKey, Value: value}, false, map[string]string{"value": "Invalid JSON"}).Render(context.Background(), ctx.Response().BodyWriter())
	}
	err = store.Database.ConfigSet(context.Background(), types.Uid(urlUID), urlTopic, urlKey, value)
	if err != nil {
		return toastError(ctx, "Failed to update config")
	}
	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{UID: urlUID, Topic: urlTopic, Key: urlKey, Value: value}).Render(context.Background(), ctx.Response().BodyWriter())
}

func deleteConfig(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	err = store.Database.ConfigDelete(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastError(ctx, "Config not found")
		}
		return toastError(ctx, "Failed to delete config")
	}
	// After deletion, show empty state if no configs remain
	items, err := store.Database.ListConfigs(context.Background(), store.ListConfigOptions{Limit: 1})
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		_ = partials.WriteTableEmptyOOB(
			context.Background(),
			ctx.Response().BodyWriter(),
			"configs-empty",
			"#configs-rows",
			"7",
			partials.EmptyStateHXCTA(
				"No configs yet",
				"Store per-module settings as key/value pairs.",
				"/service/web/configs/new",
				"#configs-rows",
				"afterbegin",
				"Create config",
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
