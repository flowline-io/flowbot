package route

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"net/http"
	"strconv"
	"strings"
)

const prefix = "service"

func WebService(app *fiber.App, group string, rs ...*Router) {
	path := "/" + prefix + "/" + group
	for _, router := range rs {
		funcName := utils.GetFunctionName(router.Function)
		// method
		switch router.Method {
		case "GET":
			app.Get(path+router.Path, Authorize(router.NotAuth, router.Function))
		case "POST":
			app.Post(path+router.Path, Authorize(router.NotAuth, router.Function))
		case "PUT":
			app.Put(path+router.Path, Authorize(router.NotAuth, router.Function))
		case "PATCH":
			app.Patch(path+router.Path, Authorize(router.NotAuth, router.Function))
		case "DELETE":
			app.Delete(path+router.Path, Authorize(router.NotAuth, router.Function))
		default:
			continue
		}
		flog.Debug("WebService %s \t%s%s \t-> %s", router.Method, path, router.Path, funcName)
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
	NotAuth       bool
}

func WithNotAuth() Option {
	return func(r *Router) {
		r.NotAuth = true
	}
}

func ErrorResponse(ctx *fiber.Ctx, text string) error {
	ctx = ctx.Status(http.StatusBadRequest)
	return ctx.SendString(text)
}

const (
	uidKey         = "uid"
	topicKey       = "topic"
	paramKey       = "param"
	accessTokenKey = "accessToken"
)

func Authorize(notAuth bool, handler fiber.Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// check skip auth
		if notAuth {
			return handler(ctx)
		}

		// check api flag
		var r http.Request
		if err := fasthttpadaptor.ConvertRequest(ctx.Context(), &r, true); err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(protocol.NewFailedResponseWithError(protocol.ErrInternalServerError, err))
		}
		accessToken := GetAccessToken(&r)
		p, err := store.Database.ParameterGet(accessToken)
		if err != nil {
			return ctx.Status(http.StatusUnauthorized).JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
		}
		if p.ID <= 0 || p.IsExpired() {
			return ctx.Status(http.StatusUnauthorized).JSON(protocol.NewFailedResponse(protocol.ErrFlagExpired))
		}

		topic, _ := types.KV(p.Params).String(topicKey)
		u, _ := types.KV(p.Params).String(uidKey)
		uid := types.Uid(u)
		isValid := false
		if !uid.IsZero() {
			isValid = true
		}

		if !isValid {
			return ctx.Status(http.StatusUnauthorized).JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized))
		}

		// set uid and topic
		ctx.Locals(uidKey, uid)
		ctx.Locals(topicKey, topic)
		ctx.Locals(paramKey, types.KV(p.Params))

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
	authorization = strings.TrimSpace(authorization)
	apikey = strings.ReplaceAll(authorization, "Bearer ", "")
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

func GetUid(ctx *fiber.Ctx) types.Uid {
	uid, _ := ctx.Locals(uidKey).(types.Uid)
	return uid
}

func GetTopic(ctx *fiber.Ctx) string {
	topic, _ := ctx.Locals(topicKey).(string)
	return topic
}

func GetIntParam(ctx *fiber.Ctx, name string) int64 {
	s := ctx.Params(name)
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
