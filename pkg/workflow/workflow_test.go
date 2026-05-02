package workflow

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAction_CapabilityWithDot(t *testing.T) {
	info := ParseAction("capability:bookmark.list")
	assert.True(t, info.IsCapability)
	assert.Equal(t, "capability", info.Type)
	assert.Equal(t, "bookmark", info.CapType)
	assert.Equal(t, "list", info.Operation)
	assert.Equal(t, "bookmark.list", info.Details)
}

func TestParseAction_CapabilityNoDot(t *testing.T) {
	info := ParseAction("capability:bookmark")
	assert.True(t, info.IsCapability)
	assert.Equal(t, "capability", info.Type)
	assert.Equal(t, "", info.CapType)
	assert.Equal(t, "", info.Operation)
	assert.Equal(t, "bookmark", info.Details)
}

func TestParseAction_Docker(t *testing.T) {
	info := ParseAction("docker:nginx:latest")
	assert.False(t, info.IsCapability)
	assert.Equal(t, "docker", info.Type)
	assert.Equal(t, "nginx:latest", info.Details)
}

func TestParseAction_Shell(t *testing.T) {
	info := ParseAction("shell:echo hello")
	assert.False(t, info.IsCapability)
	assert.Equal(t, "shell", info.Type)
	assert.Equal(t, "echo hello", info.Details)
}

func TestParseAction_PlainString(t *testing.T) {
	info := ParseAction("echo")
	assert.False(t, info.IsCapability)
	assert.Equal(t, "echo", info.Type)
	assert.Equal(t, "", info.Details)
}

func TestParseAction_Empty(t *testing.T) {
	info := ParseAction("")
	assert.False(t, info.IsCapability)
	assert.Equal(t, "", info.Type)
	assert.Equal(t, "", info.Details)
}

func TestDetermineRuntimeType_Capability(t *testing.T) {
	task := &types.Task{Run: "capability:bookmark.list"}
	assert.Equal(t, "capability", DetermineRuntimeType(task))
}

func TestDetermineRuntimeType_Docker(t *testing.T) {
	task := &types.Task{Run: "", Image: "nginx:latest"}
	assert.Equal(t, "docker", DetermineRuntimeType(task))
}

func TestDetermineRuntimeType_Shell(t *testing.T) {
	task := &types.Task{Run: "echo hello", Image: ""}
	assert.Equal(t, "shell", DetermineRuntimeType(task))
}

func TestDetermineRuntimeType_ImageTakesPrecedence(t *testing.T) {
	task := &types.Task{Run: "some-run", Image: "alpine"}
	assert.Equal(t, "docker", DetermineRuntimeType(task))
}

func TestWorkflowTaskToTask_Capability(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "capability:bookmark.list",
		Params: types.KV{"url": "https://example.com"},
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, "capability:bookmark.list", task.Run)
	assert.Contains(t, task.Env, "CAPABILITY_PARAMS")
	assert.Equal(t, `{"url":"https://example.com"}`, task.Env["CAPABILITY_PARAMS"])
}

func TestWorkflowTaskToTask_CapabilityNoParams(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "capability:bookmark.list",
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.NotContains(t, task.Env, "CAPABILITY_PARAMS")
}

func TestWorkflowTaskToTask_Docker(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "docker:nginx:latest",
		Params: types.KV{"cmd": "nginx -g daemon off;"},
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", task.Image)
	assert.Equal(t, []string{"nginx -g daemon off;"}, task.CMD)
}

func TestWorkflowTaskToTask_Docker_SliceCmd(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "docker:alpine",
		Params: types.KV{"cmd": []any{"sh", "-c", "echo hi"}},
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, []string{"sh", "-c", "echo hi"}, task.CMD)
}

func TestWorkflowTaskToTask_Shell(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "shell:echo hello",
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, "echo hello", task.Run)
}

func TestWorkflowTaskToTask_Shell_WithCmdParam(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "shell:echo hello",
		Params: types.KV{"cmd": "ls -la"},
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, "ls -la", task.Run)
}

func TestWorkflowTaskToTask_Machine(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "machine:vm1",
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, "vm1", task.Run)
}

func TestWorkflowTaskToTask_Default(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "custom-action",
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	assert.Equal(t, "custom-action", task.Run)
}

