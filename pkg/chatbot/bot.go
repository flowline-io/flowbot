package chatbot

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/slash"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/collect"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/instruct"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/setting"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/tool"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	MessageBotIncomingBehavior   = "message_bot_incoming"
	MessageGroupIncomingBehavior = "message_group_incoming"
)

var handlers map[string]Handler

func Register(name string, bot Handler) {
	if handlers == nil {
		handlers = make(map[string]Handler)
	}

	if bot == nil {
		flog.Fatal("Register: bot is nil")
	}
	if _, dup := handlers[name]; dup {
		flog.Fatal("Register: called twice for bot %s", name)
	}
	handlers[name] = bot
	_, _ = fmt.Printf("%s info %s [bot] %s registered\n", time.Now().Format(time.DateTime), utils.FileAndLine(), name)
}

func Help(rules []interface{}) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, rule := range rules {
		switch v := rule.(type) {
		case []command.Rule:
			// command
			rs := command.Ruleset(v)
			var rows []string
			for _, rule := range rs {
				rows = append(rows, fmt.Sprintf("%s : %s", rule.Define, rule.Help))
			}
			if len(rows) > 0 {
				result["command"] = rows
			}
		case []collect.Rule:
			// collect
			rs := collect.Ruleset(v)
			var rows []string
			for _, rule := range rs {
				rows = append(rows, fmt.Sprintf("%s : %s", rule.Id, rule.Help))
			}
			if len(rows) > 0 {
				result["collect"] = rows
			}
		case []cron.Rule:
			// cron
			rs := v
			var rows []string
			for _, rule := range rs {
				rows = append(rows, fmt.Sprintf("%s : %s", rule.Name, rule.Help))
			}
			if len(rows) > 0 {
				result["cron"] = rows
			}
		}
	}

	return result, nil
}

