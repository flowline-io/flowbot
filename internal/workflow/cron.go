package workflow

import (
	"encoding/json"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	"time"
)

type CronTaskManager struct {
	mgr *asynq.PeriodicTaskManager
}

func NewCronTaskManager() *CronTaskManager {
	provider := &DatabaseProvider{}
	mgr, err := asynq.NewPeriodicTaskManager(asynq.PeriodicTaskManagerOpts{
		RedisConnOpt:               defaultRedisClientOpt(),
		PeriodicTaskConfigProvider: provider,
		SyncInterval:               time.Minute,
		SchedulerOpts: &asynq.SchedulerOpts{
			PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
				flog.Info("CronTaskManager:  Enqueued task %s with payload %s with error %v",
					info.ID, string(info.Payload), err)
			},
		},
	})
	if err != nil {
		flog.Fatal(err.Error())
	}
	return &CronTaskManager{mgr: mgr}
}

func (c *CronTaskManager) Run() {
	if err := c.mgr.Run(); err != nil {
		flog.Error(err)
	}
}

func (c *CronTaskManager) Shutdown() {
	c.mgr.Shutdown()
}

type DatabaseProvider struct{}

func (d *DatabaseProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	list, err := store.Chatbot.ListWorkflowTriggerByType(model.TriggerCron)
	if err != nil {
		return nil, err
	}

	var configs []*asynq.PeriodicTaskConfig
	for _, trigger := range list {
		payload, err := json.Marshal(trigger)
		if err != nil {
			flog.Error(err)
			continue
		}
		configs = append(configs, &asynq.PeriodicTaskConfig{
			Cronspec: trigger.Rule,
			Task:     asynq.NewTask(TypeCron, payload),
			Opts: []asynq.Option{
				asynq.Queue(cronQueueName),
			},
		})
	}
	return configs, nil
}
