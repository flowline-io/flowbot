package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestBrandMark(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		class      string
		wantSubstr []string
	}{
		{
			name:  "navbar size classes",
			class: "w-6 h-6 shrink-0 text-primary",
			wantSubstr: []string{
				`class="w-6 h-6 shrink-0 text-primary"`,
				`viewBox="0 0 24 24"`,
				`stroke="currentColor"`,
				`fill="currentColor"`,
				`aria-hidden="true"`,
			},
		},
		{
			name:  "compact class",
			class: "w-4 h-4",
			wantSubstr: []string{
				`class="w-4 h-4"`,
				`<circle cx="12" cy="12" r="2.4"`,
			},
		},
		{
			name:  "empty class still renders mark geometry",
			class: "",
			wantSubstr: []string{
				`xmlns="http://www.w3.org/2000/svg"`,
				`<path d="M12 7v3.75M12 12.75l4 2.4M12 12.75l-4 2.4"`,
				`<circle cx="12" cy="5.5"`,
				`<circle cx="17.3" cy="16.3"`,
				`<circle cx="6.7" cy="16.3"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := BrandMark(tt.class).Render(context.Background(), &buf); err != nil {
				t.Fatalf("BrandMark().Render() error = %v", err)
			}
			got := buf.String()
			for _, want := range tt.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("BrandMark(%q) missing %q\nhtml = %s", tt.class, want, got)
				}
			}
		})
	}
}
