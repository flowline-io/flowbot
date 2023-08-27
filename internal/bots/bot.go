package bots

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/internal/ruleset/action"
	"github.com/flowline-io/flowbot/internal/ruleset/agent"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/ruleset/condition"
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/ruleset/event"
	"github.com/flowline-io/flowbot/internal/ruleset/form"
	"github.com/flowline-io/flowbot/internal/ruleset/instruct"
	"github.com/flowline-io/flowbot/internal/ruleset/page"
	"github.com/flowline-io/flowbot/internal/ruleset/pipeline"
	"github.com/flowline-io/flowbot/internal/ruleset/session"
	"github.com/flowline-io/flowbot/internal/ruleset/setting"
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

const BotNameSuffix = "_bot"

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

	// Action return bot action result
	Action(ctx types.Context, option string) (types.MsgPayload, error)

	// Session return bot session result
	Session(ctx types.Context, content interface{}) (types.MsgPayload, error)

	// Cron cron script daemon
	Cron(send types.SendFunc) (*cron.Ruleset, error)

	// Condition run conditional process
	Condition(ctx types.Context, forwarded types.MsgPayload) (types.MsgPayload, error)

	// Group return group result
	Group(ctx types.Context, head types.KV, content interface{}) (types.MsgPayload, error)

	// Pipeline return pipeline result
	Pipeline(ctx types.Context, head types.KV, content interface{}, operate types.PipelineOperate) (types.MsgPayload, string, int, error)

	// Agent return group result
	Agent(ctx types.Context, content types.KV) (types.MsgPayload, error)

	// Instruct return instruct list
	Instruct() (instruct.Ruleset, error)

	// Page return page
	Page(ctx types.Context, flag string) (string, error)

	// Webservice return webservice routes
	Webservice() *restful.WebService

	// Webapp return webapp
	Webapp() func(rw http.ResponseWriter, req *http.Request)
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

func (Base) Action(_ types.Context, _ string) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Session(_ types.Context, _ interface{}) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Cron(_ types.SendFunc) (*cron.Ruleset, error) {
	return nil, nil
}

func (Base) Condition(_ types.Context, _ types.MsgPayload) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Group(_ types.Context, _ types.KV, _ interface{}) (types.MsgPayload, error) {
	return nil, nil
}

