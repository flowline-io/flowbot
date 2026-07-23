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

const (
	serpAPISearchURL   = "https://serpapi.com/search"
	webSearchUserAgent = "Mozilla/5.0 (compatible; Flowbot/1.0; +https://github.com/flowline-io/flowbot)"
)

// WebSearchTool searches the web via the SerpApi Google Search API.
type WebSearchTool struct {
	HTTPClient *http.Client
	MaxOutput  int
	// APIKey is the SerpApi private key (required).
	APIKey string
	// BaseURL overrides the SerpApi search endpoint for tests.
	BaseURL string
}

// searchHit is one organic web search result.
type searchHit struct {
	Title   string
	URL     string
	Snippet string
}

// Name returns the tool identifier.
func (WebSearchTool) Name() string { return "web_search" }

// Description explains the tool to the model.
func (WebSearchTool) Description() string {
	return "Searches the web and returns titles, URLs, and snippets from organic results"
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

// Execute performs the web search.
func (t WebSearchTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
	query := strings.TrimSpace(fmt.Sprint(args["query"]))
	if query == "" || query == "<nil>" {
		return toolError(id, t.Name(), "query is required"), nil
	}
	if len(query) > MaxWebSearchQueryBytes {
		return tool.ErrorResult(id, t.Name(), "invalid_args",
			fmt.Sprintf("query exceeds %d bytes", MaxWebSearchQueryBytes),
			"shorten the search query"), nil
	}
	if strings.TrimSpace(t.APIKey) == "" {
		return toolError(id, t.Name(), "configure chat_agent.web_search.api_key (SerpApi) for web search"), nil
	}
	if onUpdate != nil {
		_ = onUpdate("searching...")
	}

	hits, errMsg := t.search(ctx, query)
	if errMsg != "" {
		return toolError(id, t.Name(), errMsg), nil
	}

	text := formatSearchHits(hits)
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

func (t WebSearchTool) search(ctx context.Context, query string) ([]searchHit, string) {
	client := t.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}

	base := strings.TrimSpace(t.BaseURL)
	if base == "" {
		base = serpAPISearchURL
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Sprintf("invalid serpapi base url: %v", err)
	}
	q := endpoint.Query()
	q.Set("engine", "google")
	q.Set("q", query)
	q.Set("api_key", strings.TrimSpace(t.APIKey))
	q.Set("output", "json")
	endpoint.RawQuery = q.Encode()

	body, errMsg := webSearchGET(ctx, client, endpoint.String())
	if errMsg != "" {
		return nil, errMsg
	}

	hits, errMsg := parseSerpAPIResponse(body)
	if errMsg != "" {
		return nil, errMsg
	}
	return hits, ""
}

func webSearchGET(ctx context.Context, client *http.Client, rawURL string) ([]byte, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, err.Error()
	}
	req.Header.Set("User-Agent", webSearchUserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Sprintf("search request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxFetchBytes)+1))
	if err != nil {
		return nil, fmt.Sprintf("read response: %v", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Sprintf("search status %d", resp.StatusCode)
	}
	return body, ""
}

type serpAPIResponse struct {
	Error          string              `json:"error"`
	SearchMetadata serpAPIMetadata     `json:"search_metadata"`
	OrganicResults []serpAPIOrganicHit `json:"organic_results"`
}

type serpAPIMetadata struct {
	Status string `json:"status"`
}

type serpAPIOrganicHit struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

func parseSerpAPIResponse(body []byte) ([]searchHit, string) {
	var parsed serpAPIResponse
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Sprintf("parse serpapi response: %v", err)
	}
	if errMsg := strings.TrimSpace(parsed.Error); errMsg != "" {
		return nil, fmt.Sprintf("serpapi error: %s", errMsg)
	}
	status := strings.TrimSpace(parsed.SearchMetadata.Status)
	if status != "" && !strings.EqualFold(status, "Success") {
		return nil, fmt.Sprintf("serpapi status: %s", status)
	}
	hits := make([]searchHit, 0, len(parsed.OrganicResults))
	for _, item := range parsed.OrganicResults {
		hits = append(hits, searchHit{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.Link),
			Snippet: strings.TrimSpace(item.Snippet),
		})
		if len(hits) >= MaxWebSearchResults {
			break
		}
	}
	return hits, ""
}

func formatSearchHits(hits []searchHit) string {
	var b strings.Builder
	n := 0
	for _, hit := range hits {
		if hit.Title == "" && hit.URL == "" && hit.Snippet == "" {
			continue
		}
		n++
		_, _ = fmt.Fprintf(&b, "%d. %s\n", n, fallback(hit.Title, "(no title)"))
		if hit.URL != "" {
			_, _ = fmt.Fprintf(&b, "   URL: %s\n", hit.URL)
		}
		if hit.Snippet != "" {
			_, _ = fmt.Fprintf(&b, "   %s\n", hit.Snippet)
		}
	}
	return strings.TrimSpace(b.String())
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
