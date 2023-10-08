package server

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/workflow/manage"
	"github.com/flowline-io/flowbot/internal/workflow/schedule"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/utils/queue"
)

// init channels
func initializeChannels() error {
	// bind to BotFather
	// uid, _, _, _, err := tstore.Users.GetAuthUniqueRecord("basic", "botfather")
	uid := types.Uid(0) // fixme
	_ = &Session{
		uid:    uid,
		subs:   make(map[string]*Subscription),
		send:   make(chan interface{}, sendQueueLimit+32),
		stop:   make(chan interface{}, 1),
		detach: make(chan string, 64),
	}

	for range channels.List() {
		//topic, _ := tstore.Topics.Get(fmt.Sprintf("grp%s", channel.Id))
		//if topic != nil && topic.Id != "" {
		//	flog.Info("channel %s registered", channel.Name)
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
	//		flog.Error(err)
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
	//	stats.Inc("ChannelPublishTotal", 1)
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
	var workerNum = config.App.Workflow.Worker
	// default worker num
	if workerNum == 0 {
		workerNum = 1
	}
	// manager
	globals.manager = manage.NewManager()
	go globals.manager.Run()
	// scheduler
	q := queue.NewDeltaFIFOWithOptions(queue.DeltaFIFOOptions{
		KeyFunction: schedule.KeyFunc,
	})
	globals.scheduler = schedule.NewScheduler(q)
	go globals.scheduler.Run()
	for i := 0; i < workerNum; i++ {
		worker := schedule.NewWorker(q)
		globals.workers = append(globals.workers, worker)
		go worker.Run()
	}
	return nil
}
