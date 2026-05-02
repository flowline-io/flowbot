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
