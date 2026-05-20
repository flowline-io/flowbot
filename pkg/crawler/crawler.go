package crawler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flc1125/go-cron/v4"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

type Crawler struct {
	jobs  map[string]Rule
	outCh chan Result
	stop  chan struct{}

	store *cache.RedisStore
	Send  func(id, name string, out []map[string]string)
}

func New(store *cache.RedisStore) *Crawler {
	return &Crawler{
		jobs:  make(map[string]Rule),
		outCh: make(chan Result, 10),
		stop:  make(chan struct{}),
		store: store,
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
						_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", r, debug.Stack()) //nolint:errcheck // This will never fail
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
	sentKey := cache.NewKey("crawler", "sent", name)
	todoKey := cache.NewKey("crawler", "todo", name)
	sendTimeKey := cache.NewKey("crawler", "sendtime", name)

	oldArr, err := s.store.Members(ctx, sentKey)
	if err != nil {
		flog.Error(err)
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

	todoArr, err := s.store.Members(ctx, todoKey)
	if err != nil {
		flog.Error(err)
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

	old = append(old, todo...)

	diff := stringSliceDiff(latest, old)

	switch mode {
	case "instant":
		_ = s.store.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLMedium)
	case "daily":
		sendString, _, err := s.store.Get(ctx, sendTimeKey)
		if err != nil {
			flog.Error(err)
		}
		oldSend := int64(0)
		if sendString != "" {
			oldSend, _ = strconv.ParseInt(sendString, 10, 64)
		}

		if time.Now().Unix()-oldSend < 24*60*60 {
			for _, item := range diff {
				d, _ := sonic.Marshal(item)
				_, _ = s.store.Add(ctx, todoKey, cache.TTLDay, string(d))
			}

			return nil
		}

		diff = append(diff, todo...)

		_ = s.store.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLMedium)
	default:
		return nil
	}

	for _, item := range diff {
		d, _ := sonic.Marshal(item)
		_, _ = s.store.Add(ctx, sentKey, cache.TTLMonth, string(d))
	}

	_ = s.store.Clear(ctx, todoKey)

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
	slices.Sort(keys)

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
