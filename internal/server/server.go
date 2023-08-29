package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/pprofs"
	"github.com/flowline-io/flowbot/pkg/queue"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/version"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/pflag"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	// Store
	_ "github.com/flowline-io/flowbot/internal/store/mysql"

	// File upload handlers
	_ "github.com/flowline-io/flowbot/pkg/media/fs"
	_ "github.com/flowline-io/flowbot/pkg/media/s3"
)

const (
	// currentVersion is the current API/protocol version
	currentVersion = "0.1"
	// minSupportedVersion is the minimum supported API version
	// minSupportedVersion = "0.1"

	// idleSessionTimeout defines duration of being idle before terminating a session.
	idleSessionTimeout = time.Second * 55

	// defaultMaxMessageSize is the default maximum message size
	defaultMaxMessageSize = 1 << 19 // 512K

	// Base URL path for serving the streaming API.
	defaultApiPath = "/"
)

func ListenAndServe() {
	executable, _ := os.Executable()

	logFlags := pflag.String("log_flags", "stdFlags",
		"Comma-separated list of log flags (as defined in https://golang.org/pkg/log/#pkg-constants without the L prefix)")
	configFile := pflag.String("config", "flowbot.json", "Path to config file.")
	listenOn := pflag.String("listen", "", "Override address and port to listen on for HTTP(S) clients.")
	apiPath := pflag.String("api_path", "", "Override the base URL path where API is served.")
	tlsEnabled := pflag.Bool("tls_enabled", false, "Override config value for enabling TLS.")
	expvarPath := pflag.String("expvar", "", "Override the URL path where runtime stats are exposed. Use '-' to disable.")
	serverStatusPath := pflag.String("server_status", "",
		"Override the URL path where the server's internal status is displayed. Use '-' to disable.")
	pprofFile := pflag.String("pprof", "", "File name to save profiling info to. Disabled if not set.")
	pprofUrl := pflag.String("pprof_url", "", "Debugging only! URL path for exposing profiling info. Disabled if not set.")
	pflag.Parse()

	logs.Init(os.Stderr, *logFlags)

	curwd, err := os.Getwd()
	if err != nil {
		logs.Err.Fatal("Couldn't get current working directory: ", err)
	}

	logs.Info.Printf("Server v%s:%s:%s; pid %d; %d process(es)",
		currentVersion, executable, version.Buildstamp,
		os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))

	*configFile = utils.ToAbsolutePath(curwd, *configFile)
	logs.Info.Printf("Using config from '%s'", *configFile)

	// Load config
	config.Load(".", curwd)

	if *listenOn != "" {
		config.App.Listen = *listenOn
	}

	// Set up HTTP server. Must use non-default mux because of expvar.
	app := fiber.New(fiber.Config{
		JSONDecoder:  jsoniter.Unmarshal,
		JSONEncoder:  jsoniter.Marshal,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,

		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Status code defaults to 500
			code := fiber.StatusInternalServerError

			// Retrieve the custom status code if it's a *fiber.Error
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			// Send custom error page
			err = ctx.Status(code).JSON(types.KV{"code": code, "message": err.Error()})
			if err != nil {
				logs.Err.Println(err)
				return ctx.Status(fiber.StatusInternalServerError).
					JSON(types.KV{"code": fiber.StatusInternalServerError, "message": err.Error()})
			}

			// Return from handler
			return nil
		},
	})
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(requestid.New())
	app.Use(logger.New())

	// Handle extra
	hookMux(app)

	// Exposing values for statistics and monitoring.
	evpath := *expvarPath
	if evpath == "" {
		evpath = config.App.ExpvarPath
	}
	stats.Init(app, evpath)
	stats.RegisterInt("Version")
	decVersion := utils.Base10Version(utils.ParseVersion(version.Buildstamp))
	if decVersion <= 0 {
		decVersion = utils.Base10Version(utils.ParseVersion(currentVersion))
	}
	stats.Set("Version", decVersion)

	// Initialize serving debug profiles (optional).
	pprofs.ServePprof(app, *pprofUrl)

	if *pprofFile != "" {
		*pprofFile = utils.ToAbsolutePath(curwd, *pprofFile)

		cpuf, err := os.Create(*pprofFile + ".cpu")
		if err != nil {
			logs.Err.Fatal("Failed to create CPU pprof file: ", err)
		}
		defer func() {
			_ = cpuf.Close()
		}()

		memf, err := os.Create(*pprofFile + ".mem")
		if err != nil {
			logs.Err.Fatal("Failed to create Mem pprof file: ", err)
		}
		defer func() {
			_ = memf.Close()
		}()

		_ = pprof.StartCPUProfile(cpuf)
		defer pprof.StopCPUProfile()
		defer func() {
			_ = pprof.WriteHeapProfile(memf)
		}()

		logs.Info.Printf("Profiling info saved to '%s.(cpu|mem)'", *pprofFile)
	}

	// init cache
	cache.InitCache()

	// init database
	mysql.Init()
	store.Init()

	// Open database
	err = store.Store.Open(config.App.Store)
	if err != nil {
		logs.Err.Fatal("Failed to open DB: ", err)
	}
	logs.Info.Println("DB adapter opened")
	if err != nil {
		logs.Err.Fatal("Failed to connect to DB: ", err)
	}
	defer func() {
		_ = store.Store.Close()
		logs.Info.Println("Closed database connection(s)")
		logs.Info.Println("All done, good bye")
	}()
	stats.RegisterDbStats()

	// Maximum message size
	globals.maxMessageSize = int64(config.App.MaxMessageSize)
	if globals.maxMessageSize <= 0 {
		globals.maxMessageSize = defaultMaxMessageSize
	}

	globals.useXForwardedFor = config.App.UseXForwardedFor

	// Websocket compression.
	globals.wsCompression = !config.App.WSCompressionDisabled

	if config.App.Media != nil {
		if config.App.Media.UseHandler == "" {
			config.App.Media = nil
		} else {
			globals.maxFileUploadSize = config.App.Media.MaxFileUploadSize
			if config.App.Media.Handlers != nil {
				var conf string
				if params := config.App.Media.Handlers[config.App.Media.UseHandler]; params != nil {
					data, err := json.Marshal(params)
					if err != nil {
						logs.Err.Fatalf("Failed to marshal media handler '%s': %s", config.App.Media.UseHandler, err)
					}
					conf = string(data)
				}
				if err = store.UseMediaHandler(config.App.Media.UseHandler, conf); err != nil {
					logs.Err.Fatalf("Failed to init media handler '%s': %s", config.App.Media.UseHandler, err)
				}
			}
			if config.App.Media.GcPeriod > 0 && config.App.Media.GcBlockSize > 0 {
				globals.mediaGcPeriod = time.Second * time.Duration(config.App.Media.GcPeriod)
				stopFilesGc := largeFileRunGarbageCollection(globals.mediaGcPeriod, config.App.Media.GcBlockSize)
				defer func() {
					stopFilesGc <- true
					logs.Info.Println("Stopped files garbage collector")
				}()
			}
		}
	}

	tlsConfig, err := utils.ParseTLSConfig(*tlsEnabled, config.App.TLS)
	if err != nil {
		logs.Err.Fatalln(err)
	}

	// Initialize bots
	hookBot(config.App.Bots, config.App.Vendors)

	// Initialize channels
	hookChannel()

	// Mounted
	hookMounted()

	// Queue
	queue.Init()
	queue.InitMessageQueue(NewAsyncMessageConsumer())

	// Event
	hookEvent()

	// Configure root path for serving API calls.
	if *apiPath != "" {
		config.App.ApiPath = *apiPath
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
	logs.Info.Printf("API served from root URL path '%s'", config.App.ApiPath)

	// Best guess location of the main endpoint.
	globals.servingAt = config.App.Listen + config.App.ApiPath
	if tlsConfig != nil {
		globals.servingAt = "https://" + globals.servingAt
	} else {
		globals.servingAt = "http://" + globals.servingAt
	}

	sspath := *serverStatusPath
	if sspath == "" || sspath == "-" {
		sspath = config.App.ServerStatusPath
	}
	if sspath != "" && sspath != "-" {
		logs.Info.Printf("Server status is available at '%s'", sspath)
		app.Get(sspath, adaptor.HTTPHandlerFunc(serveStatus))
	}

	if err = listenAndServe(app, config.App.Listen, tlsConfig, signalHandler()); err != nil {
		logs.Err.Fatal(err)
	}
}

