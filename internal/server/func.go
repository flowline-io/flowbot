package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/notify"
	// OAuth provider registration — these blank imports trigger init() which
	// self-registers each provider in the providers.OAuthProvider registry.
	_ "github.com/flowline-io/flowbot/pkg/providers/dropbox"
	_ "github.com/flowline-io/flowbot/pkg/providers/github"
	_ "github.com/flowline-io/flowbot/pkg/providers/slack"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var cacheStore *cache.RedisStore

// SetCacheStore sets the cache store for server functions.
func SetCacheStore(s *cache.RedisStore) {
	cacheStore = s
}


// directIncomingMessage handles incoming message events for direct channels.
//
// It will register the user and channel if they don't already exist, then
// dispatch the message to the appropriate handler based on the content.
func directIncomingMessage(caller *platforms.Caller, e protocol.Event) {
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
	}
	ctx.SetTimeout(10 * time.Minute)

	findPlatform, err := store.Database.GetPlatformByName(ctx.Context(), msg.Self.Platform)
	if err != nil {
		flog.Error(err)
		return
	}
	platformId := findPlatform.ID

	findMessage, err := store.Database.GetMessageByPlatform(ctx.Context(), platformId, msg.MessageId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		flog.Error(err)
		return
	}
	if findMessage != nil {
		flog.Info("message %s %s already exists", msg.Self.Platform, msg.MessageId)
		return
	}

	module.Behavior(uid, module.MessageBotIncomingBehavior, 1)

	var payload types.MsgPayload

	chatKey := cache.NewKey("chat", "session", uid.String())
	var session string
	s, ok, err := cacheStore.Get(ctx.Context(), chatKey)
	if err != nil {
		flog.Error(err)
	}
	if ok {
		session = s
	}

	payload, session = manageChatSession(ctx, chatKey, msg.AltMessage, session, payload)

	err = store.Database.CreateMessage(ctx.Context(), gen.Message{
		Flag:          types.Id(),
		PlatformID:    platformId,
		PlatformMsgID: msg.MessageId,
		Topic:         topic,
		Role:          types.User,
		Session:       session,
		Content:       schema.JSON{"text": msg.AltMessage},
		State:         int(schema.MessageCreated),
	})
	if err != nil {
		flog.Error(err)
		return
	}

	payload = buildHelpMessage(msg.AltMessage, payload)

	if session == "" {
		payload = dispatchToModules(ctx, msg.AltMessage)
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

func manageChatSession(ctx types.Context, chatKey cache.Key, msgAlt string, session string, payload types.MsgPayload) (types.MsgPayload, string) {
	if strings.ToLower(msgAlt) == "chat" {
		if session == "" {
			payload = types.TextMsg{Text: "Chat started"}
			err := cacheStore.Set(ctx.Context(), chatKey, types.Id(), cache.TTLSession)
			if err != nil {
				flog.Error(fmt.Errorf("failed to set chat key: %w", err))
			}
		} else {
			payload = types.TextMsg{Text: "Chat already started"}
		}
	}

	if strings.ToLower(msgAlt) == "end" {
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
	flog.Debug("context: %+v", ctx)

	// behavior
	module.Behavior(uid, module.MessageGroupIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	for name, handle := range module.List() {
		if !handle.IsReady() {
			flog.Info("module %s unavailable", name)
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
	users, err := store.Database.GetUsers(context.Background())
	if err != nil {
		flog.Error(fmt.Errorf("notify error %w", err))
		return
	}

	for _, item := range users {
		ctx := types.Context{
			AsUser: types.Uid(item.Flag),
		}
		err = notify.GatewaySend(ctx.Context(), types.Uid(item.Flag), "agent.status", []string{"slack", "ntfy"}, map[string]any{
			"message": message,
		})
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
	key := cache.NewKey("online", "user", med.UserId)
	_, ok, _ = cacheStore.Get(ctx, key)
	if !ok {
		cacheStore.Set(ctx, key, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLSession)
	}
}

// agentAction handle agent message
//
// when action is types.Collect, it will call bot.Collect
// when action is types.Pull, it will return list of instruct
// when action is types.Ack, it will update instruct state to done
// when action is types.Online, it will register agent
// when action is types.Offline, it will update agent online duration
func agentAction(uid types.Uid, data types.AgentData) (any, error) {
	ctx := types.Context{
		AsUser: uid,
	}
	switch data.Action {
	case types.PullAction:
		return handlePullAction(ctx, uid)
	case types.AckAction:
		return nil, handleAckAction(ctx.Context(), data)
	case types.OnlineAction:
		return nil, handleOnlineAction(ctx, uid, data)
	case types.OfflineAction:
		return nil, handleOfflineAction(ctx, uid, data)
	case types.MessageAction:
		return nil, handleMessageAction(ctx, uid, data)
	}
	return nil, nil
}

func handlePullAction(ctx types.Context, uid types.Uid) (any, error) {
	list, err := store.Database.ListInstruct(ctx.Context(), uid, false, 10)
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
}

func handleAckAction(ctx context.Context, data types.AgentData) error {
	no, ok := data.Content.String("no")
	if !ok {
		return errors.New("error instruct no")
	}

	err := store.Database.UpdateInstruct(ctx, &gen.Instruct{
		No:    no,
		State: int(schema.InstructDone),
	})
	if err != nil {
		return err
	}
	return nil
}

func handleOnlineAction(ctx types.Context, uid types.Uid, data types.AgentData) error {
	hostid, ok := data.Content.String("hostid")
	if !ok {
		return errors.New("error hostid")
	}
	hostname, _ := data.Content.String("hostname")

	err := registerAgent(uid, "", hostid, hostname)
	if err != nil {
		flog.Error(err)
		return errors.New("register agent error")
	}

	key := cache.NewKey("online", "agent", hostid)
	_, ok, _ = cacheStore.Get(ctx.Context(), key)
	if !ok {
		err = notify.GatewaySend(ctx.Context(), uid, "agent.status", []string{"slack", "ntfy"}, map[string]any{
			"hostid":   hostid,
			"hostname": hostname,
			"status":   "online",
		})
		if err != nil {
			flog.Error(fmt.Errorf("send message error %w", err))
		}
	}

	err = cacheStore.Set(ctx.Context(), key, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLShort)
	if err != nil {
		flog.Error(err)
		return errors.New("set agent online error")
	}
	return nil
}

func handleOfflineAction(ctx types.Context, uid types.Uid, data types.AgentData) error {
	hostid, ok := data.Content.String("hostid")
	if !ok {
		return errors.New("error hostid")
	}
	hostname, _ := data.Content.String("hostname")

	err := store.Database.UpdateAgentOnlineDuration(ctx.Context(), uid, "", hostid, time.Now())
	if err != nil {
		flog.Error(fmt.Errorf("update online duration error %w", err))
	}

	err = cacheStore.Del(ctx.Context(), cache.NewKey("online", "agent", hostid))
	if err != nil {
		flog.Error(fmt.Errorf("del agent online stats error %w", err))
	}

	err = notify.GatewaySend(ctx.Context(), uid, "agent.status", []string{"slack", "ntfy"}, map[string]any{
		"hostid":   hostid,
		"hostname": hostname,
		"status":   "offline",
	})
	if err != nil {
		flog.Error(fmt.Errorf("send message error %w", err))
	}
	return nil
}

func handleMessageAction(ctx types.Context, uid types.Uid, data types.AgentData) error {
	message, ok := data.Content.String("message")
	if !ok {
		return errors.New("empty message")
	}

	err := notify.GatewaySend(ctx.Context(), uid, "agent.status", []string{"slack", "ntfy"}, map[string]any{
		"message": message,
	})
	if err != nil {
		flog.Error(fmt.Errorf("send message error %w", err))
	}
	return nil
}

// registerPlatformUser registers a platform user based on the provided message event data.
// It checks if the platform user already exists by its flag, and if found, retrieves the existing user flag.
// If the platform user does not exist, it creates a new user and platform user entry in the database.
// It also associates the platform user with the platform.
// Returns the user flag and an error if any operation fails.
func registerPlatformUser(data protocol.MessageEventData) (types.Uid, error) {
	platform, err := store.Database.GetPlatformByName(context.Background(), data.Self.Platform)
	if err != nil {
		return "", err
	}

	platformUser, err := store.Database.GetPlatformUserByFlag(context.Background(), data.UserId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return "", err
	}

	if platformUser != nil && platformUser.ID > 0 {
		user, err := store.Database.GetUserById(context.Background(), platformUser.UserID)
		if err != nil {
			return "", err
		}
		return types.Uid(user.Flag), nil
	}
	user := &gen.User{
		Flag:  types.Id(),
		Name:  "user",
		Tags:  "[]",
		State: int(schema.UserActive),
	}
	err = store.Database.UserCreate(context.Background(), user)
	if err != nil {
		return "", err
	}

	_, err = store.Database.CreatePlatformUser(context.Background(), &gen.PlatformUser{
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

// registerPlatformChannel registers a platform channel based on the provided message event data.
// It checks if the platform channel already exists by its topic ID, and if found, retrieves the existing channel flag.
// If the platform channel does not exist, it creates a new channel and platform channel entry in the database.
// It also associates the platform channel with the user who triggered the event.
// Returns the channel flag and an error if any operation fails.
func registerPlatformChannel(data protocol.MessageEventData) (string, error) {
	platform, err := store.Database.GetPlatformByName(context.Background(), data.Self.Platform)
	if err != nil {
		return "", err
	}

	platformChannel, err := store.Database.GetPlatformChannelByFlag(context.Background(), data.TopicId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return "", err
	}

	if platformChannel != nil && platformChannel.ID > 0 {
		channel, err := store.Database.GetChannel(context.Background(), platformChannel.ChannelID)
		if err != nil {
			return "", err
		}
		return channel.Flag, nil
	}
	channel := &gen.Channel{
		Flag:  types.Id(),
		Name:  fmt.Sprintf("%s_%s", data.Self.Platform, data.TopicId),
		State: int(schema.ChannelActive),
	}
	_, err = store.Database.CreateChannel(context.Background(), channel)
	if err != nil {
		return "", err
	}

	_, err = store.Database.CreatePlatformChannel(context.Background(), &gen.PlatformChannel{
		PlatformID: platform.ID,
		ChannelID:  channel.ID,
		Flag:       data.TopicId,
	})
	if err != nil {
		return "", err
	}

	_, err = store.Database.CreatePlatformChannelUser(context.Background(), &gen.PlatformChannelUser{
		PlatformID:  platform.ID,
		ChannelFlag: data.TopicId,
		UserFlag:    data.UserId,
	})
	if err != nil {
		return "", err
	}

	return channel.Flag, nil
}

// registerAgent Register agent by uid, topic, hostid and hostname
//
// if the agent already exists, update its last online time, otherwise create a new agent
func registerAgent(uid types.Uid, topic, hostid, hostname string) error {
	if hostid == "" {
		return fmt.Errorf("hostid is empty")
	}
	agent, err := store.Database.GetAgentByHostid(context.Background(), uid, topic, hostid)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}

	if agent != nil && agent.ID > 0 {
		err = store.Database.UpdateAgentLastOnlineAt(context.Background(), uid, topic, hostid, time.Now())
		if err != nil {
			return err
		}
	} else {
		agent = &gen.Agent{
			UID:            uid.String(),
			Topic:          topic,
			Hostid:         hostid,
			Hostname:       hostname,
			OnlineDuration: 0,
			LastOnlineAt:   time.Now(),
		}
		_, err := store.Database.CreateAgent(context.Background(), agent)
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

	p, err := store.Database.ParameterGet(ctx.Context(), accessToken)
	if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
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
