package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/adjust/rmq/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sysatom/flowbot/internal/bots"
	botGithub "github.com/sysatom/flowbot/internal/bots/github"
	botPocket "github.com/sysatom/flowbot/internal/bots/pocket"
	"github.com/sysatom/flowbot/internal/ruleset/action"
	"github.com/sysatom/flowbot/internal/ruleset/pipeline"
	"github.com/sysatom/flowbot/internal/ruleset/session"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/cache"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/providers"
	"github.com/sysatom/flowbot/pkg/providers/dropbox"
	"github.com/sysatom/flowbot/pkg/providers/github"
	"github.com/sysatom/flowbot/pkg/providers/pocket"
	"github.com/sysatom/flowbot/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const BotFather = "BotFather"

func isBot(subs interface{}) bool {
	//// normal bot user
	//if subs.GetState() != types.StateOK {
	//	return false
	//}
	//// verified
	//trusted := subs.GetTrusted()
	//if trusted == nil {
	//	return false
	//}
	//if !isVerified(trusted) {
	//	return false
	//}
	//// check name
	//public := subs.GetPublic()
	//if public == nil {
	//	return false
	//}
	//name := utils.Fn(public)
	//if !strings.HasSuffix(name, bots.BotNameSuffix) {
	//	return false
	//}

	return true
}

func isBotUser(user *types.User) bool {
	if user == nil {
		return false
	}
	//// normal bot user
	//if user.State != types.StateOK {
	//	return false
	//}
	//// verified
	//if !isVerified(user.Trusted) {
	//	return false
	//}
	//// check name
	//name := utils.Fn(user.Public)
	//if !strings.HasSuffix(name, bots.BotNameSuffix) {
	//	return false
	//}

	return true
}

func isVerified(trusted interface{}) bool {
	if v, ok := trusted.(map[string]interface{}); ok {
		if b, ok := v["verified"]; ok {
			if vv, ok := b.(bool); ok {
				return vv
			}
		}
	}
	return false
}

func botName(subs interface{}) string {
	//public := subs.GetPublic()
	//if public == nil {
	//	return ""
	//}
	//name := utils.Fn(public)
	//name = strings.ReplaceAll(name, bots.BotNameSuffix, "")
	//return name
	return ""
}

// botSend bot send message, rcptTo: user uid: bot
func botSend(rcptTo string, uid types.Uid, out types.MsgPayload, option ...interface{}) {
	if out == nil {
		return
	}
}

func newProvider(category string) providers.OAuthProvider {
	var provider providers.OAuthProvider

	switch category {
	case pocket.ID:
		provider = pocket.NewPocket(botPocket.Config.ConsumerKey, "", "", "")
	case github.ID:
		provider = github.NewGithub(botGithub.Config.ID, botGithub.Config.Secret, "", "")
	case dropbox.ID:
		provider = dropbox.NewDropbox("", "", "", "")
	default:
		return nil
	}

	return provider
}

type Topic struct { // fixme del
	name string
}

