package coding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDDGHTML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantURL   string
		wantTitle string
	}{
		{
			name: "parses result links and snippets",
			body: `
<div class="result web-result">
  <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Famd">AMD RX 9070 GRE price</a>
  <a class="result__snippet">Street price around $599 for the GRE model.</a>
</div>`,
			wantCount: 1,
			wantURL:   "https://example.com/amd",
			wantTitle: "AMD RX 9070 GRE price",
		},
		{
			name:      "empty body yields no hits",
			body:      `<html><body>no results</body></html>`,
			wantCount: 0,
		},
		{
			name: "strips nested tags in title",
			body: `<a class="result__a" href="https://docs.example/"><b>Go</b> docs</a>
<a class="result__snippet">Official docs</a>`,
			wantCount: 1,
			wantURL:   "https://docs.example/",
			wantTitle: "Go docs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hits := parseDDGHTML(tt.body)
			require.Len(t, hits, tt.wantCount)
			if tt.wantCount == 0 {
				return
			}
			assert.Equal(t, tt.wantURL, hits[0].URL)
			assert.Equal(t, tt.wantTitle, hits[0].Title)
			assert.NotEmpty(t, hits[0].Snippet)
		})
	}
}

func TestIsDDGCaptcha(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "challenge form", body: `<form id="challenge-form" action="//duckduckgo.com/anomaly.js">`, want: true},
		{name: "anomaly script", body: `<script src="/anomaly.js"></script>`, want: true},
		{name: "normal results", body: `<a class="result__a" href="https://example.com">ok</a>`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isDDGCaptcha(tt.body))
		})
	}
}

func TestDecodeDDGHref(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			name: "decodes uddg redirect",
			href: "//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpath&rut=abc",
			want: "https://example.com/path",
		},
		{
			name: "keeps absolute url",
			href: "https://example.com/direct",
			want: "https://example.com/direct",
		},
		{
			name: "empty href",
			href: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, decodeDDGHref(tt.href))
		})
	}
}

func TestFormatSearchHits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		hits []searchHit
		want string
	}{
		{
			name: "formats numbered results",
			hits: []searchHit{{Title: "A", URL: "https://a.example", Snippet: "alpha"}},
			want: "1. A\n   URL: https://a.example\n   alpha",
		},
		{
			name: "skips empty hits",
			hits: []searchHit{{}, {Title: "B", URL: "https://b.example"}},
			want: "1. B\n   URL: https://b.example",
		},
		{
			name: "empty list",
			hits: nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatSearchHits(tt.hits))
		})
	}
}
