package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestTableFilterBar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		placeholder string
		testID      string
		want        []string
	}{
		{
			name:        "renders search input with placeholder and test id",
			placeholder: "Search tokens...",
			testID:      "token-table-filter",
			want: []string{
				`type="search"`,
				`placeholder="Search tokens..."`,
				`data-testid="token-table-filter"`,
				`x-model="search"`,
				`class="flowbot-table-filter-input"`,
			},
		},
		{
			name:        "renders toolbar with search icon and clear control",
			placeholder: "Filter configs...",
			testID:      "config-table-filter",
			want: []string{
				`class="flowbot-table-filter"`,
				`class="flowbot-table-filter-icon"`,
				`data-testid="config-table-filter-clear"`,
				`x-on:click="clearSearch()"`,
			},
		},
		{
			name:        "uses accessible label from placeholder",
			placeholder: "Search...",
			testID:      "table-filter",
			want: []string{
				`aria-label="Search..."`,
				`aria-label="Clear search"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.TableFilterBar(tt.placeholder, tt.testID).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
		})
	}
}

func TestTableFilterScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		wantSrc   string
		wantDefer bool
	}{
		{
			name:      "loads table-filter.js synchronously",
			wantSrc:   "/static/js/table-filter.js",
			wantDefer: false,
		},
		{
			name:      "script tag present",
			wantSrc:   "table-filter.js",
			wantDefer: false,
		},
		{
			name:      "no defer attribute",
			wantSrc:   `src="/static/js/table-filter.js"`,
			wantDefer: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.TableFilterScripts().Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			if !strings.Contains(html, tt.wantSrc) {
				t.Fatalf("want %q in %q", tt.wantSrc, html)
			}
			if tt.wantDefer && !strings.Contains(html, " defer") {
				t.Fatalf("want defer in %q", html)
			}
			if !tt.wantDefer && strings.Contains(html, " defer") {
				t.Fatalf("must not use defer: %q", html)
			}
		})
	}
}
