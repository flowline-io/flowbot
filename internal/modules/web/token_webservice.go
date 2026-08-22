package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var tokenWebserviceRules = []webservice.Rule{
	webservice.Get("/tokens", tokensPage),
	webservice.Get("/tokens/list", tokensList),
	webservice.Get("/tokens/new", tokensNewForm),
	webservice.Post("/tokens", tokensCreate),
	webservice.Delete("/tokens/:flag", tokensRevoke),
}

func tokensPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.ModuleDataStoreFromDB().ListTokens(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list tokens: %v", err)
	}
	ctx.Type("html")
	return pages.TokensPage(ctx.Context(), items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func tokensList(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.ModuleDataStoreFromDB().ListTokens(context.Background())
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderErrorKey(ctx, "error.load.tokens")
	}
	ctx.Type("html")
	return partials.TokenTable(ctx.Context(), items).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func tokensNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="token-form-new" hx-swap-oob="delete"></tr><tr id="tokens-empty" hx-swap-oob="delete"></tr>`))
	return partials.TokenForm(nil).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func tokensCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uidVal := strings.TrimSpace(ctx.FormValue("uid"))
	expiresVal := ctx.FormValue("expires")
	args := ctx.RequestCtx().PostArgs()
	scopesBytes := args.PeekMulti("scopes")

	errorsMsg := make(map[string]string)
	if uidVal == "" {
		errorsMsg["uid"] = webMsg(ctx, "error.validation.uid_required")
	}
	if expiresVal == "" {
		errorsMsg["expires"] = webMsg(ctx, "error.validation.expiry_required")
	}
	scopes := make([]string, 0, len(scopesBytes))
	for _, raw := range scopesBytes {
		val := string(raw)
		if val != "" {
			scopes = append(scopes, val)
		}
	}
	if len(scopes) == 0 {
		errorsMsg["scopes"] = webMsg(ctx, "error.validation.scopes_required")
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	expiresDuration, err := time.ParseDuration(expiresVal)
	if err != nil {
		errorsMsg["expires"] = webMsg(ctx, "error.validation.invalid_duration")
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	validScopes := make(map[string]bool)
	for _, s := range auth.AllScopes() {
		validScopes[s.Value] = true
	}
	for _, s := range scopes {
		if !validScopes[s] {
			errorsMsg["scopes"] = webMsgData(ctx, "error.validation.invalid_scope", map[string]any{"Scope": s})
			break
		}
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	token, err := store.ModuleDataStoreFromDB().CreateToken(
		context.Background(),
		types.Uid(uidVal),
		time.Now().Add(expiresDuration),
		scopes,
	)
	if err != nil {
		return types.Errorf(types.ErrInternal, "create token: %v", err)
	}

	now := time.Now()
	item := model.TokenItem{
		Token:     auth.HashToken(token),
		UID:       types.Uid(uidVal),
		Scopes:    scopes,
		CreatedAt: now,
		ExpiredAt: now.Add(expiresDuration),
	}

	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="tokens-empty" hx-swap-oob="delete"></tr>`))
	alert := fmt.Sprintf(
		`<div data-testid="token-created-alert" hx-swap-oob="innerHTML:#token-alert-container" class="alert alert-success"><span><strong>Token created:</strong> <code class="font-mono text-xs">%s</code></span><button class="btn btn-ghost btn-xs" data-testid="token-copy-btn" data-token=%q onclick="navigator.clipboard.writeText(this.dataset.token);this.textContent='Copied!'">Copy</button></div>`,
		token, token,
	)
	ctx.Response().BodyWriter().Write([]byte(alert))
	return partials.TokenRow(ctx.Context(), item).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func tokensRevoke(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeTokenParam(ctx)
	if err != nil {
		return err
	}
	err = store.ModuleDataStoreFromDB().RevokeToken(context.Background(), flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return toastErrorKey(ctx, "toast.token.not_found")
		}
		return toastErrorKey(ctx, "toast.token.revoke_failed")
	}
	items, err := store.ModuleDataStoreFromDB().ListTokens(context.Background())
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		_ = partials.WriteTableEmptyOOB(
			context.Background(),
			ctx.Response().BodyWriter(),
			"tokens-empty",
			"#tokens-rows",
			"7",
			partials.EmptyStateHXCTA(
				webMsg(ctx, "table.empty.tokens"),
				webMsg(ctx, "table.empty.tokens_detail"),
				"/service/web/tokens/new",
				"#tokens-rows",
				"afterbegin",
				webMsg(ctx, "table.empty.tokens_cta"),
			),
		)
	}
	return nil
}

func decodeTokenParam(ctx fiber.Ctx) (string, error) {
	flag := ctx.Params("flag")
	if flag == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "flag is required")
	}
	return flag, nil
}
