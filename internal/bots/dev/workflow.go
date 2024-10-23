package dev

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/channels/crawler"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/transmission"
)

const (
	endWorkflowActionID     = "end"
	inWorkflowActionID      = "in"
	addWorkflowActionID     = "add"
	outWorkflowActionID     = "out"
	errorWorkflowActionID   = "error"
	messageWorkflowActionID = "message"
	fetchWorkflowActionID   = "fetch"
	grepWorkflowActionID    = "grep"
	uniqueWorkflowActionID  = "unique"
	torrentWorkflowActionID = "torrent"
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
		Id:           inWorkflowActionID,
		Title:        "in",
		Desc:         "return {a, b}",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return types.KV{"a": 1, "b": 1}, nil
		},
	},
	{
		Id:           addWorkflowActionID,
		Title:        "add",
		Desc:         "a + b",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			a, _ := input.Int64("a")
			b, _ := input.Int64("b")
			return types.KV{"value": a + b}, nil
		},
	},
	{
		Id:           outWorkflowActionID,
		Title:        "out",
		Desc:         "print debug log",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			flog.Debug("%s => %+v", outWorkflowActionID, input)
			return nil, nil
		},
	},
	{
		Id:           errorWorkflowActionID,
		Title:        "error",
		Desc:         "return error",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return nil, fmt.Errorf("workflow run error %s", time.Now().Format(time.DateTime))
		},
	},
	{
		Id:           messageWorkflowActionID,
		Title:        "message",
		Desc:         "send message",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			if input == nil {
				return nil, fmt.Errorf("%s step, empty input", messageWorkflowActionID)
			}
			return nil, event.SendMessage(ctx.Context(), ctx.AsUser.String(), ctx.Topic, types.KVMsg(input))
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
				result, err := unique(ctx.Context(), id, v)
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
			endpoint, _ := providers.GetConfig(transmission.ID, transmission.EndpointKey)
			c, err := transmission.NewTransmission(endpoint.String())
			if err != nil {
				return nil, fmt.Errorf("%s step, transmission client failed, %w", torrentWorkflowActionID, err)
			}

			url, _ := input.String("url")
			if url != "" {
				t, err := c.TorrentAddUrl(ctx.Context(), url)
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
				result := make([]types.KV, 0)
				if v, ok := list.([]any); ok {
					for _, item := range v {
						val, ok := item.(map[string]any)
						if !ok {
							continue
						}
						kv := types.KV(val)
						url, _ := kv.String("url")

						flog.Info("[%s] torrent add url: %s", transmission.ID, url)

						t, err := c.TorrentAddUrl(ctx.Context(), url)
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
				}
				return types.KV{"list": result}, nil
			}

			return nil, nil
		},
	},
}
