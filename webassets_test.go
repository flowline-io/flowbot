package webassets

import (
	"io"
	"strings"
	"testing"
)

func TestFaviconSVGEmbedded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantSubstr []string
	}{
		{
			name: "favicon present with hub geometry",
			path: "favicon.svg",
			wantSubstr: []string{
				`xmlns="http://www.w3.org/2000/svg"`,
				`aria-label="Flowbot"`,
				`viewBox="0 0 32 32"`,
				`<circle cx="16" cy="16"`,
			},
		},
		{
			name: "favicon uses teal brand gradient",
			path: "favicon.svg",
			wantSubstr: []string{
				`#0F766E`,
				`#134E4A`,
				`#5EEAD4`,
			},
		},
		{
			name: "favicon has rounded tile",
			path: "favicon.svg",
			wantSubstr: []string{
				`rx="8"`,
				`linearGradient`,
			},
		},
		{
			name: "favicon is valid UTF-8 ASCII comments",
			path: "favicon.svg",
			wantSubstr: []string{
				`hub to three capability nodes`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := SubFS.Open(tt.path)
			if err != nil {
				t.Fatalf("SubFS.Open(%q) error = %v", tt.path, err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			got := string(data)
			for _, want := range tt.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("favicon missing %q", want)
				}
			}
		})
	}
}

func TestIBMPlexSansFontsEmbedded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
	}{
		{name: "regular 400", path: "fonts/ibm-plex-sans-latin-400-normal.woff2"},
		{name: "medium 500", path: "fonts/ibm-plex-sans-latin-500-normal.woff2"},
		{name: "semibold 600", path: "fonts/ibm-plex-sans-latin-600-normal.woff2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := SubFS.Open(tt.path)
			if err != nil {
				t.Fatalf("SubFS.Open(%q) error = %v", tt.path, err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if len(data) < 1000 {
				t.Fatalf("font %q too small: %d bytes", tt.path, len(data))
			}
		})
	}
}