func (Base) Pipeline(_ types.Context, _ types.KV, _ interface{}, _ types.PipelineOperate) (types.MsgPayload, string, int, error) {
	return nil, "", 0, nil
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

func (Base) Webservice() *restful.WebService {
	return nil
}

func (Base) Webapp() func(rw http.ResponseWriter, req *http.Request) {
	return nil
}

type configType struct {
	Name string `json:"name"`
}

var handlers map[string]Handler

func Register(name string, bot Handler) {
	if handlers == nil {
		handlers = make(map[string]Handler)
	}

	if bot == nil {
		panic("Register: bot is nil")
	}
	if _, dup := handlers[name]; dup {
		panic("Register: called twice for bot " + name)
	}
	handlers[name] = bot
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
		case []agent.Rule:
			// agent
			rs := agent.Ruleset(v)
			var rows []string
			for _, rule := range rs {
				rows = append(rows, fmt.Sprintf("%s : %s", rule.Id, rule.Help))
			}
			if len(rows) > 0 {
				result["agent"] = rows
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

func RunGroup(eventRules []event.Rule, ctx types.Context, head types.KV, content interface{}) (types.MsgPayload, error) {
	rs := event.Ruleset(eventRules)
	payload, err := rs.ProcessEvent(ctx, head, content)
	if err != nil {
		return nil, err
	}
	// todo
	if len(payload) > 0 {
		return payload[0], nil
	}
	return nil, nil
}

func HelpPipeline(pipelineRules []pipeline.Rule, _ types.Context, _ types.KV, content interface{}) (types.MsgPayload, error) {
	rs := pipeline.Ruleset(pipelineRules)
	in, ok := content.(string)
	if ok {
		payload, err := rs.Help(in)
		if err != nil {
			return nil, err
		}
		if payload != nil {
			return payload, nil
		}
	}
	return nil, nil
}

func TriggerPipeline(pipelineRules []pipeline.Rule, ctx types.Context, _ types.KV, content interface{}, trigger types.TriggerType) (string, pipeline.Rule, error) {
	rs := pipeline.Ruleset(pipelineRules)
	in, ok := content.(string)
	if ok {
		rule, err := rs.TriggerPipeline(ctx, trigger, in)
		if err != nil {
			return "", pipeline.Rule{}, err
		}

		pipelineFlag := ""
		if ctx.PipelineFlag == "" {
			// store pipeline
			flag, err := StorePipeline(ctx, rule, 0)
			if err != nil {
				logs.Err.Println(err)
				return "", pipeline.Rule{}, err
			}
			pipelineFlag = flag
		}

		return pipelineFlag, rule, nil
	}
	return "", pipeline.Rule{}, errors.New("error trigger")
}

func ProcessPipeline(ctx types.Context, pipelineRule pipeline.Rule, index int) (types.MsgPayload, error) {
	if index < 0 || index > len(pipelineRule.Step) {
		return nil, errors.New("error pipeline stage index")
	}
	if index == len(pipelineRule.Step) {
		return types.TextMsg{Text: "Pipeline Done"}, SetPipelineState(ctx, ctx.PipelineFlag, model.PipelineDone)
	}
	var payload types.MsgPayload
	stage := pipelineRule.Step[index]
	switch stage.Type {
	case types.FormStage:
		payload = FormMsg(ctx, stage.Flag)
	case types.ActionStage:
		payload = ActionMsg(ctx, stage.Flag)
	case types.CommandStage:
		for name, handler := range List() {
			if stage.Bot != types.Bot(name) {
				continue
			}
			for _, item := range handler.Rules() {
				switch v := item.(type) {
				case []command.Rule:
					for _, rule := range v {
						tokens, err := parser.ParseString(strings.Join(stage.Args, " "))
						if err != nil {
							return nil, err
						}
						check, err := parser.SyntaxCheck(rule.Define, tokens)
						if err != nil {
							return nil, err
						}
						if !check {
							continue
						}
						payload = rule.Handler(ctx, tokens)
					}
				}
			}
		}
	case types.InstructStage:
		data := make(map[string]interface{}) // fixme
		for i, arg := range stage.Args {
			data[fmt.Sprintf("val%d", i+1)] = arg
		}
		payload = InstructMsg(ctx, stage.Flag, data)
	case types.SessionStage:
		data := make(map[string]interface{}) // fixme
		for i, arg := range stage.Args {
			data[fmt.Sprintf("val%d", i+1)] = arg
		}
		payload = SessionMsg(ctx, stage.Flag, data)
	}

	if payload != nil {
		return payload, nil
	}
	return nil, errors.New("error pipeline process")
}

func RunPipeline(pipelineRules []pipeline.Rule, ctx types.Context, head types.KV, content interface{}, operate types.PipelineOperate) (types.MsgPayload, string, int, error) {
	switch operate {
	case types.PipelineCommandTriggerOperate:
		payload, err := HelpPipeline(pipelineRules, ctx, head, content)
		if err != nil {
			return nil, "", 0, err
		}
		if payload != nil {
			return payload, "", 0, nil
		}
		flag, rule, err := TriggerPipeline(pipelineRules, ctx, head, content, types.TriggerCommandType)
		if err != nil {
			return nil, "", 0, err
		}
		ctx.PipelineFlag = flag
		ctx.PipelineVersion = rule.Version
		payload, err = ProcessPipeline(ctx, rule, 0)
		if err != nil {
			return nil, "", 0, err
		}
		return payload, flag, rule.Version, SetPipelineStep(ctx, flag, 1)
	case types.PipelineProcessOperate:
	case types.PipelineNextOperate:
		for _, rule := range pipelineRules {
			if rule.Id == ctx.PipelineRuleId {
				payload, err := ProcessPipeline(ctx, rule, ctx.PipelineStepIndex)
				if err != nil {
					return nil, "", 0, err
				}
				return payload, ctx.PipelineFlag, ctx.PipelineVersion, SetPipelineStep(ctx, ctx.PipelineFlag, ctx.PipelineStepIndex+1)
			}
		}
	}
	return nil, "", 0, nil
}

func StorePipeline(ctx types.Context, pipelineRule pipeline.Rule, index int) (string, error) {
	flag := types.Id().String()
	return flag, store.Chatbot.PipelineCreate(model.Pipeline{
		UID:     ctx.AsUser.UserId(),
		Topic:   ctx.Original,
		Flag:    flag,
		RuleID:  pipelineRule.Id,
		Version: int32(pipelineRule.Version),
		Stage:   int32(index),
		State:   model.PipelineStart,
	})
}

func SetPipelineState(ctx types.Context, flag string, state model.PipelineState) error {
	return store.Chatbot.PipelineState(ctx.AsUser, ctx.Original, model.Pipeline{
		Flag:  flag,
		State: state,
	})
}

func SetPipelineStep(ctx types.Context, flag string, index int) error {
	return store.Chatbot.PipelineStep(ctx.AsUser, ctx.Original, model.Pipeline{
		Flag:  flag,
		Stage: int32(index),
	})
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
	exForm, err := store.Chatbot.FormGet(ctx.FormId)
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
		err = store.Chatbot.FormSet(ctx.FormId, model.Form{Values: model.JSON(values), State: model.FormStateSubmitSuccess})
		if err != nil {
			return nil, err
		}

		// store page state
		err = store.Chatbot.PageSet(ctx.FormId, model.Page{State: model.PageStateProcessedSuccess})
		if err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func RunPage(pageRules []page.Rule, ctx types.Context, flag string) (string, error) {
	rs := page.Ruleset(pageRules)
	return rs.ProcessPage(ctx, flag)
}

func PageURL(ctx types.Context, pageRuleId string, param types.KV, expiredDuration time.Duration) (string, error) {
	if param == nil {
		param = types.KV{}
	}
	param["original"] = ctx.Original
	param["topic"] = ctx.RcptTo
	param["uid"] = ctx.AsUser.UserId()
	flag, err := StoreParameter(param, time.Now().Add(expiredDuration))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/extra/p/%s/%s", types.AppUrl(), pageRuleId, flag), nil
}

func ServiceURL(ctx types.Context, group, version, path string, param types.KV) string {
	if param == nil {
		param = types.KV{}
	}
	param["original"] = ctx.Original
	param["topic"] = ctx.RcptTo
	param["uid"] = ctx.AsUser.UserId()
	flag, err := StoreParameter(param, time.Now().Add(time.Hour))
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/bot/%s/%s%s?p=%s", types.AppUrl(), group, version, path, flag)
}

func AppURL(ctx types.Context, name string, param types.KV) string {
	if param == nil {
		param = types.KV{}
	}
	param["original"] = ctx.Original
	param["topic"] = ctx.RcptTo
	param["uid"] = ctx.AsUser.UserId()
	flag, err := StoreParameter(param, time.Now().Add(time.Hour))
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/app/%s/?p=%s", types.AppUrl(), name, flag)
}

func RunAction(actionRules []action.Rule, ctx types.Context, option string) (types.MsgPayload, error) {
	// check action
	exAction, err := store.Chatbot.ActionGet(ctx.RcptTo, ctx.SeqId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if exAction.ID > 0 && exAction.State > model.ActionStateLongTerm {
		return types.TextMsg{Text: "done"}, nil
	}

	// process action
	rs := action.Ruleset(actionRules)
	payload, err := rs.ProcessAction(ctx, option)
	if err != nil {
		return nil, err
	}

	// is long term
	isLongTerm := false
	for _, rule := range rs {
		if rule.Id == ctx.ActionRuleId {
			isLongTerm = rule.IsLongTerm
		}
	}
	var state model.ActionState
	if !isLongTerm {
		state = model.ActionStateSubmitSuccess
	} else {
		state = model.ActionStateLongTerm
	}
	// store action
	err = store.Chatbot.ActionSet(ctx.RcptTo, ctx.SeqId, model.Action{UID: ctx.AsUser.UserId(), Value: option, State: state})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func RunCron(cronRules []cron.Rule, name string, send types.SendFunc) (*cron.Ruleset, error) {
	ruleset := cron.NewCronRuleset(name, cronRules)
	ruleset.Send = send
	ruleset.Daemon()
	return ruleset, nil
}

func RunCondition(conditionRules []condition.Rule, ctx types.Context, forwarded types.MsgPayload) (types.MsgPayload, error) {
	rs := condition.Ruleset(conditionRules)
	return rs.ProcessCondition(ctx, forwarded)
}

func RunAgent(agentVersion int, agentRules []agent.Rule, ctx types.Context, content types.KV) (types.MsgPayload, error) {
	rs := agent.Ruleset(agentRules)
	return rs.ProcessAgent(agentVersion, ctx, content)
}

func RunSession(sessionRules []session.Rule, ctx types.Context, content interface{}) (types.MsgPayload, error) {
	rs := session.Ruleset(sessionRules)
	return rs.ProcessSession(ctx, content)
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
					if rule.Id == id {
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

	formId := types.Id().String()
	d, err := json.Marshal(payload)
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}
	schema := types.KV{}
	err = schema.Scan(d)
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}

	var values types.KV = make(map[string]interface{})
	if v, ok := payload.(types.FormMsg); ok {
		for _, field := range v.Field {
			values[field.Key] = nil
		}
	}

	// set extra
	var extra types.KV = make(map[string]interface{})
	if ctx.PipelineFlag != "" {
		extra["pipeline_flag"] = ctx.PipelineFlag
		extra["pipeline_version"] = ctx.PipelineVersion
	}

	// store form
	err = store.Chatbot.FormSet(formId, model.Form{
		FormID: formId,
		UID:    ctx.AsUser.UserId(),
		Topic:  ctx.Original,
		Schema: model.JSON(schema),
		Values: model.JSON(values),
		Extra:  model.JSON(extra),
		State:  model.FormStateCreated,
	})
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}

	// store page
	err = store.Chatbot.PageSet(formId, model.Page{
		PageID: formId,
		UID:    ctx.AsUser.UserId(),
		Topic:  ctx.Original,
		Type:   model.PageForm,
		Schema: model.JSON(schema),
		State:  model.PageStateCreated,
	})
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}

	return types.LinkMsg{
		Title: fmt.Sprintf("%s Form[%s]", formMsg.Title, formId),
		Url:   fmt.Sprintf("%s/extra/page/%s", types.AppUrl(), formId),
	}
}

func StoreParameter(params types.KV, expiredAt time.Time) (string, error) {
	flag := strings.ToLower(types.Id().String())
	return flag, store.Chatbot.ParameterSet(flag, params, expiredAt)
}

func ActionMsg(_ types.Context, id string) types.MsgPayload {
	var title string
	var option []string
	for _, handler := range List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []action.Rule:
				for _, rule := range v {
					if rule.Id == id {
						title = rule.Title
						option = rule.Option
					}
				}
			}
		}
	}
	if len(option) <= 0 {
		return types.TextMsg{Text: "error action rule id"}
	}

	return types.ActionMsg{
		ID:     id,
		Title:  title,
		Option: option,
	}
}

