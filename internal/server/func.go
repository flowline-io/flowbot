package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/dropbox"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	openaiProvider "github.com/flowline-io/flowbot/pkg/providers/openai"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/redis/go-redis/v9"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"gorm.io/gorm"
)

func newProvider(category string) providers.OAuthProvider {
	var provider providers.OAuthProvider

	switch category {
	case pocket.ID:
		key, _ := providers.GetConfig(pocket.ID, pocket.ClientIdKey)
		provider = pocket.NewPocket(key.String(), "", "", "")
	case github.ID:
		id, _ := providers.GetConfig(github.ID, github.ClientIdKey)
		secret, _ := providers.GetConfig(github.ID, github.ClientSecretKey)
		provider = github.NewGithub(id.String(), secret.String(), "", "")
	case dropbox.ID:
		provider = dropbox.NewDropbox("", "", "", "")
	default:
		return nil
	}

	return provider
}

func directIncomingMessage(caller *platforms.Caller, e protocol.Event) {
	// check topic owner user

	msg, ok := e.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	uid, err := registerPlatformUser(msg)
	if err != nil {
		flog.Error(err)
		return
	}

	topic, err := registerPlatformChannel(msg)
	if err != nil {
		flog.Error(err)
		return
	}

	ctx := types.Context{
		Id:     e.Id,
		AsUser: uid,
		//Original:  msg.Original,
		//RcptTo:    msg.RcptTo,
		//AsUser:    uid,
		//AuthLvl:   msg.AuthLvl,
		//MetaWhat:  msg.MetaWhat,
		//Timestamp: msg.Timestamp,
	}
	ctx.SetTimeout(10 * time.Minute)

	// platform
	findPlatform, err := store.Database.GetPlatformByName(msg.Self.Platform)
	if err != nil {
		flog.Error(err)
		return
	}
	platformId := findPlatform.ID

	// check
	findMessage, err := store.Database.GetMessageByPlatform(platformId, msg.MessageId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		flog.Error(err)
		return
	}
	if findMessage != nil {
		flog.Info("message %s %s already exists", msg.Self.Platform, msg.MessageId)
		return
	}
	err = store.Database.CreateMessage(model.Message{
		Flag:          types.Id(),
		PlatformID:    platformId,
		PlatformMsgID: msg.MessageId,
		Topic:         topic,
		Content:       model.JSON{"text": msg.AltMessage},
		State:         model.MessageCreated,
	})
	if err != nil {
		flog.Error(err)
		return
	}

	// behavior
	bots.Behavior(uid, bots.MessageBotIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	// help command
	if strings.ToLower(msg.AltMessage) == "help" || strings.ToLower(msg.AltMessage) == "h" {
		m := make(types.KV)
		for name, handle := range bots.List() {
			for _, item := range handle.Rules() {
				switch v := item.(type) {
				case []command.Rule:
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

	for name, handle := range bots.List() {
		if !handle.IsReady() {
			flog.Info("bot %s unavailable", name)
			continue
		}

		// command
		if payload == nil {
			in := msg.AltMessage
			// check "/" prefix
			if strings.HasPrefix(in, "/") {
				in = strings.Replace(in, "/", "", 1)
			}
			payload, err = handle.Command(ctx, in)
			if err != nil {
				flog.Warn("topic[%s]: failed to run bot: %v", name, err)
			}

			// stats
			if payload != nil {
				stats.BotRunTotalCounter(stats.CommandRuleset).Inc()
			}
		}

		if payload != nil {
			break
		}
	}

	// llm chat
	if payload == nil {
		tokenVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.TokenKey)
		baseUrlVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.BaseUrlKey)
		modelVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.ModelKey)
		languageVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.LanguageKey)

		llm, err := openai.New(
			openai.WithToken(tokenVal.String()),
			openai.WithBaseURL(baseUrlVal.String()),
			openai.WithModel(modelVal.String()),
		)
		if err != nil {
			flog.Error(err)
			return
		}

		// Sending initial message to the model, with a list of available tools.
		messageHistory := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, msg.AltMessage),
		}

		resp, err := llm.GenerateContent(ctx.Context(), messageHistory, llms.WithTools(bots.AvailableTools()))
		if err != nil {
			flog.Error(err)
			return
		}

		messageHistory = updateMessageHistory(messageHistory, resp)

		// Execute tool calls requested by the model
		messageHistory, err = executeToolCalls(ctx, llm, messageHistory, resp)
		if err != nil {
			flog.Error(err)
			return
		}
		messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Please answer in %s", languageVal.String())))

		// Send query to the model again, this time with a history containing its
		// request to invoke a tool and our response to the tool call.
		resp, err = llm.GenerateContent(ctx.Context(), messageHistory)
		if err != nil {
			flog.Error(err)
			return
		}

		if resp != nil && len(resp.Choices) > 0 {
			payload = types.TextMsg{Text: resp.Choices[0].Content}
		}
	}

	if payload == nil {
		return
	}

	flog.Debug("incoming send message action topic %v payload %+v", msg.MessageId, payload)
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(payload),
		},
	})
	flog.Info("[event] %+v  response: %+v", msg, resp)
}

