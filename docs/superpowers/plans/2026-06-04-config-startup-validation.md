# Config Startup Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add startup configuration validation to fail fast before subsystem initialization with clear error messages and fix suggestions.

**Architecture:** Two methods added to `config.Type`: `Validate()` for pure field checks (no I/O) accumulating all errors, and `ReachabilityCheck()` for DB/Redis connection attempts with short timeouts. Uses `go-playground/validator` struct tags on sub-structs for declarative checks, imperative code for cross-field logic.

**Tech Stack:** Go, go-playground/validator, go-redis/v9, pgx/v5, testify, viper

---

### Task 1: Add validate struct tags to config sub-types

**Files:**
- Modify: `pkg/config/config.go`

- [ ] **Step 1: Add validate tags to Redis, Log, LogRotation, Flowbot structs**

Add `validate:"..."` tags to these struct fields in `pkg/config/config.go`:

**Redis** (line 252-284): Add tags to Host, Port, Password:
```go
type Redis struct {
	Host     string `json:"host" yaml:"host" mapstructure:"host" validate:"required,min=1"`
	Port     int    `json:"port" yaml:"port" mapstructure:"port" validate:"required,gte=1,lte=65535"`
	DB       int    `json:"db" yaml:"db" mapstructure:"db"`
	Password string `json:"password" yaml:"password" mapstructure:"password" validate:"required,min=1"`
	PoolSize        int           `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"`
	MinIdleConns    int           `json:"min_idle_conns" yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	MaxRetries      int           `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff" mapstructure:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff" mapstructure:"max_retry_backoff"`
	DialTimeout     time.Duration `json:"dial_timeout" yaml:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"`
	PoolTimeout     time.Duration `json:"pool_timeout" yaml:"pool_timeout" mapstructure:"pool_timeout"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	PoolFIFO        bool          `json:"pool_fifo" yaml:"pool_fifo" mapstructure:"pool_fifo"`
}
```

**Log** (line 210): Add tag to Level:
```go
type Log struct {
	Level      string `json:"level" yaml:"level" mapstructure:"level" validate:"omitempty,oneof=debug info warn error fatal panic"`
	Caller     bool   `json:"caller" yaml:"caller" mapstructure:"caller"`
	StackTrace bool   `json:"stackTrace" yaml:"stackTrace" mapstructure:"stackTrace"`
	JSONOutput bool   `json:"jsonOutput" yaml:"jsonOutput" mapstructure:"jsonOutput"`
	FileLog    bool   `json:"fileLog" yaml:"fileLog" mapstructure:"fileLog"`
	FileLogPath string `json:"fileLogPath" yaml:"fileLogPath" mapstructure:"fileLogPath"`
	ModuleLevel map[string]string `json:"moduleLevel" yaml:"moduleLevel" mapstructure:"moduleLevel"`
	Sampling    *LogSampling     `json:"sampling" yaml:"sampling" mapstructure:"sampling"`
	Rotation    *LogRotation     `json:"rotation" yaml:"rotation" mapstructure:"rotation"`
}
```

**Flowbot** (line 415): Add tag to URL:
```go
type Flowbot struct {
	URL         string `json:"url" yaml:"url" mapstructure:"url" validate:"omitempty,url"`
	ChannelPath string `json:"channel_path" yaml:"channel_path" mapstructure:"channel_path"`
	Language    string `json:"language" yaml:"language" mapstructure:"language"`
}
```

- [ ] **Step 2: Add validate tags to Tracing, Profiling structs**

**Tracing** (line 159): Add tags to Endpoint, SampleRate:
```go
type Tracing struct {
	Enabled     bool    `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Endpoint    string  `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint" validate:"required_if=Enabled true,url"`
	ServiceName string  `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	Environment string  `json:"environment" yaml:"environment" mapstructure:"environment"`
	SampleRate  float64 `json:"sample_rate" yaml:"sample_rate" mapstructure:"sample_rate" validate:"omitempty,gte=0,lte=1"`
}
```

