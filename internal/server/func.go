package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"

	// OAuth provider registration — these blank imports trigger init() which
	// self-registers each provider in the providers.OAuthProvider registry.
	_ "github.com/flowline-io/flowbot/pkg/providers/dropbox"
	_ "github.com/flowline-io/flowbot/pkg/providers/github"
	_ "github.com/flowline-io/flowbot/pkg/providers/slack"
)

var cacheStore *cache.RedisStore

// SetCacheStore sets the cache store for server functions.
func SetCacheStore(s *cache.RedisStore) {
	cacheStore = s
}

// auth pprof middleware for pprof routes
func authPprof(ctx fiber.Ctx) bool {
	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(ctx.RequestCtx(), &r, true); err != nil {
		flog.Error(fmt.Errorf("pprof auth error: %w", err))
		return true
	}

	if !strings.Contains(ctx.Path(), "/debug/pprof") {
		return true
	}

	accessToken := route.GetAccessToken(&r)
	if accessToken == "" {
		flog.Warn("pprof auth warning: missing token")
		return true
	}

	p, err := store.Database.ParameterGet(ctx.Context(), accessToken)
	if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
		flog.Warn("pprof auth warning: parameter error")
		return true
	}

	return false
}

type structValidator struct {
	validate *validator.Validate
}

// Validator needs to implement the Validate method
func (v *structValidator) Validate(out any) error {
	return v.validate.Struct(out)
}
