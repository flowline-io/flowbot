package config

import (
	"context"
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
	if t.Ability.EventPool.ExpiryDuration != "" {
		if _, err := time.ParseDuration(t.Ability.EventPool.ExpiryDuration); err != nil {
			errs = append(errs, fmt.Errorf("ability.event_pool.expiry_duration: invalid duration %q. Fix: set a valid Go duration (e.g. \"30s\") in ability.event_pool.expiry_duration in flowbot.yaml", t.Ability.EventPool.ExpiryDuration))
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

// ReachabilityCheck is a stub — full implementation provided separately.
func (t *Type) ReachabilityCheck(ctx context.Context) error {
	return fmt.Errorf("reachability check not yet implemented")
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
