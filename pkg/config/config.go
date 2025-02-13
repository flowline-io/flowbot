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

type tlsAutocertConfig struct {
	// Domains to support by autocert
	Domains []string `json:"domains" yaml:"domains" mapstructure:"domains"`
	// Name of directory where auto-certificates are cached, e.g. /etc/letsencrypt/live/your-domain-here
	CertCache string `json:"cache" yaml:"cert_cache" mapstructure:"cert_cache"`
	// Contact email for letsencrypt
	Email string `json:"email" yaml:"email" mapstructure:"email"`
}

type TLSConfig struct {
	// Flag enabling TLS
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Listen for connections on this address:port and redirect them to HTTPS port.
	RedirectHTTP string `json:"http_redirect" yaml:"redirect_http" mapstructure:"redirect_http"`
	// Enable Strict-Transport-Security by setting max_age > 0
	StrictMaxAge int `json:"strict_max_age" yaml:"strict_max_age" mapstructure:"strict_max_age"`
	// ACME autocert config, e.g. letsencrypt.org
	Autocert *tlsAutocertConfig `json:"autocert" yaml:"autocert" mapstructure:"autocert"`
	// If Autocert is not defined, provide file names of static certificate and key
	CertFile string `json:"cert_file" yaml:"certFile" mapstructure:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"keyFile" mapstructure:"key_file"`
}

type StoreType struct {
	// 16-byte key for XTEA. Used to initialize types.UidGenerator.
	UidKey string `json:"uid_key" yaml:"uid_key" mapstructure:"uid_key"`
	// Maximum number of results to return from adapter.
	MaxResults int `json:"max_results" yaml:"max_results" mapstructure:"max_results"`
	// DB adapter name to use. Should be one of those specified in `Adapters`.
	UseAdapter string `json:"use_adapter" yaml:"use_adapter" mapstructure:"use_adapter"`
	// Configurations for individual adapters.
	Adapters map[string]interface{} `json:"adapters" yaml:"adapters" mapstructure:"adapters"`
}

type Log struct {
	Level string `json:"level" yaml:"level" mapstructure:"level"`
}

type Redis struct {
	Host     string `json:"host" yaml:"host" mapstructure:"host"`
	Port     int    `json:"port" yaml:"port" mapstructure:"port"`
	DB       int    `json:"db" yaml:"db" mapstructure:"db"`
	Password string `json:"password" yaml:"pass" mapstructure:"password"`
}

type platform struct {
	Slack    Slack    `json:"slack" yaml:"slack" mapstructure:"slack"`
	Discord  Discord  `json:"discord" yaml:"discord" mapstructure:"discord"`
	Telegram Telegram `json:"telegram" yaml:"telegram" mapstructure:"telegram"`
	Tailchat Tailchat `json:"tailchat" yaml:"tailchat" mapstructure:"tailchat"`
}

type Slack struct {
	Enabled           bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	AppID             string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	ClientID          string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
	ClientSecret      string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
	SigningSecret     string `json:"signing_secret" yaml:"signing_secret" mapstructure:"signing_secret"`
	VerificationToken string `json:"verification_token" yaml:"verification_token" mapstructure:"verification_token"`
	AppToken          string `json:"app_token" yaml:"app_token" mapstructure:"app_token"`
	BotToken          string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token"`
}

type Discord struct {
	Enabled      bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	AppID        string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	PublicKey    string `json:"public_key" yaml:"public_key" mapstructure:"public_key"`
	ClientID     string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
	BotToken     string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token"`
}

type Telegram struct {
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
}

type Tailchat struct {
	Enabled   bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	ApiURL    string `json:"api_url" yaml:"api_url" mapstructure:"api_url"`
	AppID     string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	AppSecret string `json:"app_secret" yaml:"app_secret" mapstructure:"app_secret"`
}

type Engine struct {
	Type   string `json:"type" yaml:"type" mapstructure:"type"`
	Limits struct {
		Cpus   string `json:"cpus" yaml:"cpus" mapstructure:"cpus"`
		Memory string `json:"memory" yaml:"memory" mapstructure:"memory"`
	} `json:"limits" yaml:"limits" mapstructure:"limits"`
	Mounts struct {
		Bind struct {
			Allowed bool `json:"allowed" yaml:"allowed" mapstructure:"allowed"`
		} `json:"bind" yaml:"bind" mapstructure:"bind"`
	} `json:"mounts" yaml:"mounts" mapstructure:"mounts"`
	Docker struct {
		Config string `json:"config" yaml:"config" mapstructure:"config"`
	} `json:"docker" yaml:"docker" mapstructure:"docker"`
	Shell struct {
		CMD []string `json:"cmd" yaml:"cmd" mapstructure:"cmd"`
		UID string   `json:"uid" yaml:"uid" mapstructure:"uid"`
		GID string   `json:"gid" yaml:"gid" mapstructure:"gid"`
	} `json:"shell" yaml:"shell" mapstructure:"shell"`
	Machine struct {
		Host     string `json:"host" yaml:"host" mapstructure:"host"`
		Port     int    `json:"post" yaml:"port" mapstructure:"port"`
		Username string `json:"username" yaml:"username" mapstructure:"username"`
		Password string `json:"password" yaml:"password" mapstructure:"password"`
	} `json:"machine" yaml:"machine" mapstructure:"machine"`
}

type Metrics struct {
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
}

type Search struct {
	Endpoint   string            `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	MasterKey  string            `json:"master_key" yaml:"master_key" mapstructure:"master_key"`
	DataIndex  string            `json:"data_index" yaml:"data_index" mapstructure:"data_index"`
	UrlBaseMap map[string]string `json:"url_base_map" yaml:"url_base_map" mapstructure:"url_base_map"`
}

type Flowbot struct {
	URL         string `json:"url" yaml:"url" mapstructure:"url"`
	ChannelPath string `json:"channel_path" yaml:"channel_path" mapstructure:"channel_path"`
}

type Alarm struct {
	Filter       string `json:"filter" yaml:"filter" mapstructure:"filter"`
	SlackWebhook string `json:"slack_webhook" yaml:"slack_webhook" mapstructure:"slack_webhook"`
}

type Agent struct {
	Token     string `json:"token" yaml:"token" mapstructure:"token"`
	BaseUrl   string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	Model     string `json:"model" yaml:"model" mapstructure:"model"`
	ToolModel string `json:"tool_model" yaml:"tool_model" mapstructure:"tool_model"`
	Language  string `json:"language" yaml:"language" mapstructure:"language"`
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
