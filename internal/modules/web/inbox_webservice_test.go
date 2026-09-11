package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeInboxRedirectURL(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   string
		wantOK bool
	}{
		{name: "agent path", raw: "/service/web/agents/sess-1", want: "/service/web/agents/sess-1", wantOK: true},
		{name: "with query", raw: "/service/web/inbox?filter=all", want: "/service/web/inbox?filter=all", wantOK: true},
		{name: "external https", raw: "https://evil.example/phish", wantOK: false},
		{name: "protocol relative", raw: "//evil.example/x", wantOK: false},
		{name: "backslash open redirect", raw: "/\\evil.example/", wantOK: false},
		{name: "non web path", raw: "/chatagent/x", wantOK: false},
		{name: "empty", raw: "", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := safeInboxRedirectURL(tt.raw)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSafeNext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "home default", raw: "", want: "/service/web/home"},
		{name: "valid path", raw: "/service/web/tokens", want: "/service/web/tokens"},
		{name: "backslash rejected", raw: "/\\evil.example/", want: "/service/web/home"},
		{name: "protocol relative rejected", raw: "//evil.example/", want: "/service/web/home"},
		{name: "absolute url rejected", raw: "https://evil.example/", want: "/service/web/home"},
		{name: "outside web rejected", raw: "/hub/apps", want: "/service/web/home"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, safeNext(tt.raw))
		})
	}
}
