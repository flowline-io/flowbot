package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestAlpineDataPageScriptsLoadSynchronously(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		src    string
		render func() string
	}{
		{
			name: "event filters",
			src:  "/static/js/event-filters.js",
			render: func() string {
				var buf bytes.Buffer
				if err := partials.EventFilterScripts().Render(context.Background(), &buf); err != nil {
					t.Fatalf("render: %v", err)
				}
				return buf.String()
			},
		},
		{
			name: "homelab registry",
			src:  "/static/js/homelab-registry.js",
			render: func() string {
				var buf bytes.Buffer
				if err := partials.HomelabRegistryScripts().Render(context.Background(), &buf); err != nil {
					t.Fatalf("render: %v", err)
				}
				return buf.String()
			},
		},
		{
			name: "table filter",
			src:  "/static/js/table-filter.js",
			render: func() string {
				var buf bytes.Buffer
				if err := partials.TableFilterScripts().Render(context.Background(), &buf); err != nil {
					t.Fatalf("render: %v", err)
				}
				return buf.String()
			},
		},
		{
			name: "chart scripts may defer",
			src:  "/static/vendor/chart.js.min.js",
			render: func() string {
				var buf bytes.Buffer
				if err := partials.ChartScripts().Render(context.Background(), &buf); err != nil {
					t.Fatalf("render: %v", err)
				}
				return buf.String()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html := tt.render()
			if !strings.Contains(html, tt.src) {
				t.Fatalf("want script %q in %q", tt.src, html)
			}
			alpineDataScript := strings.Contains(tt.src, "event-filters") ||
				strings.Contains(tt.src, "homelab-registry") ||
				strings.Contains(tt.src, "table-filter")
			scriptTagHasDefer := scriptSrcHasDefer(html, tt.src)
			if alpineDataScript && scriptTagHasDefer {
				t.Fatalf("Alpine.data script %q must not use defer: %q", tt.src, html)
			}
			if !alpineDataScript && !scriptTagHasDefer {
				t.Fatalf("chart script should keep defer: %q", html)
			}
		})
	}
}

func scriptSrcHasDefer(html, src string) bool {
	idx := strings.Index(html, src)
	if idx < 0 {
		return false
	}
	start := strings.LastIndex(html[:idx], "<script")
	if start < 0 {
		return false
	}
	end := strings.Index(html[idx:], ">")
	if end < 0 {
		return false
	}
	tag := html[start : idx+end]
	return strings.Contains(tag, " defer")
}

func TestEventFilterScriptsIncludesClipCopy(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := partials.EventFilterScripts().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "/static/js/clip-copy.js") {
		t.Fatalf("want clip-copy.js in event filter scripts\nhtml=%s", html)
	}
	if !strings.Contains(html, "/static/js/event-filters.js") {
		t.Fatalf("want event-filters.js in event filter scripts\nhtml=%s", html)
	}
}
