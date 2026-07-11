package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var chatAgentPermissionsWebserviceRules = []webservice.Rule{
	webservice.Get("/chatagent-permissions", chatAgentPermissionsPage, route.WithNotAuth()),
	webservice.Post("/chatagent-permissions", chatAgentPermissionsSave, route.WithNotAuth()),
	webservice.Post("/chatagent-permissions/reset", chatAgentPermissionsReset, route.WithNotAuth()),
}

func chatAgentPermissionsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	view, err := chatagent.BuildPermissionsView(ctx.Context(), uid, "")
	if err != nil {
		return types.Errorf(types.ErrInternal, "load permissions: %v", err)
	}
	return renderChatAgentPermissionsPage(ctx, view, nil, permission.FormValues{})
}

func chatAgentPermissionsSave(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	view, err := chatagent.BuildPermissionsView(ctx.Context(), uid, "")
	if err != nil {
		return types.Errorf(types.ErrInternal, "load permissions: %v", err)
	}

	submitMode := strings.TrimSpace(ctx.FormValue("submit_mode"))
	if submitMode == "" {
		submitMode = "form"
	}

	var (
		cfg         permission.Config
		fieldErrors map[string]string
		submitted   permission.FormValues
	)
	switch submitMode {
	case "json":
		cfg, fieldErrors, err = parsePermissionJSON(ctx)
	default:
		submitted = permission.ParseFormPostArgs(collectFormArgs(ctx))
		cfg, fieldErrors, err = permission.BuildUserConfigFromForm(view.Defaults, submitted)
	}
	if err != nil {
		if len(fieldErrors) > 0 {
			ctx.Status(http.StatusBadRequest)
			return renderChatAgentPermissionsPage(ctx, view, fieldErrors, submitted)
		}
		if errors.Is(err, types.ErrInvalidArgument) {
			ctx.Status(http.StatusBadRequest)
			return renderChatAgentPermissionsPage(ctx, view, map[string]string{"_form": err.Error()}, submitted)
		}
		ctx.Status(http.StatusBadRequest)
		return renderChatAgentPermissionsPage(ctx, view, map[string]string{"_form": err.Error()}, submitted)
	}

	if err := chatagent.SaveUserPermissions(ctx.Context(), uid, cfg); err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			ctx.Status(http.StatusBadRequest)
			return renderChatAgentPermissionsPage(ctx, view, map[string]string{"_form": err.Error()}, submitted)
		}
		return types.Errorf(types.ErrInternal, "save permissions: %v", err)
	}
	ctx.Redirect().To("/service/web/chatagent-permissions")
	return nil
}

func chatAgentPermissionsReset(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	if err := chatagent.DeleteUserPermissions(ctx.Context(), uid); err != nil {
		return types.Errorf(types.ErrInternal, "reset permissions: %v", err)
	}
	ctx.Redirect().To("/service/web/chatagent-permissions")
	return nil
}

func parsePermissionJSON(ctx fiber.Ctx) (permission.Config, map[string]string, error) {
	body := strings.TrimSpace(ctx.FormValue("rules"))
	if body == "" {
		return nil, map[string]string{"rules": "Rules JSON is required"}, errors.New("rules required")
	}
	cfg, err := permission.ParseConfig([]byte(body))
	if err != nil {
		return nil, map[string]string{"rules": "Invalid permission JSON"}, err
	}
	if err := permission.ValidateUserConfig(cfg); err != nil {
		return nil, map[string]string{"rules": err.Error()}, types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return cfg, nil, nil
}

func collectFormArgs(ctx fiber.Ctx) map[string]string {
	args := make(map[string]string)
	ctx.Request().PostArgs().VisitAll(func(key, value []byte) {
		args[string(key)] = string(value)
	})
	return args
}

func renderChatAgentPermissionsPage(
	ctx fiber.Ctx,
	view chatagent.PermissionsView,
	fieldErrors map[string]string,
	submitted permission.FormValues,
) error {
	userJSON, err := marshalUserPermissionsJSON(view.User)
	if err != nil {
		return types.Errorf(types.ErrInternal, "marshal permissions: %v", err)
	}
	fields := partials.BuildPermissionFormFields(view)
	if len(submitted.Simple) > 0 || len(submitted.Patterns) > 0 {
		fields = partials.ApplySubmittedPermissionForm(fields, submitted)
	}
	data := partials.PermissionFormPageData{
		Fields:   fields,
		UserJSON: userJSON,
		Errors:   fieldErrors,
	}
	ctx.Type("html")
	return pages.ChatAgentPermissionsPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func marshalUserPermissionsJSON(user permission.Config) (string, error) {
	if len(user) == 0 {
		return "{}", nil
	}
	return sonic.MarshalString(user)
}

func webUID(ctx fiber.Ctx) (types.Uid, error) {
	rc := route.GetRequestContext(ctx)
	if rc == nil || rc.UID.IsZero() {
		return types.Uid(""), types.ErrUnauthorized
	}
	return rc.UID, nil
}

func ensureWebSessionOwner(ctx fiber.Ctx, sessionID string) error {
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	row, err := store.Database.GetChatSession(ctx.Context(), sessionID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.ErrNotFound
		}
		return err
	}
	if row.UID != uid.String() {
		return types.ErrForbidden
	}
	return nil
}
