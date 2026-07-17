package coding

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

const duckDuckGoAPI = "https://api.duckduckgo.com/"

// WebSearchTool searches the web using DuckDuckGo Instant Answer API.
type WebSearchTool struct {
	HTTPClient *http.Client
	MaxOutput  int
	// BaseURL overrides the DuckDuckGo endpoint for tests.
	BaseURL string
}

// Name returns the tool identifier.
func (WebSearchTool) Name() string { return "web_search" }

// Description explains the tool to the model.
func (WebSearchTool) Description() string {
	return "Looks up concise facts via DuckDuckGo Instant Answers; may return no results for niche or technical queries"
}

// Parameters returns the JSON schema for tool arguments.
func (WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
		"required": []string{"query"},
	}
}

type ddgResponse struct {
	Abstract       string `json:"Abstract"`
	AbstractText   string `json:"AbstractText"`
	AbstractSource string `json:"AbstractSource"`
	AbstractURL    string `json:"AbstractURL"`
	Heading        string `json:"Heading"`
	RelatedTopics  []any  `json:"RelatedTopics"`
}

// Execute performs the web search.
func (t WebSearchTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
	query := strings.TrimSpace(fmt.Sprint(args["query"]))
	if query == "" {
		return toolError(id, t.Name(), "query is required"), nil
	}
	if len(query) > MaxWebSearchQueryBytes {
		return tool.ErrorResult(id, t.Name(), "invalid_args",
			fmt.Sprintf("query exceeds %d bytes", MaxWebSearchQueryBytes),
			"shorten the search query"), nil
	}
	if onUpdate != nil {
		_ = onUpdate("searching...")
	}

	client := t.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}

	base := t.BaseURL
	if base == "" {
		base = duckDuckGoAPI
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}
	q := endpoint.Query()
	q.Set("q", query)
	q.Set("format", "json")
	q.Set("no_redirect", "1")
	q.Set("no_html", "1")
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return toolError(id, t.Name(), fmt.Sprintf("search request: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxFetchBytes)+1))
	if err != nil {
		return toolError(id, t.Name(), fmt.Sprintf("read response: %v", err)), nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return toolError(id, t.Name(), fmt.Sprintf("search status %d", resp.StatusCode)), nil
	}

	var parsed ddgResponse
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return toolError(id, t.Name(), fmt.Sprintf("parse response: %v", err)), nil
	}

	text := formatDDGResult(parsed)
	limit := t.MaxOutput
	if limit <= 0 {
		limit = DefaultMaxOutput
	}
	if len(text) > limit {
		text = text[:limit] + "\n...(truncated)"
	}
	if strings.TrimSpace(text) == "" {
		text = "No results found."
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

func formatDDGResult(r ddgResponse) string {
	var b strings.Builder
	if r.Heading != "" {
		_, _ = b.WriteString(r.Heading)
		_, _ = b.WriteString("\n")
	}
	summary := r.AbstractText
	if summary == "" {
		summary = r.Abstract
	}
	if summary != "" {
		_, _ = b.WriteString(summary)
		if r.AbstractSource != "" {
			_, _ = fmt.Fprintf(&b, " (source: %s)", r.AbstractSource)
		}
		if r.AbstractURL != "" {
			_, _ = fmt.Fprintf(&b, "\nURL: %s", r.AbstractURL)
		}
		_, _ = b.WriteString("\n")
	}
	for _, topic := range r.RelatedTopics {
		if item, ok := topic.(map[string]any); ok {
			if text, ok := item["Text"].(string); ok && text != "" {
				_, _ = b.WriteString("- ")
				_, _ = b.WriteString(text)
				_, _ = b.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}
