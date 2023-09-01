package config

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var App configType

// Large file handler config.
type mediaConfig struct {
	// The name of the handler to use for file uploads.
	UseHandler string `json:"use_handler" yaml:"use_handler" mapstructure:"use_handler"`
	// Maximum allowed size of an uploaded file
	MaxFileUploadSize int64 `json:"max_size" yaml:"max_file_upload_size" mapstructure:"max_file_upload_size"`
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

type Workflow struct {
	Worker int `json:"worker" yaml:"worker" mapstructure:"worker"`
}

type Redis struct {
	Host     string `json:"host" yaml:"host" mapstructure:"host"`
	Port     int    `json:"port" yaml:"port" mapstructure:"port"`
	DB       int    `json:"db" yaml:"db" mapstructure:"db"`
	Password string `json:"password" yaml:"pass" mapstructure:"password"`
}

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
	// Cache-Control value for static content.
	CacheControl int `json:"cache_control" yaml:"cache_control" mapstructure:"cache_control"`
	// If true, do not attempt to negotiate websocket per message compression (RFC 7692.4).
	// It should be disabled (set to true) if you are using MSFT IIS as a reverse proxy.
	WSCompressionDisabled bool `json:"ws_compression_disabled" yaml:"ws_compression_disabled" mapstructure:"ws_compression_disabled"`
	// URL path for mounting the directory with static files (usually TinodeWeb).
	StaticMount string `json:"static_mount" yaml:"static_mount" mapstructure:"static_mount"`
	// Local path to static files. All files in this path are made accessible by HTTP.
	StaticData string `json:"static_data" yaml:"static_data" mapstructure:"static_data"`
	// Salt used in signing API keys
	APIKeySalt string `json:"api_key_salt" yaml:"api_key_salt" mapstructure:"api_key_salt"`
	// Maximum message size allowed from client. Intended to prevent malicious client from sending
	// very large files inband (does not affect out of band uploads).
	MaxMessageSize int `json:"max_message_size" yaml:"max_message_size" mapstructure:"max_message_size"`
	// URL path for exposing runtime stats. Disabled if the path is blank.
	ExpvarPath string `json:"expvar" yaml:"expvar_path" mapstructure:"expvar_path"`
	// URL path for internal server status. Disabled if the path is blank.
	ServerStatusPath string `json:"server_status" yaml:"server_status_path" mapstructure:"server_status_path"`
	// Take IP address of the client from HTTP header 'X-Forwarded-For'.
	// Useful when tinode is behind a proxy. If missing, fallback to default RemoteAddr.
	UseXForwardedFor bool `json:"use_x_forwarded_for" yaml:"use_x_forwarded_for" mapstructure:"use_x_forwarded_for"`
	// 2-letter country code (ISO 3166-1 alpha-2) to assign to sessions by default
	// when the country isn't specified by the client explicitly and
	// it's impossible to infer it.
	DefaultCountryCode string `json:"default_country_code" yaml:"default_country_code" mapstructure:"default_country_code"`

	// download_path
	DownloadPath string `json:"download_path" yaml:"download_path" mapstructure:"download_path"`
	// api_url
	ApiUrl string `json:"api_url" yaml:"api_url" mapstructure:"api_url"`

	// Configs for subsystems
	Store StoreType    `json:"store_config" yaml:"store_config" mapstructure:"store_config"`
	TLS   TLSConfig    `json:"tls" yaml:"tls" mapstructure:"tls"`
	Media *mediaConfig `json:"media" yaml:"media" mapstructure:"media"`

	// Redis
	Redis Redis `json:"redis" yaml:"redis" mapstructure:"redis"`

	// Log
	Log Log `json:"log" yaml:"log" mapstructure:"log"`

	// Config for workflows
	Workflow Workflow `json:"workflow" yaml:"workflow" mapstructure:"workflow"`

	// Config for bots
	Bots interface{} `json:"bots" yaml:"bots" mapstructure:"bots"`

	// Config for vendors
	Vendors interface{} `json:"vendors" yaml:"vendors" mapstructure:"vendors"`

	// Platform
	Platform platform `json:"platform" yaml:"platform" mapstructure:"platform"`
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

func Load(path ...string) {
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		flog.Fatal("Failed to bind flags: %v", err)
	}
	for _, p := range path {
		viper.AddConfigPath(p)
	}
	viper.SetConfigName("flowbot.yaml")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		flog.Fatal("Failed to read config file: %v", err)
	}
	err = viper.Unmarshal(&App)
	if err != nil {
		flog.Fatal("Failed to unmarshal config: %v", err)
	}
}
