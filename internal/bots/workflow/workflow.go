package workflow

import (
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/pkg/crawler"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/executer"
	"github.com/flowline-io/flowbot/pkg/executer/runtime"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	messageWorkflowActionID = "message"
	fetchWorkflowActionID   = "fetch"
	feedWorkflowActionID    = "feed"
	grepWorkflowActionID    = "grep"
	uniqueWorkflowActionID  = "unique"
	torrentWorkflowActionID = "torrent"
	llmWorkflowActionID     = "llm"
	dockerWorkflowActionID  = "docker"
)

var workflowRules = []workflow.Rule{
	{
		Id:          messageWorkflowActionID,
		Title:       "message",
		Description: "send message",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			if len(input) == 0 {
				return nil, nil
			}
			return nil, event.SendMessage(ctx, types.KVMsg(input))
		},
	},
	{
		Id:          fetchWorkflowActionID,
		Title:       "fetch",
		Description: "fetch url data",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			url, _ := input.String("url")
			if url == "" {
				return nil, fmt.Errorf("%s step, empty url", fetchWorkflowActionID)
			}

			list, _ := input.String("list")
			if list == "" {
				return nil, fmt.Errorf("%s step, empty list", fetchWorkflowActionID)
			}

			item, _ := input.Map("item")
			if item == nil {
				return nil, fmt.Errorf("%s step, empty item", fetchWorkflowActionID)
			}

			itemMap := make(map[string]string)
			for k, v := range item {
				itemMap[k] = fmt.Sprintf("%v", v)
			}

			rule := crawler.Rule{
				Name:   fetchWorkflowActionID,
				Enable: true,
				Id:     types.Id(),
				Page: &struct {
					URL  string
					List string
					Item map[string]string
				}{URL: url, List: list, Item: itemMap},
			}

			return types.KV{"list": rule.Run()}, nil
		},
	},
	{
		Id:          feedWorkflowActionID,
		Title:       "feed",
		Description: "parse feed",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			url, _ := input.String("url")
			if url == "" {
				return nil, fmt.Errorf("%s step, empty url", feedWorkflowActionID)
			}

			item, _ := input.Map("item")
			if item == nil {
				return nil, fmt.Errorf("%s step, empty item", feedWorkflowActionID)
			}

			itemMap := make(map[string]string)
			for k, v := range item {
				itemMap[k] = fmt.Sprintf("%v", v)
			}

			rule := crawler.Rule{
				Name:   feedWorkflowActionID,
				Enable: true,
				Id:     types.Id(),
				Feed: &struct {
					URL  string
					Item map[string]string
				}{URL: url, Item: itemMap},
			}

			return types.KV{"list": rule.Run()}, nil
		},
	},
	{
		Id:          grepWorkflowActionID,
		Title:       "grep",
		Description: "grep text",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			pattern, _ := input.String("pattern")
			if pattern == "" {
				return nil, fmt.Errorf("%s step, empty pattern", uniqueWorkflowActionID)
			}

			return kvGrep(pattern, input)
		},
	},
	{
		Id:          uniqueWorkflowActionID,
		Title:       "unique",
		Description: "unique text",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			id, _ := input.String("id")
			if id == "" {
				return nil, fmt.Errorf("%s step, empty id", uniqueWorkflowActionID)
			}
			list, _ := input.Any("list")
			if list == nil {
				return nil, fmt.Errorf("%s step, empty data", uniqueWorkflowActionID)
			}
			if v, ok := list.([]any); ok {
				result, err := rdb.BloomUnique(ctx.Context(), id, v)
				if err != nil {
					return nil, fmt.Errorf("%s step, unique failed, %w", uniqueWorkflowActionID, err)
				}

				return types.KV{
					"list": result,
				}, nil
			}

			return nil, nil
		},
	},
	{
		Id:          torrentWorkflowActionID,
		Title:       "torrent",
		Description: "download torrent",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			client, err := transmission.GetClient()
			if err != nil {
				return nil, fmt.Errorf("%s step, transmission client failed, %w", torrentWorkflowActionID, err)
			}

			url, _ := input.String("url")
			if url != "" {
				t, err := client.TorrentAddUrl(ctx.Context(), url)
				if err != nil {
					return nil, fmt.Errorf("%s step, torrent add url failed, %w", torrentWorkflowActionID, err)
				}

				return types.KV{
					"id":     t.ID,
					"name":   t.Name,
					"status": t.Status,
					"error":  t.Error,
				}, nil
			}

			list, _ := input.Any("list")
			if list != nil {
				if v, ok := list.([]any); ok {
					result := make([]types.KV, 0)
					for _, item := range v {
						val, ok := item.(map[string]any)
						if !ok {
							continue
						}
						kv := types.KV(val)
						url, _ := kv.String("url")

						flog.Info("[%s] torrent add url: %s", transmission.ID, url)

						t, err := client.TorrentAddUrl(ctx.Context(), url)
						if err != nil {
							return nil, fmt.Errorf("%s step, torrent add url failed, %w", torrentWorkflowActionID, err)
						}

						result = append(result, types.KV{
							"id":     t.ID,
							"name":   t.Name,
							"status": t.Status,
							"error":  t.Error,
						})
					}
					if len(result) > 0 {
						return types.KV{"list": result}, nil
					}
				}
			}

			return nil, nil
		},
	},
	{
		Id:          llmWorkflowActionID,
		Title:       "LLM",
		Description: "LLM Chat",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			if !agents.AgentEnabled(agents.AgentChat) {
				return nil, fmt.Errorf("%s step, agent chat disabled", llmWorkflowActionID)
			}
			ctx.SetTimeout(10 * time.Minute)
			promptVal, _ := input.String("prompt")
			if promptVal == "" {
				return nil, fmt.Errorf("%s step, empty prompt", llmWorkflowActionID)
			}
			content, _ := input.String("content")
			if content == "" {
				return nil, fmt.Errorf("%s step, empty content", llmWorkflowActionID)
			}

			messages, err := prompt.FromMessages(schema.GoTemplate, schema.UserMessage(promptVal)).
				Format(ctx.Context(), map[string]any{
					"content": content,
				})
			if err != nil {
				return nil, fmt.Errorf("%s step, prompt format failed, %w", llmWorkflowActionID, err)
			}

			llm, err := agents.ChatModel(ctx.Context(), agents.AgentModelName(agents.AgentChat))
			if err != nil {
				return nil, fmt.Errorf("%s step, llm create failed, %w", llmWorkflowActionID, err)
			}

			resp, err := agents.Generate(ctx.Context(), llm, messages)
			if err != nil {
				return nil, fmt.Errorf("%s step, llm generate failed, %w", llmWorkflowActionID, err)
			}

			return types.KV{
				"text": resp.Content,
			}, nil
		},
	},
	{
		Id:          dockerWorkflowActionID,
		Title:       "Run Docker Container",
		Description: "Executes a Docker container with specified parameters",
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			image, _ := input.String("image")
			if image == "" {
				return nil, fmt.Errorf("%s step, empty image", dockerWorkflowActionID)
			}
			run, _ := input.String("run")
			if run == "" {
				return nil, fmt.Errorf("%s step, empty run", dockerWorkflowActionID)
			}

			task := &types.Task{
				ID:    utils.NewUUID(),
				Image: image, // example: "ubuntu:mantic",
				Run:   run,   // example: "echo -n hello > $OUTPUT",
			}
			ctx.SetTimeout(10 * time.Minute)
			engine := executer.New(runtime.Docker)
			err := engine.Run(ctx.Context(), task)
			if err != nil {
				return nil, fmt.Errorf("%s step, %w", dockerWorkflowActionID, err)
			}

			return types.KV{
				"id":           task.ID,
				"state":        task.State,
				"result":       task.Result,
				"error":        task.Error,
				"created_at":   task.CreatedAt,
				"started_at":   task.StartedAt,
				"completed_at": task.CompletedAt,
				"failed_at":    task.FailedAt,
			}, nil
		},
	},
}