func botIncomingMessage(t *Topic, msg *ClientComMessage) {
	// check topic owner user
	if msg.AsUser == msg.Pub.Topic {
		return
	}
	if msg.Original == "" || msg.RcptTo == "" {
		return
	}

	var err error
	var subs []interface{} // fixme

	uid := types.ParseUserId(msg.AsUser)
	ctx := types.Context{
		Id:        msg.Id,
		Original:  msg.Original,
		RcptTo:    msg.RcptTo,
		AsUser:    uid,
		AuthLvl:   msg.AuthLvl,
		MetaWhat:  msg.MetaWhat,
		Timestamp: msg.Timestamp,
	}

	// behavior
	bots.Behavior(uid, bots.MessageBotIncomingBehavior, 1)

	// user auth record

	// bot
	for _, sub := range subs {
		if !isBot(sub) {
			continue
		}

		// bot name
		name := botName(sub)
		handle, ok := bots.List()[name]
		if !ok {
			continue
		}

		if !handle.IsReady() {
			logs.Info.Printf("bot %s unavailable", t.name)
			continue
		}

		var payload types.MsgPayload

		// auth
		if payload == nil {
			// session
			if sess, ok := sessionCurrent(uid, msg.Original); ok && sess.State == model.SessionStart {
				// session cancel command
				isCancel := false
				if msg.Pub.Head == nil {
					if v, ok := msg.Pub.Content.(string); ok {
						if v == "cancel" {
							_ = store.Chatbot.SessionState(ctx.AsUser, ctx.Original, model.SessionCancel)
							payload = types.TextMsg{Text: "session cancel"}
							isCancel = true
						}
					}
				}
				if !isCancel {
					ctx.SessionRuleId = sess.RuleID
					ctx.SessionInitValues = types.KV(sess.Init)
					ctx.SessionLastValues = types.KV(sess.Values)

					// get action handler
					var botHandler bots.Handler
					for _, handler := range bots.List() {
						for _, item := range handler.Rules() {
							switch v := item.(type) {
							case []session.Rule:
								for _, rule := range v {
									if rule.Id == sess.RuleID {
										botHandler = handler
									}
								}
							}
						}
					}
					if botHandler == nil {
						payload = types.TextMsg{Text: "error session"}
					} else {
						payload, err = botHandler.Session(ctx, msg.Pub.Content)
						if err != nil {
							logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
						}
					}
				}
			}
			// action
			if payload == nil {
				if msg.Pub.Head != nil {
					var cm types.ChatMessage
					d, err := json.Marshal(msg.Pub.Content)
					if err != nil {
						logs.Err.Println(err)
					}
					err = json.Unmarshal(d, &cm)
					if err != nil {
						logs.Err.Println(err)
					}
					var seq float64
					var option string
					for _, ent := range cm.Ent {
						if ent.Tp == "EX" {
							if m, ok := ent.Data.Val.(map[string]interface{}); ok {
								if v, ok := m["seq"]; ok {
									seq = v.(float64)
								}
								if v, ok := m["resp"]; ok {
									values := v.(map[string]interface{})
									for s := range values {
										option = s
									}
								}
							}
						}
					}
					if seq > 0 {
						message, err := store.Chatbot.GetMessage(msg.RcptTo, int(seq))
						if err != nil {
							logs.Err.Println(err)
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
						var botHandler bots.Handler
						for _, handler := range bots.List() {
							for _, item := range handler.Rules() {
								switch v := item.(type) {
								case []action.Rule:
									for _, rule := range v {
										if rule.Id == actionRuleId {
											botHandler = handler
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
								logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
							}

							if payload != nil {
								botUid := types.ParseUserId(msg.Original)
								botSend(msg.RcptTo, botUid, payload, types.WithContext(ctx))

								// pipeline action stage
								pipelineFlag, _ := types.KV(message.Head).String("x-pipeline-flag")
								pipelineVersion, _ := types.KV(message.Head).Int64("x-pipeline-version")
								nextPipeline(ctx, pipelineFlag, int(pipelineVersion), msg.RcptTo, botUid)
								return
							}
						}
					}
				}
			}
			// command
			if payload == nil {
				var content interface{}
				if msg.Pub.Head == nil {
					content = msg.Pub.Content
				} else {
					// Compatible with drafty
					if m, ok := msg.Pub.Content.(map[string]interface{}); ok {
						if txt, ok := m["txt"]; ok {
							content = txt
						}
					}
				}
				// check "/" prefix
				if in, ok := content.(string); ok && strings.HasPrefix(in, "/") {
					in = strings.Replace(in, "/", "", 1)
					payload, err = handle.Command(ctx, in)
					if err != nil {
						logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
					}

					// stats
					statsInc("BotRunCommandTotal", 1)

					// error message
					if payload == nil {
						payload = types.TextMsg{Text: "error command"}
					}
				}
			}
			// pipeline command trigger
			if payload == nil {
				var content interface{}
				if msg.Pub.Head == nil {
					content = msg.Pub.Content
				} else {
					// Compatible with drafty
					if m, ok := msg.Pub.Content.(map[string]interface{}); ok {
						if txt, ok := m["txt"]; ok {
							content = txt
						}
					}
				}
				// check "~" prefix
				if in, ok := content.(string); ok && strings.HasPrefix(in, "~") {
					var pipelineFlag string
					var pipelineVersion int
					in = strings.Replace(in, "~", "", 1)
					payload, pipelineFlag, pipelineVersion, err = handle.Pipeline(ctx, msg.Pub.Head, in, types.PipelineCommandTriggerOperate)
					if err != nil {
						logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
					}
					ctx.PipelineFlag = pipelineFlag
					ctx.PipelineVersion = pipelineVersion

					// stats
					statsInc("BotTriggerPipelineTotal", 1)

					// error message
					if payload == nil {
						payload = types.TextMsg{Text: "error pipeline"}
					}
				}
			}
			// condition
			if payload == nil {
				if msg.Pub.Head != nil {
					fUid := ""
					fSeq := int64(0)
					if v, ok := msg.Pub.Head["forwarded"]; ok {
						if s, ok := v.(string); ok {
							f := strings.Split(s, ":")
							if len(f) == 2 {
								fUid = f[0]
								fSeq, _ = strconv.ParseInt(f[1], 10, 64)
							}
						}
					}

					if fUid != "" && fSeq > 0 {
						//uid2 := types.ParseUserId(fUid)
						topic := "" // fixme
						message, err := store.Chatbot.GetMessage(topic, int(fSeq))
						if err != nil {
							logs.Err.Println(err)
						}

						if message.ID > 0 {
							src, _ := types.KV(message.Content).Map("src")
							tye, _ := types.KV(message.Content).String("tye")
							d, _ := json.Marshal(src)
							pl := types.ToPayload(tye, d)
							ctx.Condition = tye
							payload, err = handle.Condition(ctx, pl)
							if err != nil {
								logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
							}

							// stats
							statsInc("BotRunConditionTotal", 1)
						}
					}
				}
			}
			// input
			if payload == nil {
				payload, err = handle.Input(ctx, msg.Pub.Head, msg.Pub.Content)
				if err != nil {
					logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
					continue
				}

				// stats
				statsInc("BotRunInputTotal", 1)
			}
		}

		// send message
		if payload == nil {
			continue
		}

		botUid := types.ParseUserId(msg.Original)
		botSend(msg.RcptTo, botUid, payload, types.WithContext(ctx))
	}
}

func groupIncomingMessage(t *Topic, msg *ClientComMessage, event types.GroupEvent) {
	var err error
	var subs []interface{} // fixme
	// check bot user incoming
	for _, sub := range subs {
		if !isBot(sub) {
			continue
		}
		//if strings.TrimPrefix(msg.AsUser, "usr") == sub.User {
		//	return
		//}
	}

	uid := types.ParseUserId(msg.AsUser)
	ctx := types.Context{
		Id:        msg.Id,
		Original:  msg.Original,
		RcptTo:    msg.RcptTo,
		AsUser:    uid,
		AuthLvl:   msg.AuthLvl,
		MetaWhat:  msg.MetaWhat,
		Timestamp: msg.Timestamp,
	}

	// behavior
	bots.Behavior(uid, bots.MessageGroupIncomingBehavior, 1)

	// user auth record

	// bot
	for _, sub := range subs {
		if !isBot(sub) {
			continue
		}

		// bot name
		name := botName(sub)
		handle, ok := bots.List()[name]
		if !ok {
			continue
		}

		if !handle.IsReady() {
			logs.Info.Printf("bot %s unavailable", t.name)
			continue
		}

		var payload types.MsgPayload

		// auth
		if payload == nil {
			// condition
			if msg.Pub != nil && msg.Pub.Head != nil {
				fUid := ""
				fSeq := int64(0)
				if v, ok := msg.Pub.Head["forwarded"]; ok {
					if s, ok := v.(string); ok {
						f := strings.Split(s, ":")
						if len(f) == 2 {
							fUid = f[0]
							fSeq, _ = strconv.ParseInt(f[1], 10, 64)
						}
					}
				}

				if fUid != "" && fSeq > 0 {
					//uid2 := types.ParseUserId(fUid)
					topic := "" // fixme
					message, err := store.Chatbot.GetMessage(topic, int(fSeq))
					if err != nil {
						logs.Err.Println(err)
					}

					if message.ID > 0 {
						src, _ := types.KV(message.Content).Map("src")
						tye, _ := types.KV(message.Content).String("tye")
						d, _ := json.Marshal(src)
						pl := types.ToPayload(tye, d)
						ctx.Condition = tye
						payload, err = handle.Condition(ctx, pl)
						if err != nil {
							logs.Warn.Printf("topic[%s]: failed to run bot: %v", t.name, err)
						}

						// stats
						statsInc("BotRunConditionTotal", 1)
					}
				}
			}
		}

		// group
		if payload == nil {
			ctx.GroupEvent = event
			var head map[string]any
			var content any
			if msg.Pub != nil {
				head = msg.Pub.Head
				content = msg.Pub.Content
			}
			payload, err = handle.Group(ctx, head, content)
			if err != nil {
				logs.Warn.Printf("topic[%s]: failed to run group bot: %v", t.name, err)
				continue
			}

			// stats
			statsInc("BotRunGroupTotal", 1)
		}

		// send message
		if payload == nil {
			continue
		}

		botUid := types.Uid(0) // fixme
		botSend(msg.RcptTo, botUid, payload)
	}
}

func nextPipeline(ctx types.Context, pipelineFlag string, pipelineVersion int, rcptTo string, botUid types.Uid) {
	if pipelineFlag != "" && pipelineVersion > 0 {
		pipelineData, err := store.Chatbot.PipelineGet(ctx.AsUser, ctx.Original, pipelineFlag)
		if err != nil {
			logs.Err.Println(err)
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
							payload, _, _, err := handler.Pipeline(ctx, nil, nil, types.PipelineNextOperate)
							if err != nil {
								logs.Err.Println(err)
								return
							}
							botSend(rcptTo, botUid, payload, types.WithContext(ctx))
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
	//	logs.Err.Println(err)
	//	return
	//}

	//for _, cred := range creds {
	//	rcptTo := tstore.EncodeUid(cred.Userid).P2PName(botUid)
	//	if rcptTo != "" {
	//		botSend(rcptTo, botUid, types.TextMsg{Text: "reboot"})
	//	}
	//}
}

func onlineStatus(usrStr string) {
	//uid := types.ParseUserId(usrStr)
	var err error
	var user *types.User // fixme
	if isBotUser(user) {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("online:%s", usrStr)
	_, err = cache.DB.Get(ctx, key).Result()
	if err == redis.Nil {
		cache.DB.Set(ctx, key, time.Now().Unix(), 30*time.Minute)
	} else if err != nil {
		return
	} else {
		cache.DB.Expire(ctx, key, 30*time.Minute)
	}
}

func sessionCurrent(uid types.Uid, topic string) (model.Session, bool) {
	sess, err := store.Chatbot.SessionGet(uid, topic)
	if err != nil {
		return model.Session{}, false
	}
	return sess, true
}

func errorResponse(rw http.ResponseWriter, text string) {
	rw.WriteHeader(http.StatusBadRequest)
	_, _ = rw.Write([]byte(text))
}

type AsyncMessageConsumer struct {
	name string
}

func NewAsyncMessageConsumer() *AsyncMessageConsumer {
	return &AsyncMessageConsumer{name: "consumer"}
}

func (c *AsyncMessageConsumer) Consume(delivery rmq.Delivery) {
	payload := delivery.Payload()

	var qp types.QueuePayload
	err := json.Unmarshal([]byte(payload), &qp)
	if err != nil {
		if err := delivery.Reject(); err != nil {
			logs.Err.Printf("failed to reject %s: %s\n", payload, err)
			return
		}
		return
	}

	uid := types.ParseUserId(qp.Uid)
	msg := types.ToPayload(qp.Type, qp.Msg)
	botSend(qp.RcptTo, uid, msg)

	if err := delivery.Ack(); err != nil {
		logs.Err.Printf("failed to ack %s: %s\n", payload, err)
		return
	}
}

func linkitAction(uid types.Uid, data types.LinkData) (interface{}, error) {
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
		//		logs.Info.Printf("bot %s unavailable", topic)
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
		//		logs.Warn.Printf("topic[%s]: failed to agent bot: %v", topic, err)
		//		continue
		//	}
		//
		//	// stats
		//	statsInc("BotRunAgentTotal", 1)
		//
		//	// send message
		//	if payload == nil {
		//		continue
		//	}
		//
		//	botSend(uid.P2PName(topicUid), topicUid, payload)
		//}
	case types.Pull:
		list, err := store.Chatbot.ListInstruct(uid, false)
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
		var user *types.User // fixme
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
