// Package chatbot forwards legacy chatbot APIs to pkg/module.
//
// Deprecated: use github.com/flowline-io/flowbot/pkg/module instead.
package chatbot

import (
	"encoding/json"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	modulepkg "github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/collect"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/setting"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/gofiber/fiber/v3"
)

const (
	// MessageBotIncomingBehavior is the legacy behavior metric key.
	//
	// Deprecated: use pkg/module when adding new module behavior metrics.
	MessageBotIncomingBehavior = modulepkg.MessageBotIncomingBehavior

	// MessageGroupIncomingBehavior is the legacy group behavior metric key.
	//
	// Deprecated: use pkg/module when adding new module behavior metrics.
	MessageGroupIncomingBehavior = modulepkg.MessageGroupIncomingBehavior
)

// Deprecated: use module.Register.
func Register(name string, handler Handler) {
	modulepkg.Register(name, handler)
}

// Deprecated: use module.Help.
func Help(rules []any) (map[string][]string, error) {
	return modulepkg.Help(rules)
}

// Deprecated: use module.RunCommand.
func RunCommand(commandRules []command.Rule, ctx types.Context, content any) (types.MsgPayload, error) {
	return modulepkg.RunCommand(commandRules, ctx, content)
}

// Deprecated: use module.RunForm.
func RunForm(formRules []form.Rule, ctx types.Context, values types.KV) (types.MsgPayload, error) {
	return modulepkg.RunForm(formRules, ctx, values)
}

// Deprecated: use module.RunPage.
func RunPage(pageRules []page.Rule, ctx types.Context, flag string, args types.KV) (string, error) {
	return modulepkg.RunPage(pageRules, ctx, flag, args)
}

// Deprecated: use module.PageURL.
func PageURL(ctx types.Context, pageRuleID string, param types.KV, expiredDuration time.Duration) (string, error) {
	return modulepkg.PageURL(ctx, pageRuleID, param, expiredDuration)
}

// Deprecated: use module.ServiceURL.
func ServiceURL(ctx types.Context, group, path string, param types.KV) string {
	return modulepkg.ServiceURL(ctx, group, path, param)
}

// Deprecated: use module.RunCron.
func RunCron(cronRules []cron.Rule, name string) (*cron.Ruleset, error) {
	return modulepkg.RunCron(cronRules, name)
}

// Deprecated: use module.RunCollect.
func RunCollect(collectRules []collect.Rule, ctx types.Context, content types.KV) (types.MsgPayload, error) {
	return modulepkg.RunCollect(collectRules, ctx, content)
}

// Deprecated: use module.RunEvent.
func RunEvent(eventRules []event.Rule, ctx types.Context, param types.KV) error {
	return modulepkg.RunEvent(eventRules, ctx, param)
}

// Deprecated: use module.RunWebhook.
func RunWebhook(webhookRules []webhook.Rule, ctx types.Context, data []byte) (types.MsgPayload, error) {
	return modulepkg.RunWebhook(webhookRules, ctx, data)
}

// Deprecated: use module.FormMsg.
func FormMsg(ctx types.Context, id string) types.MsgPayload {
	return modulepkg.FormMsg(ctx, id)
}

// Deprecated: use module.StoreForm.
func StoreForm(ctx types.Context, payload types.MsgPayload) types.MsgPayload {
	return modulepkg.StoreForm(ctx, payload)
}

// Deprecated: use module.StoreParameter.
func StoreParameter(params types.KV, expiredAt time.Time) (string, error) {
	return modulepkg.StoreParameter(params, expiredAt)
}

// Deprecated: use module.StorePage.
func StorePage(ctx types.Context, category model.PageType, title string, payload types.MsgPayload) types.MsgPayload {
	return modulepkg.StorePage(ctx, category, title, payload)
}

// Deprecated: use module.InstructMsg.
func InstructMsg(ctx types.Context, id string, data types.KV) types.MsgPayload {
	return modulepkg.InstructMsg(ctx, id, data)
}

// Deprecated: use module.StoreInstruct.
func StoreInstruct(ctx types.Context, payload types.MsgPayload) types.MsgPayload {
	return modulepkg.StoreInstruct(ctx, payload)
}

// Deprecated: use module.SettingCovertForm.
func SettingCovertForm(id string, rule setting.Rule) form.Rule {
	return modulepkg.SettingCovertForm(id, rule)
}

// Deprecated: use module.SettingGet.
func SettingGet(ctx types.Context, id string, key string) (types.KV, error) {
	return modulepkg.SettingGet(ctx, id, key)
}

// Deprecated: use module.SettingMsg.
func SettingMsg(ctx types.Context, id string) types.MsgPayload {
	return modulepkg.SettingMsg(ctx, id)
}

// Deprecated: use module.Behavior.
func Behavior(uid types.Uid, flag string, number int) {
	modulepkg.Behavior(uid, flag, number)
}

// Deprecated: use module.Webservice.
func Webservice(app *fiber.App, name string, ruleset webservice.Ruleset) {
	modulepkg.Webservice(app, name, ruleset)
}

// Deprecated: use module.Shortcut.
func Shortcut(title, link string) (string, error) {
	return modulepkg.Shortcut(title, link)
}

// Deprecated: use module.FindRuleAndHandler.
func FindRuleAndHandler[T types.Ruler](flag string, handlers map[string]Handler) (T, Handler) {
	return modulepkg.FindRuleAndHandler[T](flag, handlers)
}

// Deprecated: use module.Init.
func Init(jsonconf json.RawMessage) error {
	return modulepkg.Init(jsonconf)
}

// Deprecated: use module.Bootstrap.
func Bootstrap() error {
	return modulepkg.Bootstrap()
}

// Deprecated: use module.Cron.
func Cron() ([]*cron.Ruleset, error) {
	return modulepkg.Cron()
}

// Deprecated: use module.List.
func List() map[string]Handler {
	return modulepkg.List()
}
