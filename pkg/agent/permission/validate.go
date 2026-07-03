package permission

import "fmt"

var sensitiveDefaultAllowKeys = map[string]struct{}{
	"bash":               {},
	"edit":               {},
	"read":               {},
	KeyExternalDirectory: {},
	KeyDelegate:          {},
	KeySchedule:          {},
}

// ValidateUserConfig rejects overly broad or unsafe user permission overrides.
func ValidateUserConfig(cfg Config) error {
	for key, rs := range cfg {
		if key == KeyWildcard {
			return fmt.Errorf("permission key %q cannot be overridden", KeyWildcard)
		}
		if err := validateRuleSet(key, rs); err != nil {
			return err
		}
	}
	return nil
}

func validateRuleSet(key string, rs RuleSet) error {
	if _, sensitive := sensitiveDefaultAllowKeys[key]; sensitive {
		if len(rs.Patterns) == 0 && rs.Default == ActionAllow {
			return fmt.Errorf("permission key %q: default allow is not permitted", key)
		}
	}
	for _, rule := range rs.Patterns {
		if IsOverlyBroadPattern(rule.Pattern) {
			return fmt.Errorf("permission key %q: pattern %q is too broad", key, rule.Pattern)
		}
	}
	return nil
}
