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
			app.Get(path+router.Path, Authorize(router.Auth, router.Function))
		case "POST":
			app.Post(path+router.Path, Authorize(router.Auth, router.Function))
		case "PUT":
			app.Put(path+router.Path, Authorize(router.Auth, router.Function))
		case "PATCH":
			app.Patch(path+router.Path, Authorize(router.Auth, router.Function))
		case "DELETE":
			app.Delete(path+router.Path, Authorize(router.Auth, router.Function))
		default:
			continue
		}
		flog.Info("WebService %s \t%s%s \t-> %s", router.Method, path, router.Path, funcName)
	}
}

func Route(method string, path string, function fiber.Handler, documentation string, options ...Option) *Router {
	r := &Router{
		Method:        method,
		Path:          path,
		Function:      function,
		Documentation: documentation,
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
	Auth          bool
}

func WithAuth() Option {
	return func(r *Router) {
		r.Auth = true
	}
}

func ErrorResponse(ctx *fiber.Ctx, text string) error {
	ctx = ctx.Status(http.StatusBadRequest)
	return ctx.SendString(text)
}

func Authorize(auth bool, handler fiber.Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// check skip auth
		if !auth {
			return handler(ctx)
		}

		// check api flag
		var r http.Request
		if err := fasthttpadaptor.ConvertRequest(ctx.Context(), &r, true); err != nil {
			flog.Error(err)
			return ctx.Status(http.StatusInternalServerError).JSON(protocol.NewFailedResponse(protocol.ErrInternalServerError))
		}
		accessToken := GetAccessToken(&r)
		p, err := store.Chatbot.ParameterGet(accessToken)
		if err != nil {
			return ctx.Status(http.StatusUnauthorized).JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
		}
		if p.ID <= 0 || p.IsExpired() {
			return ctx.Status(http.StatusUnauthorized).JSON(protocol.NewFailedResponse(protocol.ErrFlagExpired))
		}

		topic, _ := types.KV(p.Params).String("topic")
		u, _ := types.KV(p.Params).String("uid")
		uid := types.Uid(u)
		isValid := false
		if !uid.IsZero() {
			isValid = true
		}

		if !isValid {
			return ctx.Status(http.StatusUnauthorized).JSON(protocol.NewFailedResponse(protocol.ErrNotAuthorized))
		}

		// set uid and topic
		ctx.Locals("uid", uid)
		ctx.Locals("topic", topic)
		ctx.Locals("param", types.KV(p.Params))

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
	apikey = req.URL.Query().Get("accessToken")
	if apikey != "" {
		return apikey
	}

	// Check form values.
	apikey = req.FormValue("accessToken")
	if apikey != "" {
		return apikey
	}

	// Check cookies.
	if c, err := req.Cookie("accessToken"); err == nil {
		apikey = c.Value
	}

	return apikey
}

// CheckAccessToken check access token valid
func CheckAccessToken(accessToken string) (uid types.Uid, isValid bool) {
	p, err := store.Chatbot.ParameterGet(accessToken)
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

func GetUid(ctx *fiber.Ctx) types.Uid {
	uid, _ := ctx.Locals("uid").(types.Uid)
	return uid
}

func GetTopic(ctx *fiber.Ctx) string {
	topic, _ := ctx.Locals("topic").(string)
	return topic
}
