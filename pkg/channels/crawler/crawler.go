package crawler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/influxdata/cron"
	"github.com/redis/go-redis/v9"
	"github.com/sysatom/flowbot/pkg/cache"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/utils"
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
	logs.Info.Println("crawler starting...")

	for name, job := range s.jobs {
		go s.ruleWorker(name, job)
	}

	go s.resultWorker()
}

func (s *Crawler) Shutdown() {
	s.stop <- struct{}{}
}

func (s *Crawler) ruleWorker(name string, r Rule) {
	logs.Info.Printf("crawler %s start", name)
	p, err := cron.ParseUTC(r.When)
	if err != nil {
		logs.Err.Println(err, name)
		return
	}
	nextTime, err := p.Next(time.Now())
	if err != nil {
		logs.Err.Println(err, name)
		return
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-s.stop:
			logs.Info.Printf("crawler %s rule worker stopped", name)
			return
		case <-ticker.C:
			if nextTime.Format("2006-01-02 15:04") != time.Now().Format("2006-01-02 15:04") {
				continue
			}
			result := func() []map[string]string {
				defer func() {
					if r := recover(); r != nil {
						logs.Warn.Printf("crawler %s ruleWorker recover ", name)
						if v, ok := r.(error); ok {
							logs.Err.Println(v, name)
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
			nextTime, err = p.Next(time.Now())
			if err != nil {
				logs.Err.Println(err, name)
			}
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
			logs.Info.Println("crawler result worker stopped")
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
	oldArr, err := cache.DB.SMembers(ctx, sentKey).Result()
	if err != nil {
		return nil
	}
	var old []map[string]string
	for _, item := range oldArr {
		var tmp map[string]string
		_ = json.Unmarshal([]byte(item), &tmp)
		if tmp != nil {
			old = append(old, tmp)
		}
	}

	// to do
	todoArr, err := cache.DB.SMembers(ctx, todoKey).Result()
	if err != nil {
		return nil
	}
	var todo []map[string]string
	for _, item := range todoArr {
		var tmp map[string]string
		_ = json.Unmarshal([]byte(item), &tmp)
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
		_ = cache.DB.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), redis.KeepTTL)
	case "daily":
		sendString, err := cache.DB.Get(ctx, sendTimeKey).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			logs.Err.Println(err)
		}
		oldSend := int64(0)
		if len(sendString) != 0 {
			oldSend, _ = strconv.ParseInt(sendString, 10, 64)
		}

		if time.Now().Unix()-oldSend < 24*60*60 {
			for _, item := range diff {
				d, _ := json.Marshal(item)
				_ = cache.DB.SAdd(ctx, todoKey, d)
			}

			return nil
		}

		diff = append(diff, todo...)

		_ = cache.DB.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), redis.KeepTTL)
	default:
		return nil
	}

	// add data
	for _, item := range diff {
		d, _ := json.Marshal(item)
		_ = cache.DB.SAdd(ctx, sentKey, d)
	}

	// clear to do
	_ = cache.DB.Del(ctx, todoKey)

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
		txt.WriteString(key)
		txt.WriteString(":")
		txt.WriteString(m[key])
	}
	h := sha1.New()
	h.Write(txt.Bytes())
	return string(h.Sum(nil))
}
