// Package config provides runtime configuration loading and management.
package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"

	plugintypes "github.com/flowline-io/flowbot/pkg/plugin/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
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

	// HTTP boundary settings (CORS, rate limit, HSTS).
	HTTP HTTPConfig `json:"http" yaml:"http" mapstructure:"http"`

	// Postgres is the primary database configuration (YAML key: postgres).
	Postgres PostgresConfig `json:"postgres" yaml:"postgres" mapstructure:"postgres"`
	// Store is the normalized internal store view populated by Normalize from Postgres.
	// It is not read from YAML.
	Store StoreType `json:"-" yaml:"-" mapstructure:"-"`
	// Media large-file storage configuration (optional; omit to disable uploads).
	Media *mediaConfig `json:"media" yaml:"media" mapstructure:"media"`

	// Redis connection configuration.
	Redis Redis `json:"redis" yaml:"redis" mapstructure:"redis"`

	// Log
	Log Log `json:"log" yaml:"log" mapstructure:"log"`

	// Config for modules
	Modules any `json:"modules" yaml:"modules" mapstructure:"modules"`

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

	// Models
	Models []Model `json:"models" yaml:"models" mapstructure:"models"`

	// ChatAgent configures the direct-message chat assistant agent.
	ChatAgent ChatAgentConfig `json:"chat_agent" yaml:"chat_agent" mapstructure:"chat_agent"`

	// Homelab app registry and lifecycle configuration
	Homelab Homelab `json:"homelab" yaml:"homelab" mapstructure:"homelab"`

	// OpenTelemetry tracing configuration
	Tracing Tracing `json:"tracing" yaml:"tracing" mapstructure:"tracing"`

	// Pyroscope continuous profiling configuration
	Profiling Profiling `json:"profiling" yaml:"profiling" mapstructure:"profiling"`

	// Capability invocation configuration (YAML key remains "ability" for backward compatibility).
	Capability AbilityConfig `json:"ability" yaml:"ability" mapstructure:"ability"`

	// Plugin system configuration
	Plugins *plugintypes.PluginConfig `json:"plugins" yaml:"plugins" mapstructure:"plugins"`

	// Retention controls optional automatic cleanup of durable history tables.
	Retention RetentionConfig `json:"retention" yaml:"retention" mapstructure:"retention"`
}

// RetentionConfig configures optional TTL cleanup for durable rows.
type RetentionConfig struct {
	// DataEventsDays deletes data_events older than this many days when > 0. Zero disables.
	DataEventsDays int `json:"data_events_days" yaml:"data_events_days" mapstructure:"data_events_days"`
}

