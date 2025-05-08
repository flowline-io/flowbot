package server

import (
	"context"
	"fmt"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"github.com/flowline-io/flowbot/version"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

var (
	// stop signal
	stopSignal <-chan bool
	// swagger
	swagHandler fiber.Handler
	// fiber app
	httpApp *fiber.App
)

func initializeLog() error {
	flog.Init(false)
	flog.SetLevel(config.App.Log.Level)
	return nil
}

func initializeTimezone() error {
	_, err := time.LoadLocation("Local")
	if err != nil {
		return fmt.Errorf("load time location error, %w", err)
	}
	return nil
}

func initializeDatabase() error {
	// init database
	mysql.Init()
	store.Init()

	// Open database
	err := store.Store.Open(config.App.Store)
	if err != nil {
		return fmt.Errorf("failed to open DB, %w", err)
	}
	go func() {
		<-stopSignal
		err = store.Store.Close()
		if err != nil {
			flog.Error(err)
		}
		flog.Debug("Closed database connection(s)")
	}()

	// migrate
	if err := store.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate DB, %w", err)
	}

	return nil
}

func initializeMedia() error {
	// Media
	if config.App.Media != nil {
		if config.App.Media.UseHandler == "" {
			config.App.Media = nil
		} else {
			globals.maxFileUploadSize = config.App.Media.MaxFileUploadSize
			if config.App.Media.Handlers != nil {
				var conf string
				if params := config.App.Media.Handlers[config.App.Media.UseHandler]; params != nil {
					data, err := jsoniter.Marshal(params)
					if err != nil {
						return fmt.Errorf("failed to marshal media handler, %w", err)
					}
					conf = string(data)
				}
				if err := store.UseMediaHandler(config.App.Media.UseHandler, conf); err != nil {
					return fmt.Errorf("failed to init media handler, %w", err)
				}
			}
			if config.App.Media.GcPeriod > 0 && config.App.Media.GcBlockSize > 0 {
				globals.mediaGcPeriod = time.Second * time.Duration(config.App.Media.GcPeriod)
				stopFilesGc := largeFileRunGarbageCollection(globals.mediaGcPeriod, config.App.Media.GcBlockSize)
				go func() {
					<-stopSignal
					stopFilesGc <- true
					flog.Info("Stopped files garbage collector")
				}()
			}
		}
	}
	return nil
}

func initializeChatbot(signal <-chan bool) error {
	// Initialize bots
	hookBot(config.App.Bots, config.App.Vendors)

	// hook
	hookStarted()

	// Platform
	hookPlatform(signal)

	return nil
}

// init workflow
func initializeWorkflow() error {
	// Task queue
	globals.taskQueue = workflow.NewQueue()
	go globals.taskQueue.Run()
	// manager
	globals.manager = workflow.NewManager()
	go globals.manager.Run()
	// cron task manager
	globals.cronTaskManager = workflow.NewCronTaskManager()
	go globals.cronTaskManager.Run()

	return nil
}

// init bots
func initializeBot() {
	// register bots
	registerBots := sets.NewString()
	for name, handler := range bots.List() {
		registerBots.Insert(name)

		state := model.BotInactive
		if handler.IsReady() {
			state = model.BotActive
		}
		bot, _ := store.Database.GetBotByName(name)
		if bot == nil {
			bot = &model.Bot{
				Name:  name,
				State: state,
			}
			if _, err := store.Database.CreateBot(bot); err != nil {
				flog.Error(err)
			}
		} else {
			bot.State = state
			err := store.Database.UpdateBot(bot)
			if err != nil {
				flog.Error(err)
			}
		}
	}

	// inactive bot
	list, err := store.Database.GetBots()
	if err != nil {
		flog.Error(err)
	}
	for _, bot := range list {
		if !registerBots.Has(bot.Name) {
			bot.State = model.BotInactive
			if err := store.Database.UpdateBot(bot); err != nil {
				flog.Error(err)
			}
		}
	}
}

// init event
func initializeEvent() error {
	router, err := event.NewRouter()
	if err != nil {
		return err
	}

	subscriber, err := event.NewSubscriber()
	if err != nil {
		return err
	}

	router.AddNoPublisherHandler(
		"onMessageChannelEvent",
		protocol.MessageChannelEvent,
		subscriber,
		onPlatformMessageEventHandler,
	)
	router.AddNoPublisherHandler(
		"onMessageDirectEvent",
		protocol.MessageDirectEvent,
		subscriber,
		onPlatformMessageEventHandler,
	)
	router.AddNoPublisherHandler(
		"onMessageSendEventHandler",
		types.MessageSendEvent,
		subscriber,
		onMessageSendEventHandler,
	)
	router.AddNoPublisherHandler(
		"onInstructPushEventHandler",
		types.InstructPushEvent,
		subscriber,
		onInstructPushEventHandler,
	)
	router.AddNoPublisherHandler(
		"onBotRunEventHandler",
		types.BotRunEvent,
		subscriber,
		onBotRunEventHandler,
	)

	go func() {
		if err = router.Run(context.Background()); err != nil {
			flog.Error(err)
		}
	}()

	return nil
}

func initializeMetrics() error {
	return metrics.InitPushWithOptions(
		context.Background(),
		fmt.Sprintf("%s/api/v1/import/prometheus", config.App.Metrics.Endpoint),
		10*time.Second,
		true,
		&metrics.PushOptions{
			ExtraLabels: fmt.Sprintf(`instance="flowbot",version="%s"`, version.Buildtags),
		},
	)
}
