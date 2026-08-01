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
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func manageChatSession(ctx types.Context, chatKey cache.Key, msgAlt, session string, payload types.MsgPayload, uid types.Uid) (types.MsgPayload, string) {
	if strings.ToLower(msgAlt) == "chat" {
		if session == "" {
			session = types.Id()
			if err := chatagent.CreateSession(ctx.Context(), uid, session); err != nil {
				flog.Error(fmt.Errorf("failed to create chat session: %w", err))
				return types.TextMsg{Text: "Failed to start chat session."}, ""
			}
			if err := cacheStore.Set(ctx.Context(), chatKey, session, cache.TTLSession); err != nil {
				flog.Error(fmt.Errorf("failed to set chat key: %w", err))
				if closeErr := ChatAgentService().CloseSession(ctx.Context(), session); closeErr != nil {
					flog.Error(fmt.Errorf("rollback chat session: %w", closeErr))
				}
				return types.TextMsg{Text: "Failed to start chat session."}, ""
			}
			payload = types.TextMsg{Text: "Chat started"}
			flog.Info("[chat-agent] session started uid=%s session=%s", uid, session)
		} else {
			payload = types.TextMsg{Text: "Chat already started"}
			flog.Debug("[chat-agent] session already active uid=%s session=%s", uid, session)
		}
	}

	if strings.ToLower(msgAlt) == "end" {
		closingSession := session
		if session != "" {
			if err := ChatAgentService().CloseSession(ctx.Context(), session); err != nil {
				flog.Error(fmt.Errorf("failed to close chat session: %w", err))
			} else {
				flog.Info("[chat-agent] session closed uid=%s session=%s", uid, closingSession)
			}
		}
		err := cacheStore.Del(ctx.Context(), chatKey)
		if err != nil {
			flog.Error(fmt.Errorf("failed to delete chat key: %w", err))
		}
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
