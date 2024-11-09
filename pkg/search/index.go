package search

import (
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
)

func InitSearchIndex() error {
	return meilisearch.NewMeiliSearch().DefaultIndexSettings()
}
