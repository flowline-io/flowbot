package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/alarm"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/pprofs"
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"github.com/flowline-io/flowbot/version"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/pflag"
)

var (
	// stop signal
	stopSignal <-chan bool
	// swagger
	swagHandler fiber.Handler
	// fiber app
	httpApp *fiber.App
	// flag variables
	appFlag struct {
		configFile *string
		listenOn   *string
		apiPath    *string
		tlsEnabled *bool
		pprofFile  *string
		pprofUrl   *string
	}
)

func initialize() error {
	var err error

	// init log
	if err = initializeLog(); err != nil {
		return err
	}
	flog.Info("initialize Log ok")

	// init timezone
	if err = initializeTimezone(); err != nil {
		return err
	}
	flog.Info("initialize Timezone ok")

	// init flag
	if err = initializeFlag(); err != nil {
		return err
	}
	flog.Info("initialize Flag ok")

	// init config
	if err = initializeConfig(); err != nil {
		return err
	}
	flog.Info("initialize Config ok")

	// init alarm
	if err = initializeAlarm(); err != nil {
		return err
	}
	flog.Info("initialize Alarm ok")

	// init http
	if err = initializeHttp(); err != nil {
		return err
	}
	flog.Info("initialize Http ok")

	// init pprof
	if err = initializePprof(); err != nil {
		return err
	}
	flog.Info("initialize Pprof ok")

	// init cache
	if err = initializeCache(); err != nil {
		return err
	}
	flog.Info("initialize Cache ok")

	// init database
	if err = initializeDatabase(); err != nil {
		return err
	}
	flog.Info("initialize Database ok")

	// init media
	if err = initializeMedia(); err != nil {
		return err
	}
	flog.Info("initialize Media ok")

	// init signal
	if err = initializeSignal(); err != nil {
		return err
	}
	flog.Info("initialize Signal ok")

	// init event
	if err = initializeEvent(); err != nil {
		return err
	}
	flog.Info("initialize Event ok")

	// init chatbot
	if err = initializeChatbot(stopSignal); err != nil {
		return err
	}
	flog.Info("initialize Chatbot ok")

	// init metrics
	if err = initializeMetrics(); err != nil {
		return err
	}
	flog.Info("initialize Metrics ok")

	// init search
	if err = initializeSearch(); err != nil {
		return err
	}
	flog.Info("initialize Search ok")

	return nil
}

func initializeLog() error {
	flog.Init(false)
	return nil
}

func initializeTimezone() error {
	_, err := time.LoadLocation("Local")
	if err != nil {
		return fmt.Errorf("load time location error, %w", err)
	}
	return nil
}

func initializeFlag() error {
	appFlag.configFile = pflag.String("config", "flowbot.yaml", "Path to config file.")
	appFlag.listenOn = pflag.String("listen", "", "Override address and port to listen on for HTTP(S) clients.")
	appFlag.apiPath = pflag.String("api_path", "", "Override the base URL path where API is served.")
	appFlag.tlsEnabled = pflag.Bool("tls_enabled", false, "Override config value for enabling TLS.")
	appFlag.pprofFile = pflag.String("pprof", "", "File name to save profiling info to. Disabled if not set.")
	appFlag.pprofUrl = pflag.String("pprof_url", "", "Debugging only! URL path for exposing profiling info. Disabled if not set.")
	pflag.Parse()
	return nil
}

func initializeConfig() error {
	executable, _ := os.Executable()

	curwd, err := os.Getwd()
	if err != nil {
		flog.Fatal("Couldn't get current working directory: %v", err)
	}

	flog.Info("version %s:%s:%s; pid %d; %d process(es)",
		version.Buildtags, executable, version.Buildstamp,
		os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))

	*appFlag.configFile = utils.ToAbsolutePath(curwd, *appFlag.configFile)
	flog.Info("Using config from '%s'", *appFlag.configFile)

	// Load config
	config.Load(".", curwd)

	if *appFlag.listenOn != "" {
		config.App.Listen = *appFlag.listenOn
	}

	// Configure root path for serving API calls.
	if *appFlag.apiPath != "" {
		config.App.ApiPath = *appFlag.apiPath
	}
	if config.App.ApiPath == "" {
		config.App.ApiPath = defaultApiPath
	} else {
		if !strings.HasPrefix(config.App.ApiPath, "/") {
			config.App.ApiPath = "/" + config.App.ApiPath
		}
		if !strings.HasSuffix(config.App.ApiPath, "/") {
			config.App.ApiPath += "/"
		}
	}
	flog.Info("API served from root URL path '%s'", config.App.ApiPath)

	// log level
	flog.SetLevel(config.App.Log.Level)

	return nil
}

func initializeHttp() error {
	// Set up HTTP server.
	httpApp = fiber.New(fiber.Config{
		DisableStartupMessage: true,

		JSONDecoder:  jsoniter.Unmarshal,
		JSONEncoder:  jsoniter.Marshal,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,

		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Send custom error page
			if err != nil {
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
			}

			// Return from handler
			return nil
		},
	})
	httpApp.Use(recover.New(recover.Config{EnableStackTrace: true}))
	httpApp.Use(requestid.New())
	httpApp.Use(healthcheck.New())
	httpApp.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))
	httpApp.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	httpApp.Use(limiter.New(limiter.Config{
		Max:               50,
		Expiration:        10 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))
	logger := flog.GetLogger()
	httpApp.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &logger,
		SkipURIs: []string{
			"/",
			"/livez",
			"/readyz",
			"/service/user/metrics",
		},
	}))

	// hook
	httpApp.Hooks().OnRoute(func(r fiber.Route) error {
		if r.Method == http.MethodHead {
			return nil
		}
		flog.Info("[route] %+7s %s", r.Method, r.Path)
		return nil
	})

	// swagger
	if swagHandler != nil {
		httpApp.Get("/swagger/*", swagHandler)
	}

	// Handle extra
	setupMux(httpApp)

	return nil
}

func initializePprof() error {
	// Initialize serving debug profiles (optional).
	pprofs.ServePprof(httpApp, *appFlag.pprofUrl)

	if *appFlag.pprofFile != "" {
		curwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory, %w", err)
		}
		*appFlag.pprofFile = utils.ToAbsolutePath(curwd, *appFlag.pprofFile)

		cpuf, err := os.Create(*appFlag.pprofFile + ".cpu")
		if err != nil {
			flog.Fatal("Failed to create CPU pprof file: %v", err)
		}
		defer func() {
			_ = cpuf.Close()
		}()

		memf, err := os.Create(*appFlag.pprofFile + ".mem")
		if err != nil {
			flog.Fatal("Failed to create Mem pprof file: %v", err)
		}
		defer func() {
			_ = memf.Close()
		}()

		_ = pprof.StartCPUProfile(cpuf)
		defer pprof.StopCPUProfile()
		defer func() {
			_ = pprof.WriteHeapProfile(memf)
		}()

		flog.Info("Profiling info saved to '%s.(cpu|mem)'", *appFlag.pprofFile)
	}
	return nil
}

func initializeCache() error {
	// init cache
	return cache.InitCache()
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

func initializeSignal() error {
	stopSignal = utils.SignalHandler()
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

func initializeSearch() error {
	err := meilisearch.NewMeiliSearch().DefaultIndexSettings()
	if err != nil {
		flog.Error(err)
	}
	return nil
}

func initializeAlarm() error {
	return alarm.InitAlarm()
}