func RunCommand(commandRules []command.Rule, ctx types.Context, content interface{}) (types.MsgPayload, error) {
	in, ok := content.(string)
	if !ok {
		return nil, nil
	}
	rs := command.Ruleset(commandRules)
	payload, err := rs.Help(in)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		return payload, nil
	}

	payload, err = rs.ProcessCommand(ctx, in)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func RunForm(formRules []form.Rule, ctx types.Context, values types.KV) (types.MsgPayload, error) {
	// check form
	exForm, err := store.Database.FormGet(ctx.FormId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if exForm.ID == 0 {
		return nil, nil
	}
	if exForm.State > model.FormStateCreated {
		return nil, nil
	}

	// process form
	rs := form.Ruleset(formRules)
	payload, err := rs.ProcessForm(ctx, values)
	if err != nil {
		return nil, err
	}

	// is long term
	isLongTerm := false
	for _, rule := range rs {
		if rule.Id == ctx.FormRuleId {
			isLongTerm = rule.IsLongTerm
		}
	}
	if !isLongTerm {
		// store form
		err = store.Database.FormSet(ctx.FormId, model.Form{Values: model.JSON(values), State: model.FormStateSubmitSuccess})
		if err != nil {
			return nil, err
		}

		// store page state
		err = store.Database.PageSet(ctx.FormId, model.Page{State: model.PageStateProcessedSuccess})
		if err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func RunPage(pageRules []page.Rule, ctx types.Context, flag string, args types.KV) (string, error) {
	rs := page.Ruleset(pageRules)
	return rs.ProcessPage(ctx, flag, args)
}

func PageURL(ctx types.Context, pageRuleId string, param types.KV, expiredDuration time.Duration) (string, error) {
	if param == nil {
		param = types.KV{}
	}
	param["platform"] = ctx.Platform
	param["topic"] = ctx.Topic
	param["uid"] = ctx.AsUser.String()
	flag, err := StoreParameter(param, time.Now().Add(expiredDuration))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/page/%s/%s", types.AppUrl(), pageRuleId, flag), nil
}

func ServiceURL(ctx types.Context, group, path string, param types.KV) string {
	if param == nil {
		param = types.KV{}
	}
	param["platform"] = ctx.Platform
	param["topic"] = ctx.Topic
	param["uid"] = ctx.AsUser.String()
	flag, err := StoreParameter(param, time.Now().Add(time.Hour))
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/service/%s%s?p=%s", types.AppUrl(), group, path, flag)
}

func RunCron(cronRules []cron.Rule, name string) (*cron.Ruleset, error) {
	ruleset := cron.NewCronRuleset(name, cronRules)
	ruleset.Daemon()
	return ruleset, nil
}

func RunCollect(collectRules []collect.Rule, ctx types.Context, content types.KV) (types.MsgPayload, error) {
	rs := collect.Ruleset(collectRules)
	return rs.ProcessAgent(ctx, content)
}

func RunEvent(eventRules []event.Rule, ctx types.Context, param types.KV) error {
	rs := event.Ruleset(eventRules)
	return rs.ProcessEvent(ctx, param)
}

func RunWorkflow(workflowRules []workflow.Rule, ctx types.Context, input types.KV) (types.KV, error) {
	rs := workflow.Ruleset(workflowRules)
	return rs.ProcessRule(ctx, input)
}

func RunWebhook(webhookRules []webhook.Rule, ctx types.Context, data []byte) (types.MsgPayload, error) {
	rs := webhook.Ruleset(webhookRules)
	return rs.ProcessRule(ctx, data)
}

func RunTool(toolRules []tool.Rule, ctx types.Context, argumentsInJSON string) (string, error) {
	rs := tool.Ruleset(toolRules)
	return rs.ProcessRule(ctx, argumentsInJSON)
}

func FormMsg(ctx types.Context, id string) types.MsgPayload {
	// get form fields
	formMsg := types.FormMsg{ID: id}
	var title string
	var field []types.FormField
	for _, handler := range List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []form.Rule:
				for _, rule := range v {
					if rule.Id != id {
						continue
					}

					title = rule.Title
					field = rule.Field

					// default value type
					for index, formField := range field {
						if formField.ValueType == "" {
							switch formField.Type {
							case types.FormFieldText, types.FormFieldPassword, types.FormFieldTextarea,
								types.FormFieldEmail, types.FormFieldUrl:
								field[index].ValueType = types.FormFieldValueString
							case types.FormFieldNumber:
								field[index].ValueType = types.FormFieldValueInt64
							}
						}
					}
				}
			}
		}
	}
	if len(field) <= 0 {
		return types.TextMsg{Text: "form field error"}
	}
	formMsg.Title = title
	formMsg.Field = field

	return StoreForm(ctx, formMsg)
}

func StoreForm(ctx types.Context, payload types.MsgPayload) types.MsgPayload {
	formMsg, ok := payload.(types.FormMsg)
	if !ok {
		return types.TextMsg{Text: "form msg error"}
	}

	formId := types.Id()
	d, err := sonic.Marshal(payload)
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}
	schema := types.KV{}
	err = schema.Scan(d)
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}

	var values = make(types.KV)
	if v, ok := payload.(types.FormMsg); ok {
		for _, field := range v.Field {
			values[field.Key] = nil
		}
	}

	// set extra
	var extra = make(types.KV)

	// store form
	err = store.Database.FormSet(formId, model.Form{
		FormID: formId,
		UID:    ctx.AsUser.String(),
		Topic:  ctx.Topic,
		Schema: model.JSON(schema),
		Values: model.JSON(values),
		Extra:  model.JSON(extra),
		State:  model.FormStateCreated,
	})
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}

	// store page
	err = store.Database.PageSet(formId, model.Page{
		PageID: formId,
		UID:    ctx.AsUser.String(),
		Topic:  ctx.Topic,
		Type:   model.PageForm,
		Schema: model.JSON(schema),
		State:  model.PageStateCreated,
	})
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}

	return types.LinkMsg{
		Title: fmt.Sprintf("%s Form[%s]", formMsg.Title, formId),
		Url:   fmt.Sprintf("%s/p/%s", types.AppUrl(), formId),
	}
}

