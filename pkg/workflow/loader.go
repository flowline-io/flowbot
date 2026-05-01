package workflow

import (
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/pkg/types"
	"gopkg.in/yaml.v3"
)

func LoadFile(path string) (*types.WorkflowMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow file: %w", err)
	}
	return ParseYAML(data)
}

func ParseYAML(data []byte) (*types.WorkflowMetadata, error) {
	var wf types.WorkflowMetadata
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow yaml: %w", err)
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
	return &wf, nil
}
