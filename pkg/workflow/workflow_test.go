package workflow

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestParseAction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		action           string
		wantIsCapability bool
		wantType         string
		wantCapType      string
		wantOperation    string
		wantDetails      string
	}{
		{
			name:             "capability-with-dot",
			action:           "capability:bookmark.list",
			wantIsCapability: true,
			wantType:         "capability",
			wantCapType:      "bookmark",
			wantOperation:    "list",
			wantDetails:      "bookmark.list",
		},
		{
			name:             "capability-no-dot",
			action:           "capability:bookmark",
			wantIsCapability: true,
			wantType:         "capability",
			wantCapType:      "",
			wantOperation:    "",
			wantDetails:      "bookmark",
		},
		{
			name:             "docker",
			action:           "docker:nginx:latest",
			wantIsCapability: false,
			wantType:         "docker",
			wantCapType:      "",
			wantOperation:    "",
			wantDetails:      "nginx:latest",
		},
		{
			name:             "shell",
			action:           "shell:echo hello",
			wantIsCapability: false,
			wantType:         "shell",
			wantCapType:      "",
			wantOperation:    "",
			wantDetails:      "echo hello",
		},
		{
			name:             "plain-string",
			action:           "echo",
			wantIsCapability: false,
			wantType:         "echo",
			wantCapType:      "",
			wantOperation:    "",
			wantDetails:      "",
		},
		{
			name:             "empty",
			action:           "",
			wantIsCapability: false,
			wantType:         "",
			wantCapType:      "",
			wantOperation:    "",
			wantDetails:      "",
		},
		{
			name:             "mapper",
			action:           "mapper:",
			wantIsCapability: false,
			wantType:         "mapper",
			wantCapType:      "",
			wantOperation:    "",
			wantDetails:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := ParseAction(tt.action)
			assert.Equal(t, tt.wantIsCapability, info.IsCapability)
			assert.Equal(t, tt.wantType, info.Type)
			assert.Equal(t, tt.wantCapType, info.CapType)
			assert.Equal(t, tt.wantOperation, info.Operation)
			assert.Equal(t, tt.wantDetails, info.Details)
		})
	}
}

func TestDetermineRuntimeType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		task *types.Task
		want string
	}{
		{
			name: "capability",
			task: &types.Task{Run: "capability:bookmark.list"},
			want: "capability",
		},
		{
			name: "docker",
			task: &types.Task{Run: "", Image: "nginx:latest"},
			want: "docker",
		},
		{
			name: "shell",
			task: &types.Task{Run: "echo hello", Image: ""},
			want: "shell",
		},
		{
			name: "image-takes-precedence",
			task: &types.Task{Run: "some-run", Image: "alpine"},
			want: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, DetermineRuntimeType(tt.task))
		})
	}
}

