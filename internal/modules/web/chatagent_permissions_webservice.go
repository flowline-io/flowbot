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
	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	appconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var chatAgentPermissionsWebserviceRules = []webservice.Rule{
	webservice.Get("/chatagent-settings", chatAgentPermissionsPage),
	webservice.Post("/chatagent-settings", chatAgentPermissionsSave),
	webservice.Post("/chatagent-settings/reset", chatAgentPermissionsReset),
	webservice.Post("/chatagent-settings/reset-server-defaults", chatAgentPermissionsResetServerDefaults),
}

func chatAgentPermissionsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	svc, err := chatAgentService()
	if err != nil {
		return err
	}
	view, err := svc.BuildPermissionsView(ctx.Context(), uid, "")
	if err != nil {
		return types.Errorf(types.ErrInternal, "load permissions: %v", err)
	}
	return renderChatAgentPermissionsPage(ctx, uid, view, nil, permission.FormValues{})
}

func chatAgentPermissionsSave(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	svc, err := chatAgentService()
	if err != nil {
		return err
	}
	view, err := svc.BuildPermissionsView(ctx.Context(), uid, "")
	if err != nil {
		return types.Errorf(types.ErrInternal, "load permissions: %v", err)
	}

	cfg, submitted, fieldErrors, err := parseChatAgentPermissionsSubmit(ctx, view)
	if err != nil || len(fieldErrors) > 0 {
		return renderChatAgentPermissionSubmitError(ctx, uid, view, fieldErrors, submitted, err)
	}

	mode, modeErr := chatagent.ParseApprovalMode(strings.TrimSpace(ctx.FormValue("approval_mode")))
	if modeErr != nil {
		ctx.Status(http.StatusBadRequest)
		return renderChatAgentPermissionsPage(ctx, uid, view, map[string]string{"_form": modeErr.Error()}, submitted)
	}

	if err := persistChatAgentPermissionsSave(ctx, uid, view, cfg, mode, submitted); err != nil {
		return err
	}
	ctx.Redirect().To("/service/web/chatagent-settings")
	return nil
}

func parseChatAgentPermissionsSubmit(ctx fiber.Ctx, view chatagent.PermissionsView) (permission.Config, permission.FormValues, map[string]string, error) {
	submitMode := strings.TrimSpace(ctx.FormValue("submit_mode"))
	if submitMode == "" {
		submitMode = "form"
	}
	switch submitMode {
	case "json":
		cfg, fieldErrors, err := parsePermissionJSON(ctx)
		return cfg, permission.FormValues{}, fieldErrors, err
	default:
		submitted := permission.ParseFormPostArgs(collectFormArgs(ctx))
		cfg, fieldErrors, err := permission.BuildUserConfigFromForm(view.Defaults, submitted)
		return cfg, submitted, fieldErrors, err
	}
}

func renderChatAgentPermissionSubmitError(
	ctx fiber.Ctx,
	uid types.Uid,
	view chatagent.PermissionsView,
	fieldErrors map[string]string,
	submitted permission.FormValues,
	err error,
) error {
	if len(fieldErrors) > 0 {
		ctx.Status(http.StatusBadRequest)
		return renderChatAgentPermissionsPage(ctx, uid, view, fieldErrors, submitted)
	}
	formErr := map[string]string{"_form": err.Error()}
	if err == nil {
		formErr["_form"] = webMsg(ctx, "error.validation.permissions_form_invalid")
	}
	ctx.Status(http.StatusBadRequest)
	return renderChatAgentPermissionsPage(ctx, uid, view, formErr, submitted)
}

func persistChatAgentPermissionsSave(
	ctx fiber.Ctx,
	uid types.Uid,
	view chatagent.PermissionsView,
	cfg permission.Config,
	mode approval.Mode,
	submitted permission.FormValues,
) error {
	if err := chatagent.SaveUserPermissions(ctx.Context(), uid, cfg); err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			ctx.Status(http.StatusBadRequest)
			return renderChatAgentPermissionsPage(ctx, uid, view, map[string]string{"_form": err.Error()}, submitted)
		}
		return types.Errorf(types.ErrInternal, "save permissions: %v", err)
	}
	if err := chatagent.SaveUserApprovalMode(ctx.Context(), uid, mode); err != nil {
		return types.Errorf(types.ErrInternal, "save approval mode: %v", err)
	}
	if strings.TrimSpace(ctx.FormValue("submit_mode")) != "json" {
		if err := saveChatAgentServerDefaultsFromForm(ctx, view, uid, submitted); err != nil {
			return err
		}
	}
	return nil
}