func groupIncomingMessage(caller *platforms.Caller, e protocol.Event) {
	msg, ok := e.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	uid, err := registerPlatformUser(msg)
	if err != nil {
		flog.Error(err)
		return
	}

	topic, err := registerPlatformChannel(msg)
	if err != nil {
		flog.Error(err)
		return
	}

	ctx := types.Context{
		Id:     e.Id,
		Topic:  topic,
		AsUser: uid,
	}
	_, _ = fmt.Println(ctx)

	// behavior
	bots.Behavior(uid, bots.MessageGroupIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	for name, handle := range bots.List() {
		if !handle.IsReady() {
			flog.Info("bot %s unavailable", name)
			continue
		}
	}

	flog.Debug("incoming send message action topic %v payload %+v", msg.MessageId, payload)
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(payload),
		},
	})
	flog.Info("event: %+v  response: %+v", msg, resp)
}

func notifyAll(message string) {
	// send message
	users, err := store.Database.GetUsers()
	if err != nil {
		flog.Error(fmt.Errorf("notify error %w", err))
		return
	}
	for _, item := range users {
		err = event.SendMessage(context.Background(), item.Flag, "", types.TextMsg{Text: message})
		if err != nil {
			flog.Error(fmt.Errorf("notify error %w", err))
			continue
		}
	}
}

func onlineStatus(msg protocol.Event) {
	med, ok := msg.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("online:%s", med.UserId)
	_, err := cache.DB.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		cache.DB.Set(ctx, key, time.Now().Unix(), 30*time.Minute)
	} else if err != nil {
		return
	} else {
		cache.DB.Expire(ctx, key, 30*time.Minute)
	}
}

func agentAction(uid types.Uid, data types.AgentData) (interface{}, error) {
	switch data.Action {
	case types.Collect:
		id, ok := data.Content.String("id")
		if !ok {
			return nil, errors.New("error collect id")
		}

		for name, handle := range bots.List() {
			if !handle.IsReady() {
				flog.Info("bot %s unavailable", name)
				continue
			}

			ctx := types.Context{
				Platform:     "",
				Topic:        "",
				AsUser:       uid,
				CollectId:    id,
				AgentVersion: data.Version,
			}
			content, _ := data.Content.Map("content")
			payload, err := handle.Collect(ctx, content)
			if err != nil {
				flog.Warn("bot[%s]: failed to agent bot: %v", name, err)
				continue
			}

			// stats
			stats.BotRunTotalCounter(stats.AgentRuleset).Inc()

			// send message
			if payload == nil {
				continue
			}

			err = event.SendMessage(context.Background(), uid.String(), "", payload)
			if err != nil {
				flog.Error(fmt.Errorf("send message error %w", err))
				continue
			}
		}
	case types.Pull:
		list, err := store.Database.ListInstruct(uid, false, 10)
		if err != nil {
			return nil, err
		}
		var instruct []types.KV
		instruct = []types.KV{}
		for _, item := range list {
			instruct = append(instruct, types.KV{
				"no":        item.No,
				"bot":       item.Bot,
				"flag":      item.Flag,
				"content":   item.Content,
				"expire_at": item.ExpireAt,
			})
		}
		return instruct, nil
	case types.Ack:
		no, ok := data.Content.String("no")
		if !ok {
			return nil, errors.New("error instruct no")
		}

		err := store.Database.UpdateInstruct(&model.Instruct{
			No:    no,
			State: model.InstructDone,
		})
		if err != nil {
			return nil, err
		}
	case types.Online:
		hostid, ok := data.Content.String("hostid")
		if !ok {
			return nil, errors.New("error hostid")
		}
		hostname, _ := data.Content.String("hostname")

		err := registerAgent(uid, "", hostid, hostname)
		if err != nil {
			flog.Error(err)
			return nil, errors.New("register agent error")
		}

		err = event.SendMessage(context.Background(), uid.String(), "", types.TextMsg{Text: fmt.Sprintf("hostid: %s online", hostid)})
		if err != nil {
			flog.Error(fmt.Errorf("send message error %w", err))
		}
	case types.Offline:
		hostid, ok := data.Content.String("hostid")
		if !ok {
			return nil, errors.New("error hostid")
		}

		err := store.Database.UpdateAgentOnlineDuration(uid, "", hostid, time.Now())
		if err != nil {
			flog.Error(fmt.Errorf("update online duration error %w", err))
		}

		err = event.SendMessage(context.Background(), uid.String(), "", types.TextMsg{Text: fmt.Sprintf("hostid: %s offline", hostid)})
		if err != nil {
			flog.Error(fmt.Errorf("send message error %w", err))
		}
	}
	return nil, nil
}

