package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/channels/crawler"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/pprofs"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"github.com/flowline-io/flowbot/version"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/pflag"
)

var (
	// stop signal
	stopSignal <-chan bool
	// tls config
	tlsConfig *tls.Config
	// swagger
	swagHandler fiber.Handler
	// fiber app
	httpApp *fiber.App
	// flag variables
	appFlag struct {
		configFile       *string
		listenOn         *string
		apiPath          *string
		tlsEnabled       *bool
		expvarPath       *string
		serverStatusPath *string
		pprofFile        *string
		pprofUrl         *string
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

	// init http
	if err = initializeHttp(); err != nil {
		return err
	}
	flog.Info("initialize Http ok")

	// init stats
	if err = initializeStats(); err != nil {
		return err
	}
	flog.Info("initialize Stats ok")

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

	// init tls
	if err = initializeTLS(); err != nil {
		return err
	}
	flog.Info("initialize TLS ok")

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

	return nil
}

func initializeLog() error {
	flog.Init()
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
	appFlag.expvarPath = pflag.String("expvar", "", "Override the URL path where runtime stats are exposed. Use '-' to disable.")
	appFlag.serverStatusPath = pflag.String("server_status", "",
		"Override the URL path where the server's internal status is displayed. Use '-' to disable.")
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

	globals.useXForwardedFor = config.App.UseXForwardedFor

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
	// Set up HTTP server. Must use non-default mux because of expvar.
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
	httpApp.Use(recover.New())
	httpApp.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))
	httpApp.Use(requestid.New())
	logger := flog.GetLogger()
	httpApp.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &logger,
		SkipURIs: []string{
			"/health",
		},
	}))
	// swagger
	if swagHandler != nil {
		httpApp.Get("/swagger/*", swagHandler)
	}

	// Handle extra
	setupMux(httpApp)

	return nil
}

func initializeStats() error {
	// Exposing values for statistics and monitoring.
	evpath := *appFlag.expvarPath
	if evpath == "" {
		evpath = config.App.ExpvarPath
	}
	stats.Init(httpApp, evpath)
	stats.RegisterInt("Version")
	decVersion := utils.Base10Version(utils.ParseVersion(version.Buildstamp))
	if decVersion <= 0 {
		decVersion = utils.Base10Version(utils.ParseVersion(version.Buildtags))
	}
	stats.Set("Version", decVersion)

	sspath := *appFlag.serverStatusPath
	if sspath == "" || sspath == "-" {
		sspath = config.App.ServerStatusPath
	}
	if sspath != "" && sspath != "-" {
		flog.Debug("Server status is available at '%s'", sspath)
		httpApp.Get(sspath, adaptor.HTTPHandlerFunc(serveStatus))
	}

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
	cache.InitCache()
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
	stats.RegisterDbStats()

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

func initializeTLS() error {
	var err error
	// TLS
	tlsConfig, err = utils.ParseTLSConfig(*appFlag.tlsEnabled, config.App.TLS)
	if err != nil {
		return fmt.Errorf("failed to parse TLS config, %w", err)
	}
	return nil
}

func initializeChatbot(signal <-chan bool) error {
	// Initialize bots
	hookBot(config.App.Bots, config.App.Vendors)

	// Initialize channels
	hookChannel()

	// Mounted
	hookMounted()

	// Platform
	hookPlatform(signal)

	return nil
}

// init channels
func initializeChannels() error {
	// register channels
	registerChannels := sets.NewString()
	for name, handler := range channels.List() {
		registerChannels.Insert(name)

		state := model.ChannelInactive
		if handler.Enable {
			state = model.ChannelActive
		}
		channel, _ := store.Database.GetChannelByName(name)
		if channel == nil {
			channel = &model.Channel{
				Name:  name,
				Flag:  types.Id(),
				State: state,
			}
			if _, err := store.Database.CreateChannel(channel); err != nil {
				flog.Error(err)
			}
		} else {
			channel.State = state
			err := store.Database.UpdateChannel(channel)
			if err != nil {
				flog.Error(err)
			}
		}
	}

	// inactive channels
	list, err := store.Database.GetChannels()
	if err != nil {
		flog.Error(err)
	}
	for _, channel := range list {
		if !registerChannels.Has(channel.Name) {
			channel.State = model.ChannelInactive
			if err := store.Database.UpdateChannel(channel); err != nil {
				flog.Error(err)
			}
		}
	}

	return nil
}

// init crawler
func initializeCrawler() error {
	c := crawler.New()
	globals.crawler = c
	c.Send = func(id, name string, out []map[string]string) {
		if len(out) == 0 {
			return
		}

		// todo find topic
		_, _ = fmt.Println(id)

		keys := []string{"No"}
		for k := range out[0] {
			keys = append(keys, k)
		}

		var content interface{}
		if len(out) <= 10 {
			sort.Strings(keys)
			builder := types.MsgBuilder{}
			for index, item := range out {
				builder.AppendTextLine(fmt.Sprintf("--- %d ---", index+1), types.TextOption{})
				for _, k := range keys {
					if k == "No" {
						continue
					}
					builder.AppendText(fmt.Sprintf("%s: ", k), types.TextOption{IsBold: true})
					if utils.IsUrl(item[k]) {
						builder.AppendTextLine(item[k], types.TextOption{IsLink: true})
					} else {
						builder.AppendTextLine(item[k], types.TextOption{})
					}
				}
			}
			_, content = builder.Content()
		} else {
			var row [][]interface{}
			for index, item := range out {
				var tmp []interface{}
				for _, k := range keys {
					if k == "No" {
						tmp = append(tmp, index+1)
						continue
					}
					tmp = append(tmp, item[k])
				}
				row = append(row, tmp)
			}
			title := fmt.Sprintf("Channel %s (%d)", name, len(out))
			res := bots.StorePage(types.Context{}, model.PageTable, title, types.TableMsg{
				Title:  title,
				Header: keys,
				Row:    row,
			})
			_, content = res.Convert()
		}
		if content == nil {
			return
		}

		// stats inc
		stats.Inc("ChannelPublishTotal", 1)

		// todo send content
		_, _ = fmt.Println("channel publish", content)
	}

	var rules []crawler.Rule
	for _, publisher := range channels.List() {
		rules = append(rules, *publisher)
	}

	err := c.Init(rules...)
	if err != nil {
		return err
	}
	c.Run()
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

	go func() {
		if err = router.Run(context.Background()); err != nil {
			flog.Error(err)
		}
	}()

	return nil
}