**Profiling** (line 173): Add tag to ServerAddress:
```go
type Profiling struct {
	Enabled       bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	ServerAddress string   `json:"server_address" yaml:"server_address" mapstructure:"server_address" validate:"required_if=Enabled true,url"`
	ServiceName   string   `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	Environment   string   `json:"environment" yaml:"environment" mapstructure:"environment"`
	ProfileTypes  []string `json:"profile_types" yaml:"profile_types" mapstructure:"profile_types"`
}
```

- [ ] **Step 3: Add validate tags to platform structs (Slack, Discord, Tailchat)**

**Slack** (line 298):
```go
type Slack struct {
	Enabled           bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	AppID             string `json:"app_id" yaml:"app_id" mapstructure:"app_id" validate:"required_if=Enabled true"`
	ClientID          string `json:"client_id" yaml:"client_id" mapstructure:"client_id" validate:"required_if=Enabled true"`
	ClientSecret      string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret" validate:"required_if=Enabled true"`
	SigningSecret     string `json:"signing_secret" yaml:"signing_secret" mapstructure:"signing_secret" validate:"required_if=Enabled true"`
	VerificationToken string `json:"verification_token" yaml:"verification_token" mapstructure:"verification_token"`
	AppToken          string `json:"app_token" yaml:"app_token" mapstructure:"app_token"`
	BotToken          string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token"`
}
```

**Discord** (line 317):
```go
type Discord struct {
	Enabled      bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	AppID        string `json:"app_id" yaml:"app_id" mapstructure:"app_id" validate:"required_if=Enabled true"`
	PublicKey    string `json:"public_key" yaml:"public_key" mapstructure:"public_key" validate:"required_if=Enabled true"`
	ClientID     string `json:"client_id" yaml:"client_id" mapstructure:"client_id" validate:"required_if=Enabled true"`
	ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret" validate:"required_if=Enabled true"`
	BotToken     string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token" validate:"required_if=Enabled true"`
}
```

**Tailchat** (line 337):
```go
type Tailchat struct {
	Enabled   bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	ApiURL    string `json:"api_url" yaml:"api_url" mapstructure:"api_url" validate:"required_if=Enabled true,url"`
	AppID     string `json:"app_id" yaml:"app_id" mapstructure:"app_id"`
	AppSecret string `json:"app_secret" yaml:"app_secret" mapstructure:"app_secret"`
}
```

- [ ] **Step 4: Run go vet to ensure struct tags compile**

Run: `go vet ./pkg/config/`

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat(config): add validate struct tags to config sub-types"
```

---

### Task 2: Write tests for Validate()

**Files:**
- Create: `pkg/config/validate_test.go`

- [ ] **Step 1: Write test file with TestValidate_Required**

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validConfig() Type {
	return Type{
		Redis: Redis{
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "secret",
		},
		Store: StoreType{
			UseAdapter: "postgres",
			Adapters: map[string]any{
				"postgres": map[string]any{
					"dsn": "postgres://user:pass@localhost/flowbot?sslmode=disable",
				},
			},
		},
	}
}

