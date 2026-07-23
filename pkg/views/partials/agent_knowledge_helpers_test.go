package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentKnowledgeListURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		q    string
		want string
	}{
		{name: "empty query", q: "", want: "/service/web/agent-knowledge/list"},
		{name: "escapes spaces", q: "api specs", want: "/service/web/agent-knowledge/list?q=api+specs"},
		{name: "trims whitespace", q: "  ops  ", want: "/service/web/agent-knowledge/list?q=ops"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, agentKnowledgeListURL(tt.q))
		})
	}
}

func TestAgentKnowledgeSummaryPreview(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		summary string
		want    string
	}{
		{name: "short ascii", summary: "hello", want: "hello"},
		{name: "truncates runes not bytes", summary: repeatRune('你', 70), want: repeatRune('你', 57) + "..."},
		{name: "exact sixty runes", summary: repeatRune('a', 60), want: repeatRune('a', 60)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, agentKnowledgeSummaryPreview(tt.summary))
		})
	}
}

func repeatRune(r rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}
