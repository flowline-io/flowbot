package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
)

// metricsAuth protects GET /metrics. Access is granted when either:
//  1. Authorization Bearer / X-AccessToken matches metrics.bearer_token, or
//  2. a valid access token carries admin:metrics (or admin:*) scope.
func metricsAuth(c fiber.Ctx) error {
	token := metricsRequestToken(c)
	if bearer := config.App.Metrics.BearerToken; bearer != "" && secureTokenEqual(token, bearer) {
		return c.Next()
	}
	return route.Authorize(route.RequireScope(auth.ScopeAdminMetrics, func(ctx fiber.Ctx) error {
		return ctx.Next()
	}))(c)
}

// metricsRequestToken extracts the scrape credential from Authorization or X-AccessToken.
func metricsRequestToken(c fiber.Ctx) string {
	if v := c.Get("X-AccessToken"); v != "" {
		return v
	}
	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(c.RequestCtx(), &r, true); err != nil {
		return auth.ExtractBearerToken(c.Get(fiber.HeaderAuthorization))
	}
	return route.GetAccessToken(&r)
}

// secureTokenEqual compares a and b without leaking length via early return.
// Both values are hashed first so ConstantTimeCompare always runs on fixed-size digests.
func secureTokenEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
