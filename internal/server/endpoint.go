package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/api/types/endpoint"
	nodeBase "github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/endpoint/impl"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/runtime"
	"github.com/rulego/rulego/utils/str"
	"net/http"
	"net/textproto"
	"strings"
)

const (
	ContentTypeKey  = "Content-Type"
	JsonContextType = "application/json"

	httpTypeId = "flowbot/http"
)

type Endpoint = RestEndpoint

type RequestMessage struct {
	Ctx      fiber.Ctx
	body     []byte
	Params   map[string]string
	msg      *types.RuleMsg
	err      error
	Metadata types.Metadata
}

func (r *RequestMessage) Body() []byte {
	if r.body == nil && r.Ctx != nil {
		r.body = r.Ctx.Body()
	}
	return r.body
}

func (r *RequestMessage) Headers() textproto.MIMEHeader {
	if r.Ctx == nil {
		return nil
	}
	return r.Ctx.GetReqHeaders()
}

func (r RequestMessage) From() string {
	if r.Ctx == nil {
		return ""
	}
	return utils.BytesToString(r.Ctx.Request().URI().FullURI())
}

func (r *RequestMessage) GetParam(key string) string {
	if r.Ctx == nil {
		return ""
	}
	if v := r.Ctx.Params(key); v == "" {
		return r.Ctx.FormValue(key)
	} else {
		return v
	}
}

func (r *RequestMessage) SetMsg(msg *types.RuleMsg) {
	r.msg = msg
}

func (r *RequestMessage) GetMsg() *types.RuleMsg {
	if r.msg == nil {
		dataType := types.TEXT
		var data string
		if r.Ctx != nil && r.Ctx.Method() == http.MethodGet {
			dataType = types.JSON
			data = str.ToString(r.Ctx.Queries())
		} else {
			if contentType := r.Headers().Get(ContentTypeKey); strings.HasPrefix(contentType, JsonContextType) {
				dataType = types.JSON
			}
			data = string(r.Body())
		}
		if r.Metadata == nil {
			r.Metadata = types.NewMetadata()
		}
		ruleMsg := types.NewMsg(0, r.From(), dataType, r.Metadata, data)
		r.msg = &ruleMsg
	}
	return r.msg
}

func (r *RequestMessage) SetStatusCode(statusCode int) {
}

func (r *RequestMessage) SetBody(body []byte) {
	r.body = body
}

func (r *RequestMessage) SetError(err error) {
	r.err = err
}

func (r *RequestMessage) GetError() error {
	return r.err
}

// ResponseMessage http响应消息
type ResponseMessage struct {
	Ctx  fiber.Ctx
	body []byte
	to   string
	msg  *types.RuleMsg
	err  error
}

func (r *ResponseMessage) Body() []byte {
	return r.body
}

func (r *ResponseMessage) Headers() textproto.MIMEHeader {
	if r.Ctx == nil {
		return nil
	}
	return r.Ctx.GetRespHeaders()
}

func (r *ResponseMessage) From() string {
	if r.Ctx == nil {
		return ""
	}
	return utils.BytesToString(r.Ctx.Request().URI().FullURI())
}

func (r *ResponseMessage) GetParam(key string) string {
	if r.Ctx == nil {
		return ""
	}
	return r.Ctx.FormValue(key)
}

func (r *ResponseMessage) SetMsg(msg *types.RuleMsg) {
	r.msg = msg
}
func (r *ResponseMessage) GetMsg() *types.RuleMsg {
	return r.msg
}

func (r *ResponseMessage) SetStatusCode(statusCode int) {
	if r.Ctx != nil {
		r.Ctx.Status(statusCode)
	}
}

func (r *ResponseMessage) SetBody(body []byte) {
	r.body = body
	if r.Ctx != nil {
		_, _ = r.Ctx.Write(body)
	}
}

func (r *ResponseMessage) SetError(err error) {
	r.err = err
}

func (r *ResponseMessage) GetError() error {
	return r.err
}

type Config struct {
	AllowCors bool
}

type RestEndpoint struct {
	impl.BaseEndpoint
	nodeBase.SharedNode[*RestEndpoint]

	Config     Config
	RuleConfig types.Config
	Server     *fiber.App
	started    bool
}

func (rest *RestEndpoint) Type() string {
	return httpTypeId
}

func (rest *RestEndpoint) New() types.Node {
	return &RestEndpoint{
		Config: Config{},
		Server: sharedApp,
	}
}

func (rest *RestEndpoint) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &rest.Config)
	if err != nil {
		return err
	}
	rest.RuleConfig = ruleConfig
	return rest.SharedNode.Init(rest.RuleConfig, rest.Type(), httpTypeId, false, func() (*RestEndpoint, error) {
		return rest.initServer()
	})
}

func (rest *RestEndpoint) Destroy() {
	_ = rest.Close()
}

func (rest *RestEndpoint) Restart() error {
	var oldRouter = make(map[string]endpoint.Router)

	rest.Lock()
	for id, router := range rest.RouterStorage {
		if !router.IsDisable() {
			oldRouter[id] = router
		}
	}
	rest.Unlock()

	rest.RouterStorage = make(map[string]endpoint.Router)
	rest.started = false

	for _, router := range oldRouter {
		if len(router.GetParams()) == 0 {
			router.SetParams("GET")
		}
		if !rest.HasRouter(router.GetId()) {
			if _, err := rest.AddRouter(router, router.GetParams()...); err != nil {
				rest.Printf("rest add router path:=%s error:%v", router.FromToString(), err)
				continue
			}
		}
	}
	return nil
}

func (rest *RestEndpoint) Close() error {
	return nil
}

func (rest *RestEndpoint) Id() string {
	return httpTypeId
}

