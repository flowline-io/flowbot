package archive

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
)

const Name = "archive"

var handler bot

func Register() {
	chatbot.Register(Name, &handler)
}

type bot struct {
	initialized bool
	chatbot.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func (bot) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}

	var config configType
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	if !config.Enabled {
		flog.Info("bot %s disabled", Name)
		return nil
	}

	handler.initialized = true

	return nil
}

func (bot) IsReady() bool {
	return handler.initialized
}

func (bot) Rules() []any {
	return []any{
		commandRules,
		cronRules,
		webserviceRules,
	}
}

func (bot) Webservice(app *fiber.App) {
	chatbot.Webservice(app, Name, webserviceRules)
}

func (bot) Command(ctx types.Context, content any) (types.MsgPayload, error) {
	return chatbot.RunCommand(commandRules, ctx, content)
}

func (bot) Cron() (*cron.Ruleset, error) {
	return chatbot.RunCron(cronRules, Name)
}

var cronRules = []cron.Rule{}

var commandRules = []command.Rule{
	{
		Define: "archive add [url]",
		Help:   "Archive a URL",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, _ := tokens[2].Value.String()

			res, err := ability.Invoke(ctx.Context(), hub.CapArchive, "add", map[string]any{
				"url": url,
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: res.Text}
		},
	},
}

type addRequest struct {
	URL string `json:"url" validate:"required,url,max=2048"`
}

var webserviceRules = []webservice.Rule{
	webservice.Post("/", addArchive),
}

func addArchive(ctx fiber.Ctx) error {
	var body addRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapArchive, "add", map[string]any{
		"url": body.URL,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}