func StorePage(ctx types.Context, category model.PageType, title string, payload types.MsgPayload) types.MsgPayload {
	pageId := types.Id().String()
	d, err := json.Marshal(payload)
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}
	schema := types.KV{}
	err = schema.Scan(d)
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}

	// store page
	err = store.Chatbot.PageSet(pageId, model.Page{
		PageID: pageId,
		UID:    ctx.AsUser.UserId(),
		Topic:  ctx.Original,
		Type:   category,
		Schema: model.JSON(schema),
		State:  model.PageStateCreated,
	})
	if err != nil {
		logs.Err.Println(err)
		return types.TextMsg{Text: "store form error"}
	}

	// fix han compatible styles
	title = fmt.Sprintf("%s %s", category, title)
	if utils.HasHan(title) {
		title = ""
	}

	return types.LinkMsg{
		Title: title,
		Url:   fmt.Sprintf("%s/extra/page/%s", types.AppUrl(), pageId),
	}
}

func SessionMsg(ctx types.Context, id string, data types.KV) types.MsgPayload {
	var title string
	for _, handler := range List() {
		for _, item := range handler.Rules() {
			switch v := item.(type) {
			case []session.Rule:
				for _, rule := range v {
					if rule.Id == id {
						title = rule.Title
					}
				}
			}
		}
	}
	if title == "" {
		return types.TextMsg{Text: "error session id"}
	}

	ctx.SessionRuleId = id
	err := SessionStart(ctx, data)
	if err != nil {
		return types.TextMsg{Text: "session error"}
	}

	return types.TextMsg{Text: title}
}