func (rest *RestEndpoint) AddRouter(router endpoint.Router, params ...interface{}) (id string, err error) {
	if len(params) <= 0 {
		return "", errors.New("need to specify HTTP method")
	} else if router == nil {
		return "", errors.New("router can not nil")
	} else {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("addRouter err :%v", e)
			}
		}()
		for _, param := range params {
			err = rest.addRouter(strings.ToUpper(str.ToString(param)), router)
			if err != nil {
				return
			}
		}

		// rebuild router tree
		rest.Server.RebuildTree()

		return router.GetId(), nil
	}
}

func (rest *RestEndpoint) RemoveRouter(routerId string, params ...interface{}) error {
	routerId = strings.TrimSpace(routerId)
	rest.Lock()
	defer rest.Unlock()
	if rest.RouterStorage != nil {
		if router, ok := rest.RouterStorage[routerId]; ok && !router.IsDisable() {
			router.Disable(true)
			return nil
		} else {
			return fmt.Errorf("router: %s not found", routerId)
		}
	}
	return nil
}

func (rest *RestEndpoint) deleteRouter(routerId string) {
	routerId = strings.TrimSpace(routerId)
	rest.Lock()
	defer rest.Unlock()
	if rest.RouterStorage != nil {
		delete(rest.RouterStorage, routerId)
	}
}

func (rest *RestEndpoint) Start() error {
	if err := rest.checkIsInitSharedNode(); err != nil {
		return err
	}
	if netResource, err := rest.SharedNode.Get(); err == nil {
		return netResource.startServer()
	} else {
		return err
	}
}

// addRouter
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
func (rest *RestEndpoint) addRouter(method string, routers ...endpoint.Router) error {
	method = strings.ToUpper(method)

	rest.Lock()
	defer rest.Unlock()

	if rest.RouterStorage == nil {
		rest.RouterStorage = make(map[string]endpoint.Router)
	}
	for _, item := range routers {
		path := strings.TrimSpace(item.FromToString())
		if id := item.GetId(); id == "" {
			item.SetId(rest.RouterKey(method, path))
		}
		item.SetParams(method)
		rest.RouterStorage[item.GetId()] = item
		if rest.SharedNode.InstanceId != "" {
			if shared, err := rest.SharedNode.Get(); err == nil {
				return shared.addRouter(method, item)
			} else {
				return err
			}
		} else {
			isWait := false
			if from := item.GetFrom(); from != nil {
				if to := from.GetTo(); to != nil {
					isWait = to.IsWait()
				}
			}
			rest.Server.Add([]string{method}, path, rest.handler(item, isWait))
		}
	}
	return nil
}

func (rest *RestEndpoint) GET(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodGet, routers...)
	return rest
}

func (rest *RestEndpoint) HEAD(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodHead, routers...)
	return rest
}

func (rest *RestEndpoint) OPTIONS(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodOptions, routers...)
	return rest
}

func (rest *RestEndpoint) POST(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodPost, routers...)
	return rest
}

func (rest *RestEndpoint) PUT(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodPut, routers...)
	return rest
}

func (rest *RestEndpoint) PATCH(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodPatch, routers...)
	return rest
}

func (rest *RestEndpoint) DELETE(routers ...endpoint.Router) *RestEndpoint {
	_ = rest.addRouter(http.MethodDelete, routers...)
	return rest
}

func (rest *RestEndpoint) checkIsInitSharedNode() error {
	if !rest.SharedNode.IsInit() {
		err := rest.SharedNode.Init(rest.RuleConfig, rest.Type(), httpTypeId, false, func() (*RestEndpoint, error) {
			return rest.initServer()
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (rest *RestEndpoint) RouterKey(method string, from string) string {
	return method + ":" + from
}

func (rest *RestEndpoint) handler(router endpoint.Router, isWait bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if e := recover(); e != nil {
				flog.Error(fmt.Errorf("fiber endpoint handler error:%v\n%v", e, runtime.Stack()))
			}
		}()
		if router.IsDisable() {
			return c.Status(fiber.StatusNotFound).SendString("404")
		}

		var params = make(map[string]string)
		err := c.Bind().URI(&params)
		if err != nil {
			return err
		}

		metadata := types.NewMetadata()
		exchange := &endpoint.Exchange{
			In: &RequestMessage{
				Ctx:      c,
				Params:   params,
				Metadata: metadata,
			},
			Out: &ResponseMessage{
				Ctx: c,
			},
		}

		for key, value := range params {
			metadata.PutValue(key, value)
		}

		for key, value := range c.Queries() {
			if len(value) > 1 {
				metadata.PutValue(key, str.ToString(value))
			} else {
				metadata.PutValue(key, value)
			}
		}
		var ctx = c.Context()
		if !isWait {
			ctx = context.Background()
		}
		rest.DoProcess(ctx, router, exchange)

		return c.Status(fiber.StatusOK).SendString("ok")
	}
}

func (rest *RestEndpoint) Printf(format string, v ...interface{}) {
	if rest.RuleConfig.Logger != nil {
		rest.RuleConfig.Logger.Printf(format, v...)
	}
}

func (rest *RestEndpoint) Started() bool {
	return rest.started
}

func (rest *RestEndpoint) GetServer() *fiber.App {
	if rest.Server != nil {
		return rest.Server
	} else if rest.SharedNode.InstanceId != "" {
		if shared, err := rest.SharedNode.Get(); err == nil {
			return shared.Server
		}
	}
	return nil
}

func (rest *RestEndpoint) initServer() (*RestEndpoint, error) {
	return rest, nil
}

func (rest *RestEndpoint) startServer() error {
	if rest.started {
		return nil
	}
	rest.started = true

	if rest.OnEvent != nil {
		rest.OnEvent(endpoint.EventInitServer, rest)
	}

	return nil
}
