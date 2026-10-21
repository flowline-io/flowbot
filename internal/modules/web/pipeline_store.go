package web

import "github.com/flowline-io/flowbot/internal/store"

// getPipelineDefStore is for web pages other than the pipeline editor.
// Pipeline, workflow, and function pages use ActiveService.
func getPipelineDefStore() *store.PipelineStore {
	if store.Database == nil {
		return nil
	}
	return store.PipelineStoreFromDB()
}
