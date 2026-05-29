package module

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"sync"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/flog"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/slash"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	MessageBotIncomingBehavior   = "message_bot_incoming"
	MessageGroupIncomingBehavior = "message_group_incoming"
)

var (
	handlers   map[string]Handler
	handlersMu sync.RWMutex
)

func Register(name string, module Handler) {
	handlersMu.Lock()
	defer handlersMu.Unlock()

	if handlers == nil {
		handlers = make(map[string]Handler)
	}

	if module == nil {
		flog.Fatal("Register: module is nil")
	}
	if _, dup := handlers[name]; dup {
		flog.Fatal("Register: called twice for module %s", name)
	}
	handlers[name] = module
	flog.Info("[module] %s registered", name)
}

// Unregister removes a previously registered module handler.
// It is a no-op if the name is not found.
// Intended primarily for test teardown.
func Unregister(name string) {
	handlersMu.Lock()
	defer handlersMu.Unlock()

	if handlers == nil {
		return
	}
	delete(handlers, name)
}

func Help(rules []any) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, rule := range rules {
		if v, ok := rule.([]command.Rule); ok {
			rs := command.Ruleset(v)
			var rows []string
			for _, rule := range rs {
				rows = append(rows, fmt.Sprintf("%s : %s", rule.Define, rule.Help))
			}
			if len(rows) > 0 {
				result["command"] = rows
			}
		}
	}

	return result, nil
}

func RunCommand(commandRules []command.Rule, ctx types.Context, content any) (types.MsgPayload, error) {
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
	exForm, err := store.Database.FormGet(ctx.Context(), ctx.FormId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return nil, err
	}
	if exForm.ID == 0 {
		return nil, nil
	}
	if exForm.State > int(schema.FormStateCreated) {
		return nil, nil
	}

	rs := form.Ruleset(formRules)
	payload, err := rs.ProcessForm(ctx, values)
	if err != nil {
		return nil, err
	}

	isLongTerm := false
	for _, rule := range rs {
		if rule.Id == ctx.FormRuleId {
			isLongTerm = rule.IsLongTerm
		}
	}
	if !isLongTerm {
		err = store.Database.FormSet(ctx.Context(), ctx.FormId, gen.Form{Values: values, State: int(schema.FormStateSubmitSuccess)})
		if err != nil {
			return nil, err
		}
	}

	return payload, nil
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

func FormMsg(ctx types.Context, id string) types.MsgPayload {
	formMsg := types.FormMsg{ID: id}
	var title string
	var field []types.FormField
	for _, handler := range List() {
		for _, item := range handler.Rules() {
			if v, ok := item.([]form.Rule); ok {
				for _, rule := range v {
					if rule.Id != id {
						continue
					}

					title = rule.Title
					field = rule.Field

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
	s := types.KV{}
	err = s.Scan(d)
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

	var extra = make(types.KV)

	err = store.Database.FormSet(ctx.Context(), formId, gen.Form{
		FormID: formId,
		UID:    ctx.AsUser.String(),
		Topic:  ctx.Topic,
		Schema: s,
		Values: values,
		Extra:  extra,
		State:  int(schema.FormStateCreated),
	})
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: "store form error"}
	}

	return types.LinkMsg{
		Title: fmt.Sprintf("%s Form[%s]", formMsg.Title, formId),
		Url:   fmt.Sprintf("%s/form/%s", types.AppUrl(), formId),
	}
}

func StoreParameter(params types.KV, expiredAt time.Time) (string, error) {
	flag := types.Id()
	return flag, store.Database.ParameterSet(context.Background(), flag, params, expiredAt)
}

func SettingGet(ctx types.Context, id string, key string) (types.KV, error) {
	return store.Database.ConfigGet(ctx.Context(), ctx.AsUser, ctx.Topic, fmt.Sprintf("%s_%s", id, key))
}

func SettingMsg(ctx types.Context, id string) types.MsgPayload {
	return FormMsg(ctx, fmt.Sprintf("%s_setting", id))
}

func Behavior(uid types.Uid, flag string, number int) {
	b, err := store.Database.BehaviorGet(context.Background(), uid, flag)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return
	}
	if b.ID > 0 {
		_ = store.Database.BehaviorIncrease(context.Background(), uid, flag, number)
	} else {
		_ = store.Database.BehaviorSet(context.Background(), gen.Behavior{
			UID:   uid.String(),
			Flag:  flag,
			Count: int32(number),
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

func FindRuleAndHandler[T types.Ruler](flag string, handlers map[string]Handler) (T, Handler) {
	keys := make([]string, 0, len(handlers))
	for k := range handlers {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		handler := handlers[k]
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

func Init(jsonconf json.RawMessage) error {
	var config []json.RawMessage

	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	configMap := make(map[string]json.RawMessage)
	for _, cc := range config {
		var item configType
		if err := sonic.Unmarshal(cc, &item); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		configMap[item.Name] = cc
	}

	handlersMu.RLock()
	for name, module := range handlers {
		handlersMu.RUnlock()
		var configItem json.RawMessage
		if v, ok := configMap[name]; ok {
			configItem = v
		} else {
			configItem = []byte(`{"enabled": true}`)
		}
		if err := module.Init(configItem); err != nil {
			return err
		}
		handlersMu.RLock()
	}
	handlersMu.RUnlock()

	return nil
}

func Bootstrap() error {
	handlersMu.RLock()
	defer handlersMu.RUnlock()

	for name, module := range handlers {
		if !module.IsReady() {
			continue
		}
		if err := module.Bootstrap(); err != nil {
			return fmt.Errorf("%s bootstrap: %w", name, err)
		}
	}
	return nil
}

func List() map[string]Handler {
	handlersMu.RLock()
	defer handlersMu.RUnlock()

	copyMap := make(map[string]Handler, len(handlers))
	maps.Copy(copyMap, handlers)
	return copyMap
}
