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

// Type of the configuration file
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

	// DevMode enables dev features like dev-login
	DevMode bool `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode"`

	// Config for bots
	Bots any `json:"bots" yaml:"bots" mapstructure:"bots"`

	// Config for vendors
	Vendors any `json:"vendors" yaml:"vendors" mapstructure:"vendors"`

	// Platform
	Platform platform `json:"platform" yaml:"platform" mapstructure:"platform"`

	// Executor
	Executor Executor `json:"executor" yaml:"executor" mapstructure:"executor"`

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

	// Homelab app registry and lifecycle configuration
	Homelab Homelab `json:"homelab" yaml:"homelab" mapstructure:"homelab"`

	// Pipeline definitions for cross-service event-driven automation
	Pipelines []Pipeline `json:"pipelines" yaml:"pipelines" mapstructure:"pipelines"`

	// Notify configuration for notification gateway
	Notify Notify `json:"notify" yaml:"notify" mapstructure:"notify"`

	// OpenTelemetry tracing configuration
	Tracing Tracing `json:"tracing" yaml:"tracing" mapstructure:"tracing"`
}

// Notify holds notification gateway configuration including templates and rules.
type Notify struct {
	// Templates defines notification message templates indexed by ID.
	Templates []NotifyTemplate `json:"templates" yaml:"templates" mapstructure:"templates"`
	// Rules defines notification filtering and aggregation rules.
	Rules []NotifyRule `json:"rules" yaml:"rules" mapstructure:"rules"`
}

// NotifyTemplate defines a notification message template with optional per-channel overrides.
type NotifyTemplate struct {
	ID              string           `json:"id" yaml:"id" mapstructure:"id"`
	Name            string           `json:"name" yaml:"name" mapstructure:"name"`
	Description     string           `json:"description" yaml:"description" mapstructure:"description"`
	DefaultFormat   string           `json:"default_format" yaml:"default_format" mapstructure:"default_format"`
	DefaultTemplate string           `json:"default_template" yaml:"default_template" mapstructure:"default_template"`
	Overrides       []NotifyOverride `json:"overrides" yaml:"overrides" mapstructure:"overrides"`
}

// NotifyOverride defines a channel-specific template override.
type NotifyOverride struct {
	Channel  string `json:"channel" yaml:"channel" mapstructure:"channel"`
	Format   string `json:"format" yaml:"format" mapstructure:"format"`
	Template string `json:"template" yaml:"template" mapstructure:"template"`
}

// NotifyRuleAction defines the action to take when a rule matches.
type NotifyRuleAction string

// Rule action constants.
const (
	NotifyRuleActionThrottle  NotifyRuleAction = "throttle"
	NotifyRuleActionAggregate NotifyRuleAction = "aggregate"
	NotifyRuleActionMute      NotifyRuleAction = "mute"
	NotifyRuleActionDrop      NotifyRuleAction = "drop"
)

// NotifyRuleMatch defines the event and channel matching criteria.
type NotifyRuleMatch struct {
	Event   string `json:"event" yaml:"event" mapstructure:"event"`
	Channel string `json:"channel" yaml:"channel" mapstructure:"channel"`
}

// NotifyRuleParams holds action-specific parameters.
type NotifyRuleParams struct {
	Window      string `json:"window" yaml:"window" mapstructure:"window"`
	Limit       int    `json:"limit" yaml:"limit" mapstructure:"limit"`
	DigestTplID string `json:"digest_template_id" yaml:"digest_template_id" mapstructure:"digest_template_id"`
	DelayedSend bool   `json:"delayed_send" yaml:"delayed_send" mapstructure:"delayed_send"`
}

// NotifyRule defines a notification filtering or aggregation rule.
type NotifyRule struct {
	ID        string           `json:"id" yaml:"id" mapstructure:"id"`
	Action    NotifyRuleAction `json:"action" yaml:"action" mapstructure:"action"`
	Match     NotifyRuleMatch  `json:"match" yaml:"match" mapstructure:"match"`
	Condition string           `json:"condition" yaml:"condition" mapstructure:"condition"`
	Priority  int              `json:"priority" yaml:"priority" mapstructure:"priority"`
	Params    NotifyRuleParams `json:"params" yaml:"params" mapstructure:"params"`
}

