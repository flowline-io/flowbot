// Package route provides HTTP route registration and discovery.
package route

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

const prefix = "service"

func WebService(app *fiber.App, group string, rs ...*Router) {
	path := "/" + prefix + "/" + group
	for _, router := range rs {
		switch router.Method {
		case "GET":
			app.Get(path+router.Path, authorizeWithLevel(router.AuthLevel, router.Function))
		case "POST":
			app.Post(path+router.Path, authorizeWithLevel(router.AuthLevel, router.Function))
		case "PUT":
			app.Put(path+router.Path, authorizeWithLevel(router.AuthLevel, router.Function))
		case "PATCH":
			app.Patch(path+router.Path, authorizeWithLevel(router.AuthLevel, router.Function))
		case "DELETE":
			app.Delete(path+router.Path, authorizeWithLevel(router.AuthLevel, router.Function))
		default:
			continue
		}
	}
}

func Route(method, path string, function fiber.Handler, options ...Option) *Router {
	r := &Router{
		Method:   method,
		Path:     path,
		Function: function,
	}
	for _, option := range options {
		option(r)
	}
	return r
}

type Option func(r *Router)

type Router struct {
	Method        string
	Path          string
	Function      fiber.Handler
	Documentation string
	AuthLevel     AuthLevel
}

type AuthLevel int

const (
	NoAuth AuthLevel = 1
)

func WithNotAuth() Option {
	return func(r *Router) {
		r.AuthLevel = NoAuth
	}
}

var routeAuditor audit.Auditor

// SetAuditor sets the global auditor used for auth event recording.
func SetAuditor(a audit.Auditor) {
	routeAuditor = a
}

func ErrorResponse(ctx fiber.Ctx, text string) error {
	ctx = ctx.Status(http.StatusBadRequest)
	return ctx.SendString(text)
}

// RequestContext is a typed struct stored in fiber.Locals after authorization.
// It carries the authenticated user identity, request metadata, and scopes.
type RequestContext struct {
	UID    types.Uid
	Topic  string
	Param  types.KV
	Scopes []string
}

const requestContextKey = "route:ctx"

const (
	accessTokenKey = "accessToken"
)

func authorizeWithLevel(authLevel AuthLevel, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if authLevel == NoAuth {
			return handler(ctx)
		}
		return Authorize(handler)(ctx)
	}
}

func Authorize(handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		accessToken, err := resolveAccessToken(ctx)
		if err != nil {
			return err
		}

		p, err := LookupAccessToken(context.Background(), accessToken)
		if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
			auditAuthReject(ctx, "auth.token.validate.fail", "invalid or expired")
			return protocol.ErrNotAuthorized.New("parameter error")
		}

		paramKV := types.KV(p.Params)
		topic, _ := paramKV.String("topic")
		uidStr, _ := paramKV.String("uid")
		uid := types.Uid(uidStr)

		if uid.IsZero() {
			auditAuthReject(ctx, "auth.token.validate.fail", "uid empty")
			return protocol.ErrNotAuthorized.New("uid empty")
		}

		scopes := parseScopes(paramKV)
		throttledUpdateLastUsed(paramKV, p.Flag, p.ExpiredAt)

		ctx.Locals(requestContextKey, &RequestContext{
			UID:    uid,
			Topic:  topic,
			Param:  paramKV,
			Scopes: scopes,
		})

		return handler(ctx)
	}
}

// LookupAccessToken resolves an access token parameter by SHA-256 hash key.
// Legacy plaintext rows are migrated to the hash key on first successful lookup.
func LookupAccessToken(ctx context.Context, raw string) (gen.Parameter, error) {
	if raw == "" {
		return gen.Parameter{}, types.ErrNotFound
	}
	hashed := auth.HashToken(raw)
	p, err := store.Database.ParameterGet(ctx, hashed)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, types.ErrNotFound) {
		return gen.Parameter{}, err
	}

	p, err = store.Database.ParameterGet(ctx, raw)
	if err != nil {
		return gen.Parameter{}, err
	}

	params := types.KV(p.Params)
	if setErr := store.Database.ParameterSet(ctx, hashed, params, p.ExpiredAt); setErr != nil {
		return gen.Parameter{}, setErr
	}
	_ = store.Database.ParameterDelete(ctx, raw)
	p.Flag = hashed
	return p, nil
}

// DeleteAccessToken removes both the hashed and legacy plaintext parameter rows for raw.
func DeleteAccessToken(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	hashed := auth.HashToken(raw)
	if err := store.Database.ParameterDelete(ctx, hashed); err != nil {
		return err
	}
	return store.Database.ParameterDelete(ctx, raw)
}

// resolveAccessToken extracts the access token from cookies or the HTTP request.
func resolveAccessToken(ctx fiber.Ctx) (string, error) {
	accessToken := ctx.Cookies(accessTokenKey)
	if accessToken != "" {
		return accessToken, nil
	}

	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(ctx.RequestCtx(), &r, true); err != nil {
		auditAuthReject(ctx, "auth.token.validate.fail", "request conversion failed")
		return "", protocol.ErrNotAuthorized.Wrap(err)
	}
	accessToken = GetAccessToken(&r)
	if accessToken == "" {
		auditAuthReject(ctx, "auth.token.validate.fail", "missing token")
		return "", protocol.ErrNotAuthorized.New("Missing token")
	}
	return accessToken, nil
}

