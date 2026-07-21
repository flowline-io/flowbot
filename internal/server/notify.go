package server

import (
	"context"

	"github.com/bytedance/sonic"

	storeDB "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	abilitynotify "github.com/flowline-io/flowbot/pkg/capability/notify"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/notify/messagepusher"
	"github.com/flowline-io/flowbot/pkg/notify/ntfy"
	"github.com/flowline-io/flowbot/pkg/notify/pushover"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	"github.com/flowline-io/flowbot/pkg/notify/slack"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types/model"

	"go.uber.org/fx"
)

var NotifyModules = fx.Options(
	fx.Invoke(
		messagepusher.Register,
		ntfy.Register,
		pushover.Register,
		slack.Register,
		initNotificationGateway,
	),
)

func initNotificationGateway(lc fx.Lifecycle, store *cache.RedisStore) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			templates, err := loadNotifyTemplatesFromDB(ctx)
			if err != nil {
				flog.Warn("failed to load notify templates from DB: %v", err)
				templates = nil
			}
			if err := notifytmpl.Init(templates); err != nil {
				return err
			}
			if err := notify.SeedAgentNotifyTemplate(ctx); err != nil {
				flog.Warn("failed to seed agent.notify template: %v", err)
			} else {
				// Reload so the seeded template is available without restart.
				if reloaded, loadErr := loadNotifyTemplatesFromDB(ctx); loadErr == nil {
					if initErr := notifytmpl.Init(reloaded); initErr != nil {
						flog.Warn("failed to reload notify templates after seed: %v", initErr)
					}
				}
			}

			enabled := true
			dbRules, err := storeDB.Database.ListNotifyRules(ctx, storeDB.ListNotifyRuleOptions{Enabled: &enabled})
			var rules []notify.Rule
			if err != nil {
				flog.Warn("failed to load notify rules from DB: %v", err)
			} else {
				rules = modelNotifyRulesToManifest(dbRules)
			}
			if err := notifyrules.Init(store, rules); err != nil {
				return err
			}

			return abilitynotify.Register()
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})
}

// loadNotifyTemplatesFromDB loads persisted templates and converts them to notify manifests.
func loadNotifyTemplatesFromDB(ctx context.Context) ([]notify.Template, error) {
	rows, err := storeDB.Database.ListNotifyTemplates(ctx, storeDB.ListNotifyTemplateOptions{})
	if err != nil {
		return nil, err
	}
	return modelNotifyTemplatesToManifest(rows)
}

// modelNotifyTemplatesToManifest converts store models to notify template manifests.
func modelNotifyTemplatesToManifest(rows []model.NotifyTemplate) ([]notify.Template, error) {
	out := make([]notify.Template, 0, len(rows))
	for _, row := range rows {
		tmpl, err := modelNotifyTemplateToManifest(row)
		if err != nil {
			flog.Warn("skipping notify template %s: %v", row.TemplateID, err)
			continue
		}
		out = append(out, tmpl)
	}
	return out, nil
}

// modelNotifyTemplateToManifest converts a single store template model to a notify manifest.
func modelNotifyTemplateToManifest(row model.NotifyTemplate) (notify.Template, error) {
	var overrides []notify.Override
	if row.OverridesJSON != "" && row.OverridesJSON != "[]" {
		if err := sonic.Unmarshal([]byte(row.OverridesJSON), &overrides); err != nil {
			return notify.Template{}, err
		}
	}
	return notify.Template{
		ID:              row.TemplateID,
		Name:            row.Name,
		Description:     row.Description,
		DefaultFormat:   row.DefaultFormat,
		DefaultTemplate: row.DefaultTemplate,
		Overrides:       overrides,
	}, nil
}

// modelNotifyRulesToManifest converts enabled store rules to notify rule manifests.
func modelNotifyRulesToManifest(rules []model.NotifyRule) []notify.Rule {
	out := make([]notify.Rule, 0, len(rules))
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		var cond string
		if r.Condition != "" {
			cond = r.Condition
		}
		var params notify.RuleParams
		if r.ParamsJSON != "" {
			if err := sonic.Unmarshal([]byte(r.ParamsJSON), &params); err != nil {
				flog.Warn("skipping notify rule %s: invalid params JSON: %v", r.RuleID, err)
				continue
			}
		}
		out = append(out, notify.Rule{
			ID:     r.RuleID,
			Action: notify.RuleAction(r.Action),
			Match: notify.RuleMatch{
				Event:   r.EventPattern,
				Channel: r.ChannelPattern,
			},
			Condition: cond,
			Priority:  r.Priority,
			Params:    params,
		})
	}
	return out
}