func saveChatAgentServerDefaultsFromForm(ctx fiber.Ctx, view chatagent.PermissionsView, uid types.Uid, submitted permission.FormValues) error {
	if err := chatagent.SaveServerDefaults(ctx.Context(), chatagent.ServerDefaultsFormInput{
		ChatModel:     strings.TrimSpace(ctx.FormValue("server_chat_model")),
		ToolModel:     strings.TrimSpace(ctx.FormValue("server_tool_model")),
		ThinkingLevel: strings.TrimSpace(ctx.FormValue("server_thinking_level")),
	}); err != nil {
		if errors.Is(err, types.ErrInvalidArgument) {
			ctx.Status(http.StatusBadRequest)
			return renderChatAgentPermissionsPage(ctx, uid, view, map[string]string{"_form": err.Error()}, submitted)
		}
		return types.Errorf(types.ErrInternal, "save server defaults: %v", err)
	}
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
	ctx.Redirect().To("/service/web/chatagent-settings")
	return nil
}

func chatAgentPermissionsResetServerDefaults(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	if err := chatagent.DeleteServerDefaults(ctx.Context()); err != nil {
		return types.Errorf(types.ErrInternal, "reset server defaults: %v", err)
	}
	ctx.Redirect().To("/service/web/chatagent-settings")
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
	for key, value := range ctx.Request().PostArgs().All() {
		args[string(key)] = string(value)
	}
	return args
}

func renderChatAgentPermissionsPage(
	ctx fiber.Ctx,
	uid types.Uid,
	view chatagent.PermissionsView,
	fieldErrors map[string]string,
	submitted permission.FormValues,
) error {
	userJSON, err := marshalUserPermissionsJSON(view.User)
	if err != nil {
		return types.Errorf(types.ErrInternal, "marshal permissions: %v", err)
	}
	fields := partials.BuildPermissionFormFields(ctx.Context(), mapPermissionsView(view))
	if len(submitted.Simple) > 0 || len(submitted.Patterns) > 0 {
		fields = partials.ApplySubmittedPermissionForm(fields, submitted)
	}
	mode, err := chatagent.LoadUserApprovalMode(ctx.Context(), uid)
	if err != nil {
		return types.Errorf(types.ErrInternal, "load approval mode: %v", err)
	}
	if submittedMode := strings.TrimSpace(ctx.FormValue("approval_mode")); submittedMode != "" {
		if parsed, parseErr := chatagent.ParseApprovalMode(submittedMode); parseErr == nil {
			mode = parsed
		}
	}
	data := partials.PermissionFormPageData{
		Fields:            fields,
		UserJSON:          userJSON,
		Errors:            fieldErrors,
		ApprovalMode:      string(mode),
		ApprovalServerDef: appconfig.ChatAgentApprovalModeDefault(),
		ServerDefaults:    buildServerDefaultsFormData(ctx.Context(), ctx),
	}
	ctx.Type("html")
	return pages.ChatAgentPermissionsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func buildServerDefaultsFormData(ctx context.Context, fiberCtx fiber.Ctx) partials.ServerDefaultsFormData {
	yamlChat := appconfig.ChatAgentChatModel()
	yamlTool := appconfig.App.ChatAgent.ToolModel
	stored, err := chatagent.LoadServerDefaults(ctx)
	if err != nil {
		stored = chatagent.ServerDefaults{}
	}
	data := partials.ServerDefaultsFormData{
		YAMLChatModel:         yamlChat,
		YAMLToolModel:         yamlTool,
		InheritValue:          chatagent.ServerDefaultFormInherit,
		ToolNoneValue:         chatagent.ServerDefaultToolNone,
		SelectableModels:      selectableModelOptions(ctx),
		SelectedChatModel:     chatagent.ServerDefaultFormInherit,
		SelectedToolModel:     chatagent.ServerDefaultFormInherit,
		SelectedThinkingLevel: chatagent.ServerDefaultFormInherit,
	}
	if stored.ChatModelSet {
		data.SelectedChatModel = stored.ChatModel
		data.ChatModelOverridden = true
	}
	if stored.ToolModelSet {
		if stored.ToolModel == "" {
			data.SelectedToolModel = chatagent.ServerDefaultToolNone
		} else {
			data.SelectedToolModel = stored.ToolModel
		}
		data.ToolModelOverridden = true
	}
	if stored.ThinkingLevelSet {
		data.SelectedThinkingLevel = stored.ThinkingLevel
		data.ThinkingOverridden = true
	}
	if fiberCtx != nil {
		data = partials.ApplySubmittedServerDefaults(
			data,
			chatagent.ServerDefaultFormInherit,
			fiberCtx.FormValue("server_chat_model"),
			fiberCtx.FormValue("server_tool_model"),
			fiberCtx.FormValue("server_thinking_level"),
		)
	}
	return data
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

func ensureWebSessionExists(ctx fiber.Ctx, sessionID string) error {
	_, err := store.ChatStoreFromDB().GetChatSession(ctx.Context(), sessionID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.ErrNotFound
		}
		return err
	}
	return nil
}

func ensureWebSessionOwner(ctx fiber.Ctx, sessionID string) error {
	uid, err := webUID(ctx)
	if err != nil {
		return err
	}
	row, err := store.ChatStoreFromDB().GetChatSession(ctx.Context(), sessionID)
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
