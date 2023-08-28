package server

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/workflow/manager"
	"github.com/flowline-io/flowbot/internal/workflow/scheduler"
	"github.com/flowline-io/flowbot/internal/workflow/worker"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/utils"
	"strings"
	"time"
)

// init base bot user
func initializeBotFather() error {
	msg := &ClientComMessage{
		Acc: &MsgClientAcc{
			User:      "new",
			State:     "ok",
			AuthLevel: "auth",
			Scheme:    "basic",
			Secret:    []byte(fmt.Sprintf("%s:170953280278461931", BotFather)),
			Login:     false,
			Tags:      []string{"bot"},
			Desc: &MsgSetDesc{
				DefaultAcs: nil,
				Public: map[string]interface{}{
					"fn": BotFather,
				},
				Trusted: map[string]interface{}{
					"staff": true,
				},
				Private: nil,
			},
		},
	}

	//var user types.User
	//var private interface{}

	// Assign actual access values, public and private.
	if msg.Acc.Desc != nil {
		if !utils.IsNullValue(msg.Acc.Desc.Public) {
			//user.Public = msg.Acc.Desc.Public
		}
		if !utils.IsNullValue(msg.Acc.Desc.Trusted) {
			//user.Trusted = msg.Acc.Desc.Trusted
		}
		if !utils.IsNullValue(msg.Acc.Desc.Private) {
			//private = msg.Acc.Desc.Private
		}
	}

	// Create user record in the database.
	//if _, err := tstore.Users.Create(&user, private); err != nil {
	//	return fmt.Errorf("create bot user: failed to create bot user, %s", err)
	//}

	// Create or update validation record in DB.
	//value := strings.ToLower(fmt.Sprintf("%s@bot.system", BotFather))
	//_, err := tstore.Users.UpsertCred(&types.Credential{
	//	User:   user.Uid().String(),
	//	Method: "email",
	//	Value:  value,
	//	Resp:   "000000",
	//	Done:   true,
	//})
	//if err != nil {
	//	return fmt.Errorf("create credential record error %s (%s)", value, err)
	//}

	return nil
}

// init bot users
func initializeBotUsers() error {
	var msgs []*ClientComMessage

	for name := range bots.List() {
		msgs = append(msgs, &ClientComMessage{
			Acc: &MsgClientAcc{
				User:      "new",
				AuthLevel: "auth",
				Scheme:    "basic",
				Secret:    []byte(fmt.Sprintf("%s%s:%d", name, bots.BotNameSuffix, time.Now().Unix())),
				Login:     false,
				Tags:      []string{"bot", name},
				Desc: &MsgSetDesc{
					Public: map[string]interface{}{
						"fn": fmt.Sprintf("%s%s", name, bots.BotNameSuffix),
					},
					Trusted: map[string]interface{}{
						"verified": true,
					},
				},
			},
		})
	}

	for _, msg := range msgs {
		// Check if login is unique.

		//var user types.User
		//var private interface{}

		//state, err := types.NewObjState("ok")
		//if err != nil {
		//	return err
		//}
		//user.State = state

		// Assign actual access values, public and private.
		if msg.Acc.Desc != nil {
			if !utils.IsNullValue(msg.Acc.Desc.Public) {
				//user.Public = msg.Acc.Desc.Public
			}
			if !utils.IsNullValue(msg.Acc.Desc.Trusted) {
				//user.Trusted = msg.Acc.Desc.Trusted
			}
			if !utils.IsNullValue(msg.Acc.Desc.Private) {
				//private = msg.Acc.Desc.Private
			}
		}

		// Create user record in the database.
		//if _, err := tstore.Users.Create(&user, private); err != nil {
		//	return fmt.Errorf("create bot user: failed to create bot user, %s", err)
		//}

		// Add authentication record. The authhdl.AddRecord may change tags.

		// Create or update validation record in DB.
		secret := string(msg.Acc.Secret)
		splitAt := strings.Index(secret, ":")
		if splitAt < 0 {
			return fmt.Errorf("secret split error %s", msg.Acc.Secret)
		}
		//uname := strings.ToLower(secret[:splitAt])
		//value := strings.ToLower(fmt.Sprintf("%s@bot.system", uname))

		//_, err = tstore.Users.UpsertCred(&types.Credential{
		//	User:   user.Uid().String(),
		//	Method: "email",
		//	Value:  value,
		//	Resp:   "000000",
		//	Done:   true,
		//})
		//if err != nil {
		//	return fmt.Errorf("create credential record error %s (%s)", value, err)
		//}
	}
	return nil
}

