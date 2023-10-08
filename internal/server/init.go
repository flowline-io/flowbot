package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/workflow/manage"
	"github.com/flowline-io/flowbot/internal/workflow/schedule"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/channels/crawler"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/queue"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"sort"
)

// init channels
func registerChannels() error {
	// register channels
	registerChannels := sets.NewString()
	for name, handler := range channels.List() {
		registerChannels.Insert(name)

		state := model.ChannelInactive
		if handler.Enable {
			state = model.ChannelActive
		}
		channel, _ := store.Chatbot.GetChannelByName(name)
		if channel == nil {
			channel = &model.Channel{
				Name:  name,
				State: state,
			}
			if _, err := store.Chatbot.CreateChannel(channel); err != nil {
				flog.Error(err)
			}
		} else {
			channel.State = state
			err := store.Chatbot.UpdateChannel(channel)
			if err != nil {
				flog.Error(err)
			}
		}
	}

	// inactive channels
	list, err := store.Chatbot.GetChannels()
	if err != nil {
		flog.Error(err)
	}
	for _, channel := range list {
		if !registerChannels.Has(channel.Name) {
			channel.State = model.ChannelInactive
			if err := store.Chatbot.UpdateChannel(channel); err != nil {
				flog.Error(err)
			}
		}
	}

	return nil
}

// init crawler
func initializeCrawler() error {
	c := crawler.New()
	globals.crawler = c
	c.Send = func(id, name string, out []map[string]string) {
		if len(out) == 0 {
			return
		}

		// todo find topic
		fmt.Println(id)

		keys := []string{"No"}
		for k := range out[0] {
			keys = append(keys, k)
		}

		var content interface{}
		if len(out) <= 10 {
			sort.Strings(keys)
			builder := types.MsgBuilder{}
			for index, item := range out {
				builder.AppendTextLine(fmt.Sprintf("--- %d ---", index+1), types.TextOption{})
				for _, k := range keys {
					if k == "No" {
						continue
					}
					builder.AppendText(fmt.Sprintf("%s: ", k), types.TextOption{IsBold: true})
					if utils.IsUrl(item[k]) {
						builder.AppendTextLine(item[k], types.TextOption{IsLink: true})
					} else {
						builder.AppendTextLine(item[k], types.TextOption{})
					}
				}
			}
			_, content = builder.Content()
		} else {
			var row [][]interface{}
			for index, item := range out {
				var tmp []interface{}
				for _, k := range keys {
					if k == "No" {
						tmp = append(tmp, index+1)
						continue
					}
					tmp = append(tmp, item[k])
				}
				row = append(row, tmp)
			}
			title := fmt.Sprintf("Channel %s (%d)", name, len(out))
			res := bots.StorePage(types.Context{}, model.PageTable, title, types.TableMsg{
				Title:  title,
				Header: keys,
				Row:    row,
			})
			_, content = res.Convert()
		}
		if content == nil {
			return
		}

		// stats inc
		stats.Inc("ChannelPublishTotal", 1)

		// todo send content
		fmt.Println("channel publish", content)
	}

	var rules []crawler.Rule
	for _, publisher := range channels.List() {
		rules = append(rules, *publisher)
	}

	err := c.Init(rules...)
	if err != nil {
		return err
	}
	c.Run()
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

// init bots
func registerBot() {
	// register bots
	registerBots := sets.NewString()
	for name, handler := range bots.List() {
		registerBots.Insert(name)

		state := model.BotInactive
		if handler.IsReady() {
			state = model.BotActive
		}
		bot, _ := store.Chatbot.GetBotByName(name)
		if bot == nil {
			bot = &model.Bot{
				Name:  name,
				State: state,
			}
			if _, err := store.Chatbot.CreateBot(bot); err != nil {
				flog.Error(err)
			}
		} else {
			bot.State = state
			err := store.Chatbot.UpdateBot(bot)
			if err != nil {
				flog.Error(err)
			}
		}
	}

	// inactive bot
	list, err := store.Chatbot.GetBots()
	if err != nil {
		flog.Error(err)
	}
	for _, bot := range list {
		if !registerBots.Has(bot.Name) {
			bot.State = model.BotInactive
			if err := store.Chatbot.UpdateBot(bot); err != nil {
				flog.Error(err)
			}
		}
	}
}