func SessionStart(ctx types.Context, initValues types.KV) error {
	sess, err := store.Chatbot.SessionGet(ctx.AsUser, ctx.Original)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if sess.ID > 0 && sess.State == model.SessionStart {
		return errors.New("already a session started")
	}
	var values = types.KV{"val": nil}
	_ = store.Chatbot.SessionCreate(model.Session{
		UID:    ctx.AsUser.UserId(),
		Topic:  ctx.Original,
		RuleID: ctx.SessionRuleId,
		Init:   model.JSON(initValues),
		Values: model.JSON(values),
		State:  model.SessionStart,
	})
	return nil
}

func SessionDone(ctx types.Context) {
	_ = store.Chatbot.SessionState(ctx.AsUser, ctx.Original, model.SessionDone)
}

func CreateShortUrl(text string) (string, error) {
	if utils.IsUrl(text) {
		url, err := store.Chatbot.UrlGetByUrl(text)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		if url.ID > 0 {
			return fmt.Sprintf("%s/u/%s", types.AppUrl(), url.Flag), nil
		}
		flag := strings.ToLower(types.Id().String())
		err = store.Chatbot.UrlCreate(model.Url{
			Flag:  flag,
			URL:   text,
			State: model.UrlStateEnable,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s/u/%s", types.AppUrl(), flag), nil
	}
	return "", errors.New("error url")
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
		No:       types.Id().String(),
		Object:   model.InstructObjectLinkit,
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

	_, err := store.Chatbot.CreateInstruct(&model.Instruct{
		UID:      ctx.AsUser.UserId(),
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

	// event
	err = pkgEvent.Emit(pkgEvent.InstructEvent, types.KV{
		"uid":       ctx.AsUser.UserId(),
		"no":        msg.No,
		"object":    msg.Object,
		"bot":       msg.Bot,
		"flag":      msg.Flag,
		"content":   msg.Content,
		"state":     msg.State,
		"expire_at": msg.ExpireAt,
	})
	if err != nil {
		logs.Err.Println(err)
	}

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
			err := store.Chatbot.ConfigSet(ctx.AsUser, ctx.Original, fmt.Sprintf("%s_%s", id, key), types.KV{
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
	return store.Chatbot.ConfigGet(ctx.AsUser, ctx.Original, fmt.Sprintf("%s_%s", id, key))
}

func SettingMsg(ctx types.Context, id string) types.MsgPayload {
	return FormMsg(ctx, fmt.Sprintf("%s_setting", id))
}

const (
	MessageBotIncomingBehavior   = "message_bot_incoming"
	MessageGroupIncomingBehavior = "message_group_incoming"
)

func Behavior(uid types.Uid, flag string, number int) {
	b, err := store.Chatbot.BehaviorGet(uid, flag)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if b.ID > 0 {
		_ = store.Chatbot.BehaviorIncrease(uid, flag, number)
	} else {
		_ = store.Chatbot.BehaviorSet(model.Behavior{
			UID:    uid.UserId(),
			Flag:   flag,
			Count_: int32(number),
		})
	}
}

func ServeFile(rw http.ResponseWriter, req *http.Request, dist embed.FS, dir string) {
	s := fs.FS(dist)
	h, err := fs.Sub(s, dir)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	vars := mux.Vars(req)
	subpath, _ := vars["subpath"]
	if subpath == "" {
		subpath = "index.html"
	}

	if strings.HasSuffix(subpath, "html") {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	if strings.HasSuffix(subpath, "css") {
		rw.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	if strings.HasSuffix(subpath, "js") {
		rw.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	}
	if strings.HasSuffix(subpath, "svg") {
		rw.Header().Set("Content-Type", "image/svg+xml")
	}

	content, err := fs.ReadFile(h, subpath)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if subpath == "index.html" {
		flag := req.URL.Query().Get("p")
		if flag == "" {
			rw.WriteHeader(http.StatusForbidden)
			return
		}

		param, err := store.Chatbot.ParameterGet(flag)
		if err != nil {
			rw.WriteHeader(http.StatusForbidden)
			return
		}

		original, _ := types.KV(param.Params).String("original")
		topic, _ := types.KV(param.Params).String("topic")
		uid, _ := types.KV(param.Params).String("uid")

		jsScript := fmt.Sprintf(`
<body><script>let Global = {};Global.original = '%s';Global.topic = '%s';Global.uid = '%s';Global.flag = '%s';Global.base = '%s';</script>
`, original, topic, uid, flag, types.AppUrl())

		html := strings.ReplaceAll(utils.BytesToString(content), "<body>", jsScript)
		content = utils.StringToBytes(html)
	}

	_, _ = rw.Write(content)
}

func Webservice(name, version string, ruleset webservice.Ruleset) *restful.WebService {
	if len(ruleset) == 0 {
		return nil
	}
	var routes []*route.Router
	for _, rule := range ruleset {
		routes = append(routes, route.Route(rule.Method, rule.Path, rule.Function, rule.Documentation, rule.Option...))
	}
	return route.WebService(name, version, routes...)
}

// Init initializes registered handlers.
func Init(jsonconf json.RawMessage) error {
	var config []json.RawMessage

	if err := json.Unmarshal(jsonconf, &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
	}

	configMap := make(map[string]json.RawMessage)
	for _, cc := range config {
		var item configType
		if err := json.Unmarshal(cc, &item); err != nil {
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

func Bootstrap() error {
	for _, bot := range handlers {
		if !bot.IsReady() {
			continue
		}
		if err := bot.Bootstrap(); err != nil {
			return err
		}
		if err := bot.OnEvent(); err != nil {
			return err
		}
	}
	return nil
}

// Cron registered handlers
func Cron(send func(rcptTo string, uid types.Uid, out types.MsgPayload, option ...interface{})) ([]*cron.Ruleset, error) {
	rss := make([]*cron.Ruleset, 0)
	for _, bot := range handlers {
		rs, err := bot.Cron(send)
		if err != nil {
			return nil, err
		}
		if rs != nil {
			rss = append(rss, rs)
		}
	}
	return rss, nil
}

func List() map[string]Handler {
	return handlers
}
