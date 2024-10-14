package cron

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/event"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/influxdata/cron"
	"gorm.io/gorm"
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
		flog.Info("cron %s start", r.cronRules[rule].Name)
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
		flog.Error(err)
		return
	}
	nextTime, err := p.Next(time.Now())
	if err != nil {
		flog.Error(err)
		return
	}
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-r.stop:
			flog.Info("cron %s rule worker stopped", rule.Name)
			return
		case <-ticker.C:
			if nextTime.Format("2006-01-02 15:04") != time.Now().Format("2006-01-02 15:04") {
				continue
			}
			msgs := func() []result {
				defer func() {
					if rc := recover(); rc != nil {
						flog.Warn("cron %s ruleWorker recover", rule.Name)
						if v, ok := rc.(error); ok {
							flog.Error(v)
						}
					}
				}()

				// bot user
				// botUid, _, _, _, _ := serverStore.Users.GetAuthUniqueRecord("basic", fmt.Sprintf("%s_bot", r.Type))

				// all normal users
				users, err := store.Database.GetUsers()
				if err != nil {
					flog.Error(err)
					return nil
				}

				var res []result
				for _, user := range users {
					// check subscription
					//uid := types.EncodeUid(int64(user.ID))
					//topic := uid.P2PName(botUid)
					uid := types.Uid(user.Flag)
					topic := "" // fixme

					// get oauth token
					oauth, err := store.Database.OAuthGet(uid, topic, r.Type)
					if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
						continue
					}

					// ctx
					ctx := types.Context{
						Topic:  topic,
						AsUser: uid,
						Token:  oauth.Token,
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
				flog.Error(err)
			}
		}
	}
}

func (r *Ruleset) resultWorker() {
	for {
		select {
		case <-r.stop:
			flog.Info("cron %s result worker stopped", r.Type)
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

	filterKey := fmt.Sprintf("cron:%s:%s:filter", res.name, res.ctx.AsUser)

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
	err := event.SendMessage(context.Background(), res.ctx.AsUser.String(), res.ctx.Topic, res.payload)
	if err != nil {
		flog.Error(err)
	}
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
