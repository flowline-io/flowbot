package workflow

import (
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/internal/agents"

	"github.com/flowline-io/flowbot/pkg/cache"

	"github.com/flowline-io/flowbot/pkg/crawler"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/executer"
	"github.com/flowline-io/flowbot/pkg/executer/runtime"
	"github.com/flowline-io/flowbot/pkg/expression"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/lobehub"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	endWorkflowActionID     = "end"
	messageWorkflowActionID = "message"
	fetchWorkflowActionID   = "fetch"
	feedWorkflowActionID    = "feed"
	grepWorkflowActionID    = "grep"
	uniqueWorkflowActionID  = "unique"
	torrentWorkflowActionID = "torrent"
	websiteWorkflowActionID = "website"
	llmWorkflowActionID     = "llm"
	exprWorkflowActionID    = "expr"
	dockerWorkflowActionID  = "docker"
)

var workflowRules = []workflow.Rule{
	{
		Id:           endWorkflowActionID,
		Title:        "end",
		Desc:         "end workflow",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return nil, nil
		},
	},
	{
		Id:           messageWorkflowActionID,
		Title:        "message",
		Desc:         "send message",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			if len(input) == 0 {
				return nil, nil
			}
			return nil, event.SendMessage(ctx, types.KVMsg(input))
		},
	},
	{
		Id:           fetchWorkflowActionID,
		Title:        "fetch",
		Desc:         "fetch url data",
		InputSchema:  nil,
		OutputSchema: nil,
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
		Id:           feedWorkflowActionID,
		Title:        "feed",
		Desc:         "parse feed",
		InputSchema:  nil,
		OutputSchema: nil,
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
		Id:           grepWorkflowActionID,
		Title:        "grep",
		Desc:         "grep text",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			pattern, _ := input.String("pattern")
			if pattern == "" {
				return nil, fmt.Errorf("%s step, empty pattern", uniqueWorkflowActionID)
			}

			return kvGrep(pattern, input)
		},
	},
	{
		Id:           uniqueWorkflowActionID,
		Title:        "unique",
		Desc:         "unique text",
		InputSchema:  nil,
		OutputSchema: nil,
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
				result, err := cache.Unique(ctx.Context(), id, v)
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
		Id:           torrentWorkflowActionID,
		Title:        "torrent",
		Desc:         "download torrent",
		InputSchema:  nil,
		OutputSchema: nil,
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
		Id:           websiteWorkflowActionID,
		Title:        "website content",
		Desc:         "Retrieve Website Content",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			url, _ := input.String("url")
			if url == "" {
				return nil, fmt.Errorf("%s step, empty url", websiteWorkflowActionID)
			}

			resp, err := lobehub.NewLobehub().WebCrawler(url)
			if err != nil {
				return nil, fmt.Errorf("%s step, get website content failed, %w", torrentWorkflowActionID, err)
			}

			return types.KV{
				"content": resp.Content,
				"title":   resp.Title,
				"url":     resp.Url,
				"website": resp.Website,
			}, nil
		},
	},
	{
		Id:           llmWorkflowActionID,
		Title:        "LLM",
		Desc:         "LLM Chat",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
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

			llm, err := agents.ChatModel(ctx.Context(), agents.Model())
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
		Id:           exprWorkflowActionID,
		Title:        "expr",
		Desc:         "expr-lang expression",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			script, _ := input.String("script")
			if script == "" {
				return nil, fmt.Errorf("%s step, empty prompt", exprWorkflowActionID)
			}

			expression.LoadEnv("input", input)
			program, err := expression.Compile(script)
			if err != nil {
				return nil, fmt.Errorf("%s step, expr compile failed, %w", exprWorkflowActionID, err)
			}

			result, err := expression.Run(program)
			if err != nil {
				return nil, fmt.Errorf("%s step, expr run failed, %w", exprWorkflowActionID, err)
			}

			switch v := result.(type) {
			case types.KV:
				return v, nil
			case map[string]any:
				return v, nil
			case []any, []types.KV, []map[string]any:
				return types.KV{
					"list": v,
				}, nil
			default:
				return types.KV{
					"data": v,
				}, nil
			}
		},
	},
	{
		Id:           dockerWorkflowActionID,
		Title:        "Run Docker Container",
		Desc:         "Executes a Docker container with specified parameters",
		InputSchema:  nil,
		OutputSchema: nil,
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
