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
		// method
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
		// funcName := utils.GetFunctionName(router.Function)
		// flog.Info("WebService %s \t%s%s \t-> %s", router.Method, path, router.Path, funcName)
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

const (
	uidKey         = "uid"
	topicKey       = "topic"
	paramKey       = "param"
	accessTokenKey = "accessToken"
	scopesKey      = "scopes"
)

func Authorize(authLevel AuthLevel, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		// Check if authentication can be skipped
		if authLevel == NoAuth {
			return handler(ctx)
		}

		// Check API flag
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

		topic, _ := types.KV(p.Params).String(topicKey)
		u, _ := types.KV(p.Params).String(uidKey)
		uid := types.Uid(u)

		if uid.IsZero() {
			return protocol.ErrNotAuthorized.New("uid empty")
		}

		// Extract scopes from parameter
		paramKV := types.KV(p.Params)
		var scopes []string
		if raw, ok := paramKV[scopesKey]; ok {
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

		// Set uid, topic, scopes, param
		ctx.Locals(uidKey, uid)
		ctx.Locals(topicKey, topic)
		ctx.Locals(paramKey, paramKV)
		ctx.Locals(scopesKey, scopes)

		return handler(ctx)
	}
}

// GetAccessToken Get API key from an HTTP request.
func GetAccessToken(req *http.Request) string {
	// Check header.
	apikey := req.Header.Get("X-AccessToken")
	if apikey != "" {
		return apikey
	}
	authorization := req.Header.Get("Authorization")
	apikey = auth.ExtractBearerToken(authorization)
	if apikey != "" {
		return apikey
	}

	// Check URL query parameters.
	apikey = req.URL.Query().Get(accessTokenKey)
	if apikey != "" {
		return apikey
	}

	// Check form values.
	apikey = req.FormValue(accessTokenKey)
	if apikey != "" {
		return apikey
	}

	// Check cookies.
	if c, err := req.Cookie(accessTokenKey); err == nil {
		apikey = c.Value
	}

	return apikey
}

// CheckAccessToken check access token valid
func CheckAccessToken(accessToken string) (uid types.Uid, isValid bool) {
	p, err := store.Database.ParameterGet(accessToken)
	if err != nil {
		return
	}
	if p.ID <= 0 || p.IsExpired() {
		return
	}

	u, _ := types.KV(p.Params).String(uidKey)
	uid = types.Uid(u)
	if uid.IsZero() {
		return
	}
	isValid = true
	return
}

func GetUid(ctx fiber.Ctx) types.Uid {
	uid, _ := ctx.Locals(uidKey).(types.Uid)
	return uid
}

func GetTopic(ctx fiber.Ctx) string {
	topic, _ := ctx.Locals(topicKey).(string)
	return topic
}

func GetIntParam(ctx fiber.Ctx, name string) int64 {
	s := ctx.Params(name)
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// GetScopes returns the scopes associated with the current request.
func GetScopes(ctx fiber.Ctx) []string {
	scopes, _ := ctx.Locals(scopesKey).([]string)
	return scopes
}

// GetParam returns the full parameter KV from the current request context.
func GetParam(ctx fiber.Ctx) types.KV {
	param, _ := ctx.Locals(paramKey).(types.KV)
	return param
}

// RequireScope returns middleware that checks whether the current request
// has the required scope. Must be chained after Authorize.
func RequireScope(scope string, handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		scopes := GetScopes(ctx)
		if !auth.HasScope(scopes, scope) {
			return protocol.ErrAccessDenied.New("insufficient scope: " + scope)
		}
		return handler(ctx)
	}
}

// ScopeHandler wraps a handler with scope check.
// Use when you need to check scope inside an already-authorized handler.
func ScopeHandler(ctx fiber.Ctx, scope string) bool {
	scopes := GetScopes(ctx)
	return auth.HasScope(scopes, scope)
}
