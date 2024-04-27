package dev

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/adguard"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/executer"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/google/uuid"
	"github.com/montanaflynn/stats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
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
			minNum, _ := tokens[1].Value.Int64()
			maxNum, _ := tokens[2].Value.Int64()

			nBing, err := rand.Int(rand.Reader, big.NewInt(maxNum+1-minNum))
			if err != nil {
				flog.Error(err)
				return nil
			}
			t := nBing.Int64() + minNum

			return types.TextMsg{Text: strconv.FormatInt(t, 10)}
		},
	},
	{
		Define: "id",
		Help:   `Generate random id`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: types.Id()}
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

			var initValue = types.KV{"number": big.Int64()}
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
		Help:   `[example] publish mq and task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := event.SendMessage(ctx.RcptTo, ctx.Original, types.TextMsg{Text: time.Now().String()})
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
			//err := event.PublishMessage(event.SendEvent, types.KV{"topic": ctx.RcptTo, "bot": Name, "message": "fire send event"})
			//if err != nil {
			//	return types.TextMsg{Text: "error"}
			//}
			//err = event.PublishMessage(event.ExampleEvent, types.KV{"now": time.Now().Unix()})
			//if err != nil {
			//	return types.TextMsg{Text: "error"}
			//}
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
	{
		Define: "docker",
		Help:   `run docker image`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flog.Debug("start docker command")
			engine := executer.New()
			task := &types.Task{
				ID:    utils.NewUUID(),
				Image: "ubuntu:mantic",
				Run:   "echo -n hello > $OUTPUT",
			}
			err := engine.Run(context.Background(), task)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}
			flog.Debug("docker command result %v", task.Result)
			return types.TextMsg{Text: task.Result}
		},
	},
	{
		Define: "workflow stat",
		Help:   `workflow job statisticians`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			jobs, err := store.Database.GetJobsByState(model.JobSucceeded)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}
			steps, err := store.Database.GetStepsByState(model.StepSucceeded)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}

			jobElapsed := make([]float64, 0, len(jobs))
			for _, job := range jobs {
				if job.StartedAt == nil || job.EndedAt == nil {
					continue
				}
				elapsed := job.EndedAt.Sub(*job.StartedAt).Seconds()
				if elapsed < 0 {
					continue
				}
				jobElapsed = append(jobElapsed, elapsed)
			}

			stepElapsed := make([]float64, 0, len(steps))
			for _, step := range steps {
				if step.StartedAt == nil || step.EndedAt == nil {
					continue
				}
				elapsed := step.EndedAt.Sub(*step.StartedAt).Seconds()
				if elapsed < 0 {
					continue
				}
				stepElapsed = append(stepElapsed, elapsed)
			}

			str := strings.Builder{}
			minVal, _ := stats.Min(jobElapsed)
			medianVal, _ := stats.Median(jobElapsed)
			maxVal, _ := stats.Max(jobElapsed)
			avgVal, _ := stats.Mean(jobElapsed)
			varVal, _ := stats.Variance(jobElapsed)
			str.WriteString(fmt.Sprintf("Jobs total %d, min: %f, median: %f, max: %f, avg: %f, variance: %f \n",
				len(jobElapsed), minVal, medianVal, maxVal, avgVal, varVal))

			minVal, _ = stats.Min(stepElapsed)
			medianVal, _ = stats.Median(stepElapsed)
			maxVal, _ = stats.Max(stepElapsed)
			avgVal, _ = stats.Mean(stepElapsed)
			varVal, _ = stats.Variance(stepElapsed)
			str.WriteString(fmt.Sprintf("Steps total %d, min: %f, median: %f, max: %f, avg: %f, variance: %f \n",
				len(stepElapsed), minVal, medianVal, maxVal, avgVal, varVal))

			return types.TextMsg{Text: str.String()}
		},
	},
	{
		Define: "workflow queue",
		Help:   `workflow queue statisticians`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			inspector := workflow.GetInspector()
			queues, err := inspector.Queues()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			str := strings.Builder{}
			for _, queueName := range queues {
				info, err := inspector.GetQueueInfo(queueName)
				if err != nil {
					return types.TextMsg{Text: err.Error()}
				}

				str.WriteString(fmt.Sprintf("queue %s: size %d memory %v processed %d failed %d \n",
					info.Queue, info.Size, humanize.Bytes(uint64(info.MemoryUsage)), info.Processed, info.Failed))
			}

			return types.TextMsg{Text: str.String()}
		},
	},
	{
		Define: "workflow history",
		Help:   `workflow task history`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			inspector := workflow.GetInspector()
			queues, err := inspector.Queues()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			str := strings.Builder{}
			for _, queueName := range queues {
				stats, err := inspector.History(queueName, 7)
				if err != nil {
					return types.TextMsg{Text: err.Error()}
				}
				str.WriteString(fmt.Sprintf("queue %s:", queueName))
				for _, info := range stats {
					str.WriteString(fmt.Sprintf("%s -> processed %d failed %d, ",
						info.Date.Format(time.DateOnly), info.Processed, info.Failed))
				}
				str.WriteString("\n")
			}

			return types.TextMsg{Text: str.String()}
		},
	},
	{
		Define: "torrent demo",
		Help:   `torrent download demo`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(transmission.ID, transmission.EndpointKey)
			client, err := transmission.NewTransmission(endpoint.String())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			url := "magnet:?xt=urn:btih:f07e0b0584745b7bcb35e98097488d34e68623d0&dn=ubuntu-17.10.1-desktop-amd64.iso&tr=http%3A%2F%2Ftorrent.ubuntu.com%3A6969%2Fannounce&tr=http%3A%2F%2Fipv6.torrent.ubuntu.com%3A6969%2Fannounce"
			torrent, err := client.TorrentAddUrl(context.Background(), url)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("torrent %s added", *torrent.Name)}
		},
	},
	{
		Define: "adguard status",
		Help:   `get adguard home status`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(adguard.ID, adguard.EndpointKey)
			username, _ := providers.GetConfig(adguard.ID, adguard.UsernameKey)
			password, _ := providers.GetConfig(adguard.ID, adguard.PasswordKey)
			client, err := adguard.NewAdGuardHome(endpoint.String(), username.String(), password.String())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

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
			client, err := adguard.NewAdGuardHome(endpoint.String(), username.String(), password.String())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			resp, err := client.GetStats()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("adguard home dns queries %d, blocked filtering %d，avg processing time %f ms",
				resp.NumDnsQueries, resp.NumBlockedFiltering, resp.AvgProcessingTime*1000)}
		},
	},
}
