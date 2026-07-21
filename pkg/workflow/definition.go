package workflow

import (
	"fmt"
	"maps"
	"reflect"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

// WorkflowRows is the store-level projection used to rebuild WorkflowMetadata.
type WorkflowRows struct {
	Workflow *gen.Workflow
	Tasks    []*gen.WorkflowTask
	Triggers []*gen.WorkflowTrigger
}

// MetadataFromRows converts normalized store rows into WorkflowMetadata.
func MetadataFromRows(rows WorkflowRows) (*types.WorkflowMetadata, error) {
	if rows.Workflow == nil {
		return nil, fmt.Errorf("workflow row is nil")
	}
	wf := rows.Workflow
	meta := &types.WorkflowMetadata{
		Name:           wf.Name,
		Describe:       wf.Describe,
		Enabled:        wf.Enabled,
		Resumable:      wf.Resumable,
		MaxConcurrency: wf.MaxConcurrency,
		Pipeline:       append([]string(nil), wf.Pipeline...),
		Inputs:         inputsFromMaps(wf.Inputs),
		Tasks:          make([]types.WorkflowTask, 0, len(rows.Tasks)),
		Triggers:       make([]types.WorkflowTriggerDef, 0, len(rows.Triggers)),
	}
	for _, t := range rows.Tasks {
		if t == nil {
			continue
		}
		meta.Tasks = append(meta.Tasks, types.WorkflowTask{
			ID:       t.TaskID,
			Action:   t.Action,
			Describe: t.Describe,
			Params:   types.KV(cloneMap(t.Params)),
			Vars:     append([]string(nil), t.Vars...),
			Conn:     append([]string(nil), t.Conn...),
			Retry:    retryFromMap(t.Retry),
		})
	}
	for _, tr := range rows.Triggers {
		if tr == nil {
			continue
		}
		meta.Triggers = append(meta.Triggers, types.WorkflowTriggerDef{
			Type:    tr.Type,
			Enabled: tr.Enabled,
			Rule:    types.KV(cloneMap(tr.Rule)),
		})
	}
	return meta, nil
}

// InputsToMaps converts WorkflowInputDef slice to JSON-friendly maps for storage.
func InputsToMaps(inputs []types.WorkflowInputDef) []map[string]any {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(inputs))
	for _, in := range inputs {
		m := map[string]any{
			"name": in.Name,
			"type": in.Type,
		}
		if in.Required {
			m["required"] = true
		}
		if in.Default != nil {
			m["default"] = in.Default
		}
		if in.Description != "" {
			m["description"] = in.Description
		}
		out = append(out, m)
	}
	return out
}

// RetryToMap converts RetryConfig to a JSON map for storage.
func RetryToMap(r *types.RetryConfig) map[string]any {
	if r == nil {
		return nil
	}
	m := map[string]any{
		"max_attempts": r.MaxAttempts,
		"backoff":      r.Backoff,
		"jitter":       r.Jitter,
	}
	if r.Delay > 0 {
		m["delay"] = r.Delay.String()
	}
	if r.MaxDelay > 0 {
		m["max_delay"] = r.MaxDelay.String()
	}
	if len(r.RetryOn) > 0 {
		m["retry_on"] = append([]string(nil), r.RetryOn...)
	}
	return m
}

// ApplyInputDefaults copies input and fills missing keys from declared defaults.
func ApplyInputDefaults(declared []types.WorkflowInputDef, input types.KV) types.KV {
	out := types.KV{}
	if input != nil {
		maps.Copy(out, input)
	}
	for _, def := range declared {
		if def.Name == "" || def.Default == nil {
			continue
		}
		if _, present := out[def.Name]; !present {
			out[def.Name] = def.Default
		}
	}
	return out
}

// ValidateInputs checks required fields and type constraints against declared input defs.
func ValidateInputs(declared []types.WorkflowInputDef, input types.KV) error {
	if input == nil {
		input = types.KV{}
	}
	for _, def := range declared {
		val, present := input[def.Name]
		if !present || val == nil {
			if def.Required && def.Default == nil {
				return fmt.Errorf("required input %q is missing", def.Name)
			}
			continue
		}
		if err := validateInputValue(def, val); err != nil {
			return err
		}
	}
	return nil
}

func validateInputValue(def types.WorkflowInputDef, val any) error {
	switch def.Type {
	case types.WorkflowInputTypeString:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("input %q must be a string", def.Name)
		}
	case types.WorkflowInputTypeNumber:
		if !isNumber(val) {
			return fmt.Errorf("input %q must be a number", def.Name)
		}
	case types.WorkflowInputTypeBoolean:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("input %q must be a boolean", def.Name)
		}
	case types.WorkflowInputTypeJSON:
		switch val.(type) {
		case map[string]any, []any, types.KV:
			// ok
		default:
			return fmt.Errorf("input %q must be a json object or array", def.Name)
		}
	default:
		return fmt.Errorf("input %q has unsupported type %q", def.Name, def.Type)
	}
	return nil
}

func isNumber(val any) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func inputsFromMaps(raw []map[string]any) []types.WorkflowInputDef {
	if len(raw) == 0 {
		return nil
	}
	out := make([]types.WorkflowInputDef, 0, len(raw))
	for _, m := range raw {
		def := types.WorkflowInputDef{
			Name:        stringFromAny(m["name"]),
			Type:        stringFromAny(m["type"]),
			Required:    boolFromAny(m["required"]),
			Default:     m["default"],
			Description: stringFromAny(m["description"]),
		}
		out = append(out, def)
	}
	return out
}

func retryFromMap(m map[string]any) *types.RetryConfig {
	if len(m) == 0 {
		return nil
	}
	r := &types.RetryConfig{
		MaxAttempts: intFromAny(m["max_attempts"]),
		Backoff:     stringFromAny(m["backoff"]),
		Jitter:      boolFromAny(m["jitter"]),
		Delay:       durationFromAny(m["delay"]),
		MaxDelay:    durationFromAny(m["max_delay"]),
	}
	if on, ok := m["retry_on"].([]any); ok {
		for _, v := range on {
			if s, ok := v.(string); ok {
				r.RetryOn = append(r.RetryOn, s)
			}
		}
	} else if on, ok := m["retry_on"].([]string); ok {
		r.RetryOn = append([]string(nil), on...)
	}
	return r
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	maps.Copy(out, m)
	return out
}

func stringFromAny(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func boolFromAny(v any) bool {
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.CanInt() {
			return int(rv.Int())
		}
		return 0
	}
}

func durationFromAny(v any) time.Duration {
	switch d := v.(type) {
	case time.Duration:
		return d
	case string:
		parsed, err := time.ParseDuration(d)
		if err != nil {
			return 0
		}
		return parsed
	case float64:
		return time.Duration(d)
	case int64:
		return time.Duration(d)
	case int:
		return time.Duration(d)
	default:
		return 0
	}
}
