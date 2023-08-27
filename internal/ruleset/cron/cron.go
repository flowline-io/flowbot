package cron

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/influxdata/cron"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/cache"
	"github.com/sysatom/flowbot/pkg/logs"
	"gorm.io/gorm"
	"time"
)

type Rule struct {
	Name   string
	Help   string
	When   string
	Action func(types.Context) []types.MsgPayload
}

type Ruleset struct {
	stop      chan struct{}
	Type      string
	outCh     chan result
	cronRules []Rule

	Send types.SendFunc
}

type result struct {
	name    string
	ctx     types.Context
	payload types.MsgPayload
}

// NewCronRuleset New returns a cron rule set
func NewCronRuleset(name string, rules []Rule) *Ruleset {
	return &Ruleset{
		stop:      make(chan struct{}),
		Type:      name,
		outCh:     make(chan result, 100),
		cronRules: rules,
	}
}

func (r *Ruleset) Daemon() {
	// process cron
	for rule := range r.cronRules {
		logs.Info.Printf("cron %s start", r.cronRules[rule].Name)
		go r.ruleWorker(r.cronRules[rule])
	}

	// result pipeline
	go r.resultWorker()
}

func (r *Ruleset) Shutdown() {
	r.stop <- struct{}{}
}

func (r *Ruleset) ruleWorker(rule Rule) {
	p, err := cron.ParseUTC(rule.When)
	if err != nil {
		logs.Err.Println("cron worker", rule.Name, err)
		return
	}
	nextTime, err := p.Next(time.Now())
	if err != nil {
		logs.Err.Println("cron worker", rule.Name, err)
		return
	}
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-r.stop:
			logs.Info.Printf("cron %s rule worker stopped", rule.Name)
			return
		case <-ticker.C:
			if nextTime.Format("2006-01-02 15:04") != time.Now().Format("2006-01-02 15:04") {
				continue
			}
			msgs := func() []result {
				defer func() {
					if rc := recover(); rc != nil {
						logs.Warn.Printf("cron %s ruleWorker recover", rule.Name)
						if v, ok := rc.(error); ok {
							logs.Err.Println(v)
						}
					}
				}()

				// bot user
				// botUid, _, _, _, _ := serverStore.Users.GetAuthUniqueRecord("basic", fmt.Sprintf("%s_bot", r.Type))
				botUid := types.Uid(0) // fixme

				// all normal users
				users, err := store.Chatbot.GetNormalUsers()
				if err != nil {
					logs.Err.Println(err)
					return nil
				}

				var res []result
				for _, user := range users {
					// check subscription
					//uid := types.EncodeUid(int64(user.ID))
					//topic := uid.P2PName(botUid)
					uid := types.Uid(user.ID)
					topic := "" // fixme

					// get oauth token
					oauth, err := store.Chatbot.OAuthGet(uid, botUid.UserId(), r.Type)
					if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
						continue
					}

					// ctx
					ctx := types.Context{
						Original: botUid.UserId(),
						AsUser:   uid,
						Token:    oauth.Token,
						RcptTo:   topic,
					}

					// run action
					ra := rule.Action(ctx)
					for i := range ra {
						res = append(res, result{
							name:    rule.Name,
							ctx:     ctx,
							payload: ra[i],
						})
					}
				}
				return res
			}()
			if len(msgs) > 0 {
				for _, item := range msgs {
					r.outCh <- item
				}
			}
			nextTime, err = p.Next(time.Now())
			if err != nil {
				logs.Err.Println("cron worker", rule.Name, err)
			}
		}
	}
}

func (r *Ruleset) resultWorker() {
	for {
		select {
		case <-r.stop:
			logs.Info.Printf("cron %s result worker stopped", r.Type)
			return
		case out := <-r.outCh:
			// filter
			res := r.filter(out)
			// pipeline
			r.pipeline(res)
		}
	}
}

func (r *Ruleset) filter(res result) result {
	// user auth record

	filterKey := fmt.Sprintf("cron:%s:%s:filter", res.name, res.ctx.AsUser.UserId())

	// content hash
	d := un(res.payload)
	s := sha1.New()
	_, _ = s.Write(d)
	hash := s.Sum(nil)

	ctx := context.Background()
	state := cache.DB.SIsMember(ctx, filterKey, hash).Val()
	if state {
		return result{}
	}

	_ = cache.DB.SAdd(ctx, filterKey, hash)
	return res
}

func (r *Ruleset) pipeline(res result) {
	if res.payload == nil {
		return
	}
	r.Send(res.ctx.RcptTo, types.ParseUserId(res.ctx.Original), res.payload)
}

func un(payload types.MsgPayload) []byte {
	switch v := payload.(type) {
	case types.TextMsg:
		return []byte(v.Text)
	case types.InfoMsg:
		return []byte(v.Title)
	case types.RepoMsg:
		return []byte(*v.FullName)
	case types.LinkMsg:
		return []byte(v.Url)
	}
	return nil
}
