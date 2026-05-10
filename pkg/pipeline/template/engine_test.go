package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- eventFieldTest holds cases for TestRenderString_EventFields ---
type eventFieldTest struct {
	name     string
	template string
	event    map[string]any
	want     string
}

func TestRenderString_EventFields(t *testing.T) {
	tests := []eventFieldTest{
		{
			name:     "EventField",
			template: `{{event "url"}}`,
			event:    map[string]any{"url": "https://example.com", "id": "123"},
			want:     "https://example.com",
		},
		{
			name:     "EventField_OldSyntax",
			template: `{{event.url}}`,
			event:    map[string]any{"url": "https://example.com", "id": "123"},
			want:     "https://example.com",
		},
		{
			name:     "EventDotAccess",
			template: `{{.Event.url}}`,
			event:    map[string]any{"url": "https://example.com", "id": "123"},
			want:     "https://example.com",
		},
		{
			name:     "EventIndex",
			template: `{{index .Event "id"}}`,
			event:    map[string]any{"id": "123"},
			want:     "123",
		},
		{
			name:     "NoTemplate",
			template: "plain text",
			event:    nil,
			want:     "plain text",
		},
		{
			name:     "MultipleEventFields",
			template: `id={{event "id"}} url={{event "url"}} title={{event "title"}}`,
			event:    map[string]any{"id": "123", "url": "https://x.com", "title": "Test"},
			want:     "id=123 url=https://x.com title=Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			var data *TemplateData
			if tt.event != nil {
				data = &TemplateData{Event: tt.event}
			}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- stepFieldTest holds cases for TestRenderString_StepFields ---
type stepFieldTest struct {
	name     string
	template string
	steps    map[string]map[string]any
	want     string
}

func TestRenderString_StepFields(t *testing.T) {
	tests := []stepFieldTest{
		{
			name:     "StepField",
			template: `{{step "archive" "url"}}`,
			steps:    map[string]map[string]any{"archive": {"url": "https://archived.example.com", "id": "a1"}},
			want:     "https://archived.example.com",
		},
		{
			name:     "StepField_OldSyntax",
			template: `{{steps.archive.url}}`,
			steps:    map[string]map[string]any{"archive": {"url": "https://archived.example.com", "id": "a1"}},
			want:     "https://archived.example.com",
		},
		{
			name:     "StepDotAccess",
			template: `{{index .Steps.archive "url"}}`,
			steps:    map[string]map[string]any{"archive": {"url": "https://archived.example.com"}},
			want:     "https://archived.example.com",
		},
		{
			name:     "MultipleSteps",
			template: `{{step "step1" "id"}}-{{step "step2" "id"}}`,
			steps:    map[string]map[string]any{"step1": {"id": "abc"}, "step2": {"id": "def"}},
			want:     "abc-def",
		},
		{
			name:     "MissingStepField",
			template: `{{step "nonexistent" "id"}}`,
			steps:    map[string]map[string]any{},
			want:     "",
		},
		{
			name:     "StepIndexAccess",
			template: `{{index .Steps.archive "url"}}`,
			steps:    map[string]map[string]any{"archive": {"url": "https://a.example.com"}},
			want:     "https://a.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Steps: tt.steps}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- conditionalTest holds cases for TestRenderString_Conditionals ---
type conditionalTest struct {
	name     string
	template string
	event    map[string]any
	want     string
}

func TestRenderString_Conditionals(t *testing.T) {
	tests := []conditionalTest{
		{
			name:     "If",
			template: `{{if .Event.url}}has-url{{else}}no-url{{end}}`,
			event:    map[string]any{"url": "https://example.com"},
			want:     "has-url",
		},
		{
			name:     "IfEmpty",
			template: `{{if .Event.url}}has-url{{else}}no-url{{end}}`,
			event:    map[string]any{"url": ""},
			want:     "no-url",
		},
		{
			name:     "IfMissing",
			template: `{{if .Event.url}}has-url{{else}}no-url{{end}}`,
			event:    map[string]any{},
			want:     "no-url",
		},
		{
			name:     "Eq",
			template: `{{if eq .Event.status "done"}}completed{{else}}pending{{end}}`,
			event:    map[string]any{"status": "done"},
			want:     "completed",
		},
		{
			name:     "Ne",
			template: `{{if ne .Event.status "done"}}not-done{{end}}`,
			event:    map[string]any{"status": "pending"},
			want:     "not-done",
		},
		{
			name:     "And",
			template: `{{if and .Event.a .Event.b}}both{{end}}`,
			event:    map[string]any{"a": "x", "b": "y"},
			want:     "both",
		},
		{
			name:     "Or",
			template: `{{if or .Event.a .Event.b}}either{{end}}`,
			event:    map[string]any{"a": "x"},
			want:     "either",
		},
		{
			name:     "Not",
			template: `{{if not .Event.a}}missing{{end}}`,
			event:    map[string]any{},
			want:     "missing",
		},
		{
			name:     "Gt",
			template: `{{if gt .Event.count 3.0}}high{{end}}`,
			event:    map[string]any{"count": float64(5)},
			want:     "high",
		},
		{
			name:     "With",
			template: `{{with .Event.user}}{{.name}} is {{.role}}{{end}}`,
			event:    map[string]any{"user": map[string]any{"name": "Alice", "role": "admin"}},
			want:     "Alice is admin",
		},
		{
			name:     "WithEmpty",
			template: `{{with .Event.user}}found{{else}}not-found{{end}}`,
			event:    map[string]any{},
			want:     "not-found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Event: tt.event}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- loopTest holds cases for TestRenderString_Loops ---
type loopTest struct {
	name         string
	template     string
	event        map[string]any
	want         string
	wantContains []string
}

func TestRenderString_Loops(t *testing.T) {
	tests := []loopTest{
		{
			name:     "Range",
			template: `{{range $i, $v := .Event.items}}{{$i}}:{{$v}},{{end}}`,
			event:    map[string]any{"items": []any{"a", "b", "c"}},
			want:     "0:a,1:b,2:c,",
		},
		{
			name:         "RangeMap",
			template:     `{{range $k, $v := .Event.tags}}{{$k}}={{$v}};{{end}}`,
			event:        map[string]any{"tags": map[string]any{"env": "prod", "region": "us"}},
			wantContains: []string{"env=prod", "region=us"},
		},
		{
			name:     "Else",
			template: `{{range .Event.items}}x{{else}}empty{{end}}`,
			event:    map[string]any{"items": []any{}},
			want:     "empty",
		},
		{
			name:     "NestedConditionAndLoop",
			template: `{{range .Event.items}}{{if .}}{{.}},{{end}}{{end}}`,
			event:    map[string]any{"items": []any{"a", "b", ""}},
			want:     "a,b,",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Event: tt.event}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			if tt.wantContains != nil {
				for _, s := range tt.wantContains {
					assert.Contains(t, result, s)
				}
			} else {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

// --- functionTest holds cases for TestRenderString_Functions ---
type functionTest struct {
	name     string
	template string
	event    map[string]any
	want     string
}

func TestRenderString_Functions(t *testing.T) {
	tests := []functionTest{
		{
			name:     "Join",
			template: `{{join .Event.tags ", "}}`,
			event:    map[string]any{"tags": []any{"a", "b", "c"}},
			want:     "a, b, c",
		},
		{
			name:     "Split",
			template: `{{index (split .Event.csv ",") 0}}`,
			event:    map[string]any{"csv": "a,b,c"},
			want:     "a",
		},
		{
			name:     "Contains",
			template: `{{if contains .Event.title "World"}}found{{end}}`,
			event:    map[string]any{"title": "Hello World"},
			want:     "found",
		},
		{
			name:     "ContainsFalse",
			template: `{{if contains .Event.title "xyz"}}found{{else}}not-found{{end}}`,
			event:    map[string]any{"title": "Hello World"},
			want:     "not-found",
		},
		{
			name:     "Default_Nil",
			template: `{{default "fallback" .Event.missing}}`,
			event:    map[string]any{},
			want:     "fallback",
		},
		{
			name:     "Default_Empty",
			template: `{{default "anonymous" .Event.name}}`,
			event:    map[string]any{"name": ""},
			want:     "anonymous",
		},
		{
			name:     "Default_WithValue",
			template: `{{default "anonymous" .Event.name}}`,
			event:    map[string]any{"name": "Alice"},
			want:     "Alice",
		},
		{
			name:     "JSON",
			template: `{{json .Event.obj}}`,
			event:    map[string]any{"obj": map[string]any{"key": "value"}},
			want:     `{"key":"value"}`,
		},
		{
			name:     "Len_String",
			template: `{{len .Event.name}}`,
			event:    map[string]any{"name": "hello"},
			want:     "5",
		},
		{
			name:     "Len_Slice",
			template: `{{len .Event.items}}`,
			event:    map[string]any{"items": []any{1, 2, 3}},
			want:     "3",
		},
		{
			name:     "Len_Map",
			template: `{{len .Event.tags}}`,
			event:    map[string]any{"tags": map[string]any{"a": 1, "b": 2}},
			want:     "2",
		},
		{
			name:     "Len_Nil",
			template: `{{len .Event.missing}}`,
			event:    map[string]any{},
			want:     "0",
		},
		{
			name:     "printf",
			template: `{{printf "id-%s" .Event.id}}`,
			event:    map[string]any{"id": "123"},
			want:     "id-123",
		},
		{
			name:     "StringWithWhitespaceInTemplate",
			template: "Hello {{ .Event.name }}!",
			event:    map[string]any{"name": "test"},
			want:     "Hello test!",
		},
		{
			name:     "EscapedBraces",
			template: `start {{event "id"}} end`,
			event:    map[string]any{"id": "123"},
			want:     "start 123 end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Event: tt.event}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- jsonpathTest holds cases for TestRenderString_Jsonpath ---
type jsonpathTest struct {
	name     string
	template string
	event    map[string]any
	steps    map[string]map[string]any
	want     string
}

func TestRenderString_Jsonpath(t *testing.T) {
	tests := []jsonpathTest{
		{
			name:     "Simple",
			template: `{{jsonpath (json .Event.nested) "key"}}`,
			event:    map[string]any{"nested": map[string]any{"key": "hello"}},
			want:     "hello",
		},
		{
			name:     "ArrayIndex",
			template: `{{jsonpath (json .Event.items) "1"}}`,
			event:    map[string]any{"items": []any{"a", "b", "c"}},
			want:     "b",
		},
		{
			name:     "ArrayNestedAccess",
			template: `{{jsonpath (json .Event.items) "1.name"}}`,
			event: map[string]any{
				"items": []any{
					map[string]any{"name": "a"},
					map[string]any{"name": "b"},
				},
			},
			want: "b",
		},
		{
			name:     "ArrayWildcard",
			template: `{{jsonpath (json .Event.items) "#.name"}}`,
			event: map[string]any{
				"items": []any{
					map[string]any{"name": "a"},
					map[string]any{"name": "b"},
				},
			},
			want: `["a","b"]`,
		},
		{
			name:     "ArrayLength",
			template: `{{jsonpath (json .Event.items) "#"}}`,
			event:    map[string]any{"items": []any{float64(1), float64(2), float64(3)}},
			want:     "3",
		},
		{
			name:     "MissingPath",
			template: `{{jsonpath (json .Event.data) "x.y"}}`,
			event:    map[string]any{"data": map[string]any{"a": float64(1)}},
			want:     "",
		},
		{
			name:     "EmptyJSON",
			template: `{{jsonpath "" "x"}}`,
			event:    map[string]any{},
			want:     "",
		},
		{
			name:     "Exists_True",
			template: `{{if jsonpathExists (json .Event.data) "a"}}yes{{else}}no{{end}}`,
			event:    map[string]any{"data": map[string]any{"a": float64(1)}},
			want:     "yes",
		},
		{
			name:     "Exists_False",
			template: `{{if jsonpathExists (json .Event.data) "x"}}yes{{else}}no{{end}}`,
			event:    map[string]any{"data": map[string]any{"a": float64(1)}},
			want:     "no",
		},
		{
			name:     "RawObject",
			template: `{{json (jsonpathRaw (json .Event.nested) "key")}}`,
			event:    map[string]any{"nested": map[string]any{"key": "value"}},
			want:     `"value"`,
		},
		{
			name:     "RawNumber",
			template: `{{jsonpathRaw (json .Event.data) "count"}}`,
			event:    map[string]any{"data": map[string]any{"count": float64(42)}},
			want:     "42",
		},
		{
			name:     "WithJsonFunction",
			template: `{{jsonpath (json .Event.data) "nested.deep"}}`,
			event:    map[string]any{"data": map[string]any{"nested": map[string]any{"deep": "found"}}},
			want:     "found",
		},
		{
			name:     "WithStepResult",
			template: `{{jsonpath (step "src" "result") "title"}}-{{jsonpath (step "src" "result") "tags.0"}}`,
			steps: map[string]map[string]any{
				"src": {"result": `{"title":"Hello","tags":["a","b"]}`},
			},
			want: "Hello-a",
		},
		{
			name:     "FilteredArray",
			template: `{{jsonpath (json .Event.users) "#(age>28).name"}}`,
			event: map[string]any{
				"users": []any{
					map[string]any{"name": "Alice", "age": float64(30)},
					map[string]any{"name": "Bob", "age": float64(25)},
				},
			},
			want: "Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Event: tt.event, Steps: tt.steps}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- inputFieldTest holds cases for TestRenderString_InputFields ---
type inputFieldTest struct {
	name     string
	template string
	input    map[string]any
	event    map[string]any
	steps    map[string]map[string]any
	nilData  bool
	want     string
}

func TestRenderString_InputFields(t *testing.T) {
	tests := []inputFieldTest{
		{
			name:     "InputField",
			template: `{{input "url"}}`,
			input:    map[string]any{"url": "https://example.com", "title": "Test"},
			want:     "https://example.com",
		},
		{
			name:     "InputField_DotSyntax",
			template: `{{input.url}}`,
			input:    map[string]any{"url": "https://example.com"},
			want:     "https://example.com",
		},
		{
			name:     "InputField_MultipleFields",
			template: `url={{input "url"}} title={{input "title"}}`,
			input:    map[string]any{"url": "https://x.com", "title": "Hello"},
			want:     "url=https://x.com title=Hello",
		},
		{
			name:     "InputField_Missing",
			template: `{{input "missing"}}`,
			input:    map[string]any{},
			want:     "",
		},
		{
			name:     "InputField_NilInput",
			template: `{{input "any"}}`,
			input:    nil,
			want:     "",
		},
		{
			name:     "InputField_NilData",
			template: `{{input "url"}}`,
			nilData:  true,
			want:     "",
		},
		{
			name:     "InputField_WithDefault",
			template: `{{default "fallback" (input "title")}}`,
			input:    map[string]any{},
			want:     "fallback",
		},
		{
			name:     "InputField_Conditional",
			template: `{{if (input "url")}}has-input{{else}}no-input{{end}}`,
			input:    map[string]any{"url": "https://x.com"},
			want:     "has-input",
		},
		{
			name:     "InputField_ConditionalMissing",
			template: `{{if (input "url")}}has-input{{else}}no-input{{end}}`,
			input:    map[string]any{},
			want:     "no-input",
		},
		{
			name:     "InputField_CombinedWithStep",
			template: `input={{input "url"}} step={{step "save" "id"}}`,
			input:    map[string]any{"url": "https://example.com"},
			steps:    map[string]map[string]any{"save": {"id": "saved-123"}},
			want:     "input=https://example.com step=saved-123",
		},
		{
			name:     "InputField_CombinedWithEvent",
			template: `title={{input "title"}} source={{event "source"}}`,
			input:    map[string]any{"title": "My Title"},
			event:    map[string]any{"source": "web"},
			want:     "title=My Title source=web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			var data *TemplateData
			if !tt.nilData {
				data = &TemplateData{
					Input: tt.input,
					Event: tt.event,
					Steps: tt.steps,
				}
			}
			result, err := e.RenderString(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- paramsTest holds cases for TestRender_Params ---
type paramsTest struct {
	name       string
	params     map[string]any
	event      map[string]any
	steps      map[string]map[string]any
	wantErr    bool
	errMsg     string
	wantAssert func(t *testing.T, result map[string]any)
}

func TestRender_Params(t *testing.T) {
	tests := []paramsTest{
		{
			name: "ParamsWithTemplates",
			params: map[string]any{
				"entity": `{{event "id"}}`,
				"link":   `{{event "url"}}`,
			},
			event: map[string]any{"id": "123", "url": "https://x.com"},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "123", result["entity"])
				assert.Equal(t, "https://x.com", result["link"])
			},
		},
		{
			name: "ParamsNoTemplates",
			params: map[string]any{
				"key": "value",
				"num": 42,
			},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "value", result["key"])
				assert.Equal(t, 42, result["num"])
			},
		},
		{
			name: "ParamsNestedMap",
			params: map[string]any{
				"nested": map[string]any{
					"inner": `{{event "id"}}`,
				},
			},
			event: map[string]any{"id": "123"},
			wantAssert: func(t *testing.T, result map[string]any) {
				nested := result["nested"].(map[string]any)
				assert.Equal(t, "123", nested["inner"])
			},
		},
		{
			name: "ParamsStringSlice",
			params: map[string]any{
				"items": []any{`{{event "id"}}`, "static"},
			},
			event: map[string]any{"id": "eid"},
			wantAssert: func(t *testing.T, result map[string]any) {
				items := result["items"].([]any)
				assert.Equal(t, "eid", items[0])
				assert.Equal(t, "static", items[1])
			},
		},
		{
			name: "ParamsError_Propagated",
			params: map[string]any{
				"bad": "{{if .Event.x}}}",
			},
			event:   map[string]any{},
			wantErr: true,
			errMsg:  `key "bad"`,
		},
		{
			name: "ParamsStepReferences",
			params: map[string]any{
				"ref_id":  "{{steps.archive.id}}",
				"ref_url": "{{steps.archive.url}}",
			},
			steps: map[string]map[string]any{
				"archive": {"id": "archive-1", "url": "https://archived.example.com"},
			},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "archive-1", result["ref_id"])
				assert.Equal(t, "https://archived.example.com", result["ref_url"])
			},
		},
		{
			name: "ParamsConditionInParams",
			params: map[string]any{
				"action": `{{if eq .Event.status "done"}}archive{{else}}skip{{end}}`,
			},
			event: map[string]any{"status": "done"},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "archive", result["action"])
			},
		},
		{
			name: "ParamsLoopInParams",
			params: map[string]any{
				"joined": "{{range .Event.items}}{{.}}-{{end}}",
			},
			event: map[string]any{"items": []any{"a", "b", "c"}},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "a-b-c-", result["joined"])
			},
		},
		{
			name: "ParamsDefaultInParams",
			params: map[string]any{
				"name": `{{default "guest" .Event.name}}`,
			},
			event: map[string]any{},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "guest", result["name"])
			},
		},
		{
			name: "ParamsWithJsonpath",
			params: map[string]any{
				"extracted": `{{jsonpath (json .Event.items) "1.id"}}`,
			},
			event: map[string]any{
				"items": []any{
					map[string]any{"id": "x"},
					map[string]any{"id": "y"},
				},
			},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "y", result["extracted"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Event: tt.event, Steps: tt.steps}
			result, err := e.Render(tt.params, data)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			if tt.wantAssert != nil {
				tt.wantAssert(t, result)
			}
		})
	}
}

// --- paramsInputTest holds cases for TestRender_ParamsInput ---
type paramsInputTest struct {
	name       string
	params     map[string]any
	input      map[string]any
	wantErr    bool
	wantAssert func(t *testing.T, result map[string]any)
}

func TestRender_ParamsInput(t *testing.T) {
	tests := []paramsInputTest{
		{
			name: "ParamsWithInput",
			params: map[string]any{
				"link":     `{{input "url"}}`,
				"headline": `{{input "title"}}`,
				"static":   "no-template",
			},
			input: map[string]any{"url": "https://x.com", "title": "Hello"},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "https://x.com", result["link"])
				assert.Equal(t, "Hello", result["headline"])
				assert.Equal(t, "no-template", result["static"])
			},
		},
		{
			name: "ParamsWithInputDotSyntax",
			params: map[string]any{
				"link": "{{input.url}}",
			},
			input: map[string]any{"url": "https://x.com"},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "https://x.com", result["link"])
			},
		},
		{
			name: "ParamsWithInputAndCondition",
			params: map[string]any{
				"level": `{{if eq (input "env") "prod"}}high{{else}}low{{end}}`,
			},
			input: map[string]any{"env": "prod"},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "high", result["level"])
			},
		},
		{
			name: "ParamsWithInputElseBranch",
			params: map[string]any{
				"level": `{{if eq (input "env") "prod"}}high{{else}}low{{end}}`,
			},
			input: map[string]any{"env": "dev"},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "low", result["level"])
			},
		},
		{
			name: "WorkflowStyleInputParams",
			params: map[string]any{
				"url":         "{{input.url}}",
				"title":       `Read: {{input "title"}}`,
				"description": `Bookmark: {{input "url"}}`,
				"tags":        []any{"reading", "bookmark"},
			},
			input: map[string]any{
				"url":   "https://example.com",
				"title": "Read This Article",
			},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "https://example.com", result["url"])
				assert.Equal(t, "Read: Read This Article", result["title"])
				assert.Equal(t, "Bookmark: https://example.com", result["description"])
				assert.Equal(t, []any{"reading", "bookmark"}, result["tags"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			data := &TemplateData{Input: tt.input}
			result, err := e.Render(tt.params, data)
			require.NoError(t, err)
			if tt.wantAssert != nil {
				tt.wantAssert(t, result)
			}
		})
	}
}

// --- preprocessTest holds cases for TestPreprocessTemplate ---
type preprocessTest struct {
	name  string
	input string
	want  string
}

func TestPreprocessTemplate(t *testing.T) {
	tests := []preprocessTest{
		// Event
		{
			name:  "Event_Single",
			input: "{{event.url}}",
			want:  `{{event "url"}}`,
		},
		{
			name:  "Event_WithPrefixSuffix",
			input: "prefix {{event.url}} suffix",
			want:  `prefix {{event "url"}} suffix`,
		},
		{
			name:  "Event_Multiple",
			input: "{{event.url}} {{event.id}}",
			want:  `{{event "url"}} {{event "id"}}`,
		},
		// Steps
		{
			name:  "Steps_Single",
			input: "{{steps.archive.url}}",
			want:  `{{step "archive" "url"}}`,
		},
		{
			name:  "Steps_Multiple",
			input: "{{steps.s1.id}} {{steps.s2.result}}",
			want:  `{{step "s1" "id"}} {{step "s2" "result"}}`,
		},
		// StepLegacy
		{
			name:  "StepLegacy_Single",
			input: "{{step1.id}}",
			want:  `{{step "step1" "id"}}`,
		},
		{
			name:  "StepLegacy_CamelCase",
			input: "{{myStep.result}}",
			want:  `{{step "myStep" "result"}}`,
		},
		{
			name:  "StepLegacy_Multiple",
			input: "{{step1.id}} {{step2.result}}",
			want:  `{{step "step1" "id"}} {{step "step2" "result"}}`,
		},
		{
			name:  "StepLegacy_NonIdOrResult",
			input: "{{foo.bar}}",
			want:  "{{foo.bar}}",
		},
		// Mixed
		{
			name:  "Mixed",
			input: "id={{event.id}} ref={{steps.step1.url}} legacy={{s1.result}}",
			want:  `id={{event "id"}} ref={{step "step1" "url"}} legacy={{step "s1" "result"}}`,
		},
		// NoMatch
		{
			name:  "NoMatch",
			input: `{{.Event.url}} {{step "a" "b"}} {{.Env.HOME}}`,
			want:  `{{.Event.url}} {{step "a" "b"}} {{.Env.HOME}}`,
		},
		// Input
		{
			name:  "Input_Single",
			input: "{{input.url}}",
			want:  `{{input "url"}}`,
		},
		{
			name:  "Input_WithPrefixSuffix",
			input: "prefix {{input.url}} suffix",
			want:  `prefix {{input "url"}} suffix`,
		},
		{
			name:  "Input_Multiple",
			input: "{{input.url}} {{input.title}}",
			want:  `{{input "url"}} {{input "title"}}`,
		},
		{
			name:  "Input_UnderscoreKey",
			input: "{{input.user_id}}",
			want:  `{{input "user_id"}}`,
		},
		{
			name:  "Input_NoMatch_BareInput",
			input: "{{input}}",
			want:  "{{input}}",
		},
		{
			name:  "Input_NoMatch_Whitespace",
			input: "{{ input.url }}",
			want:  "{{ input.url }}",
		},
		{
			name:  "Input_DoesNotConflictWithStepLegacy_Result",
			input: "{{input.result}}",
			want:  `{{input "result"}}`,
		},
		{
			name:  "Input_DoesNotConflictWithStepLegacy_Id",
			input: "{{input.id}}",
			want:  `{{input "id"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessTemplate(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- edgeCaseTest holds cases for TestRenderString_EdgeCases ---
type edgeCaseTest struct {
	name     string
	template string
	data     *TemplateData
	want     string
	wantErr  bool
	errMsg   string
}

func TestRenderString_EdgeCases(t *testing.T) {
	tests := []edgeCaseTest{
		{
			name:     "MissingEventField_OldSyntax",
			template: "{{event.url}}",
			data:     &TemplateData{Event: map[string]any{}},
			want:     "",
		},
		{
			name:     "MissingKey_ZeroValue",
			template: "{{.Event.missing}}",
			data:     &TemplateData{Event: map[string]any{}},
			want:     "<no value>",
		},
		{
			name:     "InvalidTemplate_Error",
			template: "{{if .Event.x}}}",
			data:     nil,
			wantErr:  true,
			errMsg:   "template parse",
		},
		{
			name:     "NilData",
			template: "static text",
			data:     nil,
			want:     "static text",
		},
		{
			name:     "EmptyData",
			template: "{{.Event.x}}",
			data:     &TemplateData{},
			want:     "<no value>",
		},
		{
			name:     "EnvField",
			template: "{{.Env.HOME}}",
			data:     &TemplateData{Env: map[string]string{"HOME": "/home/user", "USER": "test"}},
			want:     "/home/user",
		},
		{
			name:     "RangeWithSplit",
			template: `{{range split .Event.csv ","}}{{.}}-{{end}}`,
			data:     &TemplateData{Event: map[string]any{"csv": "a,b,c"}},
			want:     "a-b-c-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			result, err := e.RenderString(tt.template, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// --- maxDepthTest holds cases for TestRender_MaxDepth ---
type maxDepthTest struct {
	name        string
	buildParams func() map[string]any
	wantErr     bool
	errMsg      string
	wantAssert  func(t *testing.T, result map[string]any)
}

func TestRender_MaxDepth(t *testing.T) {
	tests := []maxDepthTest{
		{
			name: "MaxDepth",
			buildParams: func() map[string]any {
				params := map[string]any{"x": map[string]any{}}
				inner := params["x"].(map[string]any)
				for i := range maxRenderDepth + 5 {
					next := map[string]any{"x": map[string]any{}}
					inner["x"] = next
					inner = next["x"].(map[string]any)
					_ = i
				}
				return params
			},
			wantErr: true,
			errMsg:  "max render depth",
		},
		{
			name: "MaxDepth_NormalParams",
			buildParams: func() map[string]any {
				return map[string]any{
					"a": "value",
					"b": map[string]any{"c": "d"},
					"e": []any{"f", "g"},
				}
			},
			wantAssert: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "value", result["a"])
				assert.Equal(t, "d", result["b"].(map[string]any)["c"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			params := tt.buildParams()
			result, err := e.Render(params, nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			if tt.wantAssert != nil {
				tt.wantAssert(t, result)
			}
		})
	}
}
