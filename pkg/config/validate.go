package config

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	// Register pgx driver for database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"github.com/flowline-io/flowbot/pkg/validate"
)

// ValidationErrors accumulates multiple validation errors for batch reporting.
type ValidationErrors []error

// Error joins all errors with newlines so each failure appears on its own line.
func (ve ValidationErrors) Error() string {
	var b strings.Builder
	for i, e := range ve {
		if i > 0 {
			_ = b.WriteByte('\n')
		}
		_, _ = b.WriteString(e.Error())
	}
	return b.String()
}

// Validate performs pure field validation on the config struct. It accumulates
// all errors before returning so the user can fix everything in one pass.
// This method does not perform any I/O (no network connections).
func (t *Type) Validate() error {
	var errs ValidationErrors

	errs = t.validateStructTags(errs)
	errs = t.validateStore(errs)
	errs = t.validateDurations(errs)

	// Listen host:port
	if t.Listen != "" {
		if _, _, err := net.SplitHostPort(t.Listen); err != nil {
			errs = append(errs, fmt.Errorf("listen: invalid host:port %q. Fix: set listen in flowbot.yaml (e.g. \":6060\")", t.Listen))
		}
	}

	modelNames := make(map[string]bool)
	errs, modelNames = t.validateModels(errs, modelNames)
	errs = t.validateChatAgent(errs, modelNames)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// validateStructTags runs playground-validator struct tag checks on sub-structs.
func (t *Type) validateStructTags(errs ValidationErrors) ValidationErrors {
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
	return errs
}

// validateStore checks the store adapter configuration.
func (t *Type) validateStore(errs ValidationErrors) ValidationErrors {
	if t.Store.UseAdapter == "" {
		errs = append(errs, fmt.Errorf("store.use_adapter: must not be empty. Fix: set store_config.use_adapter in flowbot.yaml"))
		return errs
	}

	adapterMap := t.Store.Adapters
	if adapterMap == nil || len(adapterMap) == 0 {
		errs = append(errs, fmt.Errorf("store.adapters: must contain adapter %q. Fix: set store_config.adapters.%s in flowbot.yaml", t.Store.UseAdapter, t.Store.UseAdapter))
		return errs
	}

	adapterCfg, ok := adapterMap[t.Store.UseAdapter]
	if !ok {
		errs = append(errs, fmt.Errorf("store.adapters: adapter %q not found in adapters map. Fix: set store_config.adapters.%s in flowbot.yaml", t.Store.UseAdapter, t.Store.UseAdapter))
		return errs
	}

	dsn := extractDSN(adapterCfg)
	if dsn == "" {
		errs = append(errs, fmt.Errorf("store.adapters.%s.dsn: must not be empty. Fix: set store_config.adapters.%s.dsn in flowbot.yaml", t.Store.UseAdapter, t.Store.UseAdapter))
	}
	return errs
}

// validateDurations checks that all duration fields parse as valid Go duration strings.
func (t *Type) validateDurations(errs ValidationErrors) ValidationErrors {
	if t.Homelab.Discovery.ProbeTimeout != "" {
		if _, err := time.ParseDuration(t.Homelab.Discovery.ProbeTimeout); err != nil {
			errs = append(errs, fmt.Errorf("homelab.discovery.probe_timeout: invalid duration %q. Fix: set a valid Go duration (e.g. \"30s\") in homelab.discovery.probe_timeout in flowbot.yaml", t.Homelab.Discovery.ProbeTimeout))
		}
	}
	if t.Capability.EventPool.ExpiryDuration != "" {
		if _, err := time.ParseDuration(t.Capability.EventPool.ExpiryDuration); err != nil {
			errs = append(errs, fmt.Errorf("ability.event_pool.expiry_duration: invalid duration %q. Fix: set a valid Go duration (e.g. \"30s\") in ability.event_pool.expiry_duration in flowbot.yaml", t.Capability.EventPool.ExpiryDuration))
		}
	}
	return errs
}

// supportedModelProviders lists valid models[].provider values accepted by config validation.
var supportedModelProviders = []string{
	"openai",
	"openai_compatible",
	"gemini",
	"anthropic",
}

func isSupportedModelProvider(provider string) bool {
	return slices.Contains(supportedModelProviders, provider)
}

// validateModels validates model configurations and collects model names for
// agent reference checks.
func (t *Type) validateModels(errs ValidationErrors, modelNames map[string]bool) (ValidationErrors, map[string]bool) {
	for i, m := range t.Models {
		if m.Provider == "" {
			errs = append(errs, fmt.Errorf("models[%d].provider: must not be empty. Fix: set models[%d].provider in flowbot.yaml", i, i))
		} else if !isSupportedModelProvider(m.Provider) {
			errs = append(errs, fmt.Errorf(
				"models[%d].provider: unsupported value %q. Fix: set models[%d].provider to one of [openai, openai_compatible, gemini, anthropic] in flowbot.yaml",
				i, m.Provider, i,
			))
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
	}
	return errs, modelNames
}

// validateChatAgent validates chat agent model configuration when chat_model is set.
func (t *Type) validateChatAgent(errs ValidationErrors, modelNames map[string]bool) ValidationErrors {
	chat := t.ChatAgent.ChatModel
	if chat == "" {
		return errs
	}
	if len(modelNames) > 0 && !modelNames[chat] {
		errs = append(errs, fmt.Errorf(
			"chat_agent.chat_model: %q not found in models. Fix: reference an existing model name in chat_agent.chat_model in flowbot.yaml",
			chat,
		))
	}
	tool := t.ChatAgent.ToolModel
	if tool != "" {
		if len(modelNames) > 0 && !modelNames[tool] {
			errs = append(errs, fmt.Errorf(
				"chat_agent.tool_model: %q not found in models. Fix: reference an existing model name in chat_agent.tool_model in flowbot.yaml",
				tool,
			))
		}
		if len(modelNames) > 0 && modelNames[chat] && modelNames[tool] {
			chatProvider := providerForModelInList(t.Models, chat)
			toolProvider := providerForModelInList(t.Models, tool)
			if chatProvider != toolProvider {
				errs = append(errs, fmt.Errorf(
					"chat_agent: chat_model %q (provider %q) and tool_model %q (provider %q) must use the same provider. Fix: align providers in flowbot.yaml",
					chat, chatProvider, tool, toolProvider,
				))
			}
		}
	}
	return errs
}

// extractDSN extracts the DSN string from an adapter config stored as `any`.
func extractDSN(cfg any) string {
	m, ok := cfg.(map[string]any)
	if !ok {
		return ""
	}
	dsn, ok := m["dsn"].(string)
	if !ok {
		return ""
	}
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

// ReachabilityCheck attempts PostgreSQL and Redis connections with short
// timeouts to verify that dependencies are reachable. Only call this after
// Validate() passes, since it assumes required fields are non-empty.
func (t *Type) ReachabilityCheck(ctx context.Context) error {
	var errs ValidationErrors

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
					_ = db.Close()
				}
				cancel()
			}
		}
	}

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
