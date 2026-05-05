package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderString_NoTemplate(t *testing.T) {
	e := New()
	result, err := e.RenderString("plain text", nil)
	require.NoError(t, err)
	assert.Equal(t, "plain text", result)
}

func TestRenderString_EventField(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"url": "https://example.com", "id": "123"},
	}
	result, err := e.RenderString("{{event \"url\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result)
}

func TestRenderString_EventField_OldSyntax(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"url": "https://example.com", "id": "123"},
	}
	result, err := e.RenderString("{{event.url}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result)
}

func TestRenderString_EventDotAccess(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"url": "https://example.com", "id": "123"},
	}
	result, err := e.RenderString("{{.Event.url}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result)
}

func TestRenderString_EventIndex(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "123"},
	}
	result, err := e.RenderString("{{index .Event \"id\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "123", result)
}

func TestRenderString_StepField(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"archive": {"url": "https://archived.example.com", "id": "a1"},
		},
	}
	result, err := e.RenderString("{{step \"archive\" \"url\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://archived.example.com", result)
}

func TestRenderString_StepField_OldSyntax(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"archive": {"url": "https://archived.example.com", "id": "a1"},
		},
	}
	result, err := e.RenderString("{{steps.archive.url}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://archived.example.com", result)
}

func TestRenderString_StepDotAccess(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"archive": {"url": "https://archived.example.com"},
		},
	}
	result, err := e.RenderString("{{index .Steps.archive \"url\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://archived.example.com", result)
}

func TestRenderString_EnvField(t *testing.T) {
	e := New()
	data := &TemplateData{
		Env: map[string]string{"HOME": "/home/user", "USER": "test"},
	}
	result, err := e.RenderString("{{.Env.HOME}}", data)
	require.NoError(t, err)
	assert.Equal(t, "/home/user", result)
}

func TestRenderString_Condition_If(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"url": "https://example.com"},
	}
	result, err := e.RenderString("{{if .Event.url}}has-url{{else}}no-url{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "has-url", result)
}

func TestRenderString_Condition_IfEmpty(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"url": ""},
	}
	result, err := e.RenderString("{{if .Event.url}}has-url{{else}}no-url{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "no-url", result)
}

func TestRenderString_Condition_IfMissing(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// .Event.url returns zero value (nil) when key is missing, which is falsy in conditionals
	result, err := e.RenderString("{{if .Event.url}}has-url{{else}}no-url{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "no-url", result)
}

func TestRenderString_Condition_Eq(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"status": "done"},
	}
	result, err := e.RenderString("{{if eq .Event.status \"done\"}}completed{{else}}pending{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "completed", result)
}

func TestRenderString_Condition_Ne(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"status": "pending"},
	}
	result, err := e.RenderString("{{if ne .Event.status \"done\"}}not-done{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "not-done", result)
}

func TestRenderString_Condition_And(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"a": "x", "b": "y"},
	}
	result, err := e.RenderString("{{if and .Event.a .Event.b}}both{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "both", result)
}

func TestRenderString_Condition_Or(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"a": "x"},
	}
	result, err := e.RenderString("{{if or .Event.a .Event.b}}either{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "either", result)
}

func TestRenderString_Condition_Not(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// .Event.a returns zero value (nil) when key is missing; nil is falsy, so not-nil is truthy
	result, err := e.RenderString("{{if not .Event.a}}missing{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "missing", result)
}

func TestRenderString_Condition_Gt(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"count": float64(5)},
	}
	result, err := e.RenderString("{{if gt .Event.count 3.0}}high{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "high", result)
}

func TestRenderString_Loop_Range(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{"a", "b", "c"},
		},
	}
	result, err := e.RenderString("{{range $i, $v := .Event.items}}{{$i}}:{{$v}},{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "0:a,1:b,2:c,", result)
}

func TestRenderString_Loop_RangeMap(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"tags": map[string]any{"env": "prod", "region": "us"},
		},
	}
	result, err := e.RenderString("{{range $k, $v := .Event.tags}}{{$k}}={{$v}};{{end}}", data)
	require.NoError(t, err)
	assert.Contains(t, result, "env=prod")
	assert.Contains(t, result, "region=us")
}

