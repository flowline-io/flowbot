package partials

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPaletteNavPages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		wantHref  string
		wantTitle string
	}{
		{name: "includes home", wantHref: "/service/web/home", wantTitle: "Home"},
		{name: "includes pipelines", wantHref: "/service/web/pipelines", wantTitle: "Pipelines"},
		{name: "includes agents", wantHref: "/service/web/agents", wantTitle: "Agents"},
		{name: "includes homelab registry", wantHref: "/service/web/homelab", wantTitle: "Registry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pages := CommandPaletteNavPages()
			require.NotEmpty(t, pages)
			found := false
			for _, p := range pages {
				if p.Href == tt.wantHref && p.Title == tt.wantTitle {
					found = true
					assert.Equal(t, "pages", p.Group)
					assert.True(t, strings.HasPrefix(p.ID, "page:"))
					break
				}
			}
			assert.True(t, found, "missing page %s (%s)", tt.wantTitle, tt.wantHref)
		})
	}
}

func TestCommandPalettePagesJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wantSubstr string
	}{
		{name: "encodes home href", wantSubstr: `"/service/web/home"`},
		{name: "encodes pipelines title", wantSubstr: `"Pipelines"`},
		{name: "is a JSON array", wantSubstr: `[{`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CommandPalettePagesJSON()
			assert.Contains(t, got, tt.wantSubstr)
		})
	}
}
