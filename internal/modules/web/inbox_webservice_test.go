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
