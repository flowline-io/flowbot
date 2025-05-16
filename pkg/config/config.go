package config

import (
	"context"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

const (
	// Base URL path for serving the streaming API.
	defaultApiPath = "/"
)

var App Type

// Contentx of the configuration file
type Type struct {
	// HTTP(S) address:port to listen on for websocket and long polling clients. Either a
	// numeric or a canonical name, e.g. ":80" or ":https". Could include a host name, e.g.
	// "localhost:80".
	// Could be blank: if TLS is not configured, will use ":80", otherwise ":443".
	// Can be overridden from the command line, see option --listen.
	Listen string `json:"listen" yaml:"listen" mapstructure:"listen"`
	// Base URL path where the streaming and large file API calls are served, default is '/'.
	// Can be overridden from the command line, see option --api_path.
	ApiPath string `json:"api_path" yaml:"api_path" mapstructure:"api_path"`

	// Configs for subsystems
	Store StoreType    `json:"store_config" yaml:"store_config" mapstructure:"store_config"`
	Media *mediaConfig `json:"media" yaml:"media" mapstructure:"media"`

	// Redis
	Redis Redis `json:"redis" yaml:"redis" mapstructure:"redis"`

	// Log
	Log Log `json:"log" yaml:"log" mapstructure:"log"`

	// Config for bots
	Bots interface{} `json:"bots" yaml:"bots" mapstructure:"bots"`

	// Config for vendors
	Vendors interface{} `json:"vendors" yaml:"vendors" mapstructure:"vendors"`

	// Platform
	Platform platform `json:"platform" yaml:"platform" mapstructure:"platform"`

	// Engine
	Engine Engine `json:"engine" yaml:"engine" mapstructure:"engine"`

	// Metrics
	Metrics Metrics `json:"metrics" yaml:"metrics" mapstructure:"metrics"`

	// Search
	Search Search `json:"search" yaml:"search" mapstructure:"search"`

	// Project
	Flowbot Flowbot `json:"flowbot" yaml:"flowbot" mapstructure:"flowbot"`

	// Alarm
	Alarm Alarm `json:"alarm" yaml:"alarm" mapstructure:"alarm"`

	// Models
	Models []Model `json:"models" yaml:"models" mapstructure:"models"`

	// Agents
	Agents []Agent `json:"agents" yaml:"agents" mapstructure:"agents"`
}

// Large file handler config.
type mediaConfig struct {
	// The name of the handler to use for file uploads.
	UseHandler string `json:"use_handler" yaml:"use_handler" mapstructure:"use_handler"`
	// Maximum allowed size of an uploaded file
	MaxFileUploadSize int64 `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
	// Garbage collection timeout
	GcPeriod int `json:"gc_period" yaml:"gc_period" mapstructure:"gc_period"`
	// Number of entries to delete in one pass
	GcBlockSize int `json:"gc_block_size" yaml:"gc_block_size" mapstructure:"gc_block_size"`
	// Individual handler config params to pass to handlers unchanged.
	Handlers map[string]interface{} `json:"handlers" yaml:"handlers" mapstructure:"handlers"`
}

type StoreType struct {
	// Maximum number of results to return from adapter.
	MaxResults int `json:"max_results" yaml:"max_results" mapstructure:"max_results"`
	// DB adapter name to use. Should be one of those specified in `Adapters`.
	UseAdapter string `json:"use_adapter" yaml:"use_adapter" mapstructure:"use_adapter"`
	// Configurations for individual adapters.
	Adapters map[string]interface{} `json:"adapters" yaml:"adapters" mapstructure:"adapters"`
}

type Log struct {
	// Log level: debug, info, warn, error, fatal, panic
	Level string `json:"level" yaml:"level" mapstructure:"level"`
}

type Redis struct {
	// Redis host
	Host string `json:"host" yaml:"host" mapstructure:"host"`
	// Redis port
	Port int `json:"port" yaml:"port" mapstructure:"port"`
	// Redis database
	DB int `json:"db" yaml:"db" mapstructure:"db"`
	// Redis password
	Password string `json:"password" yaml:"pass" mapstructure:"password"`
}

type platform struct {
	// Slack platform configuration
	Slack Slack `json:"slack" yaml:"slack" mapstructure:"slack"`
	// Discord platform configuration
	Discord Discord `json:"discord" yaml:"discord" mapstructure:"discord"`
	// Telegram platform configuration
	Telegram Telegram `json:"telegram" yaml:"telegram" mapstructure:"telegram"`
	// Tailchat platform configuration
	Tailchat Tailchat `json:"tailchat" yaml:"tailchat" mapstructure:"tailchat"`
}

type Slack struct {
	// Slack platform configuration
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Slack app ID
	AppID string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	// Slack client ID
	ClientID string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
	// Slack client secret
	ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
	// Slack signing secret
	SigningSecret string `json:"signing_secret" yaml:"signing_secret" mapstructure:"signing_secret"`
	// Slack verification token
	VerificationToken string `json:"verification_token" yaml:"verification_token" mapstructure:"verification_token"`
	// Slack app token
	AppToken string `json:"app_token" yaml:"app_token" mapstructure:"app_token"`
	// Slack bot token
	BotToken string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token"`
}

type Discord struct {
	// Discord platform configuration
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Discord app ID
	AppID string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	// Discord public key
	PublicKey string `json:"public_key" yaml:"public_key" mapstructure:"public_key"`
	// Discord client ID
	ClientID string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
	// Discord client secret
	ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
	// Discord bot token
	BotToken string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token"`
}

type Telegram struct {
	// Telegram platform configuration
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
}

type Tailchat struct {
	// Tailchat platform configuration
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Tailchat API URL
	ApiURL string `json:"api_url" yaml:"api_url" mapstructure:"api_url"`
	// Tailchat app ID
	AppID string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	// Tailchat app secret
	AppSecret string `json:"app_secret" yaml:"app_secret" mapstructure:"app_secret"`
}

type Engine struct {
	// Engine type: docker
	Type string `json:"type" yaml:"type" mapstructure:"type"`
	// Resource limits
	Limits struct {
		// CPU limit
		Cpus string `json:"cpus" yaml:"cpus" mapstructure:"cpus"`
		// Memory limit
		Memory string `json:"memory" yaml:"memory" mapstructure:"memory"`
	} `json:"limits" yaml:"limits" mapstructure:"limits"`
	Mounts struct {
		// Bind mount
		Bind struct {
			// Allowed
			Allowed bool `json:"allowed" yaml:"allowed" mapstructure:"allowed"`
		} `json:"bind" yaml:"bind" mapstructure:"bind"`
	} `json:"mounts" yaml:"mounts" mapstructure:"mounts"`
	Docker struct {
		// Docker config
		Config string `json:"config" yaml:"config" mapstructure:"config"`
	} `json:"docker" yaml:"docker" mapstructure:"docker"`
	Shell struct {
		// Command
		CMD []string `json:"cmd" yaml:"cmd" mapstructure:"cmd"`
		// User ID
		UID string `json:"uid" yaml:"uid" mapstructure:"uid"`
		// Group ID
		GID string `json:"gid" yaml:"gid" mapstructure:"gid"`
	} `json:"shell" yaml:"shell" mapstructure:"shell"`
	Machine struct {
		// Host
		Host string `json:"host" yaml:"host" mapstructure:"host"`
		// Port
		Port int `json:"post" yaml:"port" mapstructure:"port"`
		// Username
		Username string `json:"username" yaml:"username" mapstructure:"username"`
		// Password
		Password string `json:"password" yaml:"password" mapstructure:"password"`
	} `json:"machine" yaml:"machine" mapstructure:"machine"`
}

type Metrics struct {
	// Metrics endpoint
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
}

type Search struct {
	// Search endpoint
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	// Search master key
	MasterKey string `json:"master_key" yaml:"master_key" mapstructure:"master_key"`
	// Search data index
	DataIndex string `json:"data_index" yaml:"data_index" mapstructure:"data_index"`
	// Search URL base map
	UrlBaseMap map[string]string `json:"url_base_map" yaml:"url_base_map" mapstructure:"url_base_map"`
}

type Flowbot struct {
	// Flowbot URL
	URL string `json:"url" yaml:"url" mapstructure:"url"`
	// Flowbot channel path
	ChannelPath string `json:"channel_path" yaml:"channel_path" mapstructure:"channel_path"`
	// language
	Language string `json:"language" yaml:"language" mapstructure:"language"`
}

type Alarm struct {
	// Alarm filter rules
	Filter string `json:"filter" yaml:"filter" mapstructure:"filter"`
	// Slack webhook URL
	SlackWebhook string `json:"slack_webhook" yaml:"slack_webhook" mapstructure:"slack_webhook"`
}

type Agent struct {
	// Agent Name
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	// Enabled
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Use model
	Model string `json:"model" yaml:"model" mapstructure:"model"`
}

type Model struct {
	// Protocol
	Protocol string `json:"protocol" yaml:"protocol" mapstructure:"protocol"`
	// Base URL
	BaseUrl string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	// API key
	ApiKey string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	// Useful model names
	ModelNames []string `json:"model_names" yaml:"model_names" mapstructure:"model_names"`
}

func Load(path ...string) {
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("[config] Failed to bind flags: %v", err)
	}
	for _, p := range path {
		viper.AddConfigPath(p)
	}
	viper.SetConfigName("flowbot")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatalf("[config] Failed to read config file: %v", err)
	}
	err = viper.Unmarshal(&App)
	if err != nil {
		log.Fatalf("[config] Failed to unmarshal config: %v", err)
	}
}