// parseScopes extracts string scopes from the param KV store.
func parseScopes(paramKV types.KV) []string {
	raw, ok := paramKV["scopes"]
	if !ok {
		return nil
	}

	switch v := raw.(type) {
	case []any:
		scopes := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				scopes = append(scopes, s)
			}
		}
		return scopes
	case []string:
		return v
	default:
		return nil
	}
}

// throttledUpdateLastUsed updates last_used_at in params with a 60s throttle
// to avoid a database write on every request. tokenFlag must be the storage key (hash).
func throttledUpdateLastUsed(paramKV types.KV, tokenFlag string, expiredAt time.Time) {
	if shouldUpdateLastUsed(paramKV) {
		paramKV["last_used_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		_ = store.Database.ParameterSet(context.Background(), tokenFlag, paramKV, expiredAt)
	}
}

// shouldUpdateLastUsed returns true when last_used_at is missing, unparseable,
// or older than 60s.
func shouldUpdateLastUsed(paramKV types.KV) bool {
	lastUsedRaw, ok := paramKV["last_used_at"]
	if !ok {
		return true
	}

	lastUsedStr, isStr := lastUsedRaw.(string)
	if !isStr {
		return true
	}

	lastUsed, parseErr := time.Parse(time.RFC3339Nano, lastUsedStr)
	if parseErr != nil {
		return true
	}

	return time.Since(lastUsed) >= 60*time.Second
}

// GetRequestContext returns the typed RequestContext from fiber.Locals.
// Returns nil if the request was not authorized.
func GetRequestContext(ctx fiber.Ctx) *RequestContext {
	rc, ok := ctx.Locals(requestContextKey).(*RequestContext)
	if !ok {
		return nil
	}
	return rc
}

// GetAccessToken extracts the API key from an HTTP request.
func GetAccessToken(req *http.Request) string {
	apikey := req.Header.Get("X-AccessToken")
	if apikey != "" {
		return apikey
	}
	authorization := req.Header.Get("Authorization")
	apikey = auth.ExtractBearerToken(authorization)
	if apikey != "" {
		return apikey
	}
	apikey = req.URL.Query().Get(accessTokenKey)
	if apikey != "" {
		return apikey
	}
	apikey = req.FormValue(accessTokenKey)
	if apikey != "" {
		return apikey
	}
	if c, err := req.Cookie(accessTokenKey); err == nil {
		apikey = c.Value
	}
	return apikey
}

func CheckAccessToken(accessToken string) (uid types.Uid, isValid bool) {
	p, err := LookupAccessToken(context.Background(), accessToken)
	if err != nil {
		return
	}
	if p.ID <= 0 || store.ParameterIsExpired(p) {
		return
	}
	params := types.KV(p.Params)
	u, _ := params.String("uid")
	uid = types.Uid(u)
	if uid.IsZero() {
		return
	}
	isValid = true
	return
}

func GetUid(ctx fiber.Ctx) types.Uid {
	rc := GetRequestContext(ctx)
	if rc == nil {
		return ""
	}
	return rc.UID
}

func GetTopic(ctx fiber.Ctx) string {
	rc := GetRequestContext(ctx)
	if rc == nil {
		return ""
	}
	return rc.Topic
}

func GetIntParam(ctx fiber.Ctx, name string) int64 {
	s := ctx.Params(name)
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func GetScopes(ctx fiber.Ctx) []string {
	rc := GetRequestContext(ctx)
	if rc == nil {
		return nil
	}
	return rc.Scopes
}

func GetParam(ctx fiber.Ctx) types.KV {
	rc := GetRequestContext(ctx)
	if rc == nil {
		return nil
	}
	return rc.Param
}

func RequireScope(scope string, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		scopes := GetScopes(ctx)
		if !auth.HasScope(scopes, scope) {
			auditScopeDeny(ctx, scope)
			return protocol.ErrAccessDenied.New("insufficient scope: " + scope)
		}
		return handler(ctx)
	}
}

func auditAuthReject(ctx fiber.Ctx, action, reason string) {
	if routeAuditor == nil {
		return
	}
	ip := ctx.IP()
	ua := string(ctx.Request().Header.UserAgent())
	_ = routeAuditor.RecordRejected(context.Background(), audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "token",
			IPAddress:   ip,
			UserAgent:   ua,
		},
		Action: action,
		Target: audit.Target{Type: "token"},
	}, reason)
}

func auditScopeDeny(ctx fiber.Ctx, scope string) {
	if routeAuditor == nil {
		return
	}
	rc := GetRequestContext(ctx)
	uid := ""
	if rc != nil {
		uid = string(rc.UID)
	}
	ip := ctx.IP()
	ua := string(ctx.Request().Header.UserAgent())
	_ = routeAuditor.RecordRejected(context.Background(), audit.Entry{
		Subject: &audit.Subject{
			SubjectType: "token",
			UID:         uid,
			IPAddress:   ip,
			UserAgent:   ua,
		},
		Action: "auth.scope.deny",
		Target: audit.Target{Type: "scope"},
	}, "required: "+scope)
}

func ScopeHandler(ctx fiber.Ctx, scope string) bool {
	scopes := GetScopes(ctx)
	return auth.HasScope(scopes, scope)
}
