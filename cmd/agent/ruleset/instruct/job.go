package instruct

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct/bot"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

type instructJob struct {
	cache *cache.Cache
}

func (j *instructJob) Run(ctx context.Context) error {
	res, err := client.Pull()
	if err != nil {
		flog.Error(err)
		return err
	}
	if res == nil {
		return nil
	}

	// instruct loop
	for _, item := range res.Instruct {
		// check has been run
		_, has := j.cache.Get(item.No)
		if has {
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

	return nil
}

func RunInstruct(c *cache.Cache, item client.Instruct) error {
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
			ok := c.Set(item.No, "1", 1)
			if !ok {
				return errors.New("set cache failed")
			}
		}
	}
	return nil
}
