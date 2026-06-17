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
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var tokenWebserviceRules = []webservice.Rule{
	webservice.Get("/tokens", tokensPage, route.WithNotAuth()),
	webservice.Get("/tokens/list", tokensList, route.WithNotAuth()),
	webservice.Get("/tokens/new", tokensNewForm, route.WithNotAuth()),
	webservice.Post("/tokens", tokensCreate, route.WithNotAuth()),
	webservice.Delete("/tokens/:flag", tokensRevoke, route.WithNotAuth()),
}

func tokensPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListTokens(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list tokens: %v", err)
	}
	ctx.Type("html")
	return pages.TokensPage(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensList(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListTokens(context.Background())
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load tokens")
	}
	ctx.Type("html")
	return partials.TokenTable(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	ctx.Response().BodyWriter().Write([]byte(`<tr id="token-form-new" hx-swap-oob="delete"></tr><tr id="tokens-empty" hx-swap-oob="delete"></tr>`))
	return partials.TokenForm(nil).Render(context.Background(), ctx.Response().BodyWriter())
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
		errorsMsg["uid"] = "UID is required"
	}
	if expiresVal == "" {
		errorsMsg["expires"] = "Expiry is required"
	}
	scopes := make([]string, 0, len(scopesBytes))
	for _, raw := range scopesBytes {
		val := string(raw)
		if val != "" {
			scopes = append(scopes, val)
		}
	}
	if len(scopes) == 0 {
		errorsMsg["scopes"] = "At least one scope is required"
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}

	expiresDuration, err := time.ParseDuration(expiresVal)
	if err != nil {
		errorsMsg["expires"] = "Invalid duration"
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}

	validScopes := make(map[string]bool)
	for _, s := range auth.AllScopes() {
		validScopes[s.Value] = true
	}
	for _, s := range scopes {
		if !validScopes[s] {
			errorsMsg["scopes"] = fmt.Sprintf("Invalid scope: %s", s)
			break
		}
	}
	if len(errorsMsg) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.TokenForm(errorsMsg).Render(context.Background(), ctx.Response().BodyWriter())
	}

	token, err := store.Database.CreateToken(
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
		Token:     token,
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
	return partials.TokenRow(item).Render(context.Background(), ctx.Response().BodyWriter())
}

func tokensRevoke(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	flag, err := decodeTokenParam(ctx)
	if err != nil {
		return err
	}
	err = store.Database.RevokeToken(context.Background(), flag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Token not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to revoke token")
	}
	items, err := store.Database.ListTokens(context.Background())
	if err == nil && len(items) == 0 {
		ctx.Type("html")
		ctx.Response().BodyWriter().Write([]byte(`<tr id="tokens-empty" hx-swap-oob="innerHTML:#tokens-rows"><td colspan="7" class="text-center text-base-content/50">No tokens found.</td></tr>`))
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
