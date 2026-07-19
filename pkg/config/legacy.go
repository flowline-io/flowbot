package config

import "fmt"

// RejectLegacyKeys reports migration errors when obsolete YAML top-level keys are present.
// Call with viper.AllSettings() (or an equivalent raw map) before relying on unmarshaled config.
func RejectLegacyKeys(settings map[string]any) error {
	var errs ValidationErrors

	if _, ok := settings["store_config"]; ok {
		errs = append(errs, fmt.Errorf(
			"store_config: removed. Fix: migrate to postgres.dsn (see docs/reference/config-reference.md)",
		))
	}

	redisRaw, ok := settings["redis"]
	if !ok {
		if len(errs) > 0 {
			return errs
		}
		return nil
	}
	redisMap, ok := redisRaw.(map[string]any)
	if !ok {
		if len(errs) > 0 {
			return errs
		}
		return nil
	}

	for _, key := range []string{"host", "port", "password", "db", "pass"} {
		if _, present := redisMap[key]; present {
			errs = append(errs, fmt.Errorf(
				"redis.%s: removed. Fix: set redis.url (e.g. redis://:PASSWORD@HOST:PORT/DB); see docs/reference/config-reference.md",
				key,
			))
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
