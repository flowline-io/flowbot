package server

import (
	"fmt"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func chatScope(msg protocol.MessageEventData) string {
	if msg.ThreadId != "" {
		return msg.ThreadId
	}
	if msg.TopicId != "" {
		return msg.TopicId
	}
	return "default"
}

func isThreadedChatCommand(msgAlt string) bool {
	switch strings.ToLower(strings.TrimSpace(msgAlt)) {
	case "chat", "end", "plan", "proceed":
		return true
	default:
		return false
	}
}

func chatCommandThreadID(msg protocol.MessageEventData, msgAlt string) string {
	if !isThreadedChatCommand(msgAlt) {
		return ""
	}
	return msg.ThreadId
}

func enabledChatKey(uid types.Uid) cache.Key {
	return cache.NewKey("chat", "enabled", uid.String())
}

func threadChatKey(uid types.Uid, scope string) cache.Key {
	return cache.NewKey("chat", "session", uid.String()+":"+scope)
}

func threadsSetKey(uid types.Uid) cache.Key {
	return cache.NewKey("chat", "threads", uid.String())
}

func isChatEnabled(ctx types.Context, uid types.Uid) bool {
	ok, err := cacheStore.Exists(ctx.Context(), enabledChatKey(uid))
	if err != nil {
		flog.Error(fmt.Errorf("chat enabled check: %w", err))
		return false
	}
	return ok
}

func loadThreadSessionID(ctx types.Context, uid types.Uid, scope string) string {
	sessionID, ok, err := cacheStore.Get(ctx.Context(), threadChatKey(uid, scope))
	if err != nil {
		flog.Error(fmt.Errorf("load thread session uid=%s scope=%s: %w", uid, scope, err))
		return ""
	}
	if !ok {
		return ""
	}
	flog.Debug("[chat-agent] session cache hit uid=%s scope=%s session=%s", uid, scope, sessionID)
	return sessionID
}

func bindThreadSession(ctx types.Context, uid types.Uid, scope, sessionID string) error {
	if err := cacheStore.Set(ctx.Context(), threadChatKey(uid, scope), sessionID, cache.TTLSession); err != nil {
		return fmt.Errorf("set thread session key uid=%s scope=%s: %w", uid, scope, err)
	}
	if err := cacheStore.Set(ctx.Context(), enabledChatKey(uid), "1", cache.TTLSession); err != nil {
		return fmt.Errorf("set chat enabled uid=%s: %w", uid, err)
	}
	if _, err := cacheStore.Add(ctx.Context(), threadsSetKey(uid), cache.TTLSession, scope); err != nil {
		return fmt.Errorf("add chat thread scope uid=%s scope=%s: %w", uid, scope, err)
	}
	return nil
}

func createAndBindThreadSession(ctx types.Context, uid types.Uid, scope string) (string, error) {
	sessionID := types.Id()
	if err := chatagent.CreateSession(ctx.Context(), uid, sessionID); err != nil {
		return "", fmt.Errorf("create chat session uid=%s: %w", uid, err)
	}
	if err := bindThreadSession(ctx, uid, scope, sessionID); err != nil {
		if closeErr := ChatAgentService().CloseSession(ctx.Context(), sessionID); closeErr != nil {
			flog.Error(fmt.Errorf("rollback chat session: %w", closeErr))
		}
		return "", err
	}
	flog.Info("[chat-agent] session started uid=%s scope=%s session=%s", uid, scope, sessionID)
	return sessionID, nil
}

func refreshThreadSessionCache(ctx types.Context, uid types.Uid, scope, sessionID string) {
	if sessionID == "" {
		return
	}
	if err := touchThreadSessionCache(ctx, uid, scope); err != nil {
		flog.Error(fmt.Errorf("refresh chat session cache: %w", err))
	}
}

// touchThreadSessionCache extends TTLs on existing session keys without rewriting values.
func touchThreadSessionCache(ctx types.Context, uid types.Uid, scope string) error {
	ttl := cache.TTLSession
	if err := cacheStore.Expire(ctx.Context(), threadChatKey(uid, scope), ttl); err != nil {
		return fmt.Errorf("expire thread session key uid=%s scope=%s: %w", uid, scope, err)
	}
	if err := cacheStore.Expire(ctx.Context(), enabledChatKey(uid), ttl); err != nil {
		return fmt.Errorf("expire chat enabled uid=%s: %w", uid, err)
	}
	if err := cacheStore.Expire(ctx.Context(), threadsSetKey(uid), ttl); err != nil {
		return fmt.Errorf("expire chat threads set uid=%s: %w", uid, err)
	}
	return nil
}

func ensureThreadSession(ctx types.Context, uid types.Uid, scope, sessionID, msgAlt string) string {
	if sessionID != "" || chatagent.IsChatControlCommand(msgAlt) {
		return sessionID
	}
	if !isChatEnabled(ctx, uid) {
		return ""
	}
	created, err := createAndBindThreadSession(ctx, uid, scope)
	if err != nil {
		flog.Error(fmt.Errorf("auto-create thread session: %w", err))
		return ""
	}
	return created
}

func endAllThreadSessions(ctx types.Context, uid types.Uid) {
	scopes, err := cacheStore.Members(ctx.Context(), threadsSetKey(uid))
	if err != nil {
		flog.Error(fmt.Errorf("list chat thread scopes: %w", err))
	}
	for _, scope := range scopes {
		sessionID := loadThreadSessionID(ctx, uid, scope)
		if sessionID != "" {
			if closeErr := ChatAgentService().CloseSession(ctx.Context(), sessionID); closeErr != nil {
				flog.Error(fmt.Errorf("failed to close chat session: %w", closeErr))
			} else {
				flog.Info("[chat-agent] session closed uid=%s scope=%s session=%s", uid, scope, sessionID)
			}
		}
		if delErr := cacheStore.Del(ctx.Context(), threadChatKey(uid, scope)); delErr != nil {
			flog.Error(fmt.Errorf("failed to delete chat session key: %w", delErr))
		}
	}
	if clearErr := cacheStore.Clear(ctx.Context(), threadsSetKey(uid)); clearErr != nil {
		flog.Error(fmt.Errorf("failed to clear chat threads set: %w", clearErr))
	}
	if delErr := cacheStore.Del(ctx.Context(), enabledChatKey(uid)); delErr != nil {
		flog.Error(fmt.Errorf("failed to delete chat enabled key: %w", delErr))
	}
}

func manageChatSession(ctx types.Context, uid types.Uid, scope, msgAlt, session string, payload types.MsgPayload) (types.MsgPayload, string) {
	if strings.ToLower(msgAlt) == "chat" {
		if session == "" {
			created, err := createAndBindThreadSession(ctx, uid, scope)
			if err != nil {
				flog.Error(fmt.Errorf("failed to create chat session: %w", err))
				return types.TextMsg{Text: "Failed to start chat session."}, ""
			}
			payload = types.TextMsg{Text: "Chat started"}
			session = created
		} else {
			if err := cacheStore.Set(ctx.Context(), enabledChatKey(uid), "1", cache.TTLSession); err != nil {
				flog.Error(fmt.Errorf("failed to set chat enabled: %w", err))
			}
			payload = types.TextMsg{Text: "Chat already started"}
			flog.Debug("[chat-agent] session already active uid=%s scope=%s session=%s", uid, scope, session)
		}
	}

	if strings.ToLower(msgAlt) == "end" {
		endAllThreadSessions(ctx, uid)
		payload = types.TextMsg{Text: "Chat ended"}
		session = ""
	}

	if payload, handled := handleChatPlanCommands(ctx, msgAlt, session, uid); handled {
		return payload, session
	}

	return payload, session
}

func handleChatPlanCommands(ctx types.Context, msgAlt, session string, uid types.Uid) (types.MsgPayload, bool) {
	if session == "" {
		return nil, false
	}
	switch strings.ToLower(msgAlt) {
	case "plan":
		if err := ChatAgentService().SetSessionModeAndNotify(ctx.Context(), session, chatagent.ModePlan); err != nil {
			flog.Error(fmt.Errorf("failed to enable plan mode: %w", err))
			return types.TextMsg{Text: "Failed to enable plan mode."}, true
		}
		flog.Info("[chat-agent] plan mode enabled uid=%s session=%s", uid, session)
		return types.TextMsg{Text: "Plan mode on. The agent will research and propose a plan without making changes."}, true
	case "proceed":
		if err := ChatAgentService().SetSessionModeAndNotify(ctx.Context(), session, chatagent.ModeNormal); err != nil {
			flog.Error(fmt.Errorf("failed to disable plan mode: %w", err))
			return types.TextMsg{Text: "Failed to disable plan mode."}, true
		}
		flog.Info("[chat-agent] plan mode disabled uid=%s session=%s", uid, session)
		return types.TextMsg{Text: "Plan mode off. The agent can now make changes. Re-send your request to execute."}, true
	default:
		return nil, false
	}
}

func buildHelpMessage(msgAlt string, payload types.MsgPayload) types.MsgPayload {
	if strings.ToLower(msgAlt) != "help" {
		return payload
	}

	byModule := make(map[string][]command.Rule)
	for name, handle := range module.List() {
		for _, item := range handle.Rules() {
			if v, ok := item.([]command.Rule); ok {
				byModule[name] = append(byModule[name], v...)
			}
		}
	}
	raw := formatGroupedHelpMarkdown(byModule)
	if raw == "" {
		return payload
	}
	return types.MarkdownMsg{
		Title: "Help",
		Raw:   raw,
	}
}

func formatGroupedHelpMarkdown(byModule map[string][]command.Rule) string {
	names := make([]string, 0, len(byModule))
	for name, rules := range byModule {
		if len(rules) > 0 {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	slices.Sort(names)

	sections := make([]string, 0, len(names))
	for _, name := range names {
		rules := slices.Clone(byModule[name])
		slices.SortFunc(rules, func(a, c command.Rule) int {
			return strings.Compare(a.Define, c.Define)
		})
		lines := make([]string, 0, len(rules)+1)
		lines = append(lines, "*"+name+"*")
		for _, rule := range rules {
			lines = append(lines, rule.FormatHelpLine())
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func dispatchToModules(ctx types.Context, msgAlt string) types.MsgPayload {
	var payload types.MsgPayload
	for name, handle := range module.List() {
		if !handle.IsReady() {
			flog.Info("module %s unavailable", name)
			continue
		}

		if payload == nil {
			in := msgAlt
			if strings.HasPrefix(in, "/") {
				in = strings.Replace(in, "/", "", 1)
			}
			var err error
			payload, err = handle.Command(ctx, in)
			if err != nil {
				flog.Warn("topic[%s]: failed to run bot: %v", name, err)
			}

			if payload != nil {
				stats.ModuleRunTotalCounter(stats.CommandRuleset).Inc()
			}
		}

		if payload != nil {
			break
		}
	}
	return payload
}
