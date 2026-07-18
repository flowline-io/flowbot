package template

import (
	"regexp"
	"slices"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/notify/manifest"
)

var templateFieldPattern = regexp.MustCompile(`\.([A-Za-z_][A-Za-z0-9_]*)`)

// ExtractTemplateFields returns unique root payload field names referenced in a template string.
func ExtractTemplateFields(tmplStr string) []string {
	matches := templateFieldPattern.FindAllStringSubmatch(tmplStr, -1)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// SamplePayload builds a sample payload map from a notify template's body and overrides.
func SamplePayload(tmpl manifest.Template) map[string]any {
	fields := ExtractTemplateFields(tmpl.DefaultTemplate)
	for _, o := range tmpl.Overrides {
		for _, f := range ExtractTemplateFields(o.Template) {
			if !slices.Contains(fields, f) {
				fields = append(fields, f)
			}
		}
	}
	slices.Sort(fields)

	out := make(map[string]any, len(fields)+1)
	for _, f := range fields {
		out[f] = sampleValueForField(f)
	}
	if _, ok := out["summary"]; !ok {
		if tmpl.ID != "" {
			out["summary"] = tmpl.ID
		} else {
			out["summary"] = "playground"
		}
	}
	return out
}

// SamplePayloadJSON returns pretty-printed JSON for SamplePayload.
func SamplePayloadJSON(tmpl manifest.Template) (string, error) {
	payload := SamplePayload(tmpl)
	b, err := sonic.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func sampleValueForField(name string) any {
	switch strings.ToLower(name) {
	case "url", "drone_url":
		return "https://example.com"
	case "title", "name", "message", "body", "notes", "description":
		return "example " + name
	case "id", "hostid", "project_id", "build":
		return "123"
	case "hostname":
		return "host.example"
	case "status":
		return "online"
	case "amount":
		return 42.5
	case "currency":
		return "USD"
	case "priority":
		return "normal"
	case "recurring":
		return true
	default:
		return "example"
	}
}
