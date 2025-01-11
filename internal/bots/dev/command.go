package dev

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/providers/safeline"
	"os"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/executer"
	"github.com/flowline-io/flowbot/pkg/executer/runtime"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/adguard"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
	openaiProvider "github.com/flowline-io/flowbot/pkg/providers/openai"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var commandRules = []command.Rule{
	{
		Define: "setting",
		Help:   `Bot setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.SettingMsg(ctx, Name)
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
		Define: "form",
		Help:   `[example] form`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.FormMsg(ctx, devFormID)
		},
	},
	{
		Define: "queue",
		Help:   `[example] publish mq and task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := event.SendMessage(ctx.Context(), ctx.AsUser.String(), ctx.Topic, types.TextMsg{Text: time.Now().String()})
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
		Define: "markdown",
		Help:   `[example] markdown page`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StorePage(ctx, model.PageMarkdown, "", types.MarkdownMsg{
				Raw: markdownText,
			})
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
		Help:   `[example] run docker image`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flog.Debug("start docker command")

			task := &types.Task{
				ID:    utils.NewUUID(),
				Image: "ubuntu:mantic",
				Run:   "echo -n hello > $OUTPUT",
			}
			engine := executer.New(runtime.Docker)
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
		Define: "torrent demo",
		Help:   `[example] torrent download demo`,
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
		Define: "test",
		Help:   `[example] test`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := meilisearch.NewMeiliSearch().AddDocument(types.Document{
				SourceId:    types.Id(),
				Source:      hoarder.ID,
				Title:       "test....",
				Description: "the....",
				Url:         "/url/test",
				CreatedAt:   int32(time.Now().Unix()),
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			list, total, err := meilisearch.NewMeiliSearch().Search(gitea.ID, "title", 1, 10)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("documents %v", total),
				Model: list,
			}
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
		Define: "slash",
		Help:   `[example] Slash example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, err := bots.Shortcut("test", "https://example.com")
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			return types.TextMsg{Text: url}
		},
	},
	{
		Define: "llm",
		Help:   `[example] LLM example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			token, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.TokenKey)
			baseUrl, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.BaseUrlKey)

			llm, err := openai.New(openai.WithToken(token.String()), openai.WithBaseURL(baseUrl.String()))
			if err != nil {
				flog.Error(err)
			}
			prompt := "Human: Who was the first man to walk on the moon?\nAssistant:"
			completion, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt,
				llms.WithTemperature(0.8),
			)
			if err != nil {
				flog.Error(err)
			}

			return types.TextMsg{Text: completion}
		},
	},
	{
		Define: "notify test",
		Help:   `[example] Notify example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := notify.ChannelSend(ctx.AsUser, "example", notify.Message{
				Title: "example title",
				Body:  "example body",
				Url:   "https://example.com",
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "fs test",
		Help:   `[example] filesystem example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			f, err := os.Open("./README.md")
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			defer func() { _ = f.Close() }()
			fileStat, err := f.Stat()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			fileSize := fileStat.Size()

			url, size, err := store.FS.Upload(&types.FileDef{
				User:     ctx.AsUser.String(),
				Size:     fileSize,
				MimeType: "text/markdown",
				Location: "example",
			}, f)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("url: %s, size: %d", url, size)}
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
	{
		Define: "safeline test",
		Help:   `[example] safeline example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(safeline.ID, safeline.EndpointKey)
			token, _ := providers.GetConfig(safeline.ID, safeline.TokenKey)

			client := safeline.NewSafeLine(endpoint.String(), token.String())
			resp, err := client.QPS(context.Background())

			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			return types.InfoMsg{
				Title: "safeline demo",
				Model: resp,
			}
		},
	},
}
