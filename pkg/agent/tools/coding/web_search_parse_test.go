package coding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSerpAPIResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantURL   string
		wantTitle string
		wantErr   string
	}{
		{
			name: "parses organic results",
			body: `{
  "search_metadata": {"status": "Success"},
  "organic_results": [
    {"title": "Coffee - Wikipedia", "link": "https://en.wikipedia.org/wiki/Coffee", "snippet": "Coffee is a brewed drink."}
  ]
}`,
			wantCount: 1,
			wantURL:   "https://en.wikipedia.org/wiki/Coffee",
			wantTitle: "Coffee - Wikipedia",
		},
		{
			name:      "empty organic results",
			body:      `{"search_metadata": {"status": "Success"}, "organic_results": []}`,
			wantCount: 0,
		},
		{
			name:    "serpapi error field",
			body:    `{"error": "Invalid API key."}`,
			wantErr: "serpapi error: Invalid API key.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hits, errMsg := parseSerpAPIResponse([]byte(tt.body))
			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, errMsg)
				return
			}
			require.Empty(t, errMsg)
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
