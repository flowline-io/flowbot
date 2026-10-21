package web

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"

	serversmcp "github.com/flowline-io/flowbot/internal/server/mcp"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/i18n"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var mcpSettingsWebserviceRules = []webservice.Rule{
	webservice.Get("/settings/mcp", mcpSettingsPage),
	webservice.Post("/settings/mcp/token", mcpSettingsCreateToken),
}

func mcpSettingsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	data := mcpSettingsFromQuery(ctx)
	ctx.Type("html")
	return pages.MCPSettingsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func mcpSettingsCreateToken(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	data := mcpSettingsFromForm(ctx)
	uid := getUID(ctx)
	if uid == "" {
		data.Error = webMsg(ctx, "error.validation.uid_required")
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return pages.MCPSettingsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	expiresVal := ctx.FormValue("expires")
	if expiresVal == "" {
		expiresVal = "2160h"
	}
	expires, err := time.ParseDuration(expiresVal)
	if err != nil || expires <= 0 {
		data.Error = webMsg(ctx, "error.validation.invalid_duration")
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return pages.MCPSettingsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if len(data.Scopes) == 0 || !auth.CanGrantScopes(route.GetScopes(ctx), data.Scopes) {
		data.Error = webMsg(ctx, "error.validation.scope_elevation")
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return pages.MCPSettingsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	token, err := store.ModuleDataStoreFromDB().CreateToken(context.Background(), types.Uid(uid), time.Now().Add(expires), data.Scopes)
	if err != nil {
		return types.WrapError(types.ErrInternal, "create token", err)
	}
	data.Token = token
	ctx.Type("html")
	return pages.MCPSettingsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func mcpSettingsFromQuery(ctx fiber.Ctx) pages.MCPSettingsData {
	return mcpSettingsFromValues(ctx, ctx.Query("preset", "read"), ctx.Query("url"), byteValues(ctx.Request().URI().QueryArgs().PeekMulti("scopes")))
}

func mcpSettingsFromForm(ctx fiber.Ctx) pages.MCPSettingsData {
	return mcpSettingsFromValues(ctx, ctx.FormValue("preset", "read"), ctx.FormValue("url"), byteValues(ctx.RequestCtx().PostArgs().PeekMulti("scopes")))
}

func mcpSettingsFromValues(ctx fiber.Ctx, preset, endpoint string, scopes []string) pages.MCPSettingsData {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = mcpRequestEndpoint(ctx)
	}
	return buildMCPSettings(ctx, preset, wizardScopes(preset, scopes), endpoint, "")
}

func buildMCPSettings(ctx fiber.Ctx, preset string, scopes []string, endpoint, token string) pages.MCPSettingsData {
	if preset != "read" && preset != "run" && preset != "custom" {
		preset = "read"
	}
	tools := serversmcp.FilterByScopes(serversmcp.BuildCatalog(hub.Default.List(), pkgconfig.App.MCP.Include, pkgconfig.App.MCP.Exclude), scopes)
	preview := make([]pages.MCPToolPreview, 0, len(tools))
	for _, tool := range tools {
		label, kind := tierLabel(ctx, tool.Group, tool.Operation)
		preview = append(preview, pages.MCPToolPreview{Name: tool.Name, Tier: label, Kind: kind})
	}
	custom := make([]pages.MCPScopeChoice, 0)
	selected := map[string]bool{}
	for _, scope := range scopes {
		selected[scope] = true
	}
	for _, info := range auth.MCPWizardScopes() {
		custom = append(custom, pages.MCPScopeChoice{Value: info.Value, Checked: selected[info.Value]})
	}
	return pages.MCPSettingsData{
		Enabled:   pkgconfig.App.MCP.Enabled,
		URL:       endpoint,
		Preset:    preset,
		Scopes:    scopes,
		Custom:    custom,
		Tools:     preview,
		Snippets:  mcpSnippets(endpoint),
		Token:     token,
		CSRFToken: csrf.TokenFromContext(ctx),
	}
}

func wizardScopes(preset string, custom []string) []string {
	switch preset {
	case "run":
		return auth.MCPRunScopes()
	case "custom":
		allowed := map[string]bool{}
		for _, info := range auth.MCPWizardScopes() {
			allowed[info.Value] = true
		}
		out := make([]string, 0, len(custom))
		for _, scope := range custom {
			scope = strings.TrimSpace(scope)
			if scope == "" || scope == auth.ScopeAdmin || !allowed[scope] {
				continue
			}
			out = append(out, scope)
		}
		return out
	default:
		return auth.MCPReadScopes()
	}
}

func tierLabel(ctx fiber.Ctx, group, op string) (label, kind string) {
	tier, _ := capability.Classify(group, op)
	switch tier {
	case capability.TierDestructive:
		return i18n.T(ctx, "page.mcp.tier_destructive"), "destructive"
	case capability.TierWrite:
		return i18n.T(ctx, "page.mcp.tier_write"), "write"
	default:
		return i18n.T(ctx, "page.mcp.tier_read"), "read"
	}
}

func mcpSnippets(endpoint string) []pages.MCPSnippet {
	clients := []pkgmcp.Client{pkgmcp.ClientCodex, pkgmcp.ClientClaude, pkgmcp.ClientCursor, pkgmcp.ClientGeneric}
	out := make([]pages.MCPSnippet, 0, len(clients))
	for _, client := range clients {
		snip, err := pkgmcp.RenderSnippet(client, pkgmcp.SnippetOptions{URL: endpoint})
		if err != nil {
			continue
		}
		out = append(out, pages.MCPSnippet{Client: snip.Client, Target: snip.Target, Snippet: snip.Snippet})
	}
	return out
}

func mcpRequestEndpoint(ctx fiber.Ctx) string {
	scheme := string(ctx.Request().URI().Scheme())
	if scheme == "" {
		scheme = "http"
	}
	host := string(ctx.Request().URI().Host())
	if host == "" {
		host = ctx.Hostname()
	}
	return scheme + "://" + host + "/mcp"
}

func byteValues(raw [][]byte) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		out = append(out, string(item))
	}
	return out
}
