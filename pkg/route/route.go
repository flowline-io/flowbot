package route

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"strings"
)

const prefix = "bot"

func WebService(app *fiber.App, group, version string, rs ...*Router) {
	path := "/" + prefix + "/" + group + "/" + version
	for _, router := range rs {
		funcName := utils.GetFunctionName(router.Function)
		// method
		switch router.Method {
		case "GET":
			app.Get(path+router.Path, router.Function)
		case "POST":
			app.Post(path+router.Path, router.Function)
		case "PUT":
			app.Put(path+router.Path, router.Function)
		case "PATCH":
			app.Patch(path+router.Path, router.Function)
		case "DELETE":
			app.Delete(path+router.Path, router.Function)
		default:
			continue
		}
		flog.Info("WebService %s \t%s%s \t-> %s", router.Method, path, router.Path, funcName)
	}
}

func authFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	accessToken := GetAccessToken(req.Request)
	p, err := store.Chatbot.ParameterGet(accessToken)
	if err != nil {
		_ = resp.WriteErrorString(401, "401: Not Authorized")
		return
	}
	if p.ID <= 0 || p.IsExpired() {
		return
	}

	topic, _ := types.KV(p.Params).String("topic")
	u, _ := types.KV(p.Params).String("uid")
	uid := types.Uid(u)
	isValid := false
	if !uid.IsZero() {
		isValid = true
	}

	if !isValid {
		_ = resp.WriteErrorString(401, "401: Not Authorized")
		return
	}
	req.SetAttribute("uid", uid)
	req.SetAttribute("topic", topic)
	req.SetAttribute("param", types.KV(p.Params))
	chain.ProcessFilter(req, resp)
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

func ErrorResponse(ctx *fiber.Ctx, text string) error {
	ctx = ctx.Status(http.StatusBadRequest)
	return ctx.SendString(text)
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
