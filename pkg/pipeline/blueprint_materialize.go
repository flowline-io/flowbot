package pipeline

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/flowline-io/flowbot/pkg/types"
)

// BindBlueprintInputs merges defaults with provided values and type-checks them.
func BindBlueprintInputs(defs []BlueprintInputDef, provided map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Name)
		raw, has := provided[name]
		if !has || raw == nil || raw == "" {
			if def.Default != nil {
				raw = def.Default
				has = true
			}
		}
		if !has || raw == nil || raw == "" {
			if def.Required {
				return nil, types.Errorf(types.ErrInvalidArgument, "input %s is required", name)
			}
			continue
		}
		typed, err := coerceBlueprintInput(def, raw)
		if err != nil {
			return nil, err
		}
		out[name] = typed
	}
	return out, nil
}

// MaterializeEditorYAML substitutes {{inputs.*}} into the blueprint definition and
// returns a published EditorDefinition YAML for the given instance name and enabled flag.
func MaterializeEditorYAML(doc *BlueprintDocument, instanceName string, enabled bool, inputs map[string]any) (string, error) {
	if err := ValidateName(instanceName); err != nil {
		return "", types.WrapError(types.ErrInvalidArgument, "invalid pipeline name", err)
	}
	if err := validateBlueprintDocument(doc); err != nil {
		return "", err
	}
	bound, err := BindBlueprintInputs(doc.Inputs, inputs)
	if err != nil {
		return "", err
	}
	body, err := substituteInputs(cloneAny(doc.Definition), bound)
	if err != nil {
		return "", err
	}
	defMap, ok := body.(map[string]any)
	if !ok {
		return "", types.Errorf(types.ErrInvalidArgument, "blueprint definition must be a mapping")
	}
	defMap["name"] = instanceName
	defMap["enabled"] = enabled
	raw, err := yaml.Marshal(defMap)
	if err != nil {
		return "", fmt.Errorf("marshal materialized pipeline: %w", err)
	}
	ed, err := ParseEditorYAML(string(raw))
	if err != nil {
		return "", types.WrapError(types.ErrInvalidArgument, "materialized pipeline YAML is invalid", err)
	}
	ed.Name = instanceName
	ed.Enabled = enabled
	if enabled {
		syncCronTriggersEnabled(ed, true)
	} else {
		syncCronTriggersEnabled(ed, false)
	}
	out, err := yaml.Marshal(ed)
	if err != nil {
		return "", fmt.Errorf("marshal editor definition: %w", err)
	}
	return string(out), nil
}

func coerceBlueprintInput(def BlueprintInputDef, raw any) (any, error) {
	switch def.Type {
	case BlueprintInputString:
		return coerceStringInput(def.Name, raw)
	case BlueprintInputBoolean:
		return coerceBoolInput(def.Name, raw)
	case BlueprintInputNumber:
		return coerceNumberInput(def.Name, raw)
	case BlueprintInputNotifyChannel:
		return coerceNotifyChannelInput(def, raw)
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "unsupported input type %q", def.Type)
	}
}

func coerceStringInput(name string, raw any) (any, error) {
	s, ok := raw.(string)
	if !ok {
		return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a string", name)
	}
	return s, nil
}

func coerceBoolInput(name string, raw any) (any, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a boolean", name)
		}
		return b, nil
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a boolean", name)
	}
}

func coerceNumberInput(name string, raw any) (any, error) {
	switch v := raw.(type) {
	case int:
		return v, nil
	case int64:
		return v, nil
	case float64:
		return v, nil
	case string:
		if strings.Contains(v, ".") {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a number", name)
			}
			return f, nil
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a number", name)
		}
		return n, nil
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a number", name)
	}
}

func coerceNotifyChannelInput(def BlueprintInputDef, raw any) (any, error) {
	channels, err := coerceStringList(raw)
	if err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "input %s must be a list of channel names", def.Name)
	}
	if def.Required && len(channels) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "input %s is required", def.Name)
	}
	return channels, nil
}

func coerceStringList(raw any) ([]string, error) {
	switch v := raw.(type) {
	case []string:
		return v, nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return []string{}, nil
		}
		return []string{s}, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, types.ErrInvalidArgument
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	default:
		return nil, types.ErrInvalidArgument
	}
}

func substituteInputs(v any, values map[string]any) (any, error) {
	switch t := v.(type) {
	case string:
		return substituteInputString(t, values)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			nv, err := substituteInputs(val, values)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			nv, err := substituteInputs(val, values)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

func substituteInputString(s string, values map[string]any) (any, error) {
	trimmed := strings.TrimSpace(s)
	if m := blueprintInputPlaceholder.FindStringSubmatch(trimmed); len(m) == 2 && m[0] == trimmed {
		val, ok := values[m[1]]
		if !ok {
			return nil, types.Errorf(types.ErrInvalidArgument, "unresolved input %s", m[1])
		}
		return val, nil
	}
	var unresolved string
	out := blueprintInputPlaceholder.ReplaceAllStringFunc(s, func(match string) string {
		m := blueprintInputPlaceholder.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		val, ok := values[m[1]]
		if !ok {
			unresolved = m[1]
			return match
		}
		return fmt.Sprint(val)
	})
	if unresolved != "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "unresolved input %s", unresolved)
	}
	return out, nil
}

func cloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = cloneAny(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = cloneAny(val)
		}
		return out
	default:
		return v
	}
}
