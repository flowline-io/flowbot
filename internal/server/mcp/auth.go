// Package mcp serves Flowbot capabilities over Streamable HTTP at /mcp.
package mcp

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

// Identity is the authenticated MCP caller (API token, never a web session).
type Identity struct {
	UID       types.Uid
	Scopes    []string
	IP        string
	UserAgent string
}

func authenticate(c fiber.Ctx) (Identity, error) {
	raw := headerToken(c.Get("X-AccessToken"), c.Get(fiber.HeaderAuthorization))
	if raw == "" {
		return Identity{}, protocol.ErrNotAuthorized.New("Missing token")
	}
	p, err := route.LookupAccessToken(c.Context(), raw)
	if err != nil || p.ID <= 0 || route.AccessTokenIsExpired(p) {
		return Identity{}, protocol.ErrNotAuthorized.New("parameter error")
	}
	paramKV := types.KV(p.Params)
	uidStr, _ := paramKV.String("uid")
	uid := types.Uid(uidStr)
	if uid.IsZero() {
		return Identity{}, protocol.ErrNotAuthorized.New("uid empty")
	}
	kind, _ := paramKV.String("kind")
	if kind == webauth.KindFull {
		return Identity{}, types.Errorf(types.ErrForbidden, "web sessions cannot call /mcp")
	}
	scopes := parseScopes(paramKV)
	if !auth.HasAnyScope(scopes) {
		return Identity{}, protocol.ErrNotAuthorized.New("token has no scopes")
	}
	return Identity{
		UID:       uid,
		Scopes:    scopes,
		IP:        c.IP(),
		UserAgent: c.Get(fiber.HeaderUserAgent),
	}, nil
}

func headerToken(xAccessToken, authorization string) string {
	if t := strings.TrimSpace(xAccessToken); t != "" {
		return t
	}
	authorization = strings.TrimSpace(authorization)
	const prefix = "Bearer "
	if len(authorization) < len(prefix) || !strings.EqualFold(authorization[:len(prefix)], prefix) {
		return ""
	}
	return auth.ExtractBearerToken(authorization)
}

func parseScopes(paramKV types.KV) []string {
	raw, ok := paramKV["scopes"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}
