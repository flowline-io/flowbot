package htmlpreview

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteAndPrepare(t *testing.T) {
	t.Parallel()
	toolImpl := Tool{}

	tests := []struct {
		name     string
		args     map[string]any
		wantErr  bool
		wantSub  string
		check    func(t *testing.T, stdout, html string)
		wantCSP  bool
		fragment bool
	}{
		{
			name: "fragment is wrapped with csp",
			args: map[string]any{"html": `<div id="ok">hi</script><script>document.body.dataset.x="1"</script>`},
			check: func(t *testing.T, stdout, _ string) {
				assert.Contains(t, stdout, "html presented")
				assert.Contains(t, stdout, "id: html_")
			},
			wantCSP:  true,
			fragment: true,
		},
		{
			name: "full document injects csp into head",
			args: map[string]any{
				"html":  `<!DOCTYPE html><html><head><title>x</title></head><body><script>void 0</script></body></html>`,
				"title": "Dash",
				"id":    "chart-1",
			},
			wantSub: "id: chart-1",
			wantCSP: true,
		},
		{
			name:    "empty html errors",
			args:    map[string]any{"html": "  "},
			wantErr: true,
			wantSub: "html is required",
		},
		{
			name:    "oversize html errors",
			args:    map[string]any{"html": strings.Repeat("a", MaxHTMLBytes+1)},
			wantErr: true,
			wantSub: "exceeds",
		},
		{
			name:    "invalid id errors",
			args:    map[string]any{"html": "<p>x</p>", "id": "bad id"},
			wantErr: true,
			wantSub: "alphanumeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := toolImpl.Execute(context.Background(), "c1", tt.args, nil)
			require.NoError(t, err)
			text := toolText(res)
			if tt.wantErr {
				assert.True(t, res.IsError)
				assert.Contains(t, text, tt.wantSub)
				return
			}
			require.False(t, res.IsError)
			if tt.wantSub != "" {
				assert.Contains(t, text, tt.wantSub)
			}
			if tt.check != nil {
				tt.check(t, text, "")
			}
			if tt.wantCSP {
				html := stringArg(tt.args["html"])
				title := stringArg(tt.args["title"])
				doc := PrepareDocument(html, title)
				assert.Contains(t, doc, `http-equiv="Content-Security-Policy"`)
				assert.Contains(t, doc, PreviewCSP)
				assert.Contains(t, doc, "<script>")
				assertCSPFirstInHead(t, doc)
				if tt.fragment {
					assert.Contains(t, strings.ToLower(doc), "<!doctype html>")
				}
			}
		})
	}
}

func TestPrepareDocumentKeepsExistingTitle(t *testing.T) {
	t.Parallel()
	doc := PrepareDocument(`<html><head><title>Keep</title></head><body>x</body></html>`, "Dash")
	assert.Equal(t, 1, strings.Count(strings.ToLower(doc), "<title"))
	assert.Contains(t, doc, "<title>Keep</title>")
	assertCSPFirstInHead(t, doc)
}

func TestPrepareDocumentInsertsHeadWhenMissing(t *testing.T) {
	t.Parallel()
	doc := PrepareDocument(`<html><body>x</body></html>`, "Dash")
	assertCSPFirstInHead(t, doc)
	assert.Contains(t, doc, "<title>Dash</title>")
}

func assertCSPFirstInHead(t *testing.T, doc string) {
	t.Helper()
	_, end := indexOpenTag(strings.ToLower(doc), "head")
	require.Positive(t, end)
	assert.True(t, strings.HasPrefix(doc[end:], cspMeta), "csp meta must be the first head content\ndoc=%s", doc)
}

func TestRegisterActiveToolNames(t *testing.T) {
	t.Parallel()
	reg := tool.NewRegistry()
	require.NoError(t, Register(reg))
	_, ok := reg.Get(ToolName)
	assert.True(t, ok)
	assert.Equal(t, []string{ToolName}, ActiveToolNames())
	assert.Error(t, Register(nil))
}

func TestHTMLAndTitleFromArguments(t *testing.T) {
	t.Parallel()
	html, title := HTMLAndTitleFromArguments(`{"html":"<p>x</p>"}`)
	assert.Equal(t, "<p>x</p>", html)
	assert.Equal(t, DefaultTitle, title)
	html, title = HTMLAndTitleFromArguments(`{"html":"<p>x</p>","title":"A"}`)
	assert.Equal(t, "<p>x</p>", html)
	assert.Equal(t, "A", title)
	html, title = HTMLAndTitleFromArguments("not-json")
	assert.Empty(t, html)
	assert.Empty(t, title)
}

func TestParseResultMeta(t *testing.T) {
	t.Parallel()
	got := ParseResultMeta("html presented\nid: html_ab\ntitle: Dash\nbytes: 4\nhash: sha256:ffff")
	assert.Equal(t, "html_ab", got.ID)
	assert.Equal(t, "Dash", got.Title)
}

func toolText(res msg.ToolResultMessage) string {
	for _, part := range res.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