func NewConfig(lc fx.Lifecycle) Type {
	executable, _ := os.Executable()

	curwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't get current working directory: %v", err)
	}

	log.Printf("version %s:%s:%s; pid %d; %d process(es)\n",
		version.Buildtags, executable, version.Buildstamp,
		os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))

	configFile := utils.ToAbsolutePath(curwd, "flowbot.yaml")
	log.Printf("Using config from '%s'\n", configFile)

	// Load config
	Load(".", curwd)

	// Configure root path for serving API calls.
	if App.ApiPath == "" {
		App.ApiPath = defaultApiPath
	} else {
		if !strings.HasPrefix(App.ApiPath, "/") {
			App.ApiPath = "/" + App.ApiPath
		}
		if !strings.HasSuffix(App.ApiPath, "/") {
			App.ApiPath += "/"
		}
	}
	log.Printf("API served from root URL path '%s'\n", App.ApiPath)

	// // Debug
	// if App.IsDevelopmentMode() {
	// 	viper.Debug()
	// }

	// fx hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Watch config
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("Config file changed: %s\n", e.Name)

				// Reload
				err := viper.Unmarshal(&App)
				if err != nil {
					log.Fatalf("[config] Failed to unmarshal config: %v", err)
				}
			})
			viper.WatchConfig()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return App
}