func TestValidate_Required(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
	}{
		{
			name:    "missing redis host",
			mutate:  func(c *Type) { c.Redis.Host = "" },
			wantErr: "redis.Host",
		},
		{
			name:    "missing redis password",
			mutate:  func(c *Type) { c.Redis.Password = "" },
			wantErr: "redis.Password",
		},
		{
			name:    "redis port zero",
			mutate:  func(c *Type) { c.Redis.Port = 0 },
			wantErr: "redis.Port",
		},
		{
			name:    "redis port too high",
			mutate:  func(c *Type) { c.Redis.Port = 99999 },
			wantErr: "redis.Port",
		},
		{
			name:    "missing store adapter name",
			mutate:  func(c *Type) { c.Store.UseAdapter = "" },
			wantErr: "store.use_adapter",
		},
		{
			name: "use_adapter not found in adapters map",
			mutate: func(c *Type) {
				c.Store.UseAdapter = "mysql"
			},
			wantErr: "not found",
		},
		{
			name: "missing DSN",
			mutate: func(c *Type) {
				c.Store.Adapters = map[string]any{
					"postgres": map[string]any{},
				}
			},
			wantErr: "dsn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			assert.Error(t, err)
			if tt.wantErr != "" {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			assert.Contains(t, err.Error(), "Fix:")
		})
	}
}
```

- [ ] **Step 2: Write TestValidate_Format**

```go
func TestValidate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
		noErr   bool
	}{
		{
			name:  "valid log level debug",
			mutate: func(c *Type) { c.Log.Level = "debug" },
			noErr: true,
		},
		{
			name:  "valid log level info",
			mutate: func(c *Type) { c.Log.Level = "info" },
			noErr: true,
		},
		{
			name:  "valid log level warn",
			mutate: func(c *Type) { c.Log.Level = "warn" },
			noErr: true,
		},
		{
			name:    "invalid log level",
			mutate:  func(c *Type) { c.Log.Level = "verbose" },
			wantErr: "log.Level",
		},
		{
			name: "tracing enabled, missing endpoint",
			mutate: func(c *Type) {
				c.Tracing.Enabled = true
				c.Tracing.Endpoint = ""
			},
			wantErr: "tracing.Endpoint",
		},
		{
			name: "tracing enabled, invalid URL",
			mutate: func(c *Type) {
				c.Tracing.Enabled = true
				c.Tracing.Endpoint = "not-a-url"
			},
			wantErr: "tracing.Endpoint",
		},
		{
			name: "tracing disabled, missing endpoint OK",
			mutate: func(c *Type) {
				c.Tracing.Enabled = false
				c.Tracing.Endpoint = ""
			},
			noErr: true,
		},
		{
			name: "sample rate out of range high",
			mutate: func(c *Type) {
				c.Tracing.SampleRate = 2.0
			},
			wantErr: "tracing.SampleRate",
		},
		{
			name:    "invalid listen address",
			mutate:  func(c *Type) { c.Listen = "::99999" },
			wantErr: "listen",
		},
		{
			name: "invalid probe timeout duration",
			mutate: func(c *Type) {
				c.Homelab.Discovery.ProbeTimeout = "xyz"
			},
			wantErr: "probe_timeout",
		},
		{
			name: "invalid expiry duration",
			mutate: func(c *Type) {
				c.capability.EventPool.ExpiryDuration = "bad"
			},
			wantErr: "expiry_duration",
		},
		{
			name:  "valid flowbot url",
			mutate: func(c *Type) { c.Flowbot.URL = "http://example.com" },
			noErr: true,
		},
		{
			name:    "invalid flowbot url",
			mutate:  func(c *Type) { c.Flowbot.URL = "not-a-url" },
			wantErr: "flowbot.URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.noErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.wantErr != "" {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			}
		})
	}
}
```

- [ ] **Step 3: Write TestValidate_Conditional**

```go
func TestValidate_Conditional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Type)
		wantErr string
		noErr   bool
	}{
		{
			name: "slack enabled, missing app_id",
			mutate: func(c *Type) {
				c.Platform.Slack.Enabled = true
				c.Platform.Slack.AppID = ""
			},
			wantErr: "platform.slack.AppID",
		},
		{
			name: "slack disabled, missing app_id OK",
			mutate: func(c *Type) {
				c.Platform.Slack.Enabled = false
				c.Platform.Slack.AppID = ""
			},
			noErr: true,
		},
		{
			name: "discord enabled, missing bot_token",
			mutate: func(c *Type) {
				c.Platform.Discord.Enabled = true
				c.Platform.Discord.BotToken = ""
			},
			wantErr: "platform.discord.BotToken",
		},
		{
			name: "discord disabled, missing bot_token OK",
			mutate: func(c *Type) {
				c.Platform.Discord.Enabled = false
				c.Platform.Discord.BotToken = ""
			},
			noErr: true,
		},
		{
			name: "tailchat enabled, invalid api_url",
			mutate: func(c *Type) {
				c.Platform.Tailchat.Enabled = true
				c.Platform.Tailchat.ApiURL = "not-a-url"
			},
			wantErr: "platform.tailchat.ApiURL",
		},
		{
			name: "agent references unknown model",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "openai", ModelNames: []string{"gpt4"}, BaseUrl: "https://api.openai.com"},
				}
				c.Agents = []Agent{
					{Name: "helper", Model: "nonexistent"},
				}
			},
			wantErr: "not found in models",
		},
		{
			name: "model missing provider",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "", BaseUrl: "https://api.openai.com"},
				}
			},
			wantErr: "models[0].provider",
		},
		{
			name: "model invalid base_url",
			mutate: func(c *Type) {
				c.Models = []Model{
					{Provider: "openai", BaseUrl: "not-a-url"},
				}
			},
			wantErr: "models[0].base_url",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.noErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.wantErr != "" {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			}
		})
	}
}
```

- [ ] **Step 4: Write TestValidate_Accumulated and TestValidate_HappyPath**

```go
func TestValidate_Accumulated(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Redis.Host = ""
	cfg.Redis.Password = ""
	cfg.Store.UseAdapter = ""
	cfg.Store.Adapters = nil

	err := cfg.Validate()
	assert.Error(t, err)
	errStr := err.Error()
	assert.Contains(t, errStr, "redis.Host")
	assert.Contains(t, errStr, "redis.Password")
	assert.Contains(t, errStr, "store.use_adapter")
	// Verify multiple lines (one per error)
	lines := 0
	for _, ch := range errStr {
		if ch == '\n' {
			lines++
		}
	}
	assert.GreaterOrEqual(t, lines, 2, "should have at least 2 newlines for 3+ errors")
}

