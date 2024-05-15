package workflow

import (
	"context"
	"encoding/json"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
)

type CronTaskManager struct {
	mgr *asynq.PeriodicTaskManager
}

func NewCronTaskManager() *CronTaskManager {
	provider := &DatabaseProvider{}
	mgr, err := asynq.NewPeriodicTaskManager(asynq.PeriodicTaskManagerOpts{
		RedisConnOpt:               defaultRedisClientOpt(),
		PeriodicTaskConfigProvider: provider,
		SyncInterval:               10 * time.Minute,
		SchedulerOpts: &asynq.SchedulerOpts{
			PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
				if err != nil {
					flog.Error(err)
					return
				}
				if info == nil {
					return
				}
				flog.Info("[workflow] Enqueued task %s with payload %s with error %v",
					info.ID, string(info.Payload), err)
				err = HandleCronTask(context.Background(), asynq.NewTask(info.Type, info.Payload))
				if err != nil {
					flog.Error(err)
				}
			},
			Location: time.Local,
			Logger:   flog.AsynqLogger,
			LogLevel: flog.AsynqLogLevel(config.App.Log.Level),
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
	flog.Info("cron task shutdown")
}

type DatabaseProvider struct{}

func (d *DatabaseProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	list, err := store.Database.ListWorkflowTriggerByType(model.TriggerCron)
	if err != nil {
		return nil, err
	}

	var configs []*asynq.PeriodicTaskConfig
	for _, trigger := range list {
		if trigger.State == model.WorkflowTriggerDisable {
			continue
		}
		_, err = store.Database.GetWorkflow(trigger.WorkflowID)
		if err != nil {
			flog.Warn(err.Error())
			continue
		}
		payload, err := json.Marshal(trigger)
		if err != nil {
			flog.Error(err)
			continue
		}
		var rule model.TriggerCronRule
		spec, ok := types.KV(trigger.Rule).String("spec")
		if !ok {
			continue
		}
		if spec == "" {
			continue
		}
		rule.Spec = spec
		configs = append(configs, &asynq.PeriodicTaskConfig{
			Cronspec: rule.Spec,
			Task:     asynq.NewTask(TypeCron, payload),
			Opts: []asynq.Option{
				asynq.Queue(cronQueueName),
			},
		})
	}
	return configs, nil
}
