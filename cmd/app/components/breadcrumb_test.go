package components

import (
	"testing"
)

func TestBuildBreadcrumbFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []BreadcrumbItem
	}{
		{
			name:     "root path",
			path:     "/admin",
			expected: []BreadcrumbItem{{Label: "Home", Href: "/admin"}},
		},
		{
			name: "single segment",
			path: "/admin/users",
			expected: []BreadcrumbItem{
				{Label: "Home", Href: "/admin"},
				{Label: "Users", Href: ""},
			},
		},
		{
			name: "multiple segments",
			path: "/admin/workflows/run",
			expected: []BreadcrumbItem{
				{Label: "Home", Href: "/admin"},
				{Label: "Workflows", Href: "/admin/workflows"},
				{Label: "run", Href: ""},
			},
		},
		{
			name:     "trailing slash",
			path:     "/admin/users/",
			expected: []BreadcrumbItem{{Label: "Home", Href: "/admin"}, {Label: "Users", Href: ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBreadcrumbFromPath(tt.path)
			if len(result) != len(tt.expected) {
				t.Errorf("buildBreadcrumbFromPath(%q) returned %d items, want %d", tt.path, len(result), len(tt.expected))
				return
			}
			for i, item := range result {
				if item.Label != tt.expected[i].Label {
					t.Errorf("Item %d Label = %q, want %q", i, item.Label, tt.expected[i].Label)
				}
				if item.Href != tt.expected[i].Href {
					t.Errorf("Item %d Href = %q, want %q", i, item.Href, tt.expected[i].Href)
				}
			}
		})
	}
}

func TestFormatBreadcrumbLabel(t *testing.T) {
	tests := []struct {
		segment  string
		expected string
	}{
		{"users", "Users"},
		{"containers", "Containers"},
		{"workflows", "Workflows"},
		{"bots", "Bots"},
		{"logs", "Logs"},
		{"settings", "Settings"},
		{"login", "Login"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.segment, func(t *testing.T) {
			result := formatBreadcrumbLabel(tt.segment)
			if result != tt.expected {
				t.Errorf("formatBreadcrumbLabel(%q) = %q, want %q", tt.segment, result, tt.expected)
			}
		})
	}
}
