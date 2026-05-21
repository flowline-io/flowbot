// Package template provides pipeline template rendering engine.
package template

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	txtpl "text/template"

	"github.com/bytedance/sonic"

	"github.com/tidwall/gjson"
)

// Engine renders Go text/template strings with helper functions and caching.
// Each RenderString call captures its own data via closures so concurrent
// invocations are safe without locking.
type Engine struct{}

// templateCache holds compiled templates shared across all Engine instances.
var templateCache sync.Map // string -> *txtpl.Template

// TemplateData holds the execution context for template rendering.
type TemplateData struct {
	Event map[string]any
	Steps map[string]map[string]any
	Env   map[string]string
	Input map[string]any
}

// New returns a new template Engine.
func New() *Engine {
	return &Engine{}
}

var reInput = regexp.MustCompile(`\{\{input\.(\w+)\}\}`)
var reEvent = regexp.MustCompile(`\{\{event\.(\w+)\}\}`)
var reSteps = regexp.MustCompile(`\{\{steps\.(\w+)\.(\w+)\}\}`)
var reStepLegacy = regexp.MustCompile(`\{\{(\w+)\.(id|result)\}\}`)

var pooledSonic = sonic.Config{}.Froze()

func preprocessTemplate(s string) string {
	s = reInput.ReplaceAllString(s, `{{input "$1"}}`)
	s = reEvent.ReplaceAllString(s, `{{event "$1"}}`)
	s = reSteps.ReplaceAllString(s, `{{step "$1" "$2"}}`)
	s = reStepLegacy.ReplaceAllString(s, `{{step "$1" "$2"}}`)
	return s
}

// makeFuncs returns a FuncMap whose data-dependent closures capture the
// supplied TemplateData directly, making the Engine safe for concurrent use.
func makeFuncs(data *TemplateData) txtpl.FuncMap {
	fm := txtpl.FuncMap{
		"join":           funcJoin,
		"split":          strings.Split,
		"contains":       strings.Contains,
		"default":        funcDefault,
		"json":           funcJson,
		"len":            funcLen,
		"jsonpath":       funcJsonpath,
		"jsonpathExists": funcJsonpathExists,
		"jsonpathRaw":    funcJsonpathRaw,
	}

	fm["input"] = func(field string) any {
		if data != nil && data.Input != nil {
			if v, ok := data.Input[field]; ok {
				return v
			}
		}
		return ""
	}

	fm["event"] = func(field string) any {
		if data != nil && data.Event != nil {
			if v, ok := data.Event[field]; ok {
				return v
			}
		}
		return ""
	}

	fm["step"] = func(stepName, field string) any {
		if data != nil && data.Steps != nil {
			if step, ok := data.Steps[stepName]; ok {
				if v, ok := step[field]; ok {
					return v
				}
			}
		}
		return ""
	}

	return fm
}

func funcJoin(elems any, sep string) string {
	if elems == nil {
		return ""
	}
	val := reflect.ValueOf(elems)
	if val.Kind() != reflect.Slice {
		return ""
	}
	parts := make([]string, val.Len())
	for i := range parts {
		parts[i] = fmt.Sprint(val.Index(i).Interface())
	}
	return strings.Join(parts, sep)
}

func funcDefault(def, val any) any {
	if val == nil {
		return def
	}
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.String && v.String() == "" {
		return def
	}
	return val
}

func funcJson(v any) (string, error) {
	b, err := pooledSonic.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func funcLen(v any) int {
	if v == nil {
		return 0
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return val.Len()
	}
	return 0
}

func funcJsonpath(jsonStr, path string) string {
	return gjson.Get(jsonStr, path).String()
}

func funcJsonpathExists(jsonStr, path string) bool {
	return gjson.Get(jsonStr, path).Exists()
}

func funcJsonpathRaw(jsonStr, path string) any {
	return gjson.Get(jsonStr, path).Value()
}

// RenderString renders a template string with the given TemplateData.
// Templates are cached by their preprocessed string; on cache hit the
// cached parse tree is cloned and fresh data-dependent functions
// (event, input, step) are attached via closures for the current call.
func (*Engine) RenderString(tmpl string, data *TemplateData) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	tmpl = preprocessTemplate(tmpl)

	fm := makeFuncs(data)

	var t *txtpl.Template
	if cached, ok := templateCache.Load(tmpl); ok {
		cloned, err := cached.(*txtpl.Template).Clone()
		if err != nil {
			return "", fmt.Errorf("template clone: %w", err)
		}
		t = cloned.Funcs(fm)
	} else {
		var err error
		t, err = txtpl.New("render").Funcs(fm).Parse(tmpl)
		if err != nil {
			return "", fmt.Errorf("template parse: %w", err)
		}
		templateCache.Store(tmpl, t)
	}

	tplData := map[string]any{}
	if data != nil {
		if data.Event != nil {
			tplData["Event"] = data.Event
		}
		if data.Steps != nil {
			tplData["Steps"] = data.Steps
		}
		if data.Env != nil {
			tplData["Env"] = data.Env
		}
		if data.Input != nil {
			tplData["Input"] = data.Input
		}
	}

	var buf strings.Builder
	err := t.Execute(&buf, tplData)
	if err != nil {
		return "", fmt.Errorf("template execute: %w", err)
	}

	return buf.String(), nil
}

const maxRenderDepth = 32

// Render traverses a map of parameters and renders any template strings
// found within using the given TemplateData.
func (e *Engine) Render(params map[string]any, data *TemplateData) (map[string]any, error) {
	return e.renderMap(params, data, 0)
}

func (e *Engine) renderMap(m map[string]any, data *TemplateData, depth int) (map[string]any, error) {
	if depth > maxRenderDepth {
		return nil, fmt.Errorf("max render depth %d exceeded", maxRenderDepth)
	}
	rendered := make(map[string]any, len(m))
	for key, value := range m {
		v, err := e.renderValue(value, data, depth)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key, err)
		}
		rendered[key] = v
	}
	return rendered, nil
}

func (e *Engine) renderValue(value any, data *TemplateData, depth int) (any, error) {
	switch v := value.(type) {
	case string:
		return e.RenderString(v, data)
	case map[string]any:
		return e.renderMap(v, data, depth+1)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			r, err := e.renderValue(item, data, depth+1)
			if err != nil {
				return nil, err
			}
			result[i] = r
		}
		return result, nil
	default:
		return v, nil
	}
}