func listenAndServe(app *fiber.App, addr string, tlfConf *tls.Config, stop <-chan bool) error {
	globals.shuttingDown = false

	httpdone := make(chan bool)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(map[string]any{"flowbot": currentVersion})
	})

	go func() {
		if tlfConf != nil {
			err := app.ListenTLSWithCertificate(addr, tlfConf.Certificates[0])
			if err != nil {
				logs.Err.Println(err)
			}
		} else {
			err := app.Listen(addr)
			if err != nil {
				logs.Err.Println(err)
			}
		}
		httpdone <- true
	}()

	// Wait for either a termination signal or an error
Loop:
	for {
		select {
		case <-stop:
			// Flip the flag that we are terminating and close the Accept-ing socket, so no new connections are possible.
			globals.shuttingDown = true
			// Give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := app.ShutdownWithContext(ctx); err != nil {
				// failure/timeout shutting down the server gracefully
				logs.Err.Println("HTTP server failed to terminate gracefully", err)
			}

			// While the server shuts down, termianate all sessions.
			globals.sessionStore.Shutdown()

			// Stop publishing statistics.
			stats.Shutdown()

			// Shutdown the hub. The hub will shutdown topics.
			hubdone := make(chan bool)

			// Wait for the hub to finish.
			<-hubdone
			cancel()

			// Shutdown Extra
			globals.crawler.Shutdown()
			globals.worker.Shutdown()
			globals.scheduler.Shutdown()
			globals.manager.Shutdown()
			for _, ruleset := range globals.cronRuleset {
				ruleset.Shutdown()
			}
			event.Shutdown()
			queue.Shutdown()
			cache.Shutdown()

			break Loop
		case <-httpdone:
			break Loop
		}
	}
	return nil
}