func TestRenderString_Loop_Else(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{},
		},
	}
	result, err := e.RenderString("{{range .Event.items}}x{{else}}empty{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "empty", result)
}

func TestRenderString_Join(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"tags": []any{"a", "b", "c"},
		},
	}
	result, err := e.RenderString("{{join .Event.tags \", \"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "a, b, c", result)
}

func TestRenderString_Split(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"csv": "a,b,c",
		},
	}
	result, err := e.RenderString("{{index (split .Event.csv \",\") 0}}", data)
	require.NoError(t, err)
	assert.Equal(t, "a", result)
}

func TestRenderString_Contains(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"title": "Hello World",
		},
	}
	result, err := e.RenderString("{{if contains .Event.title \"World\"}}found{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "found", result)
}

func TestRenderString_ContainsFalse(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"title": "Hello World",
		},
	}
	result, err := e.RenderString("{{if contains .Event.title \"xyz\"}}found{{else}}not-found{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "not-found", result)
}

func TestRenderString_Default_Nil(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// .Event.missing returns nil for missing key; default returns the fallback for nil
	result, err := e.RenderString("{{default \"fallback\" .Event.missing}}", data)
	require.NoError(t, err)
	assert.Equal(t, "fallback", result)
}

func TestRenderString_Default_EmptyString(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"name": ""},
	}
	result, err := e.RenderString("{{default \"anonymous\" .Event.name}}", data)
	require.NoError(t, err)
	assert.Equal(t, "anonymous", result)
}

func TestRenderString_Default_WithValue(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"name": "Alice"},
	}
	result, err := e.RenderString("{{default \"anonymous\" .Event.name}}", data)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestRenderString_JSON(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"obj": map[string]any{"key": "value"},
		},
	}
	result, err := e.RenderString("{{json .Event.obj}}", data)
	require.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, result)
}

func TestRenderString_Len_String(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"name": "hello"},
	}
	result, err := e.RenderString("{{len .Event.name}}", data)
	require.NoError(t, err)
	assert.Equal(t, "5", result)
}

func TestRenderString_Len_Slice(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"items": []any{1, 2, 3}},
	}
	result, err := e.RenderString("{{len .Event.items}}", data)
	require.NoError(t, err)
	assert.Equal(t, "3", result)
}

func TestRenderString_Len_Map(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"tags": map[string]any{"a": 1, "b": 2}},
	}
	result, err := e.RenderString("{{len .Event.tags}}", data)
	require.NoError(t, err)
	assert.Equal(t, "2", result)
}

func TestRenderString_Len_Nil(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// .Event.missing returns nil for missing key; len handles nil -> 0
	result, err := e.RenderString("{{len .Event.missing}}", data)
	require.NoError(t, err)
	assert.Equal(t, "0", result)
}

func TestRenderString_NestedConditionAndLoop(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{"a", "b", ""},
		},
	}
	result, err := e.RenderString("{{range .Event.items}}{{if .}}{{.}},{{end}}{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "a,b,", result)
}

func TestRenderString_MultipleEventFields(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "123", "url": "https://x.com", "title": "Test"},
	}
	result, err := e.RenderString("id={{event \"id\"}} url={{event \"url\"}} title={{event \"title\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "id=123 url=https://x.com title=Test", result)
}

func TestRenderString_MultipleSteps(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"step1": {"id": "abc"},
			"step2": {"id": "def"},
		},
	}
	result, err := e.RenderString("{{step \"step1\" \"id\"}}-{{step \"step2\" \"id\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "abc-def", result)
}

