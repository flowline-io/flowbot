package dev

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"fmt"
	"github.com/google/uuid"
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/event"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/parser"
	"github.com/sysatom/flowbot/pkg/queue"
	"github.com/sysatom/flowbot/pkg/utils"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var commandRules = []command.Rule{
	{
		Define: "info",
		Help:   `Bot info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return nil
		},
	},
	{
		Define: "setting",
		Help:   `Bot setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.SettingMsg(ctx, Name)
		},
	},
	{
		Define: "webapp",
		Help:   `webapp`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.LinkMsg{Url: bots.AppURL(ctx, Name, nil), Title: "webapp"}
		},
	},
	{
		Define: "rand [number] [number]",
		Help:   `Generate random numbers`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			min, _ := tokens[1].Value.Int64()
			max, _ := tokens[2].Value.Int64()

			nBing, err := rand.Int(rand.Reader, big.NewInt(max+1-min))
			if err != nil {
				logs.Err.Println("bot command rand [number] [number]", err)
				return nil
			}
			t := nBing.Int64() + min

			return types.TextMsg{Text: strconv.FormatInt(t, 10)}
		},
	},
	{
		Define: "id",
		Help:   `Generate random id`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: types.Id().String()}
		},
	},
	{
		Define: "md5 [string]",
		Help:   `md5 encode`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			txt, _ := tokens[1].Value.String()
			return types.TextMsg{Text: utils.MD5(txt)}
		},
	},
	{
		Define: "sha1 [string]",
		Help:   `sha1 encode`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			txt, _ := tokens[1].Value.String()
			return types.TextMsg{Text: utils.SHA1(txt)}
		},
	},
	{
		Define: "uuid",
		Help:   `UUID Generator`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: uuid.New().String()}
		},
	},
	{
		Define: "uid [string]",
		Help:   `Decode UID string`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			str, _ := tokens[1].Value.String()
			var uid types.Uid
			var result string
			err := uid.UnmarshalText([]byte(str))
			if err != nil {
				result = err.Error()
			} else {
				result = fmt.Sprintf("%d", uid)
			}

			return types.TextMsg{Text: result}
		},
	},
	{
		Define: "ts [number]",
		Help:   `timestamp format`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			num, _ := tokens[1].Value.Int64()
			t := time.Unix(num, 0)
			return types.TextMsg{Text: t.Format(time.RFC3339)}
		},
	},
	{
		Define: "echo [any]",
		Help:   "print",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			val := tokens[1].Value.Source
			return types.TextMsg{Text: fmt.Sprintf("%v", val)}
		},
	},
	{
		Define: "form",
		Help:   `[example] form`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.FormMsg(ctx, devFormID)
		},
	},
	{
		Define: "action",
		Help:   "[example] action",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.ActionMsg(ctx, devActionID)
		},
	},
	{
		Define: "guess",
		Help:   "Guess number game",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// rand number
			big, _ := rand.Int(rand.Reader, big.NewInt(1000))

			var initValue types.KV = types.KV{"number": big.Int64()}
			return bots.SessionMsg(ctx, guessSessionID, initValue)
		},
	},
	{
		Define: "plot",
		Help:   `[example] plot graph`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			p := plot.New()

			p.Title.Text = "Plotutil example"
			p.X.Label.Text = "X"
			p.Y.Label.Text = "Y"

			err := plotutil.AddLinePoints(p,
				"First", randomPoints(15),
				"Second", randomPoints(15),
				"Third", randomPoints(15))
			if err != nil {
				panic(err)
			}

			w := bytes.NewBufferString("")

			c := vgimg.New(vg.Points(500), vg.Points(500))
			dc := draw.New(c)
			p.Draw(dc)

			png := vgimg.PngCanvas{Canvas: c}
			if _, err := png.WriteTo(w); err != nil {
				panic(err)
			}

			return types.ImageConvert(w.Bytes(), "Plot", 500, 500)
		},
	},
	{
		Define: "queue",
		Help:   `[example] publish queue`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := queue.AsyncMessage(ctx.RcptTo, ctx.Original, types.TextMsg{Text: time.Now().String()})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "instruct",
		Help:   `[example] create instruct`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			data := types.KV{}
			data["txt"] = "example"
			return bots.InstructMsg(ctx, ExampleInstructID, data)
		},
	},
	{
		Define: "instruct list",
		Help:   `all bot instruct`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			models := make(map[string]interface{})
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
		Define: "markdown",
		Help:   `[example] markdown page`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StorePage(ctx, model.PageMarkdown, "", types.MarkdownMsg{
				Raw: markdownText,
			})
		},
	},
	{
		Define: "event",
		Help:   `fire example event`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			botUid := 0 // fixme
			err := event.Emit(event.SendEvent, types.KV{"topic": ctx.RcptTo, "topic_uid": int64(botUid), "message": "fire send event"})
			if err != nil {
				return types.TextMsg{Text: "error"}
			}
			event.AsyncEmit(event.ExampleEvent, types.KV{"now": time.Now().Unix()})
			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "page",
		Help:   `[example] dev page`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, err := bots.PageURL(ctx, devPageId, nil, 24*time.Hour)
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			return types.LinkMsg{Url: url}
		},
	},
	{
		Define: "json",
		Help:   `JSON Formatter page`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, err := bots.PageURL(ctx, jsonPageId, nil, 24*time.Hour)
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			return types.LinkMsg{Url: url}
		},
	},
}
