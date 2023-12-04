package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/version"
	"github.com/redis/go-redis/v9"
	"runtime"
	"strconv"
	"time"
)

var commandRules = []command.Rule{
	{
		Define: "version",
		Help:   `Version`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("v%s(%s)", version.CurrentVersion, version.Buildstamp)}
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
			data, err := store.Chatbot.DataGet(ctx.AsUser, ctx.Original, "stats")
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

			return types.TextListMsg{Texts: texts}
		},
	},
}
