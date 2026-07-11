package permission

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FormActionInherit means the user keeps the system default for one permission key.
const FormActionInherit = "inherit"

// FormPatternRow is one pattern rule submitted from the permissions web form.
type FormPatternRow struct {
	Pattern string
	Action  string
}

// FormValues holds parsed permission form submissions.
type FormValues struct {
	Simple   map[string]string
	Patterns map[string][]FormPatternRow
}

var (
	simplePermFieldRE  = regexp.MustCompile(`^perm\[([^\]]+)\]$`)
	patternPermFieldRE = regexp.MustCompile(`^perm\[([^\]]+)\]\[patterns\]\[(\d+)\]\[(pattern|action)\]$`)
)

// ParseFormPostArgs converts flat HTML form keys into structured permission form values.
func ParseFormPostArgs(args map[string]string) FormValues {
	out := FormValues{
		Simple:   make(map[string]string),
		Patterns: make(map[string][]FormPatternRow),
	}
	patternScratch := make(map[string]map[int]FormPatternRow)
	for key, value := range args {
		if permKey, ok := parseSimpleFormField(key); ok {
			out.Simple[permKey] = strings.TrimSpace(value)
			continue
		}
		parsePatternFormField(key, value, patternScratch)
	}
	for key, rowsByIdx := range patternScratch {
		if rows := collectPatternRows(rowsByIdx); len(rows) > 0 {
			out.Patterns[key] = rows
		}
	}
	return out
}

func parseSimpleFormField(key string) (string, bool) {
	m := simplePermFieldRE.FindStringSubmatch(key)
	if len(m) != 2 {
		return "", false
	}
	return m[1], true
}

func parsePatternFormField(key, value string, scratch map[string]map[int]FormPatternRow) {
	m := patternPermFieldRE.FindStringSubmatch(key)
	if len(m) != 4 {
		return
	}
	idx, err := strconv.Atoi(m[2])
	if err != nil {
		return
	}
	permKey := m[1]
	rows := scratch[permKey]
	if rows == nil {
		rows = make(map[int]FormPatternRow)
		scratch[permKey] = rows
	}
	row := rows[idx]
	switch m[3] {
	case "pattern":
		row.Pattern = strings.TrimSpace(value)
	case "action":
		row.Action = strings.TrimSpace(value)
	}
	rows[idx] = row
}

func collectPatternRows(rowsByIdx map[int]FormPatternRow) []FormPatternRow {
	maxIdx := -1
	for idx := range rowsByIdx {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	rows := make([]FormPatternRow, 0, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		row, ok := rowsByIdx[i]
		if !ok || (row.Pattern == "" && row.Action == "") {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

// BuildUserConfigFromForm builds a user override config from form values and defaults.
// Matching defaults are omitted. Field errors map form keys to messages.
func BuildUserConfigFromForm(defaults Config, form FormValues) (Config, map[string]string, error) {
	out := make(Config)
	fieldErrors := make(map[string]string)

	mergeSimpleFormRules(out, defaults, form.Simple, fieldErrors)
	mergePatternFormRules(out, defaults, form.Patterns, fieldErrors)

	if len(fieldErrors) > 0 {
		return nil, fieldErrors, fmt.Errorf("invalid permission form")
	}
	if err := ValidateUserConfig(out); err != nil {
		return nil, nil, err
	}
	return out, nil, nil
}

func mergeSimpleFormRules(out Config, defaults Config, simple map[string]string, fieldErrors map[string]string) {
	for key, action := range simple {
		if action == "" || action == FormActionInherit {
			continue
		}
		parsed, ok := ParseAction(action)
		if !ok {
			fieldErrors[formFieldKey(key)] = fmt.Sprintf("invalid action %q", action)
			continue
		}
		rs := RuleSet{Default: parsed}
		if ruleSetEqual(rs, defaults[key]) {
			continue
		}
		out[key] = rs
	}
}

func mergePatternFormRules(out Config, defaults Config, patterns map[string][]FormPatternRow, fieldErrors map[string]string) {
	for key, rows := range patterns {
		rs, ok := buildPatternRuleSet(key, rows, fieldErrors)
		if !ok || len(rs.Patterns) == 0 {
			continue
		}
		if ruleSetEqual(rs, defaults[key]) {
			continue
		}
		out[key] = rs
	}
}

func buildPatternRuleSet(key string, rows []FormPatternRow, fieldErrors map[string]string) (RuleSet, bool) {
	rs := RuleSet{}
	valid := true
	for i, row := range rows {
		pattern := strings.TrimSpace(row.Pattern)
		action := strings.TrimSpace(row.Action)
		if pattern == "" && action == "" {
			continue
		}
		if pattern == "" {
			fieldErrors[patternFieldKey(key, i)] = "pattern is required"
			valid = false
			continue
		}
		if IsOverlyBroadPattern(pattern) {
			fieldErrors[patternFieldKey(key, i)] = "pattern is too broad"
			valid = false
			continue
		}
		parsed, ok := ParseAction(action)
		if !ok {
			fieldErrors[patternActionFieldKey(key, i)] = fmt.Sprintf("invalid action %q", action)
			valid = false
			continue
		}
		rs.Patterns = append(rs.Patterns, PatternRule{Pattern: pattern, Action: parsed})
	}
	return rs, valid
}

func formFieldKey(key string) string {
	return "perm." + key
}

func patternFieldKey(key string, idx int) string {
	return fmt.Sprintf("perm.%s.patterns.%d.pattern", key, idx)
}

func patternActionFieldKey(key string, idx int) string {
	return fmt.Sprintf("perm.%s.patterns.%d.action", key, idx)
}

func ruleSetEqual(a, b RuleSet) bool {
	if a.Default != b.Default {
		return false
	}
	if len(a.Patterns) != len(b.Patterns) {
		return false
	}
	for i := range a.Patterns {
		if a.Patterns[i].Pattern != b.Patterns[i].Pattern || a.Patterns[i].Action != b.Patterns[i].Action {
			return false
		}
	}
	return true
}
