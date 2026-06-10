package server

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func manageChatSession(ctx types.Context, chatKey cache.Key, msgAlt string, session string, payload types.MsgPayload, uid types.Uid) (types.MsgPayload, string) {
	if strings.ToLower(msgAlt) == "chat" {
		if session == "" {
			session = types.Id()
			if err := chatagent.CreateSession(ctx.Context(), uid, session); err != nil {
				flog.Error(fmt.Errorf("failed to create chat session: %w", err))
				return types.TextMsg{Text: "Failed to start chat session."}, ""
			}
			if err := cacheStore.Set(ctx.Context(), chatKey, session, cache.TTLSession); err != nil {
				flog.Error(fmt.Errorf("failed to set chat key: %w", err))
				if closeErr := chatagent.CloseSession(ctx.Context(), session); closeErr != nil {
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
			if err := chatagent.CloseSession(ctx.Context(), session); err != nil {
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
	return payload, session
}

func buildHelpMessage(msgAlt string, payload types.MsgPayload) types.MsgPayload {
	if strings.ToLower(msgAlt) == "help" {
		m := make(types.KV)
		for name, handle := range module.List() {
			for _, item := range handle.Rules() {
				if v, ok := item.([]command.Rule); ok {
					for _, rule := range v {
						m[fmt.Sprintf("[%s] /%s", name, rule.Define)] = rule.Help
					}
				}
			}
		}
		if len(m) > 0 {
			payload = types.InfoMsg{
				Title: "Help",
				Model: m,
			}
		}
	}
	return payload
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
