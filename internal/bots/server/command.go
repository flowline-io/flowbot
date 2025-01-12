package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/adguard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/version"
	"github.com/redis/go-redis/v9"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var commandRules = []command.Rule{
	{
		Define: "version",
		Help:   `Version`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("%s (%s)", version.Buildtags, version.Buildstamp)}
		},
	},
	{
		Define: "vars",
		Help:   `vars url`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.LinkMsg{
				Title: "Vars Url",
				Url:   fmt.Sprintf("%s/debug/vars", types.AppUrl()),
			}
		},
	},
	{
		Define: "mem stats",
		Help:   `App memory stats`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			return types.InfoMsg{
				Title: "Memory stats",
				Model: memStats,
			}
		},
	},
	{
		Define: "golang stats",
		Help:   `App golang stats`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			numGoroutine := runtime.NumGoroutine()

			return types.InfoMsg{
				Title: "Golang stats",
				Model: types.KV{
					"NumGoroutine": numGoroutine,
				},
			}
		},
	},
	{
		Define: "server stats",
		Help:   `Server stats`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			data, err := store.Database.DataGet(ctx.AsUser, ctx.Topic, "stats")
			if err != nil {
				return types.TextMsg{Text: "Empty server stats"}
			}

			return types.InfoMsg{
				Title: "Server stats",
				Model: data,
			}
		},
	},
	{
		Define: "online stats",
		Help:   `Online stats`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			ctx_ := context.Background()
			keys, err := cache.DB.Keys(ctx_, "online:*").Result()
			if err != nil {
				if !errors.Is(err, redis.Nil) {
					flog.Error(err)
				}
				return types.TextMsg{Text: "Empty"}
			}

			var texts []string
			texts = append(texts, fmt.Sprintf("online %d", len(keys)))
			for _, key := range keys {
				t, err := cache.DB.Get(ctx_, key).Result()
				if err != nil {
					continue
				}
				sec, _ := strconv.ParseInt(t, 10, 64)
				texts = append(texts, fmt.Sprintf("%s -> %s", key, time.Unix(sec, 0).Format(time.RFC3339)))
			}

			return types.KVMsg{
				"list": texts,
			}
		},
	},
	{
		Define: "instruct list",
		Help:   `all bot instruct`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			models := make(types.KV)
			for name, bot := range bots.List() {
				ruleset, _ := bot.Instruct()
				for _, rule := range ruleset {
					models[fmt.Sprintf("(%s) %s", name, rule.Id)] = fmt.Sprintf("[%s]", strings.Join(rule.Args, ","))
				}
			}
			return types.InfoMsg{
				Title: "Instruct",
				Model: models,
			}
		},
	},
	{
		Define: "adguard status",
		Help:   `get adguard home status`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(adguard.ID, adguard.EndpointKey)
			username, _ := providers.GetConfig(adguard.ID, adguard.UsernameKey)
			password, _ := providers.GetConfig(adguard.ID, adguard.PasswordKey)
			client := adguard.NewAdGuardHome(endpoint.String(), username.String(), password.String())

			resp, err := client.GetStatus()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("adguard home status %+v", resp)}
		},
	},
	{
		Define: "adguard stats",
		Help:   `get adguard home statistics`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(adguard.ID, adguard.EndpointKey)
			username, _ := providers.GetConfig(adguard.ID, adguard.UsernameKey)
			password, _ := providers.GetConfig(adguard.ID, adguard.PasswordKey)
			client := adguard.NewAdGuardHome(endpoint.String(), username.String(), password.String())

			resp, err := client.GetStats()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("adguard home dns queries %d, blocked filtering %d, avg processing time %v ms",
				resp.NumDnsQueries, resp.NumBlockedFiltering, int(*resp.AvgProcessingTime)*1000)}
		},
	},
	{
		Define: "queue stats",
		Help:   `Queue Stats page`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.LinkMsg{Title: "Queue Stats", Url: fmt.Sprintf("%s/queue/stats", types.AppUrl())}
		},
	},
	{
		Define: "check",
		Help:   `self inspection`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// todo
			return types.TextMsg{Text: "ok"}
		},
	},
}
