package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestParseYAML_InputsValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		data        string
		wantErr     bool
		errContains string
		check       func(t *testing.T, wf *types.WorkflowMetadata)
	}{
		{
			name: "valid declared inputs and matching refs",
			data: `
name: with-inputs
enabled: true
inputs:
  - name: url
    type: string
    required: true
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo
    params:
      target: "{{input.url}}"
`,
			check: func(t *testing.T, wf *types.WorkflowMetadata) {
				require.Len(t, wf.Inputs, 1)
				assert.Equal(t, "url", wf.Inputs[0].Name)
				assert.True(t, wf.Enabled)
			},
		},
		{
			name: "invalid input type rejected",
			data: `
name: bad-type
inputs:
  - name: count
    type: integer
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo
`,
			wantErr:     true,
			errContains: "invalid type",
		},
		{
			name: "undeclared input template ref rejected",
			data: `
name: missing-decl
inputs:
  - name: title
    type: string
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo
    params:
      url: "{{input.url}}"
`,
			wantErr:     true,
			errContains: "undeclared input template refs",
		},
		{
			name: "enabled defaults to true when omitted",
			data: `
name: default-enabled
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo
`,
			check: func(t *testing.T, wf *types.WorkflowMetadata) {
				assert.True(t, wf.Enabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wf, err := ParseYAML([]byte(tt.data))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, wf)
			if tt.check != nil {
				tt.check(t, wf)
			}
		})
	}
}

func TestApplyInputDefaults(t *testing.T) {
	t.Parallel()
	declared := []types.WorkflowInputDef{
		{Name: "url", Type: types.WorkflowInputTypeString, Required: true},
		{Name: "optional", Type: types.WorkflowInputTypeString, Default: "x"},
		{Name: "count", Type: types.WorkflowInputTypeNumber, Default: 3},
	}
	tests := []struct {
		name  string
		input types.KV
		want  types.KV
	}{
		{
			name:  "fills missing defaults",
			input: types.KV{"url": "https://example.com"},
			want:  types.KV{"url": "https://example.com", "optional": "x", "count": 3},
		},
		{
			name:  "keeps provided values over defaults",
			input: types.KV{"url": "u", "optional": "y", "count": 9},
			want:  types.KV{"url": "u", "optional": "y", "count": 9},
		},
		{
			name:  "nil input still applies defaults",
			input: nil,
			want:  types.KV{"optional": "x", "count": 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ApplyInputDefaults(declared, tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateInputs(t *testing.T) {
	t.Parallel()
	declared := []types.WorkflowInputDef{
		{Name: "url", Type: types.WorkflowInputTypeString, Required: true},
		{Name: "count", Type: types.WorkflowInputTypeNumber},
		{Name: "flag", Type: types.WorkflowInputTypeBoolean},
		{Name: "meta", Type: types.WorkflowInputTypeJSON},
		{Name: "optional", Type: types.WorkflowInputTypeString, Default: "x"},
	}
	tests := []struct {
		name        string
		input       types.KV
		wantErr     bool
		errContains string
	}{
		{
			name: "all valid types",
			input: types.KV{
				"url":   "https://example.com",
				"count": 3,
				"flag":  true,
				"meta":  map[string]any{"k": "v"},
			},
		},
		{
			name:        "required missing",
			input:       types.KV{"count": 1},
			wantErr:     true,
			errContains: `required input "url" is missing`,
		},
		{
			name: "wrong string type",
			input: types.KV{
				"url": 42,
			},
			wantErr:     true,
			errContains: `input "url" must be a string`,
		},
		{
			name: "optional with default may be omitted",
			input: types.KV{
				"url": "ok",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateInputs(declared, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestExportYAML_Roundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		wf   *types.WorkflowMetadata
	}{
		{
			name: "key fields preserved",
			wf: &types.WorkflowMetadata{
				Name:           "export-wf",
				Describe:       "desc",
				Enabled:        true,
				Resumable:      true,
				MaxConcurrency: 2,
				Inputs: []types.WorkflowInputDef{
					{Name: "url", Type: types.WorkflowInputTypeString, Required: true, Description: "target"},
				},
				Triggers: []types.WorkflowTriggerDef{
					{Type: "manual", Enabled: true},
				},
				Pipeline: []string{"step1"},
				Tasks: []types.WorkflowTask{
					{ID: "step1", Action: "shell:echo", Params: types.KV{"msg": "{{input.url}}"}},
				},
			},
		},
		{
			name: "disabled workflow",
			wf: &types.WorkflowMetadata{
				Name:     "disabled-wf",
				Enabled:  false,
				Pipeline: []string{"a"},
				Tasks:    []types.WorkflowTask{{ID: "a", Action: "mapper:"}},
			},
		},
		{
			name: "cron trigger rule preserved",
			wf: &types.WorkflowMetadata{
				Name:     "cron-wf",
				Enabled:  true,
				Triggers: []types.WorkflowTriggerDef{{Type: "cron", Enabled: true, Rule: types.KV{"schedule": "0 * * * *"}}},
				Pipeline: []string{"a"},
				Tasks:    []types.WorkflowTask{{ID: "a", Action: "mapper:"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := ExportYAML(tt.wf)
			require.NoError(t, err)
			got, err := ParseYAML(data)
			require.NoError(t, err)
			assert.Equal(t, tt.wf.Name, got.Name)
			assert.Equal(t, tt.wf.Enabled, got.Enabled)
			assert.Equal(t, tt.wf.Resumable, got.Resumable)
			assert.Equal(t, tt.wf.MaxConcurrency, got.MaxConcurrency)
			assert.Equal(t, tt.wf.Pipeline, got.Pipeline)
			require.Len(t, got.Tasks, len(tt.wf.Tasks))
			if len(tt.wf.Inputs) > 0 {
				require.Len(t, got.Inputs, len(tt.wf.Inputs))
				assert.Equal(t, tt.wf.Inputs[0].Name, got.Inputs[0].Name)
				assert.Equal(t, tt.wf.Inputs[0].Type, got.Inputs[0].Type)
			}
			if len(tt.wf.Triggers) > 0 {
				require.Len(t, got.Triggers, len(tt.wf.Triggers))
				assert.Equal(t, tt.wf.Triggers[0].Type, got.Triggers[0].Type)
			}
		})
	}
}
