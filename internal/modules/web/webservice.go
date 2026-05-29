package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/login", loginPage, route.WithNotAuth()),
	webservice.Post("/login", loginSubmit, route.WithNotAuth()),
	webservice.Post("/logout", logout, route.WithNotAuth()),
	webservice.Get("/configs", configsPage, route.WithNotAuth()),
	webservice.Get("/configs/list", listConfigs, route.WithNotAuth()),
	webservice.Get("/configs/new", newConfigForm, route.WithNotAuth()),
	webservice.Post("/configs", createConfig, route.WithNotAuth()),
	webservice.Get("/configs/:uid/:topic/:key", getConfig, route.WithNotAuth()),
	webservice.Get("/configs/:uid/:topic/:key/edit", editConfigForm, route.WithNotAuth()),
	webservice.Put("/configs/:uid/:topic/:key", updateConfig, route.WithNotAuth()),
	webservice.Delete("/configs/:uid/:topic/:key", deleteConfig, route.WithNotAuth()),
}

func isAuthenticated(ctx fiber.Ctx) bool {
	if route.GetRequestContext(ctx) != nil {
		return true
	}
	token := ctx.Cookies("accessToken")
	if token == "" {
		return false
	}
	p, err := store.Database.ParameterGet(context.Background(), token)
	if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
		return false
	}
	paramKV := types.KV(p.Params)
	uidStr, _ := paramKV.String("uid")
	uid := types.Uid(uidStr)
	if uid.IsZero() {
		return false
	}
	topic, _ := paramKV.String("topic")
	var scopes []string
	if raw, ok := paramKV["scopes"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					scopes = append(scopes, s)
				}
			}
		case []string:
			scopes = v
		}
	}
	ctx.Locals("route:ctx", &route.RequestContext{
		UID:    uid,
		Topic:  topic,
		Param:  paramKV,
		Scopes: scopes,
	})
	return true
}

func authenticateWeb(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		return nil
	}
	return redirectToLogin(ctx)
}

func redirectToLogin(ctx fiber.Ctx) error {
	next := string(ctx.Request().URI().RequestURI())
	nextEncoded := url.QueryEscape(next)
	return ctx.Redirect().To("/service/web/login?next=" + nextEncoded)
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
	// Remove any existing new-config form row to prevent accumulation
	ctx.Response().BodyWriter().Write([]byte(`<tr id="config-form-new" hx-swap-oob="delete"></tr>`))
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
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to create config")
	}
	ctx.Type("html")
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
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to update config")
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
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Config not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to delete config")
	}
	return ctx.SendStatus(http.StatusOK)
}

func loginPage(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		next := ctx.Query("next", "/service/web/configs")
		return ctx.Redirect().To(next)
	}
	next := ctx.Query("next", "")
	ctx.Type("html")
	return pages.LoginPage(next, "").Render(context.Background(), ctx.Response().BodyWriter())
}

func loginSubmit(ctx fiber.Ctx) error {
	username := ctx.FormValue("username")
	password := ctx.FormValue("password")
	next := ctx.FormValue("next")
	cfg := authConfig()
	if username == "" || username != cfg.Username || password != cfg.Password {
		ctx.Type("html")
		return pages.LoginForm(next, "Invalid username or password").Render(context.Background(), ctx.Response().BodyWriter())
	}
	token, err := auth.NewToken()
	if err != nil {
		flog.Error(fmt.Errorf("failed to generate token: %w", err))
		ctx.Type("html")
		return pages.LoginForm(next, "Internal error").Render(context.Background(), ctx.Response().BodyWriter())
	}
	uid := types.Uid("user-" + username)
	params := types.KV{
		"uid":    string(uid),
		"topic":  "web",
		"scopes": []string{"admin:*"},
	}
	expiredAt := time.Now().Add(24 * time.Hour)
	if err := store.Database.ParameterSet(context.Background(), token, params, expiredAt); err != nil {
		flog.Error(fmt.Errorf("failed to store token: %w", err))
		ctx.Type("html")
		return pages.LoginForm(next, "Internal error").Render(context.Background(), ctx.Response().BodyWriter())
	}
	ctx.Cookie(&fiber.Cookie{
		Name:     "accessToken",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   86400,
	})
	if next == "" || !strings.HasPrefix(next, "/") || strings.Contains(next, "//") || strings.Contains(next, ":") {
		next = "/service/web/configs"
	}
	ctx.Set("HX-Redirect", next)
	return nil
}

func logout(ctx fiber.Ctx) error {
	token := ctx.Cookies("accessToken")
	if token != "" {
		if err := store.Database.ParameterDelete(context.Background(), token); err != nil {
			flog.Error(fmt.Errorf("failed to delete token on logout: %w", err))
		}
	}
	ctx.Cookie(&fiber.Cookie{
		Name:     "accessToken",
		Value:    "",
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   0,
	})
	return ctx.Redirect().To("/service/web/login")
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
	if sonic.Unmarshal([]byte(raw), &value) == nil {
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

func renderError(ctx fiber.Ctx, msg string) error {
	ctx.Type("html")
	_, err := ctx.Write([]byte(`<div class="text-red-500 text-sm py-2">` + msg + `</div>`))
	return err
}
