package notify

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const notifyConfigKeyPrefix = "notify:"

var (
	handlers   map[string]Notifyer
	handlersMu sync.RWMutex

	// databaseMu serializes notify package reads of store.Database against test writers.
	databaseMu sync.RWMutex
	// recordAsyncWG tracks in-flight recordAsync goroutines for test teardown.
	recordAsyncWG sync.WaitGroup
)

const (
	// PayloadKeySummary is the key in the GatewaySend payload map for the summary text.
	PayloadKeySummary = "summary"
	// defaultKeepRecords is the number of notification records to retain per user.
	defaultKeepRecords = 200
	// systemNotifyUID is used when recording notifications without a user subject (e.g. cron pipelines).
	systemNotifyUID = "system"
)

// Register adds a Notifyer to the global registry.
func Register(id string, notifyer Notifyer) {
	handlersMu.Lock()
	defer handlersMu.Unlock()

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

// Unregister removes a previously registered Notifyer.
// It is a no-op if the id is not found.
// Intended primarily for test teardown.
func Unregister(id string) {
	handlersMu.Lock()
	defer handlersMu.Unlock()

	if handlers == nil {
		return
	}
	delete(handlers, id)
}

// List returns a copy of the registered Notifyer map.
func List() map[string]Notifyer {
	handlersMu.RLock()
	defer handlersMu.RUnlock()

	out := make(map[string]Notifyer, len(handlers))
	maps.Copy(out, handlers)
	return out
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
		// Require a full-string match so shorter templates (e.g. {schema}://{topic})
		// do not win over more specific ones (e.g. {schema}://{host}/{targets}).
		if len(match) > 0 && match[0] == testString {
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
	var lastErr error
	sent := 0
	lines := strings.SplitSeq(text, "\n")
	for v := range lines {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		scheme, err := ParseSchema(v)
		if err != nil {
			lastErr = fmt.Errorf("[notify] parse schema error: %w", err)
			flog.Error(lastErr)
			continue
		}
		if scheme == "" {
			lastErr = fmt.Errorf("[notify] invalid URI: missing protocol scheme")
			flog.Error(lastErr)
			continue
		}
		n, ok := lookupNotifyer(scheme)
		if !ok {
			lastErr = fmt.Errorf("[notify] unknown protocol %q", scheme)
			flog.Error(lastErr)
			continue
		}

		tokens, err := ParseTemplate(v, n.Templates())
		if err != nil {
			lastErr = fmt.Errorf("[notify] %s parse template error: %w", scheme, err)
			flog.Error(lastErr)
			continue
		}
		if err := n.Send(tokens, message); err != nil {
			lastErr = fmt.Errorf("[notify] %s send message error: %w", scheme, err)
			flog.Error(lastErr)
			continue
		}
		sent++
		flog.Info("[notify] %s send message", scheme)
	}

	if sent == 0 {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("[notify] no notification sent")
	}
	return nil
}

// SendToProtocol dispatches a message using an explicit notify protocol.
// Unlike Send, the URI scheme (for example http/https used by ntfy endpoints) is not
// used for provider lookup — protocol selects the Notifyer.
func SendToProtocol(protocol, uri string, message Message) error {
	protocol = strings.TrimSpace(protocol)
	if protocol == "" {
		return fmt.Errorf("[notify] protocol is required")
	}
	n, ok := lookupNotifyer(protocol)
	if !ok {
		return fmt.Errorf("[notify] unknown protocol %q", protocol)
	}
	text := strings.TrimSpace(uri)
	if text == "" {
		return fmt.Errorf("[notify] uri is required")
	}
	if !strings.Contains(text, "://") {
		text = protocol + "://" + text
	}
	tokens, err := ParseTemplate(text, n.Templates())
	if err != nil {
		return fmt.Errorf("[notify] %s parse template error: %w", protocol, err)
	}
	if len(tokens) == 0 {
		return fmt.Errorf("[notify] %s: URI does not match any template", protocol)
	}
	if err := n.Send(tokens, message); err != nil {
		return fmt.Errorf("[notify] %s send message error: %w", protocol, err)
	}
	flog.Info("[notify] %s send message", protocol)
	return nil
}

// lookupNotifyer returns the Notifyer registered for protocol, if any.
func lookupNotifyer(protocol string) (Notifyer, bool) {
	handlersMu.RLock()
	defer handlersMu.RUnlock()
	n, ok := handlers[protocol]
	return n, ok
}

// GatewaySend is the central notification gateway entry point.
// It renders a notification template and dispatches the message to the specified channels.
// Channels are resolved from the global NotifyChannel registry first; when a UID is set,
// per-user notify:<channel> config is used as a fallback.
// Rules (throttle, mute, aggregate) are applied before sending (when rule engine is initialized).
// When channels include a successful inapp delivery, other channels are deferred and flushed
// by the escalation worker (presence / unread timeout), re-evaluating rules at flush time.
func GatewaySend(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error {
	engine := notifytmpl.GetEngine()
	if engine == nil {
		flog.Warn("[notify] template engine not initialized, skipping notification %s", templateID)
		return nil
	}

	if engine.GetTemplateID(templateID) == "" {
		return types.Errorf(types.ErrNotFound, "template %s not found", templateID)
	}

	payload = ensureCorrelationPayload(payload)
	correlationID, ok := payload[PayloadKeyCorrelationID].(string)
	if !ok {
		correlationID = ""
	}

	var summary string
	if s, ok := payload[PayloadKeySummary].(string); ok {
		summary = s
	}

	inappName := ""
	externals := make([]string, 0, len(channels))
	for _, ch := range channels {
		if ch == ChannelInapp {
			inappName = ch
			continue
		}
		externals = append(externals, ch)
	}

	if inappName == "" {
		return gatewaySendImmediate(ctx, uid, templateID, channels, summary, correlationID, payload)
	}

	inappOK, err := sendOneChannel(ctx, uid, templateID, inappName, summary, correlationID, payload)
	if err != nil {
		// still try immediate externals when inapp failed for non-rule reasons
		extErr := gatewaySendImmediate(ctx, uid, templateID, externals, summary, correlationID, payload)
		return errors.Join(err, extErr)
	}
	if !inappOK {
		return gatewaySendImmediate(ctx, uid, templateID, externals, summary, correlationID, payload)
	}

	escalateAt := computeEscalateAt(uid, payload)
	var errs []error
	for _, channel := range externals {
		if err := enqueueDeferredChannel(ctx, uid, templateID, channel, summary, correlationID, payload, escalateAt); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return types.Errorf(types.ErrInternal, "notification errors: %v", errs)
	}
	return nil
}

func gatewaySendImmediate(ctx context.Context, uid types.Uid, templateID string, channels []string, summary, correlationID string, payload map[string]any) error {
	var errs []error
	for _, channel := range channels {
		if _, err := sendOneChannel(ctx, uid, templateID, channel, summary, correlationID, payload); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return types.Errorf(types.ErrInternal, "notification errors: %v", errs)
	}
	return nil
}

// sendOneChannel evaluates rules, dispatches, and records. ok is true only on success delivery.
func sendOneChannel(ctx context.Context, uid types.Uid, templateID, channel, summary, correlationID string, payload map[string]any) (ok bool, err error) {
	eval, err := evaluateAndRenderNotification(ctx, templateID, channel, payload)
	if err != nil {
		ruleID := ""
		if eval != nil {
			ruleID = eval.ruleID
		}
		recordAsyncParams(uid, channel, templateID, summary, "failed", err.Error(), ruleID, correlationID, payload, nil)
		return false, err
	}
	if eval == nil {
		return false, nil
	}
	if eval.action != "" {
		recordAsyncParams(uid, channel, templateID, summary, eval.action, "", eval.ruleID, correlationID, payload, nil)
		return false, nil
	}
	if eval.renderResult == nil {
		return false, nil
	}
	msg := buildNotifyMessage(eval.renderResult, payload)
	if err := dispatchChannel(ctx, uid, channel, msg); err != nil {
		recordAsyncParams(uid, channel, templateID, summary, "failed", err.Error(), eval.ruleID, correlationID, payload, nil)
		return false, err
	}
	recordAsyncParams(uid, channel, templateID, summary, "success", "", eval.ruleID, correlationID, payload, nil)
	return true, nil
}

func enqueueDeferredChannel(ctx context.Context, uid types.Uid, templateID, channel, summary, correlationID string, payload map[string]any, escalateAt time.Time) error {
	eval, err := evaluateForDeferredEnqueue(ctx, templateID, channel, payload)
	if err != nil {
		ruleID := ""
		if eval != nil {
			ruleID = eval.ruleID
		}
		recordAsyncParams(uid, channel, templateID, summary, "failed", err.Error(), ruleID, correlationID, payload, nil)
		return err
	}
	if eval == nil {
		return nil
	}
	if eval.action != "" {
		recordAsyncParams(uid, channel, templateID, summary, eval.action, "", eval.ruleID, correlationID, payload, nil)
		return nil
	}
	if eval.renderResult == nil {
		return nil
	}
	ea := escalateAt
	recordAsyncParams(uid, channel, templateID, summary, "deferred", "", eval.ruleID, correlationID, payload, &ea)
	return nil
}

func ensureCorrelationPayload(payload map[string]any) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	} else {
		cp := make(map[string]any, len(payload)+1)
		maps.Copy(cp, payload)
		payload = cp
	}
	if id, ok := payload[PayloadKeyCorrelationID].(string); !ok || strings.TrimSpace(id) == "" {
		payload[PayloadKeyCorrelationID] = utils.NewUUID()
	}
	return payload
}

func computeEscalateAt(uid types.Uid, payload map[string]any) time.Time {
	delay := EscalateAfter()
	if raw, ok := payload[PayloadKeyEscalateAfter].(string); ok && strings.TrimSpace(raw) != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(raw)); err == nil && d >= 0 {
			delay = d
		}
	}
	if IsPresent(uid.String()) {
		return time.Now().Add(delay)
	}
	return time.Now()
}

