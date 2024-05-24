package bots

import (
	"encoding/json"
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/ruleset/instruct"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

type Handler interface {
	// Init initializes the bot.
	Init(jsonconf json.RawMessage) error

	// IsReady сhecks if the bot is initialized.
	IsReady() bool

	// Bootstrap Lifecycle hook
	Bootstrap() error

	// OnEvent event
	OnEvent() error

	// Help return bot help
	Help() (map[string][]string, error)

	// Rules return bot ruleset
	Rules() []interface{}

	// Input return input result
	Input(ctx types.Context, head types.KV, content interface{}) (types.MsgPayload, error)

	// Command return bot result
	Command(ctx types.Context, content interface{}) (types.MsgPayload, error)

	// Form return bot form result
	Form(ctx types.Context, values types.KV) (types.MsgPayload, error)

	// Session return bot session result
	Session(ctx types.Context, content interface{}) (types.MsgPayload, error)

	// Cron cron script daemon
	Cron(send types.SendFunc) (*cron.Ruleset, error)

	// Agent return group result
	Agent(ctx types.Context, content types.KV) (types.MsgPayload, error)

	// Instruct return instruct list
	Instruct() (instruct.Ruleset, error)

	// Page return page
	Page(ctx types.Context, flag string) (string, error)

	// Webservice return webservice routes
	Webservice(app *fiber.App)

	// Webapp return webapp
	Webapp() func(rw http.ResponseWriter, req *http.Request)

	// Workflow return workflow result
	Workflow(ctx types.Context, input types.KV) (types.KV, error)

	// Webhook return webhook result
	Webhook(ctx types.Context, content types.KV) (types.MsgPayload, error)
}

type Base struct{}

func (Base) Bootstrap() error {
	return nil
}

func (Base) WebService() *restful.WebService {
	return nil
}

func (Base) OnEvent() error {
	return nil
}

func (b Base) Help() (map[string][]string, error) {
	return Help(b.Rules())
}

func (Base) Rules() []interface{} {
	return nil
}

func (Base) Input(_ types.Context, _ types.KV, _ interface{}) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Command(_ types.Context, _ interface{}) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Form(_ types.Context, _ types.KV) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Session(_ types.Context, _ interface{}) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Cron(_ types.SendFunc) (*cron.Ruleset, error) {
	return nil, nil
}

func (Base) Agent(_ types.Context, _ types.KV) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Instruct() (instruct.Ruleset, error) {
	return nil, nil
}

func (Base) Page(_ types.Context, _ string) (string, error) {
	return "", nil
}

func (Base) Webservice(_ *fiber.App) {
}

func (Base) Webapp() func(rw http.ResponseWriter, req *http.Request) {
	return nil
}

func (Base) Workflow(_ types.Context, _ types.KV) (types.KV, error) {
	return nil, nil
}

func (Base) Webhook(_ types.Context, _ types.KV) (types.MsgPayload, error) {
	return nil, nil
}
