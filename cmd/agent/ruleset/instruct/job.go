package instruct

import (
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct/bot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

type instructJob struct {
	cache *bigcache.BigCache
}

func (j *instructJob) Run() {
	res, err := client.Pull()
	if err != nil {
		flog.Error(err)
		return
	}
	if res == nil {
		return
	}

	// instruct loop
	for _, item := range res.Instruct {
		// check has been run
		has, _ := j.cache.Get(item.No)
		if len(has) > 0 {
			continue
		}
		// check expired
		expiredAt, err := time.Parse("2006-01-02T15:04:05Z", item.ExpireAt)
		if err != nil {
			continue
		}
		if time.Now().After(expiredAt) {
			continue
		}
		err = RunInstruct(j.cache, item)
		if err != nil {
			flog.Error(fmt.Errorf("instruct run job failed %s %s %s", item.Bot, item.No, err))
		}
	}
}

func RunInstruct(cache *bigcache.BigCache, item client.Instruct) error {
	for id, dos := range bot.DoInstruct {
		if item.Bot != id {
			continue
		}
		for _, do := range dos {
			if item.Flag != do.Flag {
				continue
			}
			// run instruct
			data := types.KV{}
			if v, ok := item.Content.(map[string]any); ok {
				data = v
			}
			err := do.Run(data)
			if err != nil {
				return err
			}
			err = client.Ack(item.No)
			if err != nil {
				return err
			}
			flog.Info("[instruct] %s %s ack", item.Bot, item.No)
			err = cache.Set(item.No, []byte("1"))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
