package route

import (
	"fmt"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"net/http"
	"strings"
)

const prefix = "bot"

func NewContainer() *restful.Container {
	restfulContainer := restful.NewContainer()
	restfulContainer.ServeMux = http.NewServeMux()
	restfulContainer.Router(restful.CurlyRouter{})

	restfulContainer.RecoverHandler(func(panicReason interface{}, w http.ResponseWriter) {
		logStackOnRecover(panicReason, w)
	})
	restfulContainer.ServiceErrorHandler(func(serviceError restful.ServiceError, req *restful.Request, resp *restful.Response) {
		logServiceError(serviceError, req, resp)
	})

	// CORS
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedDomains: []string{".*"},
		CookiesAllowed: false,
		Container:      restfulContainer}
	restfulContainer.Filter(cors.Filter)
	restfulContainer.Filter(restfulContainer.OPTIONSFilter)

	return restfulContainer
}

func newWebService(group string, version string) *restful.WebService {
	ws := new(restful.WebService)
	path := "/" + prefix + "/" + group + "/" + version
	ws.Path(path)
	ws.Doc(fmt.Sprintf("API at %s", path))
	return ws
}

func WebService(group, version string, rs ...*Router) *restful.WebService {
	path := "/" + prefix + "/" + group + "/" + version
	ws := newWebService(group, version)
	for _, router := range rs {
		funcName := utils.GetFunctionName(router.Function)
		_, operationName := utils.ParseFunctionName(funcName)
		tags := []string{group}
		var builder *restful.RouteBuilder
		// method
		switch router.Method {
		case "GET":
			builder = ws.GET(router.Path)
		case "POST":
			builder = ws.POST(router.Path)
		case "PUT":
			builder = ws.PUT(router.Path)
		case "PATCH":
			builder = ws.PATCH(router.Path)
		case "DELETE":
			builder = ws.DELETE(router.Path)
		default:
			continue
		}
		// auth
		if router.Auth {
			builder.Filter(authFilter)
		}
		// params
		if len(router.Params) > 0 {
			for _, param := range router.Params {
				switch param.Type {
				case PathParamType:
					builder.Param(ws.PathParameter(param.Name, param.Description).DataType(param.DataType))
				case QueryParamType:
					builder.Param(ws.QueryParameter(param.Name, param.Description).DataType(param.DataType))
				case FormParamType:
					builder.Param(ws.FormParameter(param.Name, param.Description).DataType(param.DataType))
				}
			}
		}
		ws.Route(builder.
			To(router.Function).
			Produces(restful.MIME_JSON).
			Consumes(restful.MIME_JSON).
			Doc(router.Documentation).
			Operation(operationName).
			Metadata(restfulspec.KeyOpenAPITags, tags).
			Returns(http.StatusOK, "OK", router.ReturnSample).
			Writes(router.WriteSample))
		flog.Info("WebService %s \t%s%s \t-> %s", router.Method, path, router.Path, funcName)
	}
	return ws
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

func Route(method string, path string, function restful.RouteFunction, documentation string, options ...Option) *Router {
	r := &Router{
		Method:        method,
		Path:          path,
		Function:      function,
		Documentation: documentation,
		Params:        make([]*Param, 0),
	}
	for _, option := range options {
		option(r)
	}
	return r
}

type Option func(r *Router)

func WithReturns(returns interface{}) Option {
	return func(r *Router) {
		r.ReturnSample = returns
	}
}

func WithWrites(writes interface{}) Option {
	return func(r *Router) {
		r.WriteSample = writes
	}
}

func WithParam(param *Param) Option {
	return func(r *Router) {
		r.Params = append(r.Params, param)
	}
}

func WithAuth() Option {
	return func(r *Router) {
		r.Auth = true
	}
}

type Router struct {
	Method        string
	Path          string
	Function      restful.RouteFunction
	Documentation string
	ReturnSample  interface{}
	WriteSample   interface{}
	Params        []*Param
	Auth          bool
}

type ParamType string

const (
	PathParamType  ParamType = "path"
	QueryParamType ParamType = "query"
	FormParamType  ParamType = "form"
)

type Param struct {
	Type        ParamType
	Name        string
	Description string
	DataType    string
}

func WithPathParam(name, description, dataType string) Option {
	return WithParam(PathParam(name, description, dataType))
}

func PathParam(name, description, dataType string) *Param {
	return &Param{
		Type:        PathParamType,
		Name:        name,
		Description: description,
		DataType:    dataType,
	}
}

func WithQueryParam(name, description, dataType string) Option {
	return WithParam(QueryParam(name, description, dataType))
}

func QueryParam(name, description, dataType string) *Param {
	return &Param{
		Type:        QueryParamType,
		Name:        name,
		Description: description,
		DataType:    dataType,
	}
}

func WithFormParam(name, description, dataType string) Option {
	return WithParam(FormParam(name, description, dataType))
}

func FormParam(name, description, dataType string) *Param {
	return &Param{
		Type:        FormParamType,
		Name:        name,
		Description: description,
		DataType:    dataType,
	}
}

func ErrorResponse(resp *restful.Response, text string) {
	resp.WriteHeader(http.StatusBadRequest)
	_, _ = resp.Write([]byte(text))
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
