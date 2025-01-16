package bots

import (
	"encoding/json"
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/instruct"
	"github.com/gofiber/fiber/v2"
)

type Handler interface {
	// Init initializes the bot.
	Init(jsonconf json.RawMessage) error

	// IsReady сhecks if the bot is initialized.
	IsReady() bool

	// Bootstrap Lifecycle hook
	Bootstrap() error

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

	// Cron cron script daemon
	Cron() (*cron.Ruleset, error)

	// Collect return collect result
	Collect(ctx types.Context, content types.KV) (types.MsgPayload, error)

	// Instruct return instruct list
	Instruct() (instruct.Ruleset, error)

	// Page return page
	Page(ctx types.Context, flag string, args types.KV) (string, error)

	// Webservice return webservice routes
	Webservice(app *fiber.App)

	// Workflow return workflow result
	Workflow(ctx types.Context, input types.KV) (types.KV, error)

	// Webhook return webhook result
	Webhook(ctx types.Context, method string, data []byte) (types.MsgPayload, error)

	// LangChain return langchain result
	LangChain(ctx types.Context, args types.KV) (string, error)

	// Event return event result
	Event(ctx types.Context, param types.KV) error
}

type Base struct{}

func (Base) Bootstrap() error {
	return nil
}

func (Base) WebService() *restful.WebService {
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

func (Base) Cron() (*cron.Ruleset, error) {
	return nil, nil
}

func (Base) Collect(_ types.Context, _ types.KV) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Instruct() (instruct.Ruleset, error) {
	return nil, nil
}

func (Base) Page(_ types.Context, _ string, _ types.KV) (string, error) {
	return "", nil
}

func (Base) Webservice(_ *fiber.App) {
}

func (Base) Workflow(_ types.Context, _ types.KV) (types.KV, error) {
	return nil, nil
}

func (Base) Webhook(_ types.Context, _ string, _ []byte) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) LangChain(_ types.Context, _ types.KV) (string, error) {
	return "", nil
}

func (Base) Event(_ types.Context, _ types.KV) error {
	return nil
}
