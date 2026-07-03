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
	raw, err := sonic.MarshalString(view.Effective)
	if err != nil {
		return types.Errorf(types.ErrInternal, "marshal permissions: %v", err)
	}
	ctx.Type("html")
	return pages.ChatAgentPermissionsPage(raw).Render(context.Background(), ctx.Response().BodyWriter())
}

func chatAgentPermissionsSave(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	body := strings.TrimSpace(ctx.FormValue("rules"))
	if body == "" {
		ctx.Status(http.StatusBadRequest)
		return renderError(ctx, "Rules JSON is required")
	}
	cfg, err := permission.ParseConfig([]byte(body))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return renderError(ctx, "Invalid permission JSON")
	}
	if err := chatagent.SaveUserPermissions(ctx.Context(), uid, cfg); err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			ctx.Status(http.StatusBadRequest)
			return renderError(ctx, err.Error())
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
