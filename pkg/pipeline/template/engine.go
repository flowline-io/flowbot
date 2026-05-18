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
type Engine struct {
	cache sync.Map // string -> *txtpl.Template
	mu    sync.Mutex
	data  *TemplateData // current execution data, swapped per call
}

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

func preprocessTemplate(s string) string {
	s = reInput.ReplaceAllString(s, `{{input "$1"}}`)
	s = reEvent.ReplaceAllString(s, `{{event "$1"}}`)
	s = reSteps.ReplaceAllString(s, `{{step "$1" "$2"}}`)
	s = reStepLegacy.ReplaceAllString(s, `{{step "$1" "$2"}}`)
	return s
}

// funcs returns a FuncMap whose data-dependent closures read from e.data,
// which is set per execution. This ensures cached templates always use
// current data rather than stale pointers from a previous parse.
// The closures read e.data without locking because they are only called
// during template Execute, which runs inside RenderString's locked section.
func (e *Engine) funcs() txtpl.FuncMap {
	return txtpl.FuncMap{
		"input": func(field string) any {
			data := e.data
			if data != nil && data.Input != nil {
				if v, ok := data.Input[field]; ok {
					return v
				}
			}
			return ""
		},
		"event": func(field string) any {
			data := e.data
			if data != nil && data.Event != nil {
				if v, ok := data.Event[field]; ok {
					return v
				}
			}
			return ""
		},
		"step": func(stepName, field string) any {
			data := e.data
			if data != nil && data.Steps != nil {
				if step, ok := data.Steps[stepName]; ok {
					if v, ok := step[field]; ok {
						return v
					}
				}
			}
			return ""
		},
		"join": func(elems any, sep string) string {
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
		},
		"split":    strings.Split,
		"contains": strings.Contains,
		"default": func(def, val any) any {
			if val == nil {
				return def
			}
			v := reflect.ValueOf(val)
			if v.Kind() == reflect.String && v.String() == "" {
				return def
			}
			return val
		},
		"json": func(v any) (string, error) {
			b, err := sonic.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"len": func(v any) int {
			if v == nil {
				return 0
			}
			val := reflect.ValueOf(v)
			switch val.Kind() {
			case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
				return val.Len()
			}
			return 0
		},
		"jsonpath": func(jsonStr, path string) string {
			return gjson.Get(jsonStr, path).String()
		},
		"jsonpathExists": func(jsonStr, path string) bool {
			return gjson.Get(jsonStr, path).Exists()
		},
		"jsonpathRaw": func(jsonStr, path string) any {
			return gjson.Get(jsonStr, path).Value()
		},
	}
}

// setData stores the current TemplateData on the Engine so that cached
// template closures can read it during execution. The caller must hold
// e.mu or ensure single-goroutine access.
func (e *Engine) setData(data *TemplateData) {
	e.data = data
}

// RenderString renders a template string with the given TemplateData.
// Templates are cached by their preprocessed string; on cache hit the
// same parsed template is reused, but the data-dependent functions
// (event, input, step) always read from the current data via e.data.
func (e *Engine) RenderString(tmpl string, data *TemplateData) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	tmpl = preprocessTemplate(tmpl)

	e.mu.Lock()
	e.data = data
	defer func() {
		e.data = nil
		e.mu.Unlock()
	}()

	var t *txtpl.Template
	if cached, ok := e.cache.Load(tmpl); ok {
		t = cached.(*txtpl.Template)
	} else {
		var err error
		t, err = txtpl.New("render").Funcs(e.funcs()).Parse(tmpl)
		if err != nil {
			return "", fmt.Errorf("template parse: %w", err)
		}
		e.cache.Store(tmpl, t)
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