func TestWorkflowTaskToTask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		wt          types.WorkflowTask
		wantErr     bool
		errContains string
		check       func(t *testing.T, task *types.Task)
	}{
		{
			name: "capability",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "capability:bookmark.list",
				Params: types.KV{"url": "https://example.com"},
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "capability:bookmark.list", task.Run)
				assert.Contains(t, task.Env, "CAPABILITY_PARAMS")
				assert.JSONEq(t, `{"url":"https://example.com"}`, task.Env["CAPABILITY_PARAMS"])
			},
		},
		{
			name: "capability-no-params",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "capability:bookmark.list",
			},
			check: func(t *testing.T, task *types.Task) {
				assert.NotContains(t, task.Env, "CAPABILITY_PARAMS")
			},
		},
		{
			name: "docker",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "docker:nginx:latest",
				Params: types.KV{"cmd": "nginx -g daemon off;"},
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "nginx:latest", task.Image)
				assert.Equal(t, []string{"nginx -g daemon off;"}, task.CMD)
			},
		},
		{
			name: "docker-slice-cmd",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "docker:alpine",
				Params: types.KV{"cmd": []any{"sh", "-c", "echo hi"}},
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, []string{"sh", "-c", "echo hi"}, task.CMD)
			},
		},
		{
			name: "shell",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "shell:echo hello",
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "echo hello", task.Run)
			},
		},
		{
			name: "shell-with-cmd-param",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "shell:echo hello",
				Params: types.KV{"cmd": "ls -la"},
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "ls -la", task.Run)
			},
		},
		{
			name: "machine",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "machine:vm1",
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "vm1", task.Run)
			},
		},
		{
			name: "default",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "custom-action",
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "custom-action", task.Run)
			},
		},
		{
			name: "mapper",
			wt: types.WorkflowTask{
				ID:     "map1",
				Action: "mapper:",
				Params: types.KV{"target_url": "https://example.com"},
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, "mapper:", task.Run)
			},
		},
		{
			name: "marshal-error",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "capability:bookmark.list",
				Params: types.KV{"ch": make(chan int)},
			},
			wantErr:     true,
			errContains: "marshal params",
		},
		{
			name: "slice-cmd-mixed-types",
			wt: types.WorkflowTask{
				ID:     "step1",
				Action: "docker:alpine",
				Params: types.KV{"cmd": []any{"echo", "hello"}},
			},
			check: func(t *testing.T, task *types.Task) {
				assert.Equal(t, []string{"echo", "hello"}, task.CMD)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			task, err := WorkflowTaskToTask(tt.wt)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, task)
			}
		})
	}
}

func TestParseYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		errContains string
		check       func(t *testing.T, wf *types.WorkflowMetadata)
	}{
		{
			name: "valid",
			data: []byte(`
name: test-workflow
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo hello
`),
			check: func(t *testing.T, wf *types.WorkflowMetadata) {
				assert.Equal(t, "test-workflow", wf.Name)
				assert.Equal(t, []string{"step1"}, wf.Pipeline)
				require.Len(t, wf.Tasks, 1)
				assert.Equal(t, "step1", wf.Tasks[0].ID)
				assert.Equal(t, "shell:echo hello", wf.Tasks[0].Action)
			},
		},
		{
			name: "missing-name",
			data: []byte(`
pipeline:
  - step1
tasks:
  - id: step1
    action: echo
`),
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name: "empty-pipeline",
			data: []byte(`
name: test
pipeline: []
tasks:
  - id: step1
    action: echo
`),
			wantErr:     true,
			errContains: "pipeline is required",
		},
		{
			name: "empty-tasks",
			data: []byte(`
name: test
pipeline:
  - step1
tasks: []
`),
			wantErr:     true,
			errContains: "tasks are required",
		},
		{
			name:        "invalid-yaml",
			data:        []byte(`{{{invalid`),
			wantErr:     true,
			errContains: "parse workflow",
		},
		{
			name: "conn-cycle",
			data: []byte(`
name: test
pipeline:
  - a
tasks:
  - id: a
    action: echo
    conn: [b]
  - id: b
    action: echo
    conn: [a]
`),
			wantErr:     true,
			errContains: "cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wf, err := ParseYAML(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, wf)
			}
		})
	}
}

func TestResolveParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		params      types.KV
		results     map[string]string
		input       types.KV
		wantErr     bool
		errContains string
		check       func(t *testing.T, resolved types.KV)
	}{
		{
			name:    "simple-replacement",
			params:  types.KV{"ref": "{{step1.id}}"},
			results: map[string]string{"step1": "abc123"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "abc123", resolved["ref"])
			},
		},
		{
			name:    "no-match",
			params:  types.KV{"ref": "hello world"},
			results: map[string]string{"step1": "abc"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "hello world", resolved["ref"])
			},
		},
		{
			name:    "non-string-value",
			params:  types.KV{"count": 42},
			results: map[string]string{"step1": "abc"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, 42, resolved["count"])
			},
		},
		{
			name:    "multiple-keys",
			params:  types.KV{"a": "{{step1.id}}", "b": "{{step2.id}}"},
			results: map[string]string{"step1": "r1", "step2": "r2"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "r1", resolved["a"])
				assert.Equal(t, "r2", resolved["b"])
			},
		},
		{
			name: "condition-in-params",
			params: types.KV{
				"action": "{{if eq (step \"step1\" \"result\") \"success\"}}proceed{{else}}retry{{end}}",
			},
			results: map[string]string{"step1": "success"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "proceed", resolved["action"])
			},
		},
		{
			name: "condition-else",
			params: types.KV{
				"action": "{{if eq (step \"step1\" \"result\") \"success\"}}proceed{{else}}retry{{end}}",
			},
			results: map[string]string{"step1": "failed"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "retry", resolved["action"])
			},
		},
		{
			name:    "old-syntax-step-result",
			params:  types.KV{"output": "{{steps.step1.result}}"},
			results: map[string]string{"step1": "my-output"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "my-output", resolved["output"])
			},
		},
		{
			name:    "new-syntax-step",
			params:  types.KV{"output": "{{step \"step1\" \"id\"}}"},
			results: map[string]string{"step1": "id-value"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "id-value", resolved["output"])
			},
		},
		{
			name: "default-when-missing",
			params: types.KV{
				"label": "{{default \"no-result\" .Steps.noexist.id}}",
			},
			results: map[string]string{},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "no-result", resolved["label"])
			},
		},
		{
			name: "join-step-results",
			params: types.KV{
				"out": "{{step \"s1\" \"result\"}}|{{step \"s2\" \"result\"}}",
			},
			results: map[string]string{"s1": "a", "s2": "b"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "a|b", resolved["out"])
			},
		},
		{
			name:        "invalid-template",
			params:      types.KV{"bad": "{{if xxx}}"},
			results:     map[string]string{},
			wantErr:     true,
			errContains: "",
		},
		{
			name: "contains-check",
			params: types.KV{
				"match": "{{if contains (step \"step1\" \"result\") \"ok\"}}yes{{else}}no{{end}}",
			},
			results: map[string]string{"step1": "all-ok-done"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "yes", resolved["match"])
			},
		},
		{
			name: "loop-over-steps",
			params: types.KV{
				"all": "{{range $k, $v := .Steps}}{{$k}}={{index $v \"id\"}};{{end}}",
			},
			results: map[string]string{"a": "r1", "b": "r2"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Contains(t, resolved["all"], "a=r1")
				assert.Contains(t, resolved["all"], "b=r2")
			},
		},
		{
			name: "nested-map-value",
			params: types.KV{
				"inner": map[string]any{
					"ref": "{{step \"step1\" \"id\"}}",
				},
			},
			results: map[string]string{"step1": "nested-id"},
			check: func(t *testing.T, resolved types.KV) {
				inner := resolved["inner"].(map[string]any)
				assert.Equal(t, "nested-id", inner["ref"])
			},
		},
		{
			name: "string-slice-value",
			params: types.KV{
				"items": []any{"{{step \"a\" \"result\"}}", "{{step \"b\" \"result\"}}"},
			},
			results: map[string]string{"a": "x", "b": "y"},
			check: func(t *testing.T, resolved types.KV) {
				items := resolved["items"].([]any)
				assert.Equal(t, "x", items[0])
				assert.Equal(t, "y", items[1])
			},
		},
		{
			name:    "empty-results",
			params:  types.KV{"ref": "{{step \"nonexist\" \"id\"}}"},
			results: map[string]string{},
			check: func(t *testing.T, resolved types.KV) {
				assert.Empty(t, resolved["ref"])
			},
		},
		{
			name: "mapper-like",
			params: types.KV{
				"target_url":   "{{step \"src\" \"result\"}}",
				"target_title": "static-title",
			},
			results: map[string]string{"src": "https://example.com"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "https://example.com", resolved["target_url"])
				assert.Equal(t, "static-title", resolved["target_title"])
			},
		},
		{
			name: "mapper-with-jsonpath",
			params: types.KV{
				"id":    `{{jsonpath (step "api" "result") "data.id"}}`,
				"name":  `{{jsonpath (step "api" "result") "data.name"}}`,
				"extra": "fixed",
			},
			results: map[string]string{"api": `{"data":{"id":"123","name":"test"}}`},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "123", resolved["id"])
				assert.Equal(t, "test", resolved["name"])
				assert.Equal(t, "fixed", resolved["extra"])
			},
		},
		{
			name: "mapper-conditional",
			params: types.KV{
				"status": "{{if contains (step \"check\" \"result\") \"ok\"}}mapped-ok{{else}}mapped-fail{{end}}",
			},
			results: map[string]string{"check": "all-ok-done"},
			check: func(t *testing.T, resolved types.KV) {
				assert.Equal(t, "mapped-ok", resolved["status"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resolved, err := resolveParams(tt.params, tt.results, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resolved)
			}
		})
	}
}

