package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/dropbox"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"gorm.io/gorm"
)

// newProvider returns a new OAuth provider based on the given category.
//
// The supported categories are:
//
// - pocket.ID
// - github.ID
// - dropbox.ID
//
// The OAuth provider will be created with the configuration values
// stored in the database. If the category is not supported, nil is
// returned.
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

// directIncomingMessage handles incoming message events for direct channels.
//
// It will register the user and channel if they don't already exist, then
// iterate over all available bots and run their command handlers if they
// have any.
//
// If any bot returns a non-nil payload, it will be sent back to the direct
// channel as a message.
//
// If no bot returns a payload, a default message will be sent.
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

	// behavior
	chatbot.Behavior(uid, chatbot.MessageBotIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	// get chat key
	chatKey := fmt.Sprintf("chat:%s", uid)
	session, err := rdb.Client.Get(ctx.Context(), chatKey).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			flog.Error(err)
		}
	}

	// chat command
	// start Multi-turn conversation
	if strings.ToLower(msg.AltMessage) == "chat" {
		if session == "" {
			payload = types.TextMsg{Text: "Chat started"}
			err = rdb.Client.Set(ctx.Context(), chatKey, types.Id(), 24*time.Hour).Err()
			if err != nil {
				flog.Error(fmt.Errorf("failed to set chat key: %w", err))
			}
		} else {
			payload = types.TextMsg{Text: "Chat already started"}
		}
	}

	// chat end command
	// end Multi-turn conversation
	if strings.ToLower(msg.AltMessage) == "end" {
		err = rdb.Client.Del(ctx.Context(), chatKey).Err()
		if err != nil {
			flog.Error(fmt.Errorf("failed to delete chat key: %w", err))
		}
		payload = types.TextMsg{Text: "Chat ended"}
		session = ""
	}

	// user message
	err = store.Database.CreateMessage(model.Message{
		Flag:          types.Id(),
		PlatformID:    platformId,
		PlatformMsgID: msg.MessageId,
		Topic:         topic,
		Role:          types.User,
		Session:       session,
		Content:       model.JSON{"text": msg.AltMessage},
		State:         model.MessageCreated,
	})
	if err != nil {
		flog.Error(err)
		return
	}

	// help command
	if strings.ToLower(msg.AltMessage) == "help" {
		m := make(types.KV)
		for name, handle := range chatbot.List() {
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

	// rule chat
	if session == "" {
		for name, handle := range chatbot.List() {
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
	}

	// tool chat
	if payload == nil && session == "" {
		tools, err := chatbot.AvailableTools(ctx)
		if err != nil {
			flog.Error(err)
			return
		}
		agent, err := agents.ReactAgent(ctx.Context(), agents.AgentModelName(agents.AgentReact), tools)
		if err != nil {
			flog.Error(err)
			return
		}

		messages, err := agents.DefaultTemplate().Format(ctx.Context(), map[string]any{
			"content": msg.AltMessage,
		})
		if err != nil {
			flog.Error(err)
			return
		}

		resp, err := agent.Generate(ctx.Context(), messages)
		if err != nil {
			flog.Error(err)
			return
		}

		if resp != nil && resp.Content != "" {
			payload = types.TextMsg{Text: resp.Content}
		}
	}

	// multi-turn conversation
	if payload == nil && session != "" {
		list, err := store.Database.GetMessagesBySession(session)
		if err != nil {
			flog.Error(fmt.Errorf("failed to get history messages: %w", err))
			return
		}

		chatHistory := make([]*schema.Message, 0, len(list))
		for _, item := range list {
			content, _ := types.KV(item.Content).String("text")
			chatHistory = append(chatHistory, &schema.Message{
				Role:    schema.RoleType(item.Role),
				Content: content,
			})
		}

		messages, err := agents.DefaultMultiChatTemplate().Format(ctx.Context(), map[string]any{
			"chat_history": chatHistory,
		})
		if err != nil {
			flog.Error(fmt.Errorf("failed to format message: %w", err))
			return
		}

		llm, err := agents.ChatModel(ctx.Context(), agents.AgentModelName(agents.AgentChat))
		if err != nil {
			flog.Error(fmt.Errorf("failed to get chat model: %w", err))
			return
		}

		resp, err := agents.Generate(ctx.Context(), llm, messages)
		if err != nil {
			flog.Error(fmt.Errorf("failed to generate response: %w", err))
			return
		}

		if resp != nil && resp.Content != "" {
			payload = types.TextMsg{Text: resp.Content}

			// assistant message
			err = store.Database.CreateMessage(model.Message{
				Flag:          types.Id(),
				PlatformID:    platformId,
				PlatformMsgID: msg.MessageId,
				Topic:         topic,
				Role:          types.Assistant,
				Session:       session,
				Content:       model.JSON{"text": resp.Content},
				State:         model.MessageCreated,
			})
			if err != nil {
				flog.Error(err)
				return
			}
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

// groupIncomingMessage processes incoming message events for group channels.
//
// It will register the user and channel if they don't already exist, then
// iterate over all available bots and run their command handlers if they
// have any.
//
// If any bot returns a non-nil payload, it will be sent back to the group
// channel as a message.
//
// If no bot returns a payload, a default message will be sent.
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
	chatbot.Behavior(uid, chatbot.MessageGroupIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	for name, handle := range chatbot.List() {
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

// notifyAll send message to all users
//
// This function will send a message to all users in the database.
// If an error occurs, it will be logged and the function will continue to the next user.
func notifyAll(message string) {
	// send message
	users, err := store.Database.GetUsers()
	if err != nil {
		flog.Error(fmt.Errorf("notify error %w", err))
		return
	}

	for _, item := range users {
		ctx := types.Context{
			AsUser: types.Uid(item.Flag),
		}
		err = event.SendMessage(ctx, types.TextMsg{Text: message})
		if err != nil {
			flog.Error(fmt.Errorf("notify error %w", err))
			continue
		}
	}
}

// onlineStatus handles MessageEvent protocol event and updates user online status.
// It will set user online status to 30 minutes if user is not online,
// or reset user online status to 30 minutes if user is already online.
func onlineStatus(msg protocol.Event) {
	med, ok := msg.Data.(protocol.MessageEventData)
	if !ok {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("online:%s", med.UserId)
	_, err := rdb.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		rdb.Client.Set(ctx, key, time.Now().Unix(), 30*time.Minute)
	} else if err != nil {
		return
	} else {
		rdb.Client.Expire(ctx, key, 30*time.Minute)
	}
}

// agentAction handle agent message
//
// when action is types.Collect, it will call bot.Collect
// when action is types.Pull, it will return list of instruct
// when action is types.Ack, it will update instruct state to done
// when action is types.Online, it will register agent
// when action is types.Offline, it will update agent online duration
func agentAction(uid types.Uid, data types.AgentData) (interface{}, error) {
	ctx := types.Context{
		AsUser: uid,
	}
	switch data.Action {
	case types.CollectAction:
		id, ok := data.Content.String("id")
		if !ok {
			return nil, errors.New("error collect id")
		}

		for name, handle := range chatbot.List() {
			if !handle.IsReady() {
				flog.Info("bot %s unavailable", name)
				continue
			}

			ctx := types.Context{
				Platform:      "",
				Topic:         "",
				AsUser:        uid,
				CollectRuleId: id,
				AgentVersion:  data.Version,
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

			err = event.SendMessage(ctx, payload)
			if err != nil {
				flog.Error(fmt.Errorf("send message error %w", err))
				continue
			}
		}
	case types.PullAction:
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
	case types.AckAction:
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
	case types.OnlineAction:
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

		check, err := rdb.Client.Get(ctx.Context(), fmt.Sprintf("online:%s", hostid)).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, errors.New("get agent online error")
		}
		if check == "" {
			// send message
			err = event.SendMessage(ctx, types.TextMsg{Text: fmt.Sprintf("hostid: %s %s online", hostid, hostname)})
			if err != nil {
				flog.Error(fmt.Errorf("send message error %w", err))
			}
		}

		// leave online stats
		_, err = rdb.Client.Set(ctx.Context(), fmt.Sprintf("online:%s", hostid), time.Now().Unix(), 2*time.Minute).Result()
		if err != nil {
			flog.Error(err)
			return nil, errors.New("set agent online error")
		}
	case types.OfflineAction:
		hostid, ok := data.Content.String("hostid")
		if !ok {
			return nil, errors.New("error hostid")
		}
		hostname, _ := data.Content.String("hostname")

		err := store.Database.UpdateAgentOnlineDuration(uid, "", hostid, time.Now())
		if err != nil {
			flog.Error(fmt.Errorf("update online duration error %w", err))
		}

		_, err = rdb.Client.Del(ctx.Context(), fmt.Sprintf("online:%s", hostid)).Result()
		if err != nil {
			flog.Error(fmt.Errorf("del agent online stats error %w", err))
		}

		err = event.SendMessage(ctx, types.TextMsg{Text: fmt.Sprintf("hostid: %s %s stop", hostid, hostname)})
		if err != nil {
			flog.Error(fmt.Errorf("send message error %w", err))
		}
	case types.MessageAction:
		message, ok := data.Content.String("message")
		if !ok {
			return nil, errors.New("empty message")
		}

		err := event.SendMessage(ctx, types.TextMsg{Text: message})
		if err != nil {
			flog.Error(fmt.Errorf("send message error %w", err))
		}
	}
	return nil, nil
}

// registerPlatformUser registers a platform user based on the provided message event data.
// It checks if the platform user already exists by its flag, and if found, retrieves the existing user flag.
// If the platform user does not exist, it creates a new user and platform user entry in the database.
// It also associates the platform user with the platform.
// Returns the user flag and an error if any operation fails.
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

// registerPlatformChannel registers a platform channel based on the provided message event data.
// It checks if the platform channel already exists by its topic ID, and if found, retrieves the existing channel flag.
// If the platform channel does not exist, it creates a new channel and platform channel entry in the database.
// It also associates the platform channel with the user who triggered the event.
// Returns the channel flag and an error if any operation fails.
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

// registerAgent Register agent by uid, topic, hostid and hostname
//
// if the agent already exists, update its last online time, otherwise create a new agent
func registerAgent(uid types.Uid, topic, hostid, hostname string) error {
	if hostid == "" {
		return fmt.Errorf("hostid is empty")
	}
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

// auth pprof middleware for pprof routes
func authPprof(ctx fiber.Ctx) bool {
	var r http.Request
	if err := fasthttpadaptor.ConvertRequest(ctx.RequestCtx(), &r, true); err != nil {
		flog.Error(fmt.Errorf("pprof auth error: %w", err))
		return true
	}

	if !strings.Contains(ctx.Path(), "/debug/pprof") {
		return true
	}

	accessToken := route.GetAccessToken(&r)
	if accessToken == "" {
		flog.Warn("pprof auth warning: missing token")
		return true
	}

	p, err := store.Database.ParameterGet(accessToken)
	if err != nil || p.ID <= 0 || p.IsExpired() {
		flog.Warn("pprof auth warning: parameter error")
		return true
	}

	return false
}

type structValidator struct {
	validate *validator.Validate
}

// Validator needs to implement the Validate method
func (v *structValidator) Validate(out any) error {
	return v.validate.Struct(out)
}