func TestRenderString_MissingEventField_OldSyntax(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	result, err := e.RenderString("{{event.url}}", data)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_MissingKey_ZeroValue(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// Dot-access on missing key renders as "<no value>" in text/template.
	// Pipeline authors should use {{event "field"}} for safe rendering.
	result, err := e.RenderString("{{.Event.missing}}", data)
	require.NoError(t, err)
	assert.Equal(t, "<no value>", result)
}

func TestRenderString_MissingStepField(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{},
	}
	result, err := e.RenderString("{{step \"nonexistent\" \"id\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_InvalidTemplate_Error(t *testing.T) {
	e := New()
	_, err := e.RenderString("{{if .Event.x}}}", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template parse")
}

func TestRenderString_NilData(t *testing.T) {
	e := New()
	result, err := e.RenderString("static text", nil)
	require.NoError(t, err)
	assert.Equal(t, "static text", result)
}

func TestRenderString_EmptyData(t *testing.T) {
	e := New()
	data := &TemplateData{}
	// RenderString passes nil Event map, so .Event is nil; accessing .Event.x renders zero value
	result, err := e.RenderString("{{.Event.x}}", data)
	require.NoError(t, err)
	assert.Equal(t, "<no value>", result)
}

func TestRenderString_StepIndexAccess(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"archive": {"url": "https://a.example.com"},
		},
	}
	result, err := e.RenderString("{{index .Steps.archive \"url\"}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://a.example.com", result)
}

func TestRenderString_With_Semantic(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"user": map[string]any{"name": "Alice", "role": "admin"},
		},
	}
	result, err := e.RenderString("{{with .Event.user}}{{.name}} is {{.role}}{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "Alice is admin", result)
}

func TestRenderString_With_Empty(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// .Event.user returns nil for missing key; with redirects to else branch for nil
	result, err := e.RenderString("{{with .Event.user}}found{{else}}not-found{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "not-found", result)
}

func TestRenderString_printf(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "123"},
	}
	result, err := e.RenderString("{{printf \"id-%s\" .Event.id}}", data)
	require.NoError(t, err)
	assert.Equal(t, "id-123", result)
}

func TestRender_ParamsWithTemplates(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "123", "url": "https://x.com"},
	}
	params := map[string]any{
		"entity": "{{event \"id\"}}",
		"link":   "{{event \"url\"}}",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "123", result["entity"])
	assert.Equal(t, "https://x.com", result["link"])
}

func TestRender_ParamsNoTemplates(t *testing.T) {
	e := New()
	params := map[string]any{"key": "value", "num": 42}
	result, err := e.Render(params, nil)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
	assert.Equal(t, 42, result["num"])
}

func TestRender_ParamsNestedMap(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "123"},
	}
	params := map[string]any{
		"nested": map[string]any{
			"inner": "{{event \"id\"}}",
		},
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	nested := result["nested"].(map[string]any)
	assert.Equal(t, "123", nested["inner"])
}

func TestRender_ParamsStringSlice(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "eid"},
	}
	params := map[string]any{
		"items": []any{"{{event \"id\"}}", "static"},
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	items := result["items"].([]any)
	assert.Equal(t, "eid", items[0])
	assert.Equal(t, "static", items[1])
}

func TestRender_ParamsError_Propagated(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	params := map[string]any{
		"bad": "{{if .Event.x}}}",
	}
	_, err := e.Render(params, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key \"bad\"")
}

func TestRender_ParamsStepReferences(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"archive": {"id": "archive-1", "url": "https://archived.example.com"},
		},
	}
	params := map[string]any{
		"ref_id":  "{{steps.archive.id}}",
		"ref_url": "{{steps.archive.url}}",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "archive-1", result["ref_id"])
	assert.Equal(t, "https://archived.example.com", result["ref_url"])
}

func TestRender_ParamsConditionInParams(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"status": "done"},
	}
	params := map[string]any{
		"action": "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "archive", result["action"])
}

func TestRender_ParamsLoopInParams(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"items": []any{"a", "b", "c"}},
	}
	params := map[string]any{
		"joined": "{{range .Event.items}}{{.}}-{{end}}",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "a-b-c-", result["joined"])
}

func TestRender_ParamsDefaultInParams(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{},
	}
	// .Event.name returns nil for missing key; default returns fallback for nil
	params := map[string]any{
		"name": "{{default \"guest\" .Event.name}}",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "guest", result["name"])
}

func TestPreprocessTemplate_Event(t *testing.T) {
	result := preprocessTemplate("{{event.url}}")
	assert.Equal(t, `{{event "url"}}`, result)

	result = preprocessTemplate("prefix {{event.url}} suffix")
	assert.Equal(t, `prefix {{event "url"}} suffix`, result)

	result = preprocessTemplate("{{event.url}} {{event.id}}")
	assert.Equal(t, `{{event "url"}} {{event "id"}}`, result)
}