func TestValidateDAG(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		tasks       []types.WorkflowTask
		wantErr     bool
		errContains string
	}{
		{
			name: "no-cycle",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"b"}},
				{ID: "b", Conn: []string{"c"}},
				{ID: "c"},
			},
		},
		{
			name: "direct-cycle",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"b"}},
				{ID: "b", Conn: []string{"a"}},
			},
			wantErr:     true,
			errContains: "cycle detected",
		},
		{
			name: "indirect-cycle",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"b"}},
				{ID: "b", Conn: []string{"c"}},
				{ID: "c", Conn: []string{"a"}},
			},
			wantErr:     true,
			errContains: "cycle detected",
		},
		{
			name: "self-cycle",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"a"}},
			},
			wantErr:     true,
			errContains: "cycle detected",
		},
		{
			name: "empty-conn",
			tasks: []types.WorkflowTask{
				{ID: "a"},
				{ID: "b"},
				{ID: "c"},
			},
		},
		{
			name: "unknown-dependency",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"nonexistent"}},
			},
			wantErr:     true,
			errContains: "references unknown dependency",
		},
		{
			name: "multiple-roots",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"c"}},
				{ID: "b", Conn: []string{"c"}},
				{ID: "c"},
			},
		},
		{
			name: "diamond",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"b", "c"}},
				{ID: "b", Conn: []string{"d"}},
				{ID: "c", Conn: []string{"d"}},
				{ID: "d"},
			},
		},
		{
			name: "disconnected-with-cycle",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"b"}},
				{ID: "b", Conn: []string{"a"}},
				{ID: "c"},
			},
			wantErr:     true,
			errContains: "cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDAG(tt.tasks)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestRunner(t *testing.T) {
	t.Parallel()
	t.Run("new-runner-has-engines", func(t *testing.T) {
		t.Parallel()
		r := NewRunner()
		assert.Contains(t, r.engines, "capability")
		assert.Contains(t, r.engines, "shell")
		assert.Contains(t, r.engines, "docker")
		assert.Contains(t, r.engines, "machine")
	})

	t.Run("mapper-step-only", func(t *testing.T) {
		t.Parallel()
		runner := NewRunner()
		wf := types.WorkflowMetadata{
			Name:     "mapper-only",
			Pipeline: []string{"m1", "m2"},
			Tasks: []types.WorkflowTask{
				{
					ID:     "m1",
					Action: "mapper:",
					Params: types.KV{"key_a": "value_a", "key_b": "static"},
				},
				{
					ID:     "m2",
					Action: "mapper:",
					Params: types.KV{
						"from_m1": `{{jsonpath (step "m1" "result") "key_a"}}`,
						"extra":   "from-m2",
					},
				},
			},
		}
		err := runner.Execute(context.Background(), wf, nil, "")
		require.NoError(t, err)
	})

	t.Run("mapper-chain-with-jsonpath", func(t *testing.T) {
		t.Parallel()
		runner := NewRunner()
		wf := types.WorkflowMetadata{
			Name:     "mapper-chain",
			Pipeline: []string{"produce", "transform", "final"},
			Tasks: []types.WorkflowTask{
				{
					ID:     "produce",
					Action: "mapper:",
					Params: types.KV{
						"label": "first",
						"score": float64(10),
					},
				},
				{
					ID:     "transform",
					Action: "mapper:",
					Params: types.KV{
						"name":  `{{jsonpath (step "produce" "result") "label"}}`,
						"value": `{{jsonpath (step "produce" "result") "score"}}`,
					},
				},
				{
					ID:     "final",
					Action: "mapper:",
					Params: types.KV{
						"result": `{{step "transform" "result"}}`,
					},
				},
			},
		}
		err := runner.Execute(context.Background(), wf, nil, "")
		require.NoError(t, err)
	})
}

