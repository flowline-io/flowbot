package route

import (
	"net/http"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

const prefix = "service"

func WebService(app *fiber.App, group string, rs ...*Router) {
	path := "/" + prefix + "/" + group
	for _, router := range rs {
		switch router.Method {
		case "GET":
			app.Get(path+router.Path, Authorize(router.AuthLevel, router.Function))
		case "POST":
			app.Post(path+router.Path, Authorize(router.AuthLevel, router.Function))
		case "PUT":
			app.Put(path+router.Path, Authorize(router.AuthLevel, router.Function))
		case "PATCH":
			app.Patch(path+router.Path, Authorize(router.AuthLevel, router.Function))
		case "DELETE":
			app.Delete(path+router.Path, Authorize(router.AuthLevel, router.Function))
		default:
			continue
		}
	}
}

func Route(method string, path string, function fiber.Handler, options ...Option) *Router {
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

func Authorize(authLevel AuthLevel, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if authLevel == NoAuth {
			return handler(ctx)
		}

		var r http.Request
		if err := fasthttpadaptor.ConvertRequest(ctx.RequestCtx(), &r, true); err != nil {
			return protocol.ErrNotAuthorized.Wrap(err)
		}

		accessToken := GetAccessToken(&r)
		if accessToken == "" {
			return protocol.ErrNotAuthorized.New("Missing token")
		}

		p, err := store.Database.ParameterGet(accessToken)
		if err != nil || p.ID <= 0 || p.IsExpired() {
			return protocol.ErrNotAuthorized.New("parameter error")
		}

		paramKV := types.KV(p.Params)
		topic, _ := paramKV.String("topic")
		uidStr, _ := paramKV.String("uid")
		uid := types.Uid(uidStr)

		if uid.IsZero() {
			return protocol.ErrNotAuthorized.New("uid empty")
		}

		var scopes []string
		if raw, ok := paramKV["scopes"]; ok {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						scopes = append(scopes, s)
					}
				}
			case []string:
				scopes = v
			}
		}

		ctx.Locals(requestContextKey, &RequestContext{
			UID:    uid,
			Topic:  topic,
			Param:  paramKV,
			Scopes: scopes,
		})

		return handler(ctx)
	}
}

// GetRequestContext returns the typed RequestContext from fiber.Locals.
// Returns nil if the request was not authorized.
func GetRequestContext(ctx fiber.Ctx) *RequestContext {
	rc, _ := ctx.Locals(requestContextKey).(*RequestContext)
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
	p, err := store.Database.ParameterGet(accessToken)
	if err != nil {
		return
	}
	if p.ID <= 0 || p.IsExpired() {
		return
	}
	u, _ := types.KV(p.Params).String("uid")
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
			return protocol.ErrAccessDenied.New("insufficient scope: " + scope)
		}
		return handler(ctx)
	}
}

func ScopeHandler(ctx fiber.Ctx, scope string) bool {
	scopes := GetScopes(ctx)
	return auth.HasScope(scopes, scope)
}
