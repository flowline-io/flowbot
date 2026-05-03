package kanboard

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	kb "github.com/flowline-io/flowbot/pkg/ability/kanban"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
)

func TestKanboardConformance(t *testing.T) {
	conformance.RunKanbanConformance(t, func(t *testing.T, cfg conformance.KanbanConfig) kb.Service {
		// UpdateTask and MoveTask internally call GetTask after the mutation.
		// Use the config's UpdateTask or MoveTask as the GetTask response so
		// the returned task reflects the expected post-mutation state.
		taskForGet := cfgToProviderTask(cfg.Task)
		if taskForGet == nil && cfg.UpdateTask != nil {
			taskForGet = cfgToProviderTask(cfg.UpdateTask)
		}
		if taskForGet == nil && cfg.MoveTask != nil {
			taskForGet = cfgToProviderTask(cfg.MoveTask)
		}
		c := &fakeClient{
			tasks:        cfgToProviderTasks(cfg.Tasks),
			tasksErr:     cfg.TasksErr,
			task:         taskForGet,
			taskErr:      cfg.TaskErr,
			createTaskID: int64(cfg.CreateTaskID),
			createErr:    cfg.CreateErr,
			updateResult: true,
			updateErr:    cfg.UpdateErr,
			moveResult:   true,
			moveErr:      cfg.MoveErr,
			deleteErr:    cfg.DeleteErr,
			closeErr:     cfg.CloseErr,
			columns:      cfgToColumns(cfg.Columns),
			columnsErr:   cfg.ColumnsErr,
			searchTasks:  cfgToProviderTasks(cfg.SearchTasks),
			searchErr:    cfg.SearchErr,
		}
		return NewWithClient(c)
	})
}

func cfgToProviderTasks(items []*ability.Task) []*provider.Task {
	tasks := make([]*provider.Task, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, abilityTaskToProvider(item))
	}
	return tasks
}

func cfgToProviderTask(item *ability.Task) *provider.Task {
	if item == nil {
		return nil
	}
	return abilityTaskToProvider(item)
}

func abilityTaskToProvider(item *ability.Task) *provider.Task {
	if item == nil {
		return nil
	}
	return &provider.Task{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		ProjectID:   item.ProjectID,
		ColumnID:    item.ColumnID,
		Tags:        tagsToAny(item.Tags),
		Reference:   item.Reference,
	}
}

func cfgToColumns(cols []map[string]any) func() []map[string]any {
	return func() []map[string]any { return cols }
}

// Ensure the fake client satisfies the adapter's client interface.
var _ client = (*fakeClient)(nil)

// Ensure the conformance integrates with the kanban Service interface.
var _ kb.Service = (*Adapter)(nil)