func StoreParameter(params types.KV, expiredAt time.Time) (string, error) {
	flag := types.Id()
	return flag, store.Database.ParameterSet(flag, params, expiredAt)
}

func StorePage(ctx types.Context, category model.PageType, title string, payload types.MsgPayload) types.MsgPayload {
	pageId := types.Id()
	d, err := sonic.Marshal(payload)
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}
	schema := types.KV{}
	err = schema.Scan(d)
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}

	// store page
	err = store.Database.PageSet(pageId, model.Page{
		PageID: pageId,
		UID:    ctx.AsUser.String(),
		Topic:  ctx.Topic,
		Type:   category,
		Schema: model.JSON(schema),
		State:  model.PageStateCreated,
	})
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}

	// fix han compatible styles
	title = fmt.Sprintf("%s %s", category, title)
	if utils.HasHan(title) {
		title = ""
	}

	return types.LinkMsg{
		Title: title,
		Url:   fmt.Sprintf("%s/p/%s", types.AppUrl(), pageId),
	}
}

func InstructMsg(ctx types.Context, id string, data types.KV) types.MsgPayload {
	var botName string
	for name, handler := range List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []instruct.Rule:
				for _, rule := range v {
					if rule.Id == id {
						botName = name
					}
				}
			}
		}
	}

	return StoreInstruct(ctx, types.InstructMsg{
		No:       types.Id(),
		Object:   model.InstructObjectAgent,
		Bot:      botName,
		Flag:     id,
		Content:  data,
		Priority: model.InstructPriorityDefault,
		State:    model.InstructCreate,
		ExpireAt: time.Now().Add(time.Hour),
	})
}

func StoreInstruct(ctx types.Context, payload types.MsgPayload) types.MsgPayload {
	msg, ok := payload.(types.InstructMsg)
	if !ok {
		return types.TextMsg{Text: "error instruct msg type"}
	}

	_, err := store.Database.CreateInstruct(&model.Instruct{
		UID:      ctx.AsUser.String(),
		No:       msg.No,
		Object:   msg.Object,
		Bot:      msg.Bot,
		Flag:     msg.Flag,
		Content:  model.JSON(msg.Content),
		Priority: msg.Priority,
		State:    msg.State,
		ExpireAt: msg.ExpireAt,
	})
	if err != nil {
		return types.TextMsg{Text: "store instruct error"}
	}

	// event todo
	//err = pkgEvent.PublishMessage(pkgEvent.InstructEvent, types.KV{
	//	"uid":       ctx.AsUser.String(),
	//	"no":        msg.No,
	//	"object":    msg.Object,
	//	"bot":       msg.Bot,
	//	"flag":      msg.Flag,
	//	"content":   msg.Content,
	//	"state":     msg.State,
	//	"expire_at": msg.ExpireAt,
	//})
	//if err != nil {
	//	flog.Error(err)
	//}

	return types.TextMsg{Text: fmt.Sprintf("Instruct[%s:%s]", msg.Flag, msg.No)}
}

func SettingCovertForm(id string, rule setting.Rule) form.Rule {
	var result form.Rule
	result.Id = fmt.Sprintf("%s_setting", id)
	result.Title = fmt.Sprintf("%s Bot Setting", utils.FirstUpper(id))
	result.Field = []types.FormField{}

	for _, row := range rule {
		result.Field = append(result.Field, types.FormField{
			Key:   row.Key,
			Type:  row.Type,
			Label: row.Title,
		})
	}

	result.Handler = func(ctx types.Context, values types.KV) types.MsgPayload {
		for key, value := range values {
			err := store.Database.ConfigSet(ctx.AsUser, ctx.Topic, fmt.Sprintf("%s_%s", id, key), types.KV{
				"value": value,
			})
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("setting [%s] %s error", ctx.FormId, key)}
			}
		}
		return types.TextMsg{Text: fmt.Sprintf("ok, setting [%s]", ctx.FormId)}
	}

	return result
}