func registerPlatformUser(data protocol.MessageEventData) (types.Uid, error) {
	platform, err := store.Database.GetPlatformByName(data.Self.Platform)
	if err != nil {
		return "", err
	}

	platformUser, err := store.Database.GetPlatformUserByFlag(data.UserId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	if platformUser != nil && platformUser.ID > 0 {
		user, err := store.Database.GetUserById(platformUser.UserID)
		if err != nil {
			return "", err
		}
		return types.Uid(user.Flag), nil
	} else {
		user := &model.User{
			Flag:  types.Id(),
			Name:  "user",
			Tags:  "[]",
			State: model.UserActive,
		}
		err = store.Database.UserCreate(user)
		if err != nil {
			return "", err
		}

		_, err = store.Database.CreatePlatformUser(&model.PlatformUser{
			PlatformID: platform.ID,
			UserID:     user.ID,
			Flag:       data.UserId,
			Name:       "user",
			IsBot:      false,
		})
		if err != nil {
			return "", err
		}
		return types.Uid(user.Flag), nil
	}
}

func registerPlatformChannel(data protocol.MessageEventData) (string, error) {
	platform, err := store.Database.GetPlatformByName(data.Self.Platform)
	if err != nil {
		return "", err
	}

	platformChannel, err := store.Database.GetPlatformChannelByFlag(data.TopicId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	if platformChannel != nil && platformChannel.ID > 0 {
		channel, err := store.Database.GetChannel(platformChannel.ChannelID)
		if err != nil {
			return "", err
		}
		return channel.Flag, nil
	} else {
		channel := &model.Channel{
			Flag:  types.Id(),
			Name:  fmt.Sprintf("%s_%s", data.Self.Platform, data.TopicId),
			State: model.ChannelActive,
		}
		_, err = store.Database.CreateChannel(channel)
		if err != nil {
			return "", err
		}

		_, err = store.Database.CreatePlatformChannel(&model.PlatformChannel{
			PlatformID: platform.ID,
			ChannelID:  channel.ID,
			Flag:       data.TopicId,
		})
		if err != nil {
			return "", err
		}

		_, err = store.Database.CreatePlatformChannelUser(&model.PlatformChannelUser{
			PlatformID:  platform.ID,
			ChannelFlag: data.TopicId,
			UserFlag:    data.UserId,
		})
		if err != nil {
			return "", err
		}

		return channel.Flag, nil
	}
}

func registerAgent(uid types.Uid, topic, hostid, hostname string) error {
	agent, err := store.Database.GetAgentByHostid(uid, topic, hostid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if agent != nil && agent.ID > 0 {
		err = store.Database.UpdateAgentLastOnlineAt(uid, topic, hostid, time.Now())
		if err != nil {
			return err
		}
	} else {
		agent = &model.Agent{
			UID:            uid.String(),
			Topic:          topic,
			Hostid:         hostid,
			Hostname:       hostname,
			OnlineDuration: 0,
			LastOnlineAt:   time.Now(),
		}
		_, err := store.Database.CreateAgent(agent)
		if err != nil {
			return err
		}
	}

	return nil
}