func TestParseYAML_Valid(t *testing.T) {
	data := []byte(`
name: test-workflow
pipeline:
  - step1
tasks:
  - id: step1
    action: shell:echo hello
`)
	wf, err := ParseYAML(data)
	require.NoError(t, err)
	assert.Equal(t, "test-workflow", wf.Name)
	assert.Equal(t, []string{"step1"}, wf.Pipeline)
	require.Len(t, wf.Tasks, 1)
	assert.Equal(t, "step1", wf.Tasks[0].ID)
	assert.Equal(t, "shell:echo hello", wf.Tasks[0].Action)
}

func TestParseYAML_MissingName(t *testing.T) {
	data := []byte(`
pipeline:
  - step1
tasks:
  - id: step1
    action: echo
`)
	_, err := ParseYAML(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestParseYAML_EmptyPipeline(t *testing.T) {
	data := []byte(`
name: test
pipeline: []
tasks:
  - id: step1
    action: echo
`)
	_, err := ParseYAML(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline is required")
}

func TestParseYAML_EmptyTasks(t *testing.T) {
	data := []byte(`
name: test
pipeline:
  - step1
tasks: []
`)
	_, err := ParseYAML(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tasks are required")
}

func TestParseYAML_InvalidYAML(t *testing.T) {
	_, err := ParseYAML([]byte(`{{{invalid`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse workflow")
}

func TestResolveParams_SimpleReplacement(t *testing.T) {
	params := types.KV{"ref": "{{step1.id}}"}
	results := map[string]string{"step1": "abc123"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "abc123", resolved["ref"])
}

func TestResolveParams_NoMatch(t *testing.T) {
	params := types.KV{"ref": "hello world"}
	results := map[string]string{"step1": "abc"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "hello world", resolved["ref"])
}

func TestResolveParams_NonStringValue(t *testing.T) {
	params := types.KV{"count": 42}
	results := map[string]string{"step1": "abc"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, 42, resolved["count"])
}

func TestResolveParams_MultipleKeys(t *testing.T) {
	params := types.KV{"a": "{{step1.id}}", "b": "{{step2.id}}"}
	results := map[string]string{"step1": "r1", "step2": "r2"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "r1", resolved["a"])
	assert.Equal(t, "r2", resolved["b"])
}

func TestResolveParams_ConditionInParams(t *testing.T) {
	params := types.KV{
		"action": "{{if eq (step \"step1\" \"result\") \"success\"}}proceed{{else}}retry{{end}}",
	}
	results := map[string]string{"step1": "success"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "proceed", resolved["action"])
}

func TestResolveParams_ConditionElse(t *testing.T) {
	params := types.KV{
		"action": "{{if eq (step \"step1\" \"result\") \"success\"}}proceed{{else}}retry{{end}}",
	}
	results := map[string]string{"step1": "failed"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "retry", resolved["action"])
}

func TestResolveParams_OldSyntaxStepResult(t *testing.T) {
	params := types.KV{"output": "{{steps.step1.result}}"}
	results := map[string]string{"step1": "my-output"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "my-output", resolved["output"])
}

func TestResolveParams_NewSyntaxStep(t *testing.T) {
	params := types.KV{"output": "{{step \"step1\" \"id\"}}"}
	results := map[string]string{"step1": "id-value"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "id-value", resolved["output"])
}

func TestResolveParams_DefaultWhenMissing(t *testing.T) {
	params := types.KV{
		"label": "{{default \"no-result\" .Steps.noexist.id}}",
	}
	results := map[string]string{}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "no-result", resolved["label"])
}

func TestResolveParams_JoinStepResults(t *testing.T) {
	params := types.KV{
		"out": "{{step \"s1\" \"result\"}}|{{step \"s2\" \"result\"}}",
	}
	results := map[string]string{"s1": "a", "s2": "b"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "a|b", resolved["out"])
}

func TestResolveParams_InvalidTemplate(t *testing.T) {
	params := types.KV{"bad": "{{if xxx}}"}
	results := map[string]string{}
	_, err := resolveParams(params, results)
	require.Error(t, err)
}

func TestResolveParams_ContainsCheck(t *testing.T) {
	params := types.KV{
		"match": "{{if contains (step \"step1\" \"result\") \"ok\"}}yes{{else}}no{{end}}",
	}
	results := map[string]string{"step1": "all-ok-done"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "yes", resolved["match"])
}

func TestResolveParams_LoopOverSteps(t *testing.T) {
	params := types.KV{
		"all": "{{range $k, $v := .Steps}}{{$k}}={{index $v \"id\"}};{{end}}",
	}
	results := map[string]string{"a": "r1", "b": "r2"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Contains(t, resolved["all"], "a=r1")
	assert.Contains(t, resolved["all"], "b=r2")
}

func TestResolveParams_NestedMapValue(t *testing.T) {
	params := types.KV{
		"inner": map[string]any{
			"ref": "{{step \"step1\" \"id\"}}",
		},
	}
	results := map[string]string{"step1": "nested-id"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	inner := resolved["inner"].(map[string]any)
	assert.Equal(t, "nested-id", inner["ref"])
}

func TestResolveParams_StringSliceValue(t *testing.T) {
	params := types.KV{
		"items": []any{"{{step \"a\" \"result\"}}", "{{step \"b\" \"result\"}}"},
	}
	results := map[string]string{"a": "x", "b": "y"}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	items := resolved["items"].([]any)
	assert.Equal(t, "x", items[0])
	assert.Equal(t, "y", items[1])
}

func TestResolveParams_EmptyResults(t *testing.T) {
	params := types.KV{"ref": "{{step \"nonexist\" \"id\"}}"}
	results := map[string]string{}
	resolved, err := resolveParams(params, results)
	require.NoError(t, err)
	assert.Equal(t, "", resolved["ref"])
}

func TestNewRunner_HasEngines(t *testing.T) {
	r := NewRunner()
	assert.Contains(t, r.engines, "capability")
	assert.Contains(t, r.engines, "shell")
	assert.Contains(t, r.engines, "docker")
	assert.Contains(t, r.engines, "machine")
}

func TestWorkflowTaskToTask_MarshalError(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "capability:bookmark.list",
		Params: types.KV{"ch": make(chan int)},
	}
	_, err := WorkflowTaskToTask(wt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal params")
}

func TestValidateDAG_NoCycle(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"b"}},
		{ID: "b", Conn: []string{"c"}},
		{ID: "c"},
	}
	err := ValidateDAG(tasks)
	assert.NoError(t, err)
}

func TestValidateDAG_DirectCycle(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"b"}},
		{ID: "b", Conn: []string{"a"}},
	}
	err := ValidateDAG(tasks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestValidateDAG_IndirectCycle(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"b"}},
		{ID: "b", Conn: []string{"c"}},
		{ID: "c", Conn: []string{"a"}},
	}
	err := ValidateDAG(tasks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestValidateDAG_SelfCycle(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"a"}},
	}
	err := ValidateDAG(tasks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestValidateDAG_EmptyConn(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}
	err := ValidateDAG(tasks)
	assert.NoError(t, err)
}

func TestValidateDAG_UnknownDependency(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"nonexistent"}},
	}
	err := ValidateDAG(tasks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "references unknown dependency")
}

func TestValidateDAG_MultipleRoots(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"c"}},
		{ID: "b", Conn: []string{"c"}},
		{ID: "c"},
	}
	err := ValidateDAG(tasks)
	assert.NoError(t, err)
}

func TestValidateDAG_Diamond(t *testing.T) {
	tasks := []types.WorkflowTask{
		{ID: "a", Conn: []string{"b", "c"}},
		{ID: "b", Conn: []string{"d"}},
		{ID: "c", Conn: []string{"d"}},
		{ID: "d"},
	}
	err := ValidateDAG(tasks)
	assert.NoError(t, err)
}

func TestWorkflowTaskToTask_SliceCmdMixedTypes(t *testing.T) {
	wt := types.WorkflowTask{
		ID:     "step1",
		Action: "docker:alpine",
		Params: types.KV{"cmd": []any{"echo", "hello"}},
	}
	task, err := WorkflowTaskToTask(wt)
	require.NoError(t, err)
	// json.Unmarshal converts numbers to float64, so test with string slice
	assert.Equal(t, []string{"echo", "hello"}, task.CMD)
}