// init channels
func initializeChannels() error {
	// bind to BotFather
	// uid, _, _, _, err := tstore.Users.GetAuthUniqueRecord("basic", "botfather")
	uid := types.Uid(0) // fixme
	sess := &Session{
		uid:    uid,
		subs:   make(map[string]*Subscription),
		send:   make(chan interface{}, sendQueueLimit+32),
		stop:   make(chan interface{}, 1),
		detach: make(chan string, 64),
	}
	fmt.Println(sess)

	for range channels.List() {
		//topic, _ := tstore.Topics.Get(fmt.Sprintf("grp%s", channel.Id))
		//if topic != nil && topic.Id != "" {
		//	logs.Info.Printf("channel %s registered", channel.Name)
		//	continue
		//}

		//var msg = &ClientComMessage{
		//	Sub: &MsgClientSub{
		//		Topic: channel.Name,
		//		Set: &MsgSetQuery{
		//			Desc: &MsgSetDesc{
		//				Public: map[string]interface{}{
		//					"fn":   fmt.Sprintf("%s%s", channel.Name, channels.ChannelNameSuffix),
		//					"note": fmt.Sprintf("%s channel", channel.Name),
		//				},
		//				Trusted: map[string]interface{}{
		//					"verified": true,
		//				},
		//			},
		//			Tags: []string{"channel"},
		//		},
		//		Created: false,
		//		Newsub:  false,
		//	},
		//
		//	Original:  fmt.Sprintf("nch%s", channel.Id),
		//	RcptTo:    fmt.Sprintf("grp%s", channel.Id),
		//	AsUser:    uid.UserId(),
		//	AuthLvl:   int(auth.LevelRoot),
		//	Timestamp: time.Now(),
		//	init:      true,
		//	sess:      sess,
		//}
		//
		//globals.hub.join <- msg
	}

	return nil
}

// init crawler
func initializeCrawler() error {
	//uid, _, _, _, err := store.Users.GetAuthUniqueRecord("basic", "botfather")
	//if err != nil {
	//	return err
	//}
	//
	//c := crawler.New()
	//globals.crawler = c
	//c.Send = func(id, name string, out []map[string]string) {
	//	if len(out) == 0 {
	//		return
	//	}
	//	topic := fmt.Sprintf("grp%s", id)
	//	dst, err := store.Topics.Get(topic)
	//	if err != nil {
	//		logs.Err.Println("init crawler", err)
	//		return
	//	}
	//	if dst == nil {
	//		return
	//	}
	//
	//	keys := []string{"No"}
	//	for k := range out[0] {
	//		keys = append(keys, k)
	//	}
	//	var head map[string]interface{}
	//	var content interface{}
	//	if len(out) <= 10 {
	//		sort.Strings(keys)
	//		builder := extraTypes.MsgBuilder{}
	//		for index, item := range out {
	//			builder.AppendTextLine(fmt.Sprintf("--- %d ---", index+1), extraTypes.TextOption{})
	//			for _, k := range keys {
	//				if k == "No" {
	//					continue
	//				}
	//				builder.AppendText(fmt.Sprintf("%s: ", k), extraTypes.TextOption{IsBold: true})
	//				if utils.IsUrl(item[k]) {
	//					builder.AppendTextLine(item[k], extraTypes.TextOption{IsLink: true})
	//				} else {
	//					builder.AppendTextLine(item[k], extraTypes.TextOption{})
	//				}
	//			}
	//		}
	//		head, content = builder.Content()
	//	} else {
	//		var row [][]interface{}
	//		for index, item := range out {
	//			var tmp []interface{}
	//			for _, k := range keys {
	//				if k == "No" {
	//					tmp = append(tmp, index+1)
	//					continue
	//				}
	//				tmp = append(tmp, item[k])
	//			}
	//			row = append(row, tmp)
	//		}
	//		title := fmt.Sprintf("Channel %s (%d)", name, len(out))
	//		res := bots.StorePage(extraTypes.Context{}, model.PageTable, title, extraTypes.TableMsg{
	//			Title:  title,
	//			Header: keys,
	//			Row:    row,
	//		})
	//		head, content = res.Convert()
	//	}
	//	if content == nil {
	//		return
	//	}
	//
	//	// stats inc
	//	statsInc("ChannelPublishTotal", 1)
	//
	//	msg := &ClientComMessage{
	//		Pub: &MsgClientPub{
	//			Topic:   topic,
	//			Head:    head,
	//			Content: content,
	//		},
	//		AsUser:    uid.UserId(),
	//		Timestamp: types.TimeNow(),
	//	}
	//
	//	t := &Topic{
	//		name:   topic,
	//		cat:    types.TopicCatGrp,
	//		status: topicStatusLoaded,
	//		lastID: dst.SeqId,
	//		perUser: map[types.Uid]perUserData{
	//			uid: {
	//				modeGiven: types.ModeCFull,
	//				modeWant:  types.ModeCFull,
	//				private:   nil,
	//			},
	//		},
	//	}
	//	t.handleClientMsg(msg)
	//}
	//
	//var rules []crawler.Rule
	//for _, publisher := range channels.List() {
	//	rules = append(rules, *publisher)
	//}
	//
	//err = c.Init(rules...)
	//if err != nil {
	//	return err
	//}
	//c.Run()
	return nil
}

// init workflow
func initializeWorkflow() error {
	ctx := context.Background()
	globals.manager = manager.NewManager()
	go globals.manager.Run(ctx)
	globals.scheduler = scheduler.NewScheduler()
	go globals.scheduler.Run(ctx)
	globals.worker = worker.NewWorker()
	go globals.worker.Run(ctx)
	return nil
}