func FuzzParseAction(f *testing.F) {
	f.Add("capability:bookmark.list")
	f.Add("docker:nginx:latest")
	f.Add("shell:echo hello")
	f.Add("")
	f.Add("mapper:")
	f.Add("capability:bookmark")
	f.Add("plain-action")

	f.Fuzz(func(t *testing.T, action string) {
		info := ParseAction(action)
		if info.IsCapability {
			assert.Equal(t, "capability", info.Type, "IsCapability=true but Type mismatch")
		}
		if info.CapType != "" && info.Operation != "" {
			expected := info.CapType + "." + info.Operation
			assert.Equal(t, expected, info.Details, "CapType+Operation != Details")
		}
	})
}

func FuzzExtractCMDSlice(f *testing.F) {
	f.Add([]byte(`"echo hello"`))
	f.Add([]byte(`["sh","-c","echo hi"]`))
	f.Add([]byte(`42`))
	f.Add([]byte(`null`))
	f.Add([]byte(`true`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var val any
		_ = sonic.Unmarshal(data, &val)
		result := extractCMDSlice(val)
		_ = result
	})
}

func FuzzValidateDAG(f *testing.F) {
	f.Add([]byte(`[{"id":"a","conn":["b"]},{"id":"b"}]`))
	f.Add([]byte(`[{"id":"a","conn":["a"]}]`))
	f.Add([]byte(`[{"id":"a"},{"id":"b"},{"id":"c"}]`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var raw []struct {
			ID   string   `json:"id"`
			Conn []string `json:"conn"`
		}
		if err := sonic.Unmarshal(data, &raw); err != nil {
			t.Skip()
		}
		tasks := make([]types.WorkflowTask, len(raw))
		for i, r := range raw {
			tasks[i] = types.WorkflowTask{ID: r.ID, Conn: r.Conn}
		}
		err := ValidateDAG(tasks)
		// Errors from unknown deps or cycles are expected, panics are not.
		_ = err
	})
}

func FuzzValidateDAGPanics(f *testing.F) {
	f.Add([]byte(`[{"id":"a","conn":["b"]},{"id":"b"}]`))
	f.Add([]byte(`[{"id":"a","conn":[]}]`))
	f.Add([]byte(`[{"id":""}]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var raw []struct {
			ID   string   `json:"id"`
			Conn []string `json:"conn"`
		}
		if err := sonic.Unmarshal(data, &raw); err != nil {
			t.Skip()
		}
		tasks := make([]types.WorkflowTask, len(raw))
		for i, r := range raw {
			tasks[i] = types.WorkflowTask{ID: r.ID, Conn: r.Conn}
		}
		// For tasks without Conn, ensure no nil panic
		err := ValidateDAG(tasks)
		_ = err
	})
}

func FuzzResultCopy(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"a":"b"}`))
	f.Add([]byte(`{"step1":"result1","step2":"result2"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var src map[string]string
		if err := sonic.Unmarshal(data, &src); err != nil {
			t.Skip()
		}
		dst := resultCopy(src)
		assert.Len(t, dst, len(src), "resultCopy length mismatch")
		for k, v := range src {
			assert.Equal(t, v, dst[k], "resultCopy[%s]", k)
		}
		if src != nil {
			assert.NotNil(t, dst, "resultCopy returned nil for non-nil source")
		}
	})
}
