package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/ruleset/action"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/ruleset/pipeline"
	"github.com/flowline-io/flowbot/internal/ruleset/session"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/dropbox"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		m := make(map[string]interface{})
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

	// session
	if sess, ok := sessionCurrent(uid, topic); ok && sess.State == model.SessionStart {
		// session cancel command
		isCancel := false
		if msg.AltMessage == "cancel" {
			_ = store.Database.SessionState(ctx.AsUser, ctx.Original, model.SessionCancel)
			payload = types.TextMsg{Text: "session cancel"}
			isCancel = true
		}
		if !isCancel {
			ctx.SessionRuleId = sess.RuleID
			ctx.SessionInitValues = types.KV(sess.Init)
			ctx.SessionLastValues = types.KV(sess.Values)

			// get action handler
			name := "unknown"
			var botHandler bots.Handler
			for n, handler := range bots.List() {
				for _, item := range handler.Rules() {
					switch v := item.(type) {
					case []session.Rule:
						for _, rule := range v {
							if rule.Id == sess.RuleID {
								botHandler = handler
								name = n
							}
						}
					}
				}
			}
			if botHandler == nil {
				payload = types.TextMsg{Text: "error session"}
			} else {
				payload, err = botHandler.Session(ctx, msg.AltMessage)
				if err != nil {
					flog.Warn("topic[%s]: failed to run bot: %v", name, err)
				}
			}
		}
	}

	// action
	if payload == nil {
		seq := msg.Seq
		option := msg.Option
		if seq > 0 {
			message, err := store.Database.GetMessage(topic) // fixme
			if err != nil {
				flog.Error(err)
			}
			actionRuleId := ""
			if src, ok := types.KV(message.Content).Map("src"); ok {
				if id, ok := src["id"]; ok {
					actionRuleId = id.(string)
				}
			}
			ctx.SeqId = int(seq)
			ctx.ActionRuleId = actionRuleId

			// get action handler
			name := "unknown"
			var botHandler bots.Handler
			for n, handler := range bots.List() {
				for _, item := range handler.Rules() {
					switch v := item.(type) {
					case []action.Rule:
						for _, rule := range v {
							if rule.Id == actionRuleId {
								botHandler = handler
								name = n
							}
						}
					}
				}
			}
			if botHandler == nil {
				payload = types.TextMsg{Text: "error action"}
			} else {
				payload, err = botHandler.Action(ctx, option)
				if err != nil {
					flog.Warn("topic[%s]: failed to run bot: %v", name, err)
				}

				if payload != nil {
					//botUid := types.Uid(0) // todo types.ParseUserId(msg.Original)
					//botSend(topic, botUid, payload, types.WithContext(ctx))

					// pipeline action stage
					//pipelineFlag, _ := types.KV(message.Head).String("x-pipeline-flag")
					//pipelineVersion, _ := types.KV(message.Head).Int64("x-pipeline-version")
					//nextPipeline(ctx, pipelineFlag, int(pipelineVersion), topic, botUid)
					return
				}
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
				stats.Inc("BotRunCommandTotal", 1)
			}
		}
		// pipeline command trigger
		if payload == nil {
			in := msg.AltMessage
			// check "~" prefix
			if strings.HasPrefix(in, "~") {
				var pipelineFlag string
				var pipelineVersion int
				in = strings.Replace(in, "~", "", 1)
				payload, pipelineFlag, pipelineVersion, err = handle.Pipeline(ctx, nil, in, types.PipelineCommandTriggerOperate)
				if err != nil {
					flog.Warn("topic[%s]: failed to run bot: %v", name, err)
				}
				ctx.PipelineFlag = pipelineFlag
				ctx.PipelineVersion = pipelineVersion

				// stats
				stats.Inc("BotTriggerPipelineTotal", 1)

				// error message
				if payload == nil {
					payload = types.TextMsg{Text: "error pipeline"}
				}
			}
		}
		// condition
		if payload == nil {
			fUid := ""
			fSeq := int64(0)
			if msg.Forwarded != "" {
				f := strings.Split(msg.Forwarded, ":")
				if len(f) == 2 {
					fUid = f[0]
					fSeq, _ = strconv.ParseInt(f[1], 10, 64)
				}
			}

			if fUid != "" && fSeq > 0 {
				//uid2 := types.ParseUserId(fUid)
				topic := ""                                      // fixme
				message, err := store.Database.GetMessage(topic) // fixme
				if err != nil {
					flog.Error(err)
				}

				if message.ID > 0 {
					src, _ := types.KV(message.Content).Map("src")
					tye, _ := types.KV(message.Content).String("tye")
					d, _ := json.Marshal(src)
					pl := types.ToPayload(tye, d)
					ctx.Condition = tye
					payload, err = handle.Condition(ctx, pl)
					if err != nil {
						flog.Warn("topic[%s]: failed to run bot: %v", name, err)
					}

					// stats
					stats.Inc("BotRunConditionTotal", 1)
				}
			}
		}
		// input
		if payload == nil {
			/*
				payload, err = handle.Input(ctx, nil, msg.AltMessage)
				if err != nil {
					flog.Warn("topic[%s]: failed to run bot: %v", name, err)
					return
				}

				if payload != nil {
					// stats
					stats.Inc("BotRunInputTotal", 1)
				}
			*/
		}

		if payload != nil {
			break
		}
	}

	// send message
	if payload == nil {
		// empty
		payload = types.TextMsg{Text: "empty"}
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
		Id:       e.Id,
		Original: topic,
		AsUser:   uid,
		//Original:  msg.Original,
		//RcptTo:    msg.RcptTo,
		//AuthLvl:   msg.AuthLvl,
		//MetaWhat:  msg.MetaWhat,
		//Timestamp: msg.Timestamp,
	}

	// behavior
	bots.Behavior(uid, bots.MessageGroupIncomingBehavior, 1)

	// user auth record todo

	var payload types.MsgPayload

	for name, handle := range bots.List() {

		if !handle.IsReady() {
			flog.Info("bot %s unavailable", name)
			continue
		}

		// condition
		if payload == nil {
			fUid := ""
			fSeq := int64(0)
			if forwarded := msg.Forwarded; forwarded != "" {
				f := strings.Split(forwarded, ":")
				if len(f) == 2 {
					fUid = f[0]
					fSeq, _ = strconv.ParseInt(f[1], 10, 64)
				}
			}

			if fUid != "" && fSeq > 0 {
				//uid2 := types.ParseUserId(fUid)
				topic := ""                                      // fixme
				message, err := store.Database.GetMessage(topic) // fixme
				if err != nil {
					flog.Error(err)
				}

				if message.ID > 0 {
					src, _ := types.KV(message.Content).Map("src")
					tye, _ := types.KV(message.Content).String("tye")
					d, _ := json.Marshal(src)
					pl := types.ToPayload(tye, d)
					ctx.Condition = tye
					payload, err = handle.Condition(ctx, pl)
					if err != nil {
						flog.Warn("topic[%s]: failed to run bot: %v", name, err)
					}

					// stats
					stats.Inc("BotRunConditionTotal", 1)
				}
			}
		}
		// group
		if payload == nil {
			payload, err = handle.Group(ctx, nil, msg.AltMessage)
			if err != nil {
				flog.Warn("topic[%s]: failed to run group bot: %v", name, err)
				return
			}

			// stats
			stats.Inc("BotRunGroupTotal", 1)
		}

		if payload != nil {
			break
		}
	}

	// send message
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
	flog.Info("event: %+v  response: %+v", msg, resp)
}

