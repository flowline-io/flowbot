package search

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	"github.com/meilisearch/meilisearch-go"
)

func NewClient() Client {
	return Client{
		manager: meilisearch.New(config.App.Search.Endpoint, meilisearch.WithAPIKey(config.App.Search.MasterKey)),
	}
}

type Client struct {
	manager meilisearch.ServiceManager
}

func (c Client) AddDocument(data Document) error {
	taskInfo, err := c.manager.Index(indexName).AddDocuments(types.KV{
		"id":          idKey(data.Source, data.Id),
		"source":      data.Source,
		"title":       data.Title,
		"description": data.Description,
		"url":         data.Url,
		"created_at":  data.CreatedAt,
	}, "id")
	if err != nil {
		return err
	}
	flog.Info("[search] index %s add document %s-%s status: %s", indexName, data.Source, data.Id, taskInfo.Status)

	return nil
}

func (c Client) Search(source, query string, page, pageSize int32) ([]*Document, int64, error) {
	filter := ""
	if source != "" {
		filter = fmt.Sprintf("source = %s", source)
	}
	resp, err := c.manager.Index(indexName).Search(query, &meilisearch.SearchRequest{
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

	var list []*Document
	err = json.Unmarshal(data, &list)
	if err != nil {
		return nil, 0, err
	}

	return list, resp.TotalHits, nil
}
