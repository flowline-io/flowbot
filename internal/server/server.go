package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/queue"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/version"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jsoniter "github.com/json-iterator/go"
	jcr "github.com/tinode/jsonco"
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

// Large file handler config.
type mediaConfig struct {
	// The name of the handler to use for file uploads.
	UseHandler string `json:"use_handler"`
	// Maximum allowed size of an uploaded file
	MaxFileUploadSize int64 `json:"max_size"`
	// Garbage collection timeout
	GcPeriod int `json:"gc_period"`
	// Number of entries to delete in one pass
	GcBlockSize int `json:"gc_block_size"`
	// Individual handler config params to pass to handlers unchanged.
	Handlers map[string]json.RawMessage `json:"handlers"`
}

// Contentx of the configuration file
type configType struct {
	// HTTP(S) address:port to listen on for websocket and long polling clients. Either a
	// numeric or a canonical name, e.g. ":80" or ":https". Could include a host name, e.g.
	// "localhost:80".
	// Could be blank: if TLS is not configured, will use ":80", otherwise ":443".
	// Can be overridden from the command line, see option --listen.
	Listen string `json:"listen"`
	// Base URL path where the streaming and large file API calls are served, default is '/'.
	// Can be overridden from the command line, see option --api_path.
	ApiPath string `json:"api_path"`
	// Cache-Control value for static content.
	CacheControl int `json:"cache_control"`
	// If true, do not attempt to negotiate websocket per message compression (RFC 7692.4).
	// It should be disabled (set to true) if you are using MSFT IIS as a reverse proxy.
	WSCompressionDisabled bool `json:"ws_compression_disabled"`
	// URL path for mounting the directory with static files (usually TinodeWeb).
	StaticMount string `json:"static_mount"`
	// Local path to static files. All files in this path are made accessible by HTTP.
	StaticData string `json:"static_data"`
	// Salt used in signing API keys
	APIKeySalt []byte `json:"api_key_salt"`
	// Maximum message size allowed from client. Intended to prevent malicious client from sending
	// very large files inband (does not affect out of band uploads).
	MaxMessageSize int `json:"max_message_size"`
	// If true, ordinary users cannot delete their accounts.
	PermanentAccounts bool `json:"permanent_accounts"`
	// URL path for exposing runtime stats. Disabled if the path is blank.
	ExpvarPath string `json:"expvar"`
	// URL path for internal server status. Disabled if the path is blank.
	ServerStatusPath string `json:"server_status"`
	// Take IP address of the client from HTTP header 'X-Forwarded-For'.
	// Useful when tinode is behind a proxy. If missing, fallback to default RemoteAddr.
	UseXForwardedFor bool `json:"use_x_forwarded_for"`
	// 2-letter country code (ISO 3166-1 alpha-2) to assign to sessions by default
	// when the country isn't specified by the client explicitly and
	// it's impossible to infer it.
	DefaultCountryCode string `json:"default_country_code"`

	// Configs for subsystems
	Store json.RawMessage `json:"store_config"`
	Push  json.RawMessage `json:"push"`
	TLS   json.RawMessage `json:"tls"`
	Media *mediaConfig    `json:"media"`
	Redis json.RawMessage `json:"redis"`

	// Configs for extra
	Chatbot json.RawMessage `json:"chatbot"`
	Bot     json.RawMessage `json:"bots"`
	Vendor  json.RawMessage `json:"vendors"`
}

