package config

import (
	"log"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var App configType

// Contentx of the configuration file
type configType struct {
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

	// Agent
	Agent Agent `json:"agent" yaml:"agent" mapstructure:"agent"`
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
}

type Alarm struct {
	// Alarm filter rules
	Filter string `json:"filter" yaml:"filter" mapstructure:"filter"`
	// Slack webhook URL
	SlackWebhook string `json:"slack_webhook" yaml:"slack_webhook" mapstructure:"slack_webhook"`
}

type Agent struct {
	// Agent token
	Token string `json:"token" yaml:"token" mapstructure:"token"`
	// Agent base URL
	BaseUrl string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	// Agent model
	Model string `json:"model" yaml:"model" mapstructure:"model"`
	// Agent tool model
	ToolModel string `json:"tool_model" yaml:"tool_model" mapstructure:"tool_model"`
	// Agent language
	Language string `json:"language" yaml:"language" mapstructure:"language"`
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