func TestPreprocessTemplate_Steps(t *testing.T) {
	result := preprocessTemplate("{{steps.archive.url}}")
	assert.Equal(t, `{{step "archive" "url"}}`, result)

	result = preprocessTemplate("{{steps.s1.id}} {{steps.s2.result}}")
	assert.Equal(t, `{{step "s1" "id"}} {{step "s2" "result"}}`, result)
}

func TestPreprocessTemplate_StepLegacy(t *testing.T) {
	result := preprocessTemplate("{{step1.id}}")
	assert.Equal(t, `{{step "step1" "id"}}`, result)

	result = preprocessTemplate("{{myStep.result}}")
	assert.Equal(t, `{{step "myStep" "result"}}`, result)

	result = preprocessTemplate("{{step1.id}} {{step2.result}}")
	assert.Equal(t, `{{step "step1" "id"}} {{step "step2" "result"}}`, result)

	// Only matches .id or .result suffixes
	result = preprocessTemplate("{{foo.bar}}")
	assert.Equal(t, "{{foo.bar}}", result)
}

func TestPreprocessTemplate_Mixed(t *testing.T) {
	result := preprocessTemplate("id={{event.id}} ref={{steps.step1.url}} legacy={{s1.result}}")
	assert.Equal(t, `id={{event "id"}} ref={{step "step1" "url"}} legacy={{step "s1" "result"}}`, result)
}

func TestPreprocessTemplate_NoMatch(t *testing.T) {
	input := "{{.Event.url}} {{step \"a\" \"b\"}} {{.Env.HOME}}"
	result := preprocessTemplate(input)
	assert.Equal(t, input, result)
}

func TestRenderString_StringWithWhitespaceInTemplate(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"name": "test"},
	}
	result, err := e.RenderString("Hello {{ .Event.name }}!", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello test!", result)
}

func TestRender_MaxDepth(t *testing.T) {
	e := New()
	// Build a deeply nested map
	params := map[string]any{"x": map[string]any{}}
	inner := params["x"].(map[string]any)
	for i := range maxRenderDepth + 5 {
		next := map[string]any{"x": map[string]any{}}
		inner["x"] = next
		inner = next["x"].(map[string]any)
		_ = i
	}
	_, err := e.Render(params, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max render depth")
}

func TestRender_MaxDepth_NormalParams(t *testing.T) {
	e := New()
	params := map[string]any{
		"a": "value",
		"b": map[string]any{"c": "d"},
		"e": []any{"f", "g"},
	}
	result, err := e.Render(params, nil)
	require.NoError(t, err)
	assert.Equal(t, "value", result["a"])
	assert.Equal(t, "d", result["b"].(map[string]any)["c"])
}

func TestRenderString_RangeWithSplit(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"csv": "a,b,c"},
	}
	result, err := e.RenderString("{{range split .Event.csv \",\"}}{{.}}-{{end}}", data)
	require.NoError(t, err)
	assert.Equal(t, "a-b-c-", result)
}

func TestRenderString_EscapedBraces(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{"id": "123"},
	}
	result, err := e.RenderString("start {{event \"id\"}} end", data)
	require.NoError(t, err)
	assert.Equal(t, "start 123 end", result)
}

func TestRenderString_Jsonpath_Simple(t *testing.T) {
	e := New()
	// Use the json function to produce a JSON string, avoiding inline quote escaping issues.
	data := &TemplateData{
		Event: map[string]any{
			"nested": map[string]any{"key": "hello"},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.nested) "key"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestRenderString_Jsonpath_ArrayIndex(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{"a", "b", "c"},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.items) "1"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "b", result)
}

func TestRenderString_Jsonpath_ArrayNestedAccess(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{
				map[string]any{"name": "a"},
				map[string]any{"name": "b"},
			},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.items) "1.name"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "b", result)
}

