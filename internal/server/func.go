package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/dropbox"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	openaiProvider "github.com/flowline-io/flowbot/pkg/providers/openai"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/redis/go-redis/v9"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
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
				stats.Inc(types.BotRunCommandTotalStatsName, 1)
			}
		}

		if payload != nil {
			break
		}
	}

	// llm chat
	if payload == nil {
		token, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.TokenKey)
		baseUrl, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.BaseUrlKey)

		llm, err := openai.New(openai.WithToken(token.String()), openai.WithBaseURL(baseUrl.String()))
		if err != nil {
			flog.Error(err)
			return
		}

		prompt := prompts.NewPromptTemplate(
			"Human: {{.content}}\nAssistant:",
			[]string{"content"},
		)

		result, err := prompt.Format(map[string]any{
			"content": msg.AltMessage,
		})
		if err != nil {
			flog.Error(err)
			return
		}

		completion, err := llms.GenerateFromSinglePrompt(context.Background(), llm, result,
			llms.WithTemperature(0.8),
		)
		if err != nil {
			flog.Error(err)
			return
		}

		payload = types.TextMsg{Text: completion}
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
	fmt.Println(ctx)

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

func notifyAfterReboot() {
	// send message
	users, err := store.Database.GetUsers()
	if err != nil {
		flog.Error(fmt.Errorf("notify reboot error %w", err))
		return
	}
	for _, item := range users {
		err = event.SendMessage(item.Flag, "", types.TextMsg{Text: "reboot"})
		if err != nil {
			flog.Error(fmt.Errorf("notify reboot error %w", err))
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

func errorResponse(rw http.ResponseWriter, text string) {
	rw.WriteHeader(http.StatusBadRequest)
	_, _ = rw.Write([]byte(text))
}

func flowkitAction(uid types.Uid, data types.FlowkitData) (interface{}, error) {
	switch data.Action {
	case types.Agent:
		id, ok := data.Content.String("id")
		if !ok {
			return nil, errors.New("error agent id")
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
				AgentId:      id,
				AgentVersion: data.Version,
			}
			payload, err := handle.Agent(ctx, data.Content)
			if err != nil {
				flog.Warn("bot[%s]: failed to agent bot: %v", name, err)
				continue
			}

			// stats
			stats.Inc(types.BotRunAgentTotalStatsName, 1)

			// send message
			if payload == nil {
				continue
			}

			err = event.SendMessage(uid.String(), "", payload)
			if err != nil {
				flog.Error(fmt.Errorf("send message error %w", err))
				continue
			}
		}

	case types.Pull:
		list, err := store.Database.ListInstruct(uid, false)
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
	case types.Info:
		var user *model.User // fixme
		return utils.Fn(user), nil
	case types.Bots:
		var list []types.KV
		for name, bot := range bots.List() {
			instruct, err := bot.Instruct()
			if err != nil {
				continue
			}
			if len(instruct) <= 0 {
				continue
			}
			list = append(list, types.KV{
				"id":   name,
				"name": name,
			})
		}
		return list, nil
	case types.Help:
		if id, ok := data.Content.String("id"); ok {
			if bot, ok := bots.List()[id]; ok {
				return bot.Help()
			}
			return types.KV{}, nil
		}
	case types.Ack:
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
		return channel.Flag, nil
	}
}