func TestValidate_HappyPath(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}
```

- [ ] **Step 5: Write TestReachabilityCheck tests (skipped by default)**

```go
func TestReachabilityCheck_RedisUnreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reachability test in short mode")
	}

	cfg := validConfig()
	cfg.Redis.Host = "255.255.255.255"
	cfg.Redis.Port = 9999
	cfg.Redis.Password = "nope"

	// Note: Validate() must be called first or fields must be populated
	err := cfg.ReachabilityCheck(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis")
}

func TestReachabilityCheck_PostgresUnreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reachability test in short mode")
	}

	cfg := validConfig()
	cfg.Store.Adapters = map[string]any{
		"postgres": map[string]any{
			"dsn": "postgres://nonexistent:bad@255.255.255.255:9999/fake?sslmode=disable",
		},
	}

	err := cfg.ReachabilityCheck(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres")
}
```

- [ ] **Step 6: Run tests to confirm they fail (Validate() not implemented yet)**

Run: `go test ./pkg/config/ -run "TestValidate" -v -count=1`
Expected: FAIL with "Validate undefined"

- [ ] **Step 7: Commit**

```bash
git add pkg/config/validate_test.go
git commit -m "test(config): add Validate and ReachabilityCheck tests"
```

---

### Task 3: Implement ValidationErrors and Validate()

**Files:**
- Create: `pkg/config/validate.go`

- [ ] **Step 1: Create validate.go with types and Validate()**

```go
package config

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/flowline-io/flowbot/pkg/validate"
)

// ValidationErrors accumulates multiple validation errors for batch reporting.
type ValidationErrors []error

// Error joins all errors with newlines so each failure appears on its own line.
func (ve ValidationErrors) Error() string {
	var b strings.Builder
	for i, e := range ve {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(e.Error())
	}
	return b.String()
}