func TestRenderString_Jsonpath_ArrayWildcard(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{
				map[string]any{"name": "a"},
				map[string]any{"name": "b"},
			},
		},
	}
	// #.name extracts all array elements' name fields as a JSON array string.
	result, err := e.RenderString(`{{jsonpath (json .Event.items) "#.name"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, `["a","b"]`, result)
}

func TestRenderString_Jsonpath_ArrayLength(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{float64(1), float64(2), float64(3)},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.items) "#"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "3", result)
}

func TestRenderString_Jsonpath_MissingPath(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"data": map[string]any{"a": float64(1)},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.data) "x.y"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_JsonpathEmptyJSON(t *testing.T) {
	e := New()
	result, err := e.RenderString(`{{jsonpath "" "x"}}`, &TemplateData{})
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_JsonpathExists_True(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"data": map[string]any{"a": float64(1)},
		},
	}
	result, err := e.RenderString(`{{if jsonpathExists (json .Event.data) "a"}}yes{{else}}no{{end}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "yes", result)
}

func TestRenderString_JsonpathExists_False(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"data": map[string]any{"a": float64(1)},
		},
	}
	result, err := e.RenderString(`{{if jsonpathExists (json .Event.data) "x"}}yes{{else}}no{{end}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "no", result)
}

func TestRenderString_JsonpathRaw_Object(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"nested": map[string]any{"key": "value"},
		},
	}
	result, err := e.RenderString(`{{json (jsonpathRaw (json .Event.nested) "key")}}`, data)
	require.NoError(t, err)
	assert.Equal(t, `"value"`, result)
}

func TestRenderString_JsonpathRaw_Number(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"data": map[string]any{"count": float64(42)},
		},
	}
	// jsonpathRaw returns the raw gjson.Value() (float64), rendered as "42" by text/template.
	result, err := e.RenderString(`{{jsonpathRaw (json .Event.data) "count"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "42", result)
}

func TestRenderString_JsonpathWithJsonFunction(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"data": map[string]any{"nested": map[string]any{"deep": "found"}},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.data) "nested.deep"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "found", result)
}