// evalResult holds the result of notification evaluation, including rule actions.
type evalResult struct {
	renderResult *notifytmpl.RenderResult
	action       string // "dropped", "muted", "throttled", "aggregated", or ""
	ruleID       string // matched rule id when a rule applied or matched
}

// evaluateAndRenderNotification applies rules and renders the template for a single channel.
// Returns nil result and nil error when the message should be skipped due to rules.
func evaluateAndRenderNotification(ctx context.Context, templateID, channel string, payload map[string]any) (*evalResult, error) {
	return evaluateAndRender(ctx, templateID, channel, payload, true)
}

// evaluateForDeferredEnqueue applies drop/mute/aggregate at enqueue time but does not
// consume throttle quota (throttle is enforced at flush).
func evaluateForDeferredEnqueue(ctx context.Context, templateID, channel string, payload map[string]any) (*evalResult, error) {
	return evaluateAndRender(ctx, templateID, channel, payload, false)
}

func evaluateAndRender(ctx context.Context, templateID, channel string, payload map[string]any, applyThrottle bool) (*evalResult, error) {
	var matchedRuleID string
	ruleEngine := notifyrules.GetEngine()
	if ruleEngine != nil {
		ruleResult := ruleEngine.Evaluate(ctx, templateID, channel)
		if ruleResult != nil {
			matchedRuleID = ruleResult.RuleID
			switch ruleResult.Action {
			case RuleActionDrop:
				flog.Info("[notify] message dropped by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
				return &evalResult{action: "dropped", ruleID: matchedRuleID}, nil
			case RuleActionMute:
				flog.Info("[notify] message muted by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
				return &evalResult{action: "muted", ruleID: matchedRuleID}, nil
			case RuleActionThrottle:
				if applyThrottle {
					if handleThrottleRule(ctx, ruleResult, templateID, channel) {
						return &evalResult{action: "throttled", ruleID: matchedRuleID}, nil
					}
				}
				// Enqueue path: throttle match does not block deferred creation.
			case RuleActionAggregate:
				if handleAggregateRule(ctx, ruleResult, templateID, channel, payload) {
					return &evalResult{action: "aggregated", ruleID: matchedRuleID}, nil
				}
			}
		}
	}

	engine := notifytmpl.GetEngine()
	result, err := engine.Render(templateID, channel, payload)
	if err != nil {
		flog.Warn("[notify] failed to render template %s for channel %s: %v", templateID, channel, err)
		return &evalResult{ruleID: matchedRuleID}, err
	}
	return &evalResult{renderResult: result, ruleID: matchedRuleID}, nil
}

// handleThrottleRule checks throttle limits for a rule and returns true if the message should be skipped.
func handleThrottleRule(ctx context.Context, ruleResult *notifyrules.EvalResult, templateID, channel string) bool {
	if ruleResult.Window == "" || ruleResult.Limit <= 0 {
		return false
	}
	window, err := time.ParseDuration(ruleResult.Window)
	if err != nil {
		flog.Warn("[notify] invalid throttle window %s: %v", ruleResult.Window, err)
		return false
	}
	engine := notifyrules.GetEngine()
	allowed, err := engine.CheckThrottle(ctx, ruleResult.RuleID, templateID, channel, window, ruleResult.Limit)
	if err != nil {
		flog.Warn("[notify] throttle check error: %v", err)
		return false
	}
	if !allowed {
		flog.Info("[notify] message throttled by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
		return true
	}
	return false
}

// handleAggregateRule enqueues a message for aggregation and returns true if the message was handled.
func handleAggregateRule(ctx context.Context, ruleResult *notifyrules.EvalResult, templateID, channel string, payload map[string]any) bool {
	if ruleResult.Window == "" {
		return false
	}
	window, err := time.ParseDuration(ruleResult.Window)
	if err != nil {
		flog.Warn("[notify] invalid aggregate window %s: %v", ruleResult.Window, err)
		return false
	}
	engine := notifyrules.GetEngine()
	if err := engine.EnqueueForAggregation(ctx, ruleResult.RuleID, templateID, channel, payload); err != nil {
		flog.Warn("[notify] aggregate enqueue error: %v", err)
	}
	if _, err := engine.SetAggregateTimer(ctx, ruleResult.RuleID, templateID, channel, window); err != nil {
		flog.Warn("[notify] aggregate timer error: %v", err)
	}
	flog.Info("[notify] message queued for aggregation by rule %s", ruleResult.RuleID)
	return true
}

// buildNotifyMessage constructs a Message from a rendered template result and payload extras.
func buildNotifyMessage(result *notifytmpl.RenderResult, payload map[string]any) Message {
	msg := Message{
		Title:    result.Title,
		Body:     result.Body,
		Priority: Normal,
	}

	if title, ok := payload[PayloadKeyTitle].(string); ok && strings.TrimSpace(title) != "" {
		msg.Title = title
	}
	if url, ok := payload[PayloadKeyURL].(string); ok {
		msg.Url = url
	} else if url, ok := payload["url"].(string); ok {
		msg.Url = url
	}

	if p, ok := payload["priority"]; ok {
		switch v := p.(type) {
		case Priority:
			msg.Priority = v
		case int:
			if pri, ok := utils.IntToInt32(v); ok {
				msg.Priority = Priority(pri)
			}
		case float64:
			if pri, ok := utils.IntToInt32(int(v)); ok {
				msg.Priority = Priority(pri)
			}
		}
	}

	return msg
}

// loadDatabase returns the current store.Database under a read lock.
func loadDatabase() store.Adapter {
	databaseMu.RLock()
	defer databaseMu.RUnlock()
	return store.Database
}

// UserNotifyChannels returns channel names configured for the user under notify:<channel> keys.
func UserNotifyChannels(ctx context.Context, uid types.Uid) ([]string, error) {
	if loadDatabase() == nil {
		return nil, nil
	}
	items, err := store.ModuleDataStoreFromDB().ListConfigByPrefix(ctx, uid, "", notifyConfigKeyPrefix)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		keys = append(keys, item.Key)
	}
	return channelsFromNotifyConfigKeys(keys), nil
}

// channelsFromNotifyConfigKeys extracts channel names from notify:<channel> config keys.
func channelsFromNotifyConfigKeys(keys []string) []string {
	channels := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		ch := strings.TrimPrefix(key, notifyConfigKeyPrefix)
		if ch == "" || ch == key {
			continue
		}
		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		channels = append(channels, ch)
	}
	return channels
}

// dispatchChannel sends a rendered message to a named channel.
// It prefers the global NotifyChannel registry (settings UI), then falls back to
// per-user notify:<channel> config when a UID is present.
func dispatchChannel(ctx context.Context, uid types.Uid, channel string, msg Message) error {
	if err := sendGlobalChannel(ctx, channel, msg); err != nil {
		if !errors.Is(err, types.ErrNotFound) {
			return err
		}
	} else {
		return nil
	}
	return sendToUserChannel(ctx, uid, channel, msg)
}

// sendGlobalChannel looks up a channel by name in the global notify_channels table and sends.
func sendGlobalChannel(ctx context.Context, channel string, msg Message) error {
	if loadDatabase() == nil {
		return types.ErrNotFound
	}
	ch, err := store.NotifyConfigStoreFromDB().GetNotifyChannelByNameRaw(ctx, channel)
	if err != nil {
		return err
	}
	if !ch.Enabled {
		return types.Errorf(types.ErrInvalidArgument, "channel %s is disabled", channel)
	}
	if err := SendToProtocol(ch.Protocol, ch.URI, msg); err != nil {
		flog.Warn("[notify] failed to send to global channel %s: %v", channel, err)
		return err
	}
	return nil
}

// sendToUserChannel looks up the user's channel configuration and sends the message.
func sendToUserChannel(ctx context.Context, uid types.Uid, channel string, msg Message) error {
	if uid.IsZero() {
		return types.Errorf(types.ErrNotFound, "channel %s not found", channel)
	}

	if loadDatabase() == nil {
		return types.Errorf(types.ErrNotFound, "channel %s not found", channel)
	}
	kv, err := store.ModuleDataStoreFromDB().ConfigGet(ctx, uid, "", notifyConfigKeyPrefix+channel)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.Errorf(types.ErrNotFound, "channel %s not configured for user", channel)
		}
		flog.Warn("[notify] failed to load channel %s for user %s: %v", channel, uid, err)
		return err
	}
	templateURI, ok := kv.String("value")
	if !ok || templateURI == "" {
		return types.Errorf(types.ErrNotFound, "channel %s not configured for user", channel)
	}
	if err := Send(templateURI, msg); err != nil {
		flog.Warn("[notify] failed to send to channel %s: %v", channel, err)
		return err
	}
	return nil
}

