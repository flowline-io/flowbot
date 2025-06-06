package workflow

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

type CronTaskManager struct {
	mgr *asynq.PeriodicTaskManager
}

func NewCronTaskManager(lc fx.Lifecycle, _ *redis.Client, _ store.Adapter) *CronTaskManager {
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
				flog.Info("[workflow] Enqueued cron task %s type %s", info.ID, info.Type)
			},
			Location: time.Local,
			Logger:   flog.AsynqLogger,
			LogLevel: flog.AsynqLogLevel(config.App.Log.Level),
		},
	})
	if err != nil {
		flog.Fatal("error %v", err)
	}
	i := &CronTaskManager{mgr: mgr}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go i.Run()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			i.Shutdown()
			return nil
		},
	})

	return i
}

func (c *CronTaskManager) Run() {
	if err := c.mgr.Start(); err != nil {
		flog.Error(err)
	}
}

func (c *CronTaskManager) Shutdown() {
	c.mgr.Shutdown()
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
			flog.Warn("error %v", err)
			continue
		}
		payload, err := sonic.Marshal(trigger)
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