func TestRenderString_JsonpathWithStepResult(t *testing.T) {
	e := New()
	data := &TemplateData{
		Steps: map[string]map[string]any{
			"src": {"result": `{"title":"Hello","tags":["a","b"]}`},
		},
	}
	result, err := e.RenderString(`{{jsonpath (step "src" "result") "title"}}-{{jsonpath (step "src" "result") "tags.0"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "Hello-a", result)
}

func TestRenderString_JsonpathFilteredArray(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"users": []any{
				map[string]any{"name": "Alice", "age": float64(30)},
				map[string]any{"name": "Bob", "age": float64(25)},
			},
		},
	}
	result, err := e.RenderString(`{{jsonpath (json .Event.users) "#(age>28).name"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

// ---- input function tests ----

func TestRenderString_InputField(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://example.com", "title": "Test"},
	}
	result, err := e.RenderString(`{{input "url"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result)
}

func TestRenderString_InputField_DotSyntax(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://example.com"},
	}
	result, err := e.RenderString("{{input.url}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result)
}

func TestRenderString_InputField_MultipleFields(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://x.com", "title": "Hello"},
	}
	result, err := e.RenderString(`url={{input "url"}} title={{input "title"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "url=https://x.com title=Hello", result)
}

func TestRenderString_InputField_Missing(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{},
	}
	result, err := e.RenderString(`{{input "missing"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_InputField_NilInput(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: nil,
	}
	result, err := e.RenderString(`{{input "any"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_InputField_NilData(t *testing.T) {
	e := New()
	result, err := e.RenderString(`{{input "url"}}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestRenderString_InputField_WithDefault(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{},
	}
	result, err := e.RenderString(`{{default "fallback" (input "title")}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "fallback", result)
}

func TestRenderString_InputField_Conditional(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://x.com"},
	}
	result, err := e.RenderString(`{{if (input "url")}}has-input{{else}}no-input{{end}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "has-input", result)
}

func TestRenderString_InputField_ConditionalMissing(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{},
	}
	result, err := e.RenderString(`{{if (input "url")}}has-input{{else}}no-input{{end}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "no-input", result)
}

func TestRenderString_InputField_CombinedWithStep(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://example.com"},
		Steps: map[string]map[string]any{
			"save": {"id": "saved-123"},
		},
	}
	result, err := e.RenderString(`input={{input "url"}} step={{step "save" "id"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "input=https://example.com step=saved-123", result)
}

func TestRenderString_InputField_CombinedWithEvent(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"title": "My Title"},
		Event: map[string]any{"source": "web"},
	}
	result, err := e.RenderString(`title={{input "title"}} source={{event "source"}}`, data)
	require.NoError(t, err)
	assert.Equal(t, "title=My Title source=web", result)
}

func TestRender_ParamsWithInput(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://x.com", "title": "Hello"},
	}
	params := map[string]any{
		"link":      `{{input "url"}}`,
		"headline":  `{{input "title"}}`,
		"static":    "no-template",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "https://x.com", result["link"])
	assert.Equal(t, "Hello", result["headline"])
	assert.Equal(t, "no-template", result["static"])
}

func TestRender_ParamsWithInputDotSyntax(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"url": "https://x.com"},
	}
	params := map[string]any{
		"link": "{{input.url}}",
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "https://x.com", result["link"])
}

func TestRender_ParamsWithInputAndCondition(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"env": "prod"},
	}
	params := map[string]any{
		"level": `{{if eq (input "env") "prod"}}high{{else}}low{{end}}`,
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "high", result["level"])
}

func TestRender_ParamsWithInputElseBranch(t *testing.T) {
	e := New()
	data := &TemplateData{
		Input: map[string]any{"env": "dev"},
	}
	params := map[string]any{
		"level": `{{if eq (input "env") "prod"}}high{{else}}low{{end}}`,
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "low", result["level"])
}

// ---- preprocess input.X ----

func TestPreprocessTemplate_Input(t *testing.T) {
	result := preprocessTemplate("{{input.url}}")
	assert.Equal(t, `{{input "url"}}`, result)

	result = preprocessTemplate("prefix {{input.url}} suffix")
	assert.Equal(t, `prefix {{input "url"}} suffix`, result)

	result = preprocessTemplate("{{input.url}} {{input.title}}")
	assert.Equal(t, `{{input "url"}} {{input "title"}}`, result)
}

func TestPreprocessTemplate_InputUnderscoreKey(t *testing.T) {
	// \w+ matches word characters including underscore
	result := preprocessTemplate("{{input.user_id}}")
	assert.Equal(t, `{{input "user_id"}}`, result)
}

func TestPreprocessTemplate_InputNoMatch(t *testing.T) {
	// Only transforms {{input.X}} where there is a dot. {{input}} is left as-is.
	result := preprocessTemplate("{{input}}")
	assert.Equal(t, "{{input}}", result)

	// Input with whitespace is not matched
	result = preprocessTemplate("{{ input.url }}")
	assert.Equal(t, "{{ input.url }}", result)
}

func TestPreprocessTemplate_InputDoesNotConflictWithStepLegacy(t *testing.T) {
	// input.result matches the legacy step pattern but input is handled first
	result := preprocessTemplate("{{input.result}}")
	assert.Equal(t, `{{input "result"}}`, result)

	// input.id should also be handled as input, not step
	result = preprocessTemplate("{{input.id}}")
	assert.Equal(t, `{{input "id"}}`, result)
}

// ---- workflow-style end-to-end input test ----

func TestRender_WorkflowStyleInputParams(t *testing.T) {
	e := New()
	// Simulates the save_and_track.yaml workflow: params with input.url and input.title
	data := &TemplateData{
		Input: map[string]any{
			"url":   "https://example.com",
			"title": "Read This Article",
		},
	}
	params := map[string]any{
		"url":         "{{input.url}}",
		"title":       `Read: {{input "title"}}`,
		"description": `Bookmark: {{input "url"}}`,
		"tags":        []any{"reading", "bookmark"},
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result["url"])
	assert.Equal(t, "Read: Read This Article", result["title"])
	assert.Equal(t, "Bookmark: https://example.com", result["description"])
	assert.Equal(t, []any{"reading", "bookmark"}, result["tags"])
}

func TestRender_ParamsWithJsonpath(t *testing.T) {
	e := New()
	data := &TemplateData{
		Event: map[string]any{
			"items": []any{
				map[string]any{"id": "x"},
				map[string]any{"id": "y"},
			},
		},
	}
	params := map[string]any{
		"extracted": `{{jsonpath (json .Event.items) "1.id"}}`,
	}
	result, err := e.Render(params, data)
	require.NoError(t, err)
	assert.Equal(t, "y", result["extracted"])
}