func ListenAndServe() {
	executable, _ := os.Executable()

	logFlags := flag.String("log_flags", "stdFlags",
		"Comma-separated list of log flags (as defined in https://golang.org/pkg/log/#pkg-constants without the L prefix)")
	configfile := flag.String("config", "flowbot.json", "Path to config file.")
	listenOn := flag.String("listen", "", "Override address and port to listen on for HTTP(S) clients.")
	apiPath := flag.String("api_path", "", "Override the base URL path where API is served.")
	tlsEnabled := flag.Bool("tls_enabled", false, "Override config value for enabling TLS.")
	expvarPath := flag.String("expvar", "", "Override the URL path where runtime stats are exposed. Use '-' to disable.")
	serverStatusPath := flag.String("server_status", "",
		"Override the URL path where the server's internal status is displayed. Use '-' to disable.")
	pprofFile := flag.String("pprof", "", "File name to save profiling info to. Disabled if not set.")
	pprofUrl := flag.String("pprof_url", "", "Debugging only! URL path for exposing profiling info. Disabled if not set.")
	flag.Parse()

	logs.Init(os.Stderr, *logFlags)

	curwd, err := os.Getwd()
	if err != nil {
		logs.Err.Fatal("Couldn't get current working directory: ", err)
	}

	logs.Info.Printf("Server v%s:%s:%s; pid %d; %d process(es)",
		currentVersion, executable, version.Buildstamp,
		os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))

	*configfile = utils.ToAbsolutePath(curwd, *configfile)
	logs.Info.Printf("Using config from '%s'", *configfile)

	var config configType
	if file, err := os.Open(*configfile); err != nil {
		logs.Err.Fatal("Failed to read config file: ", err)
	} else {
		jr := jcr.New(file)
		if err = json.NewDecoder(jr).Decode(&config); err != nil {
			switch jerr := err.(type) {
			case *json.UnmarshalTypeError:
				lnum, cnum, _ := jr.LineAndChar(jerr.Offset)
				logs.Err.Fatalf("Unmarshall error in config file in %s at %d:%d (offset %d bytes): %s",
					jerr.Field, lnum, cnum, jerr.Offset, jerr.Error())
			case *json.SyntaxError:
				lnum, cnum, _ := jr.LineAndChar(jerr.Offset)
				logs.Err.Fatalf("Syntax error in config file at %d:%d (offset %d bytes): %s",
					lnum, cnum, jerr.Offset, jerr.Error())
			default:
				logs.Err.Fatal("Failed to parse config file: ", err)
			}
		}
		_ = file.Close()
	}

	if *listenOn != "" {
		config.Listen = *listenOn
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
		evpath = config.ExpvarPath
	}
	statsInit(app, evpath)
	statsRegisterInt("Version")
	decVersion := utils.Base10Version(utils.ParseVersion(version.Buildstamp))
	if decVersion <= 0 {
		decVersion = utils.Base10Version(utils.ParseVersion(currentVersion))
	}
	statsSet("Version", decVersion)

	// Initialize serving debug profiles (optional).
	servePprof(app, *pprofUrl)

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

	// Initialize store.
	hookStore()

	err = store.Store.Open(config.Store)
	logs.Info.Println("DB adapter opened")
	if err != nil {
		logs.Err.Fatal("Failed to connect to DB: ", err)
	}
	defer func() {
		_ = store.Store.Close()
		logs.Info.Println("Closed database connection(s)")
		logs.Info.Println("All done, good bye")
	}()
	statsRegisterDbStats()

	// Maximum message size
	globals.maxMessageSize = int64(config.MaxMessageSize)
	if globals.maxMessageSize <= 0 {
		globals.maxMessageSize = defaultMaxMessageSize
	}

	globals.useXForwardedFor = config.UseXForwardedFor

	// Websocket compression.
	globals.wsCompression = !config.WSCompressionDisabled

	if config.Media != nil {
		if config.Media.UseHandler == "" {
			config.Media = nil
		} else {
			globals.maxFileUploadSize = config.Media.MaxFileUploadSize
			if config.Media.Handlers != nil {
				var conf string
				if params := config.Media.Handlers[config.Media.UseHandler]; params != nil {
					conf = string(params)
				}
				if err = store.UseMediaHandler(config.Media.UseHandler, conf); err != nil {
					logs.Err.Fatalf("Failed to init media handler '%s': %s", config.Media.UseHandler, err)
				}
			}
			if config.Media.GcPeriod > 0 && config.Media.GcBlockSize > 0 {
				globals.mediaGcPeriod = time.Second * time.Duration(config.Media.GcPeriod)
				stopFilesGc := largeFileRunGarbageCollection(globals.mediaGcPeriod, config.Media.GcBlockSize)
				defer func() {
					stopFilesGc <- true
					logs.Info.Println("Stopped files garbage collector")
				}()
			}
		}
	}

	tlsConfig, err := utils.ParseTLSConfig(*tlsEnabled, config.TLS)
	if err != nil {
		logs.Err.Fatalln(err)
	}

	// Initialize config
	hookConfig(config.Chatbot)

	// Initialize bots
	hookBot(config.Bot, config.Vendor)

	// Initialize channels
	hookChannel()

	// Mounted
	hookMounted()

	// Queue
	hookQueue()

	// Event
	hookEvent()

	// Configure root path for serving API calls.
	if *apiPath != "" {
		config.ApiPath = *apiPath
	}
	if config.ApiPath == "" {
		config.ApiPath = defaultApiPath
	} else {
		if !strings.HasPrefix(config.ApiPath, "/") {
			config.ApiPath = "/" + config.ApiPath
		}
		if !strings.HasSuffix(config.ApiPath, "/") {
			config.ApiPath += "/"
		}
	}
	logs.Info.Printf("API served from root URL path '%s'", config.ApiPath)

	// Best guess location of the main endpoint.
	globals.servingAt = config.Listen + config.ApiPath
	if tlsConfig != nil {
		globals.servingAt = "https://" + globals.servingAt
	} else {
		globals.servingAt = "http://" + globals.servingAt
	}

	sspath := *serverStatusPath
	if sspath == "" || sspath == "-" {
		sspath = config.ServerStatusPath
	}
	if sspath != "" && sspath != "-" {
		logs.Info.Printf("Server status is available at '%s'", sspath)
		app.Get(sspath, adaptor.HTTPHandlerFunc(serveStatus))
	}

	if err = listenAndServe(app, config.Listen, tlsConfig, signalHandler()); err != nil {
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
			statsShutdown()

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
