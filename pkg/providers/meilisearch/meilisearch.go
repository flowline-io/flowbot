package meilisearch

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	"github.com/meilisearch/meilisearch-go"
)

type MeiliSearch struct {
	manager meilisearch.ServiceManager
}

func NewMeiliSearch() MeiliSearch {
	return MeiliSearch{
		manager: meilisearch.New(config.App.Search.Endpoint, meilisearch.WithAPIKey(config.App.Search.MasterKey)),
	}
}

func (c MeiliSearch) AddDocument(data types.Document) error {
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
		"created_at":  data.CreatedAt,
	}, "id")
	if err != nil {
		return err
	}
	flog.Info("[search] index %s add document %s-%s status: %s", config.App.Search.DataIndex, data.Source, data.Id, taskInfo.Status)

	return nil
}

func (c MeiliSearch) Search(source, query string, page, pageSize int32) ([]*types.Document, int64, error) {
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

	data, err := jsoniter.Marshal(resp.Hits)
	if err != nil {
		return nil, 0, err
	}

	var list []*types.Document
	err = json.Unmarshal(data, &list)
	if err != nil {
		return nil, 0, err
	}

	return list, resp.EstimatedTotalHits, nil
}

func (c MeiliSearch) DefaultIndexSettings() error {
	taskInfo, err := c.manager.Index(config.App.Search.DataIndex).UpdateSettings(&meilisearch.Settings{
		SortableAttributes:   []string{"created_at"},
		FilterableAttributes: []string{"source"},
		SearchableAttributes: []string{"source_id", "source", "title", "description"},
	})
	flog.Info("[search] index %s update settings status: %+v", config.App.Search.DataIndex, taskInfo)
	return err
}

func idKey(source string, id any) string {
	return fmt.Sprintf("%s-%v", source, id)
}