// Tracing configures OpenTelemetry distributed tracing.
type Tracing struct {
	// Enabled toggles trace export
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Endpoint is the OTLP HTTP endpoint (e.g. http://localhost:4318/v1/traces)
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	// ServiceName identifies this service in traces
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	// Environment tag (development, staging, production)
	Environment string `json:"environment" yaml:"environment" mapstructure:"environment"`
	// SampleRate controls trace sampling (0.0-1.0)
	SampleRate float64 `json:"sample_rate" yaml:"sample_rate" mapstructure:"sample_rate"`
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
	Handlers map[string]any `json:"handlers" yaml:"handlers" mapstructure:"handlers"`
}

type StoreType struct {
	// Maximum number of results to return from adapter.
	MaxResults int `json:"max_results" yaml:"max_results" mapstructure:"max_results"`
	// DB adapter name to use. Should be one of those specified in `Adapters`.
	UseAdapter string `json:"use_adapter" yaml:"use_adapter" mapstructure:"use_adapter"`
	// Configurations for individual adapters.
	Adapters map[string]any `json:"adapters" yaml:"adapters" mapstructure:"adapters"`
}

type Log struct {
	// Log level: debug, info, warn, error, fatal, panic
	Level string `json:"level" yaml:"level" mapstructure:"level"`
	// Caller enables caller (file:line) info in all log levels
	Caller bool `json:"caller" yaml:"caller" mapstructure:"caller"`
	// StackTrace enables full stack traces on errors
	StackTrace bool `json:"stackTrace" yaml:"stackTrace" mapstructure:"stackTrace"`
	// JSONOutput writes JSON to stdout instead of human-readable console format
	JSONOutput bool `json:"jsonOutput" yaml:"jsonOutput" mapstructure:"jsonOutput"`
	// FileLog enables file logging (defaults to XDG config dir)
	FileLog bool `json:"fileLog" yaml:"fileLog" mapstructure:"fileLog"`
	// FileLogPath overrides the default log file path
	FileLogPath string `json:"fileLogPath" yaml:"fileLogPath" mapstructure:"fileLogPath"`
	// ModuleLevel sets per-module log levels, e.g. {"pipeline": "debug"}
	ModuleLevel map[string]string `json:"moduleLevel" yaml:"moduleLevel" mapstructure:"moduleLevel"`
	// Sampling configures burst sampling for high-frequency log points
	Sampling *LogSampling `json:"sampling" yaml:"sampling" mapstructure:"sampling"`
	// Rotation configures log file rotation
	Rotation *LogRotation `json:"rotation" yaml:"rotation" mapstructure:"rotation"`
}

// LogSampling configures burst sampling to reduce noise from high-frequency log points.
type LogSampling struct {
	// Burst allows this many events in the period before sampling kicks in
	Burst int `json:"burst" yaml:"burst" mapstructure:"burst"`
	// Period is the sampling window in seconds
	Period int `json:"period" yaml:"period" mapstructure:"period"`
}

// LogRotation configures log file rotation using lumberjack.
type LogRotation struct {
	// MaxSize is the maximum size in megabytes before rotation
	MaxSize int `json:"maxSize" yaml:"maxSize" mapstructure:"maxSize"`
	// MaxAge is the maximum number of days to retain old log files
	MaxAge int `json:"maxAge" yaml:"maxAge" mapstructure:"maxAge"`
	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int `json:"maxBackups" yaml:"maxBackups" mapstructure:"maxBackups"`
	// Compress determines if rotated log files are gzipped
	Compress bool `json:"compress" yaml:"compress" mapstructure:"compress"`
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

type Executor struct {
	// Executor type: docker
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
		Port int `json:"port" yaml:"port" mapstructure:"port"`
		// Username
		Username string `json:"username" yaml:"username" mapstructure:"username"`
		// Password
		Password string `json:"password" yaml:"password" mapstructure:"password"`
		// HostKey is the base64-encoded SSH host public key
		HostKey string `json:"host_key" yaml:"host_key" mapstructure:"host_key"`
	} `json:"machine" yaml:"machine" mapstructure:"machine"`
}

type Metrics struct {
	// Enabled
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Metrics endpoint
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
}

type Search struct {
	// Enabled
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
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

type Homelab struct {
	Root        string             `json:"root" yaml:"root" mapstructure:"root"`
	AppsDir     string             `json:"apps_dir" yaml:"apps_dir" mapstructure:"apps_dir"`
	ComposeFile string             `json:"compose_file" yaml:"compose_file" mapstructure:"compose_file"`
	Runtime     HomelabRuntime     `json:"runtime" yaml:"runtime" mapstructure:"runtime"`
	Allowlist   []string           `json:"allowlist" yaml:"allowlist" mapstructure:"allowlist"`
	Permissions HomelabPermissions `json:"permissions" yaml:"permissions" mapstructure:"permissions"`
	Discovery   HomelabDiscovery   `json:"discovery" yaml:"discovery" mapstructure:"discovery"`
}

type HomelabRuntime struct {
	Mode         string `json:"mode" yaml:"mode" mapstructure:"mode"`
	DockerSocket string `json:"docker_socket" yaml:"docker_socket" mapstructure:"docker_socket"`
	SSHHost      string `json:"ssh_host" yaml:"ssh_host" mapstructure:"ssh_host"`
	SSHPort      int    `json:"ssh_port" yaml:"ssh_port" mapstructure:"ssh_port"`
	SSHUser      string `json:"ssh_user" yaml:"ssh_user" mapstructure:"ssh_user"`
	SSHPassword  string `json:"ssh_password" yaml:"ssh_password" mapstructure:"ssh_password"`
	SSHKey       string `json:"ssh_key" yaml:"ssh_key" mapstructure:"ssh_key"`
	SSHHostKey   string `json:"ssh_host_key" yaml:"ssh_host_key" mapstructure:"ssh_host_key"`
}

type HomelabPermissions struct {
	Status  bool `json:"status" yaml:"status" mapstructure:"status"`
	Logs    bool `json:"logs" yaml:"logs" mapstructure:"logs"`
	Start   bool `json:"start" yaml:"start" mapstructure:"start"`
	Stop    bool `json:"stop" yaml:"stop" mapstructure:"stop"`
	Restart bool `json:"restart" yaml:"restart" mapstructure:"restart"`
	Pull    bool `json:"pull" yaml:"pull" mapstructure:"pull"`
	Update  bool `json:"update" yaml:"update" mapstructure:"update"`
	Exec    bool `json:"exec" yaml:"exec" mapstructure:"exec"`
}

type HomelabDiscovery struct {
	ProbeEnabled       bool     `json:"probe_enabled" yaml:"probe_enabled" mapstructure:"probe_enabled"`
	ProbeTimeout       string   `json:"probe_timeout" yaml:"probe_timeout" mapstructure:"probe_timeout"`
	ProbeConcurrency   int      `json:"probe_concurrency" yaml:"probe_concurrency" mapstructure:"probe_concurrency"`
	ProbeNetworks      []string `json:"probe_networks" yaml:"probe_networks" mapstructure:"probe_networks"`
	ProbePortStrategy  string   `json:"probe_port_strategy" yaml:"probe_port_strategy" mapstructure:"probe_port_strategy"`
	FingerprintEnabled bool     `json:"fingerprint_enabled" yaml:"fingerprint_enabled" mapstructure:"fingerprint_enabled"`
	LabelPriority      bool     `json:"label_priority" yaml:"label_priority" mapstructure:"label_priority"`
}

type Pipeline struct {
	Name        string          `json:"name" yaml:"name" mapstructure:"name"`
	Description string          `json:"description" yaml:"description" mapstructure:"description"`
	Enabled     bool            `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Resumable   bool            `json:"resumable" yaml:"resumable" mapstructure:"resumable"`
	Trigger     PipelineTrigger `json:"trigger" yaml:"trigger" mapstructure:"trigger"`
	Steps       []PipelineStep  `json:"steps" yaml:"steps" mapstructure:"steps"`
}

type PipelineStep struct {
	Name       string             `json:"name" yaml:"name" mapstructure:"name"`
	Capability string             `json:"capability" yaml:"capability" mapstructure:"capability"`
	Operation  string             `json:"operation" yaml:"operation" mapstructure:"operation"`
	Params     map[string]any     `json:"params" yaml:"params" mapstructure:"params"`
	Retry      *PipelineStepRetry `json:"retry" yaml:"retry" mapstructure:"retry"`
}

// PipelineStepRetry mirrors types.RetryConfig for config parsing.
type PipelineStepRetry struct {
	MaxAttempts int      `json:"max_attempts" yaml:"max_attempts" mapstructure:"max_attempts"`
	Delay       string   `json:"delay" yaml:"delay" mapstructure:"delay"`
	Backoff     string   `json:"backoff" yaml:"backoff" mapstructure:"backoff"`
	MaxDelay    string   `json:"max_delay" yaml:"max_delay" mapstructure:"max_delay"`
	Jitter      bool     `json:"jitter" yaml:"jitter" mapstructure:"jitter"`
	RetryOn     []string `json:"retry_on" yaml:"retry_on" mapstructure:"retry_on"`
}

type PipelineTrigger struct {
	Event string `json:"event" yaml:"event" mapstructure:"event"`
}

type Alarm struct {
	// Enabled
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
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
	// Provider
	Provider string `json:"provider" yaml:"provider" mapstructure:"provider"`
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

func NewConfig(lc fx.Lifecycle) *Type {
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

	// fx hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Watch config
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("Config file changed: %s\n", e.String())

				// Reload
				err := viper.Unmarshal(&App)
				if err != nil {
					log.Printf("[config] Failed to unmarshal config: %v", err)
				}
			})
			viper.WatchConfig()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return &App
}
