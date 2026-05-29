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

var webserviceRules = []webservice.Rule{
	webservice.Get("/configs", configsPage),
	webservice.Get("/configs/list", listConfigs),
	webservice.Get("/configs/new", newConfigForm),
	webservice.Post("/configs", createConfig),
	webservice.Get("/configs/:uid/:topic/:key", getConfig),
	webservice.Get("/configs/:uid/:topic/:key/edit", editConfigForm),
	webservice.Put("/configs/:uid/:topic/:key", updateConfig),
	webservice.Delete("/configs/:uid/:topic/:key", deleteConfig),
}

func requireAuth(ctx fiber.Ctx) error {
	if route.GetRequestContext(ctx) == nil {
		return ctx.SendStatus(http.StatusUnauthorized)
	}
	return nil
}

func configsPage(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
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
	if err := requireAuth(ctx); err != nil {
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
	if err := requireAuth(ctx); err != nil {
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
	if err := requireAuth(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return partials.ConfigForm(model.ConfigItem{}, true, nil).Render(context.Background(), ctx.Response().BodyWriter())
}

func createConfig(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
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
	var value types.KV
	if valueRaw != "" {
		if err := sonic.Unmarshal([]byte(valueRaw), &value); err != nil {
			errorsMsg["value"] = "Invalid JSON"
		}
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.ConfigForm(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}, true, errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}
	err := store.Database.ConfigSet(context.Background(), types.Uid(uid), topic, key, value)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to create config")
	}
	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{UID: uid, Topic: topic, Key: key, Value: value}).Render(context.Background(), ctx.Response().BodyWriter())
}

func editConfigForm(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
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
	if err := requireAuth(ctx); err != nil {
		return err
	}
	urlUID, urlTopic, urlKey, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	valueRaw := ctx.FormValue("value")
	var value types.KV
	if valueRaw != "" {
		if err := sonic.Unmarshal([]byte(valueRaw), &value); err != nil {
			ctx.Status(http.StatusUnprocessableEntity)
			ctx.Type("html")
			return partials.ConfigForm(model.ConfigItem{UID: urlUID, Topic: urlTopic, Key: urlKey, Value: value}, false, map[string]string{"value": "Invalid JSON"}).Render(context.Background(), ctx.Response().BodyWriter())
		}
	}
	err = store.Database.ConfigSet(context.Background(), types.Uid(urlUID), urlTopic, urlKey, value)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to update config")
	}
	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{UID: urlUID, Topic: urlTopic, Key: urlKey, Value: value}).Render(context.Background(), ctx.Response().BodyWriter())
}

func deleteConfig(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	err = store.Database.ConfigDelete(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Config not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to delete config")
	}
	return ctx.SendStatus(http.StatusOK)
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

func renderError(ctx fiber.Ctx, msg string) error {
	ctx.Type("html")
	_, err := ctx.Write([]byte(`<div class="text-red-500 text-sm py-2">` + msg + `</div>`))
	return err
}
