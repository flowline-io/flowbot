package web

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/workflow"
)

// InstallPageServices wires the pipeline and workflow services from store.Database.
// Production installs them from internal/server. Tests and E2E mounts call this
// so the pipeline and workflow pages can use ActiveService.
func InstallPageServices() {
	if store.Database == nil || store.Database.GetClient() == nil {
		return
	}
	svc, _ := store.NewPipelineService(store.PipelineStoreFromDB(), store.EventStoreFromDB())
	pipeline.SetActiveService(svc)

	if prev := workflow.ActiveService(); prev != nil {
		prev.Stop()
	}
	catalog, runs := store.WorkflowServiceAdapters()
	workflow.SetReloadService(workflow.NewService(catalog, runs, nil, nil))
}
