package crawler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flc1125/go-cron/v4"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/redis/go-redis/v9"
)

type Crawler struct {
	jobs  map[string]Rule
	outCh chan Result
	stop  chan struct{}

	Send func(id, name string, out []map[string]string)
}

func New() *Crawler {
	return &Crawler{
		jobs:  make(map[string]Rule),
		outCh: make(chan Result, 10),
		stop:  make(chan struct{}),
	}
}

func (s *Crawler) Init(rules ...Rule) error {
	for _, r := range rules {
		// check
		if r.Name == "" {
			continue
		}
		if r.When == "" {
			continue
		}
		if r.Page != nil {
			if !utils.IsUrl(r.Page.URL) {
				continue
			}
		}
		if r.Json != nil {
			if !utils.IsUrl(r.Json.URL) {
				continue
			}
		}

		s.jobs[r.Name] = r
	}
	return nil
}

func (s *Crawler) Run() {
	flog.Debug("crawler starting...")

	for name, job := range s.jobs {
		go s.ruleWorker(name, job)
	}

	go s.resultWorker()
}

func (s *Crawler) Shutdown() {
	s.stop <- struct{}{}
}

func (s *Crawler) ruleWorker(name string, r Rule) {
	flog.Debug("crawler %s start", name)
	p := cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	schedule, err := p.Parse(r.When)
	if err != nil {
		flog.Error(err)
		return
	}
	nextTime := schedule.Next(time.Now())

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-s.stop:
			flog.Info("crawler %s rule worker stopped", name)
			return
		case <-ticker.C:
			if nextTime.Format("2006-01-02 15:04") != time.Now().Format("2006-01-02 15:04") {
				continue
			}
			result := func() []map[string]string {
				defer func() {
					if r := recover(); r != nil {
						_, _ = os.Stderr.WriteString(fmt.Sprintf("panic: %v\n%s\n", r, debug.Stack())) //nolint:errcheck // This will never fail
						if v, ok := r.(error); ok {
							flog.Error(v)
						}
					}
				}()
				return r.Run()
			}()
			if len(result) > 0 {
				s.outCh <- Result{
					Name:   name,
					ID:     r.Id,
					Mode:   r.Mode,
					Result: result,
				}
			}
			nextTime = schedule.Next(time.Now())
		}
	}
}

func (s *Crawler) resultWorker() {
	for {
		select {
		case out := <-s.outCh:
			// filter
			diff := s.filter(out.Name, out.Mode, out.Result)
			// send
			s.Send(out.ID, out.Name, diff)
		case <-s.stop:
			flog.Info("crawler result worker stopped")
			return
		}
	}
}

func (s *Crawler) filter(name, mode string, latest []map[string]string) []map[string]string {
	ctx := context.Background()
	sentKey := fmt.Sprintf("crawler:%s:sent", name)
	todoKey := fmt.Sprintf("crawler:%s:todo", name)
	sendTimeKey := fmt.Sprintf("crawler:%s:sendtime", name)

	// sent
	oldArr, err := rdb.Client.SMembers(ctx, sentKey).Result()
	if err != nil {
		return nil
	}
	var old []map[string]string
	for _, item := range oldArr {
		var tmp map[string]string
		_ = sonic.Unmarshal([]byte(item), &tmp)
		if tmp != nil {
			old = append(old, tmp)
		}
	}

	// to do
	todoArr, err := rdb.Client.SMembers(ctx, todoKey).Result()
	if err != nil {
		return nil
	}
	var todo []map[string]string
	for _, item := range todoArr {
		var tmp map[string]string
		_ = sonic.Unmarshal([]byte(item), &tmp)
		if tmp != nil {
			todo = append(todo, tmp)
		}
	}

	// merge
	old = append(old, todo...)

	// diff
	diff := stringSliceDiff(latest, old)

	switch mode {
	case "instant":
		_ = rdb.Client.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), redis.KeepTTL)
	case "daily":
		sendString, err := rdb.Client.Get(ctx, sendTimeKey).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			flog.Error(err)
		}
		oldSend := int64(0)
		if len(sendString) != 0 {
			oldSend, _ = strconv.ParseInt(sendString, 10, 64)
		}

		if time.Now().Unix()-oldSend < 24*60*60 {
			for _, item := range diff {
				d, _ := sonic.Marshal(item)
				_ = rdb.Client.SAdd(ctx, todoKey, d)
			}

			return nil
		}

		diff = append(diff, todo...)

		_ = rdb.Client.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), redis.KeepTTL)
	default:
		return nil
	}

	// add data
	for _, item := range diff {
		d, _ := sonic.Marshal(item)
		_ = rdb.Client.SAdd(ctx, sentKey, d)
	}

	// clear to do
	_ = rdb.Client.Del(ctx, todoKey)

	return diff
}

func stringSliceDiff(s1, s2 []map[string]string) []map[string]string {
	if len(s1) == 0 {
		return s2
	}

	mb := make(map[string]struct{}, len(s2))
	for _, x := range s2 {
		hash := mapHash(x)
		mb[hash] = struct{}{}
	}
	var diff []map[string]string
	for _, x := range s1 {
		hash := mapHash(x)
		if _, ok := mb[hash]; !ok {
			diff = append(diff, x)
		}
	}
	return diff
}

func mapHash(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	txt := bytes.Buffer{}
	for _, key := range keys {
		_, _ = txt.WriteString(key)
		_, _ = txt.WriteString(":")
		_, _ = txt.WriteString(m[key])
	}
	h := sha1.New()
	_, _ = h.Write(txt.Bytes())
	return string(h.Sum(nil))
}
