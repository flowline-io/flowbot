package notify

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types"
)

var handlers map[string]Notifyer

func Register(id string, notifyer Notifyer) {
	if handlers == nil {
		handlers = make(map[string]Notifyer)
	}

	if notifyer == nil {
		flog.Fatal("Register: notifyer is nil")
	}
	if _, dup := handlers[id]; dup {
		flog.Fatal("Register: called twice for notifyer %s", id)
	}
	handlers[id] = notifyer
}

func List() map[string]Notifyer {
	return handlers
}

func ParseTemplate(testString string, templates []string) (types.KV, error) {
	var patterns []string

	regex, err := regexp.Compile(`{(\w+)}`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	for _, v := range templates {
		s := regex.ReplaceAllString(v, `(?P<$1>[a-zA-Z0-9\.\-_]+)`)
		patterns = append(patterns, s)
	}

	result := make(types.KV)
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(testString)
		if len(match) > 0 {
			tmp := make(types.KV)
			for i, name := range re.SubexpNames() {
				if i != 0 && name != "" {
					tmp[name] = match[i]
				}
			}
			result = tmp
			break
		}
	}

	return result, nil
}

func ParseSchema(testString string) (string, error) {
	regex, err := regexp.Compile(`^([a-zA-Z0-9\-_]+)://`)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}
	s := regex.FindString(testString)
	s = strings.TrimSuffix(s, "://")
	return s, nil
}

func Send(text string, message Message) error {
	lines := strings.SplitSeq(text, "\n")
	for v := range lines {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		scheme, err := ParseSchema(v)
		if err != nil {
			flog.Info("[notify] %s parse schema error: %s", scheme, err)
			continue
		}
		if _, ok := handlers[scheme]; !ok {
			continue
		}

		tokens, err := ParseTemplate(v, handlers[scheme].Templates())
		if err != nil {
			flog.Info("[notify] %s parse template error: %s", scheme, err)
			continue
		}
		if err := handlers[scheme].Send(tokens, message); err != nil {
			flog.Info("[notify] %s send message error: %s", scheme, err)
		}
		flog.Info("[notify] %s send message", scheme)
	}

	return nil
}

func ChannelSend(uid types.Uid, name string, message Message) error {
	kv, err := store.Database.ConfigGet(uid, "", fmt.Sprintf("notify:%s", name))
	if err != nil {
		return err
	}
	template, ok := kv.String("value")
	if !ok {
		return errors.New("[notify] template not found")
	}

	return Send(template, message)
}

// GatewaySend is the central notification gateway entry point.
// It renders a notification template and dispatches the message to the specified channels.
// If uid is not zero, it looks up the user's channel configuration from the store.
// Rules (throttle, mute, aggregate) are applied before sending (when rule engine is initialized).
func GatewaySend(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error {
	engine := notifytmpl.GetEngine()
	if engine == nil {
		flog.Warn("[notify] template engine not initialized, skipping notification %s", templateID)
		return nil
	}

	// check if template exists
	if engine.GetTemplateID(templateID) == "" {
		return types.Errorf(types.ErrNotFound, "template %s not found", templateID)
	}

	// evaluate rules before sending
	ruleEngine := notifyrules.GetEngine()
	var errs []error
	for _, channel := range channels {
		// check rules (throttle, mute, aggregate)
		if ruleEngine != nil {
			ruleResult := ruleEngine.Evaluate(ctx, templateID, channel)
			if ruleResult != nil {
				switch ruleResult.Action {
				case config.NotifyRuleActionDrop:
					flog.Info("[notify] message dropped by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
					continue
				case config.NotifyRuleActionMute:
					flog.Info("[notify] message muted by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
					continue
				case config.NotifyRuleActionThrottle:
					if ruleResult.Window != "" && ruleResult.Limit > 0 {
						window, err := time.ParseDuration(ruleResult.Window)
						if err != nil {
							flog.Warn("[notify] invalid throttle window %s: %v", ruleResult.Window, err)
						} else {
							allowed, err := ruleEngine.CheckThrottle(ctx, ruleResult.RuleID, templateID, channel, window, ruleResult.Limit)
							if err != nil {
								flog.Warn("[notify] throttle check error: %v", err)
							} else if !allowed {
								flog.Info("[notify] message throttled by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
								continue
							}
						}
					}
				case config.NotifyRuleActionAggregate:
					if ruleResult.Window != "" {
						window, err := time.ParseDuration(ruleResult.Window)
						if err != nil {
							flog.Warn("[notify] invalid aggregate window %s: %v", ruleResult.Window, err)
						} else {
							// enqueue for later aggregation
							if err := ruleEngine.EnqueueForAggregation(ctx, ruleResult.RuleID, templateID, channel, payload); err != nil {
								flog.Warn("[notify] aggregate enqueue error: %v", err)
							}
							// set timer if first element
							if _, err := ruleEngine.SetAggregateTimer(ctx, ruleResult.RuleID, templateID, channel, window); err != nil {
								flog.Warn("[notify] aggregate timer error: %v", err)
							}
							flog.Info("[notify] message queued for aggregation by rule %s", ruleResult.RuleID)
							continue
						}
					}
				}
			}
		}

		// render template for this channel
		result, err := engine.Render(templateID, channel, payload)
		if err != nil {
			flog.Warn("[notify] failed to render template %s for channel %s: %v", templateID, channel, err)
			errs = append(errs, err)
			continue
		}
		if result == nil {
			continue
		}

		msg := Message{
			Title:    result.Title,
			Body:     result.Body,
			Priority: Normal,
		}

		// extract URL from payload if present
		if url, ok := payload["url"].(string); ok {
			msg.Url = url
		}

		// extract priority from payload if present
		if p, ok := payload["priority"]; ok {
			switch v := p.(type) {
			case Priority:
				msg.Priority = v
			case int:
				msg.Priority = Priority(v)
			case float64:
				msg.Priority = Priority(int(v))
			}
		}

		// look up user's channel configuration
		if !uid.IsZero() {
			kv, err := store.Database.ConfigGet(uid, "", fmt.Sprintf("notify:%s", channel))
			if err != nil {
				flog.Warn("[notify] channel %s not configured for user %s", channel, uid)
				continue
			}
			templateURI, ok := kv.String("value")
			if !ok {
				continue
			}
			if err := Send(templateURI, msg); err != nil {
				flog.Warn("[notify] failed to send to channel %s: %v", channel, err)
				errs = append(errs, err)
			}
		} else {
			flog.Warn("[notify] no user UID for notification %s, channel %s", templateID, channel)
		}
	}

	if len(errs) > 0 {
		return types.Errorf(types.ErrInternal, "notification errors: %v", errs)
	}
	return nil
}
