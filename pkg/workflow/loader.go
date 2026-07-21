// Package workflow provides the workflow definition loader and runtime.
package workflow

import (
	"fmt"
	"os"
	"regexp"
	"slices"

	"github.com/goccy/go-yaml"

	"github.com/flowline-io/flowbot/pkg/types"
)

// inputRefPattern matches best-effort {{input.name}} template references.
var inputRefPattern = regexp.MustCompile(`\{\{\s*input\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// validInputTypes are the allowed WorkflowInputDef.Type values.
var validInputTypes = map[string]struct{}{
	types.WorkflowInputTypeString:  {},
	types.WorkflowInputTypeNumber:  {},
	types.WorkflowInputTypeBoolean: {},
	types.WorkflowInputTypeJSON:    {},
}

// LoadFile reads and parses a workflow YAML file from disk.
func LoadFile(path string) (*types.WorkflowMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow file: %w", err)
	}
	return ParseYAML(data)
}

// ParseYAML unmarshals workflow YAML, validates structure, input types, and input.* template refs.
func ParseYAML(data []byte) (*types.WorkflowMetadata, error) {
	var wf types.WorkflowMetadata
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow yaml: %w", err)
	}

	// Missing enabled defaults to true (Go bool zero value is false).
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["enabled"]; !ok {
			wf.Enabled = true
		}
		applyTriggerEnabledDefaults(&wf, raw)
	} else {
		wf.Enabled = true
	}

	if wf.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}
	if len(wf.Pipeline) == 0 {
		return nil, fmt.Errorf("workflow pipeline is required")
	}
	if len(wf.Tasks) == 0 {
		return nil, fmt.Errorf("workflow tasks are required")
	}
	if err := validateInputDefs(wf.Inputs); err != nil {
		return nil, err
	}
	if err := validateInputTemplateRefs(&wf); err != nil {
		return nil, err
	}
	if err := ValidateDAG(wf.Tasks); err != nil {
		return nil, fmt.Errorf("workflow dag: %w", err)
	}
	return &wf, nil
}

// ExportYAML marshals a workflow definition to YAML exchange format.
func ExportYAML(wf *types.WorkflowMetadata) ([]byte, error) {
	if wf == nil {
		return nil, fmt.Errorf("workflow is nil")
	}
	data, err := yaml.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("export workflow yaml: %w", err)
	}
	return data, nil
}

func applyTriggerEnabledDefaults(wf *types.WorkflowMetadata, raw map[string]any) {
	rawTriggers, ok := raw["triggers"].([]any)
	if !ok {
		for i := range wf.Triggers {
			wf.Triggers[i].Enabled = true
		}
		return
	}
	for i := range wf.Triggers {
		if i >= len(rawTriggers) {
			wf.Triggers[i].Enabled = true
			continue
		}
		trig, ok := rawTriggers[i].(map[string]any)
		if !ok {
			wf.Triggers[i].Enabled = true
			continue
		}
		if _, ok := trig["enabled"]; !ok {
			wf.Triggers[i].Enabled = true
		}
	}
}

func validateInputDefs(inputs []types.WorkflowInputDef) error {
	seen := make(map[string]struct{}, len(inputs))
	for _, in := range inputs {
		if in.Name == "" {
			return fmt.Errorf("workflow input name is required")
		}
		if _, ok := seen[in.Name]; ok {
			return fmt.Errorf("duplicate workflow input %q", in.Name)
		}
		seen[in.Name] = struct{}{}
		if _, ok := validInputTypes[in.Type]; !ok {
			return fmt.Errorf("workflow input %q has invalid type %q", in.Name, in.Type)
		}
	}
	return nil
}

func validateInputTemplateRefs(wf *types.WorkflowMetadata) error {
	declared := make(map[string]struct{}, len(wf.Inputs))
	for _, in := range wf.Inputs {
		declared[in.Name] = struct{}{}
	}
	refs := collectInputRefs(wf)
	var undeclared []string
	for _, ref := range refs {
		if _, ok := declared[ref]; !ok {
			undeclared = append(undeclared, ref)
		}
	}
	if len(undeclared) == 0 {
		return nil
	}
	slices.Sort(undeclared)
	return fmt.Errorf("undeclared input template refs: %v", undeclared)
}

func collectInputRefs(wf *types.WorkflowMetadata) []string {
	seen := make(map[string]struct{})
	var refs []string
	add := func(s string) {
		for _, m := range inputRefPattern.FindAllStringSubmatch(s, -1) {
			if len(m) < 2 {
				continue
			}
			name := m[1]
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			refs = append(refs, name)
		}
	}
	for _, task := range wf.Tasks {
		add(task.Describe)
		add(task.Action)
		for _, v := range task.Vars {
			add(v)
		}
		for _, v := range task.Conn {
			add(v)
		}
		scanAnyForInputRefs(task.Params, add)
	}
	return refs
}

func scanAnyForInputRefs(v any, add func(string)) {
	switch t := v.(type) {
	case string:
		add(t)
	case types.KV:
		for _, val := range t {
			scanAnyForInputRefs(val, add)
		}
	case map[string]any:
		for _, val := range t {
			scanAnyForInputRefs(val, add)
		}
	case []any:
		for _, val := range t {
			scanAnyForInputRefs(val, add)
		}
	case []string:
		for _, val := range t {
			add(val)
		}
	}
}