// Validate performs pure field validation on the config struct. It accumulates
// all errors before returning so the user can fix everything in one pass.
// This method does not perform any I/O (no network connections).
func (t *Type) Validate() error {
	var errs ValidationErrors

	// Struct tag validation on sub-structs
	if err := validate.Validate.Struct(t.Redis); err != nil {
		errs = appendTagErrors(errs, err, "redis")
	}
	if err := validate.Validate.Struct(t.Log); err != nil {
		errs = appendTagErrors(errs, err, "log")
	}
	if t.Log.Rotation != nil {
		if t.Log.Rotation.MaxSize <= 0 {
			errs = append(errs, fmt.Errorf("log.rotation.maxSize: must be > 0 when rotation is configured. Fix: set log.rotation.maxSize in flowbot.yaml"))
		}
		if t.Log.Rotation.MaxBackups < 0 {
			errs = append(errs, fmt.Errorf("log.rotation.maxBackups: must be >= 0. Fix: set log.rotation.maxBackups in flowbot.yaml"))
		}
	}
	if err := validate.Validate.Struct(t.Tracing); err != nil {
		errs = appendTagErrors(errs, err, "tracing")
	}
	if err := validate.Validate.Struct(t.Profiling); err != nil {
		errs = appendTagErrors(errs, err, "profiling")
	}
	if err := validate.Validate.Struct(t.Flowbot); err != nil {
		errs = appendTagErrors(errs, err, "flowbot")
	}
	if err := validate.Validate.Struct(t.Platform.Slack); err != nil {
		errs = appendTagErrors(errs, err, "platform.slack")
	}
	if err := validate.Validate.Struct(t.Platform.Discord); err != nil {
		errs = appendTagErrors(errs, err, "platform.discord")
	}
	if err := validate.Validate.Struct(t.Platform.Tailchat); err != nil {
		errs = appendTagErrors(errs, err, "platform.tailchat")
	}

	// Imperative checks

	// Listen host:port
	if t.Listen != "" {
		if _, _, err := net.SplitHostPort(t.Listen); err != nil {
			errs = append(errs, fmt.Errorf("listen: invalid host:port %q. Fix: set listen in flowbot.yaml (e.g. \":6060\")", t.Listen))
		}
	}

	// Store adapter
	if t.Store.UseAdapter == "" {
		errs = append(errs, fmt.Errorf("store.use_adapter: must not be empty. Fix: set store_config.use_adapter in flowbot.yaml"))
	} else {
		adapterMap := t.Store.Adapters
		if adapterMap == nil || len(adapterMap) == 0 {
			errs = append(errs, fmt.Errorf("store.adapters: must contain adapter %q. Fix: set store_config.adapters.%s in flowbot.yaml", t.Store.UseAdapter, t.Store.UseAdapter))
		} else {
			adapterCfg, ok := adapterMap[t.Store.UseAdapter]
			if !ok {
				errs = append(errs, fmt.Errorf("store.adapters: adapter %q not found in adapters map. Fix: set store_config.adapters.%s in flowbot.yaml", t.Store.UseAdapter, t.Store.UseAdapter))
			} else {
				dsn := extractDSN(adapterCfg)
				if dsn == "" {
					errs = append(errs, fmt.Errorf("store.adapters.%s.dsn: must not be empty. Fix: set store_config.adapters.%s.dsn in flowbot.yaml", t.Store.UseAdapter, t.Store.UseAdapter))
				}
			}
		}
	}

	// Duration strings
	if t.Homelab.Discovery.ProbeTimeout != "" {
		if _, err := time.ParseDuration(t.Homelab.Discovery.ProbeTimeout); err != nil {
			errs = append(errs, fmt.Errorf("homelab.discovery.probe_timeout: invalid duration %q. Fix: set a valid Go duration (e.g. \"30s\") in homelab.discovery.probe_timeout in flowbot.yaml", t.Homelab.Discovery.ProbeTimeout))
		}
	}
	if t.capability.EventPool.ExpiryDuration != "" {
		if _, err := time.ParseDuration(t.capability.EventPool.ExpiryDuration); err != nil {
			errs = append(errs, fmt.Errorf("capability.event_pool.expiry_duration: invalid duration %q. Fix: set a valid Go duration (e.g. \"30s\") in capability.event_pool.expiry_duration in flowbot.yaml", t.capability.EventPool.ExpiryDuration))
		}
	}

	// Models
	modelNames := make(map[string]bool)
	for i, m := range t.Models {
		if m.Provider == "" {
			errs = append(errs, fmt.Errorf("models[%d].provider: must not be empty. Fix: set models[%d].provider in flowbot.yaml", i, i))
		}
		if m.BaseUrl != "" {
			if !strings.HasPrefix(m.BaseUrl, "http://") && !strings.HasPrefix(m.BaseUrl, "https://") {
				errs = append(errs, fmt.Errorf("models[%d].base_url: invalid URL %q. Fix: set a valid URL in models[%d].base_url in flowbot.yaml", i, m.BaseUrl, i))
			}
		}
		for _, name := range m.ModelNames {
			if name != "" {
				modelNames[name] = true
			}
		}

	// Agents
	for i, a := range t.Agents {
		if a.Name == "" {
			errs = append(errs, fmt.Errorf("agents[%d].name: must not be empty. Fix: set agents[%d].name in flowbot.yaml", i, i))
		}
		if a.Model != "" && len(modelNames) > 0 && !modelNames[a.Model] {
			errs = append(errs, fmt.Errorf("agents[%d].model: %q not found in models. Fix: reference an existing model name in agents[%d].model in flowbot.yaml", i, a.Model, i))
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// extractDSN extracts the DSN string from an adapter config stored as `any`.
func extractDSN(cfg any) string {
	m, ok := cfg.(map[string]any)
	if !ok {
		return ""
	}
	dsn, _ := m["dsn"].(string)
	return dsn
}

// appendTagErrors converts go-playground validator errors into ValidationErrors
// with a field path prefix and fix suggestion.
func appendTagErrors(errs ValidationErrors, err error, prefix string) ValidationErrors {
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return errs
	}
	for _, fe := range verrs {
		errs = append(errs, fmt.Errorf("%s.%s: %s. Fix: set %s.%s in flowbot.yaml", prefix, fe.Field(), formatTagError(fe), prefix, fe.Field()))
	}
	return errs
}

// formatTagError returns a human-readable description for a validation tag failure.
func formatTagError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required", "required_if":
		return "must not be empty"
	case "url":
		return "must be a valid URL"
	case "gte":
		return fmt.Sprintf("must be >= %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be <= %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	default:
		return fmt.Sprintf("validation failed on %s", fe.Tag())
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./pkg/config/ -run "TestValidate" -v -count=1`
Expected: All TestValidate tests PASS

- [ ] **Step 3: Verify Model struct field names**

Run: `grep -A5 "type Model struct" pkg/config/config.go`

Confirm `Model` uses `ModelNames []string` (not `Name string`). The `validate.go` implementation in Step 1 already uses `ModelNames` for cross-referencing agents, and the test in Task 2 uses `ModelNames`.

- [ ] **Step 4: Run tests again if changes were needed**

Run: `go test ./pkg/config/ -run "TestValidate" -v -count=1`

- [ ] **Step 5: Commit**

```bash
git add pkg/config/validate.go pkg/config/validate_test.go
git commit -m "feat(config): add Validate() with accumulated field validation"
```

---

### Task 4: Implement ReachabilityCheck()

**Files:**
- Modify: `pkg/config/validate.go`

- [ ] **Step 1: Add ReachabilityCheck() to validate.go**

Add the following method and imports to `pkg/config/validate.go`. Add imports at the top:

```go
import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/flowline-io/flowbot/pkg/validate"
)
```

Add the method after the existing `Validate()` function:

```go
// ReachabilityCheck attempts PostgreSQL and Redis connections with short
// timeouts to verify that dependencies are reachable. Only call this after
// Validate() passes, since it assumes required fields are non-empty.
func (t *Type) ReachabilityCheck(ctx context.Context) error {
	var errs ValidationErrors

	// PostgreSQL
	adapterMap := t.Store.Adapters
	if adapterMap != nil && t.Store.UseAdapter != "" {
		if adapterCfg, ok := adapterMap[t.Store.UseAdapter]; ok {
			dsn := extractDSN(adapterCfg)
			if dsn != "" {
				dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				db, err := sql.Open("pgx", dsn)
				if err != nil {
					errs = append(errs, fmt.Errorf("postgres: cannot open connection: %w. Fix: verify DSN in store_config.adapters.%s.dsn", err, t.Store.UseAdapter))
				} else {
					if err := db.PingContext(dbCtx); err != nil {
						errs = append(errs, fmt.Errorf("postgres: ping failed: %w. Fix: verify PostgreSQL is running and reachable", err))
					}
					db.Close()
				}
				cancel()
			}
		}
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         net.JoinHostPort(t.Redis.Host, strconv.Itoa(t.Redis.Port)),
		Password:     t.Redis.Password,
		DB:           t.Redis.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer rdb.Close()
	redisCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := rdb.Ping(redisCtx).Err(); err != nil {
		errs = append(errs, fmt.Errorf("redis: ping failed: %w. Fix: verify Redis is running at %s", err, net.JoinHostPort(t.Redis.Host, strconv.Itoa(t.Redis.Port))))
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
```

- [ ] **Step 2: Run tests to verify**

Run: `go test ./pkg/config/ -run "TestReachability" -v -count=1`
Expected: Tests pass (skipped in short mode)

Run: `go test ./pkg/config/ -run "TestReachability" -v -count=1 -short`
Expected: Tests skip with "skipping reachability test in short mode"

- [ ] **Step 3: Commit**

```bash
git add pkg/config/validate.go
git commit -m "feat(config): add ReachabilityCheck() for DB and Redis connection verification"
```

---

### Task 5: Wire validation into NewConfig() and hot-reload

**Files:**
- Modify: `pkg/config/config.go`

- [ ] **Step 1: Add Validate() and ReachabilityCheck() calls to NewConfig()**

In `pkg/config/config.go`, after the `App.ApiPath` normalization block (line 640) and before the fx lifecycle hook (line 643), add:

```go
	// Validate config before starting any subsystems
	if err := App.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed:\n%w", err)
	}
	if err := App.ReachabilityCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("dependency check failed:\n%w", err)
	}
```

The updated `NewConfig()` function (lines 643-666) becomes:

```go
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

				// Reload
				err := viper.Unmarshal(&App)
				if err != nil {
					log.Printf("[config] Failed to unmarshal config: %v", err)
					return
				}
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
```

Note: add `return` after the unmarshal error in `OnConfigChange` so it doesn't proceed to `App.Validate()` on empty config.

- [ ] **Step 2: Add context import to config.go**

The `NewConfig()` function now uses `context.Background()`. `context` is already imported in `config.go` (line 5: `"context"`).

- [ ] **Step 3: Run tests to ensure nothing breaks**

Run: `go test ./pkg/config/ -v -count=1`
Expected: All tests pass (existing + new)

- [ ] **Step 4: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat(config): wire Validate() and ReachabilityCheck() into startup and hot-reload"
```

---

### Task 6: Remove ad-hoc validation checks

**Files:**
- Modify: `pkg/rdb/rdb.go`
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Remove ad-hoc check from rdb.go**

In `pkg/rdb/rdb.go`, remove lines 28-30 (the `if addr == ":" || password == ""` check):

**Before:**
```go
func NewClient(lc fx.Lifecycle, _ *config.Type) (*redis.Client, error) {
	addr := net.JoinHostPort(config.App.Redis.Host, strconv.Itoa(config.App.Redis.Port))
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		return nil, fmt.Errorf("redis config error")
	}
	Client = redis.NewClient(redisOptions(config.App.Redis))
```

**After:**
```go
func NewClient(lc fx.Lifecycle, _ *config.Type) (*redis.Client, error) {
	addr := net.JoinHostPort(config.App.Redis.Host, strconv.Itoa(config.App.Redis.Port))
	password := config.App.Redis.Password
	Client = redis.NewClient(redisOptions(config.App.Redis))
```

Now unused `addr` and `password`. Remove those lines too:

```go
func NewClient(lc fx.Lifecycle, _ *config.Type) (*redis.Client, error) {
	Client = redis.NewClient(redisOptions(config.App.Redis))
```

- [ ] **Step 2: Remove ad-hoc check from postgres adapter.go**

In `internal/store/postgres/adapter.go`, remove lines 100-102 (the `conf.DSN == ""` check):

**Before:**
```go
	if conf.DSN == "" {
		return errors.New("postgres: DSN is required")
	}

	if conf.SqlTimeout <= 0 {
```

**After:**
```go
	if conf.SqlTimeout <= 0 {
```

- [ ] **Step 3: Check for unused imports after removal**

Run: `go vet ./pkg/rdb/`
Run: `go vet ./internal/store/postgres/`

Fix any unused import errors. In `rdb.go`, if `fmt` is no longer used, remove it. If `net` and `strconv` are only used in `redisOptions()`, keep them.

Specifically for `rdb.go`: `net` and `strconv` are still used in `redisOptions()` at line 66. `fmt` may become unused — check if other parts of the file use it. If yes, keep; if no, remove it from imports.

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/rdb/ -v -count=1 -short`
Run: `go test ./internal/store/postgres/ -v -count=1 -short`
Expected: Tests pass

- [ ] **Step 5: Commit**

```bash
git add pkg/rdb/rdb.go internal/store/postgres/adapter.go
git commit -m "refactor: remove ad-hoc config checks now handled by Validate()"
```

---

### Task 7: Run full lint and test suite

- [ ] **Step 1: Run lint**

Run: `go tool task lint`
Expected: No new lint errors. Fix any reported.

- [ ] **Step 2: Run format**

Run: `go tool task format`

- [ ] **Step 3: Run all unit tests (short mode)**

Run: `go tool task test`
Expected: All tests pass including new config tests.

- [ ] **Step 4: Run vet**

Run: `go vet ./...`
Expected: No errors.

- [ ] **Step 5: Final commit if any lint/format changes were needed**

```bash
git add -u
git commit -m "chore: lint and format fixes for config validation"
```
