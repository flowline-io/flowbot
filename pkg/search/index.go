package search

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/meilisearch/meilisearch-go"
)

const indexName = "data"

type Document struct {
	Id          string `json:"id"`
	Source      string `json:"source"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	CreatedAt   int32  `json:"created_at"`
}

func idKey(source string, id any) string {
	return fmt.Sprintf("%s-%v", source, id)
}

func InitSearch() error {
	taskInfo, err := NewClient().manager.Index(indexName).UpdateSettings(&meilisearch.Settings{
		SortableAttributes:   []string{"created_at"},
		FilterableAttributes: []string{"source"},
	})
	flog.Info("[search] index %s update settings status: %v", indexName, taskInfo.Status)

	return err
}
