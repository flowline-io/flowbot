package dev

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/executer"
	"github.com/flowline-io/flowbot/pkg/executer/runtime"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
	"github.com/flowline-io/flowbot/pkg/providers/safeline"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/utils"
)

var commandRules = []command.Rule{
	{
		Define: "dev setting",
		Help:   `[example] setting`,
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
		Define: "form test",
		Help:   `[example] form`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.FormMsg(ctx, devFormID)
		},
	},
	{
		Define: "queue test",
		Help:   `[example] publish mq and task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := event.SendMessage(ctx, types.TextMsg{Text: time.Now().String()})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "instruct test",
		Help:   `[example] create instruct`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			data := types.KV{}
			data["txt"] = "example"
			return bots.InstructMsg(ctx, ExampleInstructID, data)
		},
	},
	{
		Define: "page test",
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
		Define: "docker test",
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
		Define: "torrent test",
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
		Define: "slash test",
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
		Define: "llm test",
		Help:   `[example] LLM example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			messages, err := agents.DefaultTemplate().Format(ctx.Context(), map[string]any{
				"content": "Who was the first man to walk on the moon?",
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			llm, err := agents.ChatModel(ctx.Context(), agents.Model())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			resp, err := agents.Generate(ctx.Context(), llm, messages)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: resp.Content}
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
	{
		Define: "event test",
		Help:   `[example] event example`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := event.BotEventFire(ctx, types.ExampleBotEventID, types.KV{
				"k1": "v1",
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "test",
		Help:   `[example] test`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flog.Info("dev bot environment config: %s", config.Environment)

			err := meilisearch.NewMeiliSearch().AddDocument(types.Document{
				SourceId:    types.Id(),
				Source:      hoarder.ID,
				Title:       "the title....",
				Description: "the description....",
				Url:         "/url/test",
				Timestamp:   time.Now().Unix(),
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
}