func SettingGet(ctx types.Context, id string, key string) (types.KV, error) {
	return store.Database.ConfigGet(ctx.AsUser, ctx.Topic, fmt.Sprintf("%s_%s", id, key))
}

func SettingMsg(ctx types.Context, id string) types.MsgPayload {
	return FormMsg(ctx, fmt.Sprintf("%s_setting", id))
}

func Behavior(uid types.Uid, flag string, number int) {
	b, err := store.Database.BehaviorGet(uid, flag)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if b.ID > 0 {
		_ = store.Database.BehaviorIncrease(uid, flag, number)
	} else {
		_ = store.Database.BehaviorSet(model.Behavior{
			UID:    uid.String(),
			Flag:   flag,
			Count_: int32(number),
		})
	}
}

func Webservice(app *fiber.App, name string, ruleset webservice.Ruleset) {
	if len(ruleset) == 0 {
		return
	}
	var routes []*route.Router
	for _, rule := range ruleset {
		routes = append(routes, route.Route(rule.Method, rule.Path, rule.Function, rule.Option...))
	}
	route.WebService(app, name, routes...)
}

func Shortcut(title, link string) (string, error) {
	endpoint, _ := providers.GetConfig(slash.ID, slash.EndpointKey)

	name, err := utils.RandomString(6)
	if err != nil {
		return "", err
	}

	client := slash.GetClient()
	err = client.CreateShortcut(slash.Shortcut{
		Name:  name,
		Link:  link,
		Title: title,
		Tags:  []string{"flowbot"},
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/s/%s", endpoint, name), nil
}

// AvailableTools  the tools/functions we're making available for the model.
func AvailableTools(ctx types.Context) ([]llmTool.BaseTool, error) {
	var tools []llmTool.BaseTool
	for _, handler := range handlers {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []tool.Rule:
				for _, rule := range v {
					t, err := rule(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to create tool: %w", err)
					}
					tools = append(tools, t)
				}
			}
		}
	}
	return tools, nil
}

func FindRuleAndHandler[T types.Ruler](flag string, handlers map[string]Handler) (T, Handler) {
	for _, handler := range handlers {
		for _, item := range handler.Rules() {
			if rules, ok := item.([]T); ok {
				for _, rule := range rules {
					if rule.ID() == flag {
						return rule, handler
					}
				}
			}
		}
	}
	var zero T
	return zero, nil
}

type configType struct {
	Name string `json:"name"`
}

// Init initializes registered handlers.
func Init(jsonconf json.RawMessage) error {
	var config []json.RawMessage

	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	configMap := make(map[string]json.RawMessage)
	for _, cc := range config {
		var item configType
		if err := sonic.Unmarshal(cc, &item); err != nil {
			return errors.New("failed to parse config: " + err.Error())
		}

		configMap[item.Name] = cc
	}
	for name, bot := range handlers {
		var configItem json.RawMessage
		if v, ok := configMap[name]; ok {
			configItem = v
		} else {
			// default config
			configItem = []byte(`{"enabled": true}`)
		}
		if err := bot.Init(configItem); err != nil {
			return err
		}
	}

	return nil
}

// Bootstrap bots bootstrap
func Bootstrap() error {
	for _, bot := range handlers {
		if !bot.IsReady() {
			continue
		}
		if err := bot.Bootstrap(); err != nil {
			return err
		}
	}
	return nil
}

// Cron registered handlers
func Cron() ([]*cron.Ruleset, error) {
	rss := make([]*cron.Ruleset, 0)
	for _, bot := range handlers {
		rs, err := bot.Cron()
		if err != nil {
			return nil, err
		}
		if rs != nil {
			rss = append(rss, rs)
		}
	}
	return rss, nil
}

// List registered handlers
func List() map[string]Handler {
	return handlers
}