// GetNotifyStore returns the NotifyStore from the global database adapter,
// or nil if the store is not available.
func GetNotifyStore() *store.NotifyStore {
	db := loadDatabase()
	if db == nil || db.GetClient() == nil {
		return nil
	}
	return store.NotifyStoreFromDB()
}

// WaitForRecordAsyncForTest blocks until all in-flight recordAsync goroutines finish.
func WaitForRecordAsyncForTest() {
	recordAsyncWG.Wait()
}

// recordAsync writes a notification delivery record in a goroutine with a 2s timeout.
// It also triggers deferred rolling window cleanup (best-effort).
func recordAsync(uid types.Uid, channel, templateID, summary, status, errMsg, ruleID string, payload map[string]any) {
	var correlationID string
	if payload != nil {
		if id, ok := payload[PayloadKeyCorrelationID].(string); ok {
			correlationID = id
		}
	}
	recordAsyncParams(uid, channel, templateID, summary, status, errMsg, ruleID, correlationID, payload, nil)
}

func recordAsyncParams(uid types.Uid, channel, templateID, summary, status, errMsg, ruleID, correlationID string, payload map[string]any, escalateAt *time.Time) {
	payloadCopy := make(map[string]any, len(payload))
	maps.Copy(payloadCopy, payload)
	recordAsyncWG.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		ns := GetNotifyStore()
		if ns == nil {
			return
		}
		recordUID := uid.String()
		if uid.IsZero() {
			recordUID = systemNotifyUID
		}
		if _, err := ns.RecordParams(ctx, store.RecordParams{
			UID:           recordUID,
			Channel:       channel,
			TemplateID:    templateID,
			Summary:       summary,
			Status:        status,
			ErrorMsg:      errMsg,
			RuleID:        ruleID,
			CorrelationID: correlationID,
			Payload:       payloadCopy,
			EscalateAt:    escalateAt,
		}); err != nil {
			flog.Warn("[notify] failed to record notification: %v", err)
			return
		}
		if err := ns.DeleteOldest(ctx, recordUID, defaultKeepRecords); err != nil {
			flog.Warn("[notify] failed to cleanup old notifications: %v", err)
		}
	})
}
