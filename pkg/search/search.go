package search

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/goccy/go-json"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/fx"
)

var Instance *Client

type Client struct {
	manager meilisearch.ServiceManager
}

func NewClient(lc fx.Lifecycle, _ config.Type) *Client {
	Instance = &Client{
		manager: meilisearch.New(config.App.Search.Endpoint, meilisearch.WithAPIKey(config.App.Search.MasterKey)),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				err := Instance.DefaultIndexSettings()
				if err != nil {
					flog.Error(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return Instance
}

func (c *Client) AddDocument(data types.Document) error {
	// metrics
	stats.SearchProcessedDocumentTotalCounter(config.App.Search.DataIndex).Inc()

	// add
	taskInfo, err := c.manager.Index(config.App.Search.DataIndex).AddDocuments(types.KV{
		"id":          idKey(data.Source, data.SourceId),
		"source_id":   data.SourceId,
		"source":      data.Source,
		"title":       data.Title,
		"description": data.Description,
		"url":         data.Url,
		"timestamp":   data.Timestamp,
	}, "id")
	if err != nil {
		return err
	}
	flog.Debug("[search] index %s add document %s-%s status: %s", config.App.Search.DataIndex, data.Source, data.Id, taskInfo.Status)

	return nil
}

func (c *Client) Search(source, query string, page, pageSize int32) (types.DocumentList, int64, error) {
	// metrics
	stats.SearchTotalCounter(config.App.Search.DataIndex).Inc()

	// filter
	filter := ""
	if source != "" {
		filter = fmt.Sprintf("source = %s", source)
	}
	resp, err := c.manager.Index(config.App.Search.DataIndex).Search(query, &meilisearch.SearchRequest{
		Offset: int64((page - 1) * pageSize),
		Limit:  int64(pageSize),
		Sort:   []string{"created_at:desc"},
		Filter: filter,
	})
	if err != nil {
		return nil, 0, err
	}
	utils.PrettyPrintJsonStyle(resp)

	data, err := sonic.Marshal(resp.Hits)
	if err != nil {
		return nil, 0, err
	}

	var list types.DocumentList
	err = json.Unmarshal(data, &list)
	if err != nil {
		return nil, 0, err
	}
	list.FillUrlBase(config.App.Search.UrlBaseMap)

	return list, resp.EstimatedTotalHits, nil
}

func (c *Client) DefaultIndexSettings() error {
	taskInfo, err := c.manager.Index(config.App.Search.DataIndex).UpdateSettings(&meilisearch.Settings{
		SortableAttributes:   []string{"created_at"},
		FilterableAttributes: []string{"source"},
		SearchableAttributes: []string{"source_id", "source", "title", "description"},
	})
	flog.Debug("[search] index %s update settings status: %+v", config.App.Search.DataIndex, taskInfo)
	return err
}

func idKey(source string, id any) string {
	return fmt.Sprintf("%s-%v", source, id)
}