// Tracing configures OpenTelemetry distributed tracing.
type Tracing struct {
	// Enabled toggles trace export
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Endpoint is the OTLP HTTP endpoint (e.g. http://localhost:4318/v1/traces)
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint" validate:"required_if=Enabled true,omitempty,url"`
	// ServiceName identifies this service in traces
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	// Environment tag (development, staging, production)
	Environment string `json:"environment" yaml:"environment" mapstructure:"environment"`
	// SampleRate controls trace sampling (0.0-1.0)
	SampleRate float64 `json:"sample_rate" yaml:"sample_rate" mapstructure:"sample_rate" validate:"omitempty,gte=0,lte=1"`
}

// Profiling configures Pyroscope continuous profiling.
type Profiling struct {
	// Enabled toggles continuous profiling
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// ServerAddress is the Pyroscope server URL (e.g. http://localhost:4040)
	ServerAddress string `json:"server_address" yaml:"server_address" mapstructure:"server_address" validate:"required_if=Enabled true,omitempty,url"`
	// ServiceName identifies this service in profiles
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	// Environment tag (development, staging, production)
	Environment string `json:"environment" yaml:"environment" mapstructure:"environment"`
	// ProfileTypes lists the profile types to enable (cpu, alloc_objects, etc.)
	ProfileTypes []string `json:"profile_types" yaml:"profile_types" mapstructure:"profile_types"`
}

const (
	defaultMediaMaxSize     int64 = 104857600 // 100 MiB
	defaultMediaGcPeriod          = 60
	defaultMediaGcBlockSize       = 100
)

// Large file handler config.
type mediaConfig struct {
	// The name of the handler to use for file uploads.
	UseHandler string `json:"use_handler" yaml:"use_handler" mapstructure:"use_handler"`
	// Maximum allowed size of an uploaded file (0 = default 100 MiB).
	MaxFileUploadSize int64 `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
	// Garbage collection period in seconds (0 = default 60).
	GcPeriod int `json:"gc_period" yaml:"gc_period" mapstructure:"gc_period"`
	// Number of entries to delete in one pass (0 = default 100).
	GcBlockSize int `json:"gc_block_size" yaml:"gc_block_size" mapstructure:"gc_block_size"`
	// Individual handler config params to pass to handlers unchanged.
	Handlers map[string]any `json:"handlers" yaml:"handlers" mapstructure:"handlers"`
}

// PostgresConfig holds PostgreSQL connection and optional pool overrides.
type PostgresConfig struct {
	// DSN is the PostgreSQL connection string.
	DSN string `json:"dsn" yaml:"dsn" mapstructure:"dsn" validate:"required,min=1"`
	// MaxResults caps query result sets (0 = adapter default).
	MaxResults int `json:"max_results,omitempty" yaml:"max_results,omitempty" mapstructure:"max_results"`
	// MaxOpenConns is the maximum open connections (0 = pool default).
	MaxOpenConns int `json:"max_open_conns,omitempty" yaml:"max_open_conns,omitempty" mapstructure:"max_open_conns"`
	// MaxIdleConns is the maximum idle connections (0 = pool default).
	MaxIdleConns int `json:"max_idle_conns,omitempty" yaml:"max_idle_conns,omitempty" mapstructure:"max_idle_conns"`
	// ConnMaxLifetime is connection max lifetime in seconds (0 = pool default).
	ConnMaxLifetime int `json:"conn_max_lifetime,omitempty" yaml:"conn_max_lifetime,omitempty" mapstructure:"conn_max_lifetime"`
	// ConnMaxIdleTime is idle connection max lifetime in seconds (0 = pool default).
	ConnMaxIdleTime int `json:"conn_max_idle_time,omitempty" yaml:"conn_max_idle_time,omitempty" mapstructure:"conn_max_idle_time"`
	// SQLTimeout is the ping/query timeout in seconds (0 = adapter default).
	SQLTimeout int `json:"sql_timeout,omitempty" yaml:"sql_timeout,omitempty" mapstructure:"sql_timeout"`
	// HealthCheckInterval is pool health check interval in seconds.
	HealthCheckInterval int `json:"pool_health_check_interval,omitempty" yaml:"pool_health_check_interval,omitempty" mapstructure:"pool_health_check_interval"`
	// HealthCheckTimeout is pool health check timeout in seconds.
	HealthCheckTimeout int `json:"pool_health_check_timeout,omitempty" yaml:"pool_health_check_timeout,omitempty" mapstructure:"pool_health_check_timeout"`
}

// StoreType is the normalized store adapter view used by internal/store.
// It is not loaded from YAML; call Normalize after unmarshaling Postgres.
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
	Level string `json:"level" yaml:"level" mapstructure:"level" validate:"omitempty,oneof=debug info warn error fatal panic"`
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

// Redis stores connection and pool configuration for the Redis client.
type Redis struct {
	// URL is the Redis connection URI, e.g. redis://:password@127.0.0.1:6379/0.
	// Password must be non-empty (validated separately from struct tags).
	URL string `json:"url" yaml:"url" mapstructure:"url" validate:"required,min=1"`
	// Maximum number of connections in the pool (0 = go-redis default: 10*GOMAXPROCS)
	PoolSize int `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"`
	// Minimum number of idle connections maintained in the pool (0 = default: none)
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	// Maximum number of retries before giving up (0 = default: 3)
	MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`
	// Minimum backoff between retries (0 = default: 8ms)
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff" mapstructure:"min_retry_backoff"`
	// Maximum backoff between retries (0 = default: 512ms)
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff" mapstructure:"max_retry_backoff"`
	// Dial timeout for establishing new connections (0 = default: 5s)
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout" mapstructure:"dial_timeout"`
	// Timeout for socket reads (0 = fallback to 60s for backward compatibility)
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`
	// Timeout for socket writes (0 = fallback to 60s for backward compatibility)
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"`
	// Timeout for waiting for a connection from the pool (0 = default: ReadTimeout + 1s)
	PoolTimeout time.Duration `json:"pool_timeout" yaml:"pool_timeout" mapstructure:"pool_timeout"`
	// Maximum idle time for a connection before closing (0 = default: 30min)
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
	// Maximum lifetime of a connection (0 = default: no limit)
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	// Use FIFO (first-in-first-out) instead of LIFO for pool connections
	PoolFIFO bool `json:"pool_fifo" yaml:"pool_fifo" mapstructure:"pool_fifo"`
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
	AppID         string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	ClientID      string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
	ClientSecret  string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
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
	AppID string `json:"app_id" yaml:"app_id" mapstructure:"app_id" validate:"required_if=Enabled true"`
	// Discord public key
	PublicKey string `json:"public_key" yaml:"public_key" mapstructure:"public_key" validate:"required_if=Enabled true"`
	// Discord client ID
	ClientID string `json:"client_id" yaml:"client_id" mapstructure:"client_id" validate:"required_if=Enabled true"`
	// Discord client secret
	ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret" validate:"required_if=Enabled true"`
	// Discord bot token
	BotToken string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token" validate:"required_if=Enabled true"`
}

type Telegram struct {
	// Telegram platform configuration
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
}

type Tailchat struct {
	// Tailchat platform configuration
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Tailchat API URL
	ApiURL string `json:"api_url" yaml:"api_url" mapstructure:"api_url" validate:"required_if=Enabled true,omitempty,url"`
	// Tailchat app ID
	AppID string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	// Tailchat app secret
	AppSecret string `json:"app_secret" yaml:"app_secret" mapstructure:"app_secret"`
	// WebhookToken is the shared secret required on inbound /platform/tailchat callbacks
	// via the X-Tailchat-Token header. When Tailchat is enabled, an empty token rejects all requests.
	WebhookToken string `json:"webhook_token" yaml:"webhook_token" mapstructure:"webhook_token"`
}

type ExecutorLimits struct {
	// CPU limit
	Cpus string `json:"cpus" yaml:"cpus" mapstructure:"cpus"`
	// Memory limit
	Memory string `json:"memory" yaml:"memory" mapstructure:"memory"`
}

type ExecutorMountBind struct {
	// Allowed
	Allowed bool `json:"allowed" yaml:"allowed" mapstructure:"allowed"`
}

type ExecutorMounts struct {
	// Bind mount
	Bind ExecutorMountBind `json:"bind" yaml:"bind" mapstructure:"bind"`
}

type ExecutorDockerConfig struct {
	// Docker config
	Config string `json:"config" yaml:"config" mapstructure:"config"`
}

type ExecutorShellConfig struct {
	// Command
	CMD []string `json:"cmd" yaml:"cmd" mapstructure:"cmd"`
	// User ID
	UID string `json:"uid" yaml:"uid" mapstructure:"uid"`
	// Group ID
	GID string `json:"gid" yaml:"gid" mapstructure:"gid"`
}

type ExecutorMachineConfig struct {
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
}

type Executor struct {
	// Executor type: docker
	Type string `json:"type" yaml:"type" mapstructure:"type"`
	// Resource limits
	Limits  ExecutorLimits        `json:"limits" yaml:"limits" mapstructure:"limits"`
	Mounts  ExecutorMounts        `json:"mounts" yaml:"mounts" mapstructure:"mounts"`
	Docker  ExecutorDockerConfig  `json:"docker" yaml:"docker" mapstructure:"docker"`
	Shell   ExecutorShellConfig   `json:"shell" yaml:"shell" mapstructure:"shell"`
	Machine ExecutorMachineConfig `json:"machine" yaml:"machine" mapstructure:"machine"`
}

type Metrics struct {
	// Enabled controls push to the remote metrics endpoint (VictoriaMetrics / Prometheus push).
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Endpoint is the remote push URL when Enabled is true.
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	// BearerToken is an optional dedicated scrape secret for GET /metrics.
	// When set, Authorization: Bearer <token> (or X-AccessToken) matching this value is accepted.
	// Otherwise a valid access token with admin:metrics (or admin:*) scope is required.
	BearerToken string `json:"bearer_token" yaml:"bearer_token" mapstructure:"bearer_token"`
}

// HTTPConfig holds HTTP boundary hardening settings.
type HTTPConfig struct {
	// CORS configures cross-origin resource sharing. Empty allow_origins disables CORS reflection.
	CORS HTTPCORSConfig `json:"cors" yaml:"cors" mapstructure:"cors"`
	// RateLimit configures the global HTTP request rate limiter.
	RateLimit HTTPRateLimitConfig `json:"rate_limit" yaml:"rate_limit" mapstructure:"rate_limit"`
	// TLSBehindProxy enables HSTS when true (HTTPS or TLS-terminating reverse proxy / frp).
	// HSTS is also sent when modules.web.auth.cookie_secure is enabled (see ShouldSendHSTS).
	TLSBehindProxy bool `json:"tls_behind_proxy" yaml:"tls_behind_proxy" mapstructure:"tls_behind_proxy"`
	// TrustedProxies lists proxy IPs or CIDRs whose X-Forwarded-For (and related) headers
	// are trusted for client IP resolution (login rate limit, etc.). Empty disables header trust.
	TrustedProxies []string `json:"trusted_proxies" yaml:"trusted_proxies" mapstructure:"trusted_proxies"`
}

// HTTPCORSConfig holds CORS allow-origin whitelist settings.
type HTTPCORSConfig struct {
	// AllowOrigins is the Origin whitelist. Empty means no Origin is reflected (same-origin Web UI does not need CORS).
	AllowOrigins []string `json:"allow_origins" yaml:"allow_origins" mapstructure:"allow_origins"`
}

// HTTPRateLimitConfig holds global HTTP rate limiter settings.
type HTTPRateLimitConfig struct {
	// Max is the maximum requests per Expiration window (default 200 when unset or <= 0).
	Max int `json:"max" yaml:"max" mapstructure:"max"`
	// Expiration is the sliding window duration (default 10s when unset or <= 0).
	Expiration time.Duration `json:"expiration" yaml:"expiration" mapstructure:"expiration"`
}

type Search struct {
	// Search URL base map
	UrlBaseMap map[string]string `json:"url_base_map" yaml:"url_base_map" mapstructure:"url_base_map"`
}

type Flowbot struct {
	// Flowbot URL
	URL string `json:"url" yaml:"url" mapstructure:"url" validate:"omitempty,url"`
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

// AbilityEventPool configures the goroutine pool for event emission.
type AbilityEventPool struct {
	// Size is the max number of goroutines in the pool (0 = ants default).
	Size int `json:"size" yaml:"size" mapstructure:"size"`
	// ExpiryDuration is the idle worker eviction interval (e.g. "30s").
	ExpiryDuration string `json:"expiry_duration" yaml:"expiry_duration" mapstructure:"expiry_duration"`
}

// AbilityConfig holds ability invocation configuration.
type AbilityConfig struct {
	// EventPool configures the goroutine pool for event emission.
	EventPool AbilityEventPool `json:"event_pool" yaml:"event_pool" mapstructure:"event_pool"`
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
	Event       string          `json:"event" yaml:"event" mapstructure:"event"`
	Cron        string          `json:"cron" yaml:"cron" mapstructure:"cron"`
	CronTimeout string          `json:"cron_timeout" yaml:"cron_timeout" mapstructure:"cron_timeout"`
	Webhook     *WebhookTrigger `json:"webhook" yaml:"webhook" mapstructure:"webhook"`
}

// WebhookPayloadMode specifies how incoming webhook payloads are handled.
type WebhookPayloadMode string

const (
	WebhookPayloadRaw    WebhookPayloadMode = "raw"
	WebhookPayloadMapped WebhookPayloadMode = "mapped"
)

// WebhookAuth holds webhook authentication configuration.
type WebhookAuth struct {
	Token       string `json:"token" yaml:"token" mapstructure:"token"`
	HMACSecret  string `json:"hmac_secret" yaml:"hmac_secret" mapstructure:"hmac_secret"`
	HMACHeader  string `json:"hmac_header" yaml:"hmac_header" mapstructure:"hmac_header"`
	TokenHeader string `json:"token_header" yaml:"token_header" mapstructure:"token_header"`
}

// WebhookTrigger configures a webhook-based pipeline trigger.
type WebhookTrigger struct {
	Path      string             `json:"path" yaml:"path" mapstructure:"path"`
	Method    string             `json:"method" yaml:"method" mapstructure:"method"`
	Auth      *WebhookAuth       `json:"auth" yaml:"auth" mapstructure:"auth"`
	Payload   WebhookPayloadMode `json:"payload" yaml:"payload" mapstructure:"payload"`
	EventType string             `json:"event_type" yaml:"event_type" mapstructure:"event_type"`
}

// ChatAgentConfig configures the direct-message chat assistant agent runtime.
type ChatAgentConfig struct {
	// Workspace root for file and shell tools; required when the chat agent is enabled.
	Workspace string `json:"workspace" yaml:"workspace" mapstructure:"workspace"`
	// ShellTimeout limits terminal and code execution duration.
	ShellTimeout time.Duration `json:"shell_timeout" yaml:"shell_timeout" mapstructure:"shell_timeout"`
	// MaxToolOutput truncates tool stdout beyond this size in bytes.
	MaxToolOutput int `json:"max_tool_output" yaml:"max_tool_output" mapstructure:"max_tool_output"`
	// MaxSteps limits agent Observe-Think-Act iterations per user turn.
	MaxSteps int `json:"max_steps" yaml:"max_steps" mapstructure:"max_steps"`
	// RunTimeout limits total duration for one assistant turn in direct chat.
	RunTimeout time.Duration `json:"run_timeout" yaml:"run_timeout" mapstructure:"run_timeout"`
	// StreamIdleTimeout cancels an LLM stream when no deltas arrive for this duration. Zero uses 60s.
	StreamIdleTimeout time.Duration `json:"stream_idle_timeout" yaml:"stream_idle_timeout" mapstructure:"stream_idle_timeout"`
	// SystemPrompt replaces the default system prompt when non-empty.
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`
	// AppendSystemPrompt is appended to the system prompt body.
	AppendSystemPrompt string `json:"append_system_prompt" yaml:"append_system_prompt" mapstructure:"append_system_prompt"`
	// PromptGuidelines adds extra guideline bullets to the default system prompt.
	PromptGuidelines []string `json:"prompt_guidelines" yaml:"prompt_guidelines" mapstructure:"prompt_guidelines"`
	// ContextFiles lists project instruction files relative to workspace; defaults to AGENTS.md and README.md.
	ContextFiles []string `json:"context_files" yaml:"context_files" mapstructure:"context_files"`
	// Compaction configures automatic history compaction for long chat sessions.
	Compaction CompactionConfig `json:"compaction" yaml:"compaction" mapstructure:"compaction"`
	// ChatModel selects the chat agent model; non-empty enables the chat agent.
	ChatModel string `json:"chat_model" yaml:"chat_model" mapstructure:"chat_model"`
	// ToolModel enables dual-model routing when set; used after tool execution.
	ToolModel string `json:"tool_model" yaml:"tool_model" mapstructure:"tool_model"`
	// SubagentMaxDepth caps nested task-tool delegation; defaults to 1 (no nested subagents).
	SubagentMaxDepth int `json:"subagent_max_depth" yaml:"subagent_max_depth" mapstructure:"subagent_max_depth"`
	// SubagentDefaultModel overrides the model used by subagents without an explicit model; defaults to ChatModel.
	SubagentDefaultModel string `json:"subagent_default_model" yaml:"subagent_default_model" mapstructure:"subagent_default_model"`
	// SubagentMaxSteps limits Observe-Think-Act iterations within one subagent run; defaults to MaxSteps.
	SubagentMaxSteps int `json:"subagent_max_steps" yaml:"subagent_max_steps" mapstructure:"subagent_max_steps"`
	// LLMRetry configures transient LLM call retries for the chat agent.
	LLMRetry LLMRetryConfig `json:"llm_retry" yaml:"llm_retry" mapstructure:"llm_retry"`
	// Sensors configures post-tool computational sensors.
	Sensors ChatAgentSensorsConfig `json:"sensors" yaml:"sensors" mapstructure:"sensors"`
	// Sandbox configures optional Docker isolation for shell and code tools.
	Sandbox ChatAgentSandboxConfig `json:"sandbox" yaml:"sandbox" mapstructure:"sandbox"`
	// WebSearch configures backends for the web_search tool.
	WebSearch ChatAgentWebSearchConfig `json:"web_search" yaml:"web_search" mapstructure:"web_search"`
}

// ChatAgentWebSearchConfig configures the web_search tool via SerpApi.
type ChatAgentWebSearchConfig struct {
	// APIKey is the SerpApi private key (https://serpapi.com/search-api).
	APIKey string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
}

// LLMRetryConfig configures transient LLM call retries.
type LLMRetryConfig struct {
	// MaxAttempts is the total number of execution attempts. Zero uses package defaults (3).
	MaxAttempts int `json:"max_attempts" yaml:"max_attempts" mapstructure:"max_attempts"`
	// InitialInterval is the delay before the first retry. Zero uses 1s.
	InitialInterval time.Duration `json:"initial_interval" yaml:"initial_interval" mapstructure:"initial_interval"`
	// MaxInterval caps the delay between retries. Zero uses 30s.
	MaxInterval time.Duration `json:"max_interval" yaml:"max_interval" mapstructure:"max_interval"`
	// Multiplier controls delay growth. Zero uses 2.0.
	Multiplier float64 `json:"multiplier" yaml:"multiplier" mapstructure:"multiplier"`
}

// ChatAgentSensorsConfig configures post-tool sensors for the chat agent.
type ChatAgentSensorsConfig struct {
	// LintOnWrite enables observation-only lint logging after writing Go files.
	LintOnWrite bool `json:"lint_on_write" yaml:"lint_on_write" mapstructure:"lint_on_write"`
}

// ChatAgentSandboxConfig configures Docker isolation for shell and code tools.
type ChatAgentSandboxConfig struct {
	// Enabled turns on Docker sandbox execution when true.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// Image is the sandbox container image.
	Image string `json:"image" yaml:"image" mapstructure:"image"`
	// Network is the Docker network mode (empty uses default bridge isolation).
	Network string `json:"network" yaml:"network" mapstructure:"network"`
	// Memory limits container memory (e.g. "512m"); empty uses Docker defaults.
	Memory string `json:"memory" yaml:"memory" mapstructure:"memory"`
	// ServerURL is the Flowbot API URL reachable from inside the sandbox container.
	// Injected as FLOWBOT_SERVER_URL and written to the mounted CLI config.
	ServerURL string `json:"server_url" yaml:"server_url" mapstructure:"server_url"`
	// AccessToken is a dedicated Hub access token for the flowbot CLI inside the sandbox.
	// When empty, credentials are not injected. Injected as FLOWBOT_TOKEN and a mounted token file.
	AccessToken string `json:"access_token" yaml:"access_token" mapstructure:"access_token"`
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

func Load(path ...string) error {
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return fmt.Errorf("bind flags: %w", err)
	}
	for _, p := range path {
		viper.AddConfigPath(p)
	}
	viper.SetConfigName("flowbot")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := expandConfigFileEnv(viper.ConfigFileUsed()); err != nil {
		return fmt.Errorf("expand config env: %w", err)
	}
	err = viper.Unmarshal(&App)
	if err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	if err := RejectLegacyKeys(viper.AllSettings()); err != nil {
		return fmt.Errorf("legacy config keys:\n%w", err)
	}
	App.Normalize()
	return nil
}

// expandConfigFileEnv re-reads the config file with ${VAR} / $VAR substituted from the
// process environment so secrets need not be committed in plaintext YAML.
func expandConfigFileEnv(path string) error {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	expanded := os.ExpandEnv(string(raw))
	return viper.ReadConfig(strings.NewReader(expanded))
}

func NewConfig(lc fx.Lifecycle) (*Type, error) {
	executable, _ := os.Executable()

	curwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	log.Printf("version %s:%s:%s; pid %d; %d process(es)\n",
		version.Buildtags, executable, version.Buildstamp,
		os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))

	configFile := utils.ToAbsolutePath(curwd, "flowbot.yaml")
	log.Printf("Using config from '%s'\n", configFile)

	// Load config
	if err := Load(".", curwd); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

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

	// Validate config before starting any subsystems
	if err := App.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed:\n%w", err)
	}
	if err := App.ReachabilityCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("dependency check failed:\n%w", err)
	}

	// fx hooks
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// Watch config
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("Config file changed: %s\n", e.String())

				// Reload with env expansion
				if err := expandConfigFileEnv(viper.ConfigFileUsed()); err != nil {
					log.Printf("[config] Failed to expand config env: %v", err)
					return
				}
				err := viper.Unmarshal(&App)
				if err != nil {
					log.Printf("[config] Failed to unmarshal config: %v", err)
					return
				}
				if err := RejectLegacyKeys(viper.AllSettings()); err != nil {
					log.Printf("[config] Reloaded config has legacy keys, keeping previous: %v", err)
					return
				}
				App.Normalize()
				// Validate reloaded config, warn if invalid but don't crash
				if err := App.Validate(); err != nil {
					log.Printf("[config] Reloaded config is invalid, keeping previous: %v", err)
				}
			})
			viper.WatchConfig()

			return nil
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})

	return &App, nil
}