func nextPipeline(ctx types.Context, pipelineFlag string, pipelineVersion int, rcptTo string, botUid types.Uid) {
	if pipelineFlag != "" && pipelineVersion > 0 {
		pipelineData, err := store.Database.PipelineGet(ctx.AsUser, ctx.Original, pipelineFlag)
		if err != nil {
			flog.Error(err)
			return
		}
		for _, handler := range bots.List() {
			for _, item := range handler.Rules() {
				switch v := item.(type) {
				case []pipeline.Rule:
					for _, rule := range v {
						if rule.Id == pipelineData.RuleID {
							ctx.PipelineFlag = pipelineFlag
							ctx.PipelineVersion = pipelineVersion
							ctx.PipelineRuleId = pipelineData.RuleID
							ctx.PipelineStepIndex = int(pipelineData.Stage)
							//payload, _, _, err := handler.Pipeline(ctx, nil, nil, types.PipelineNextOperate)
							//if err != nil {
							//	flog.Error(err)
							//	return
							//}
							//botSend(rcptTo, botUid, payload, types.WithContext(ctx))
						}
					}
				}
			}
		}
	}
}

func notifyAfterReboot() {
	//botUid := types.Uid(0) // fixme
	//
	//creds, err := store.Chatbot.GetCredentials()
	//if err != nil {
	//	flog.Error(err)
	//	return
	//}

	//for _, cred := range creds {
	//	rcptTo := tstore.EncodeUid(cred.Userid).P2PName(botUid)
	//	if rcptTo != "" {
	//		botSend(rcptTo, botUid, types.TextMsg{Text: "reboot"})
	//	}
	//}
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

func sessionCurrent(uid types.Uid, topic string) (model.Session, bool) {
	sess, err := store.Database.SessionGet(uid, topic)
	if err != nil {
		return model.Session{}, false
	}
	return sess, true
}

func errorResponse(rw http.ResponseWriter, text string) {
	rw.WriteHeader(http.StatusBadRequest)
	_, _ = rw.Write([]byte(text))
}

func flowkitAction(uid types.Uid, data types.FlowkitData) (interface{}, error) {
	switch data.Action {
	case types.Agent:
		//userUid := uid
		//
		//id, ok := data.Content.String("id")
		//if !ok {
		//	return nil, errors.New("error agent id")
		//}
		//
		//subs, err := tstore.Users.FindSubs(userUid, [][]string{{"bot"}}, nil, true)
		//if err != nil {
		//	return nil, err
		//}
		//
		//// user auth record
		//
		//for _, sub := range subs {
		//	if !isBot(sub) {
		//		continue
		//	}
		//
		//	topic := sub.User
		//	topicUid := types.ParseUid(topic)
		//
		//	// bot name
		//	name := botName(sub)
		//	handle, ok := bots.List()[name]
		//	if !ok {
		//		continue
		//	}
		//
		//	if !handle.IsReady() {
		//		flog.Info("bot %s unavailable", topic)
		//		continue
		//	}
		//
		//	ctx := types.Context{
		//		Original:     topicUid.UserId(),
		//		RcptTo:       topic,
		//		AsUser:       userUid,
		//		AgentId:      id,
		//		AgentVersion: data.Version,
		//	}
		//	payload, err := handle.Agent(ctx, data.Content)
		//	if err != nil {
		//		flog.Warn("topic[%s]: failed to agent bot: %v", topic, err)
		//		continue
		//	}
		//
		//	// stats
		//	stats.Inc("BotRunAgentTotal", 1)
		//
		//	// send message
		//	if payload == nil {
		//		continue
		//	}
		//
		//	botSend(uid.P2PName(topicUid), topicUid, payload)
		//}
	case types.Pull:
		list, err := store.Database.ListInstruct(uid, false)
		if err != nil {
			return nil, err
		}
		var instruct []map[string]interface{}
		instruct = []map[string]interface{}{}
		for _, item := range list {
			instruct = append(instruct, map[string]interface{}{
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
		var list []map[string]interface{}
		for name, bot := range bots.List() {
			instruct, err := bot.Instruct()
			if err != nil {
				continue
			}
			if len(instruct) <= 0 {
				continue
			}
			list = append(list, map[string]interface{}{
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
			return map[string]interface{}{}, nil
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
