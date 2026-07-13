package kanboard

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
)

func TestKanboardConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"runs kanban conformance test suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conformance.RunKanbanConformance(t, func(_ *testing.T, cfg conformance.KanbanConfig) conformance.KanbanService {
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
		})
	}
}

func cfgToProviderTasks(items []*capability.Task) []*provider.Task {
	tasks := make([]*provider.Task, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, abilityTaskToProvider(item))
	}
	return tasks
}

func cfgToProviderTask(item *capability.Task) *provider.Task {
	if item == nil {
		return nil
	}
	return abilityTaskToProvider(item)
}

func abilityTaskToProvider(item *capability.Task) *provider.Task {
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
var _ Service = (*Adapter)(nil)
