package coding

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

const (
	duckDuckGoHTML     = "https://html.duckduckgo.com/html/"
	duckDuckGoInstant  = "https://api.duckduckgo.com/"
	braveSearchAPI     = "https://api.search.brave.com/res/v1/web/search"
	webSearchUserAgent = "Mozilla/5.0 (compatible; Flowbot/1.0; +https://github.com/flowline-io/flowbot)"
)

var (
	ddgResultLinkRe = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*\bresult__a\b[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	ddgSnippetRe    = regexp.MustCompile(`(?is)<(?:a|td)[^>]*class="[^"]*\bresult__snippet\b[^"]*"[^>]*>(.*?)</(?:a|td)>`)
)

// WebSearchTool searches the web via SearXNG, Brave, or DuckDuckGo HTML results.
type WebSearchTool struct {
	HTTPClient *http.Client
	MaxOutput  int
	// BaseURL overrides the DuckDuckGo HTML endpoint for tests.
	BaseURL string
	// InstantBaseURL overrides the DuckDuckGo Instant Answer endpoint for tests.
	InstantBaseURL string
	// SearxURL is a SearXNG-compatible JSON search endpoint (e.g. http://127.0.0.1:8080/search).
	SearxURL string
	// BraveAPIKey enables the Brave Search API when non-empty.
	BraveAPIKey string
	// BraveBaseURL overrides the Brave Search endpoint for tests.
	BraveBaseURL string
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

	if strings.TrimSpace(t.SearxURL) != "" {
		hits, errMsg := t.searchSearx(ctx, client, query)
		if errMsg != "" || len(hits) > 0 {
			return hits, errMsg
		}
	}
	if strings.TrimSpace(t.BraveAPIKey) != "" {
		hits, errMsg := t.searchBrave(ctx, client, query)
		if errMsg != "" || len(hits) > 0 {
			return hits, errMsg
		}
	}

	hits, captcha, errMsg := t.searchDDGHTML(ctx, client, query)
	if errMsg != "" {
		return nil, errMsg
	}
	if captcha {
		return nil, "DuckDuckGo blocked the request (captcha/bot check); configure chat_agent.web_search.searx_url or brave_api_key for reliable search"
	}
	if len(hits) > 0 {
		return hits, ""
	}

	return t.searchInstantAnswer(ctx, client, query)
}

func (t WebSearchTool) searchSearx(ctx context.Context, client *http.Client, query string) ([]searchHit, string) {
	endpoint, err := url.Parse(strings.TrimSpace(t.SearxURL))
	if err != nil {
		return nil, fmt.Sprintf("invalid searx_url: %v", err)
	}
	q := endpoint.Query()
	q.Set("q", query)
	q.Set("format", "json")
	endpoint.RawQuery = q.Encode()

	body, errMsg := webSearchGET(ctx, client, endpoint.String(), nil)
	if errMsg != "" {
		return nil, errMsg
	}

	var parsed searxResponse
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Sprintf("parse searx response: %v", err)
	}
	hits := make([]searchHit, 0, len(parsed.Results))
	for _, item := range parsed.Results {
		hits = append(hits, searchHit{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: strings.TrimSpace(item.Content),
		})
		if len(hits) >= MaxWebSearchResults {
			break
		}
	}
	return hits, ""
}

func (t WebSearchTool) searchBrave(ctx context.Context, client *http.Client, query string) ([]searchHit, string) {
	base := strings.TrimSpace(t.BraveBaseURL)
	if base == "" {
		base = braveSearchAPI
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Sprintf("invalid brave base url: %v", err)
	}
	q := endpoint.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", MaxWebSearchResults))
	endpoint.RawQuery = q.Encode()

	headers := map[string]string{
		"Accept":               "application/json",
		"X-Subscription-Token": strings.TrimSpace(t.BraveAPIKey),
	}
	body, errMsg := webSearchGET(ctx, client, endpoint.String(), headers)
	if errMsg != "" {
		return nil, errMsg
	}

	var parsed braveResponse
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Sprintf("parse brave response: %v", err)
	}
	hits := make([]searchHit, 0, len(parsed.Web.Results))
	for _, item := range parsed.Web.Results {
		hits = append(hits, searchHit{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: strings.TrimSpace(item.Description),
		})
		if len(hits) >= MaxWebSearchResults {
			break
		}
	}
	return hits, ""
}

func (t WebSearchTool) searchDDGHTML(ctx context.Context, client *http.Client, query string) ([]searchHit, bool, string) {
	base := t.BaseURL
	if base == "" {
		base = duckDuckGoHTML
	}
	form := url.Values{}
	form.Set("q", query)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, false, err.Error()
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", webSearchUserAgent)
	req.Header.Set("Referer", "https://html.duckduckgo.com/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Sprintf("search request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxFetchBytes)+1))
	if err != nil {
		return nil, false, fmt.Sprintf("read response: %v", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, false, fmt.Sprintf("search status %d", resp.StatusCode)
	}

	text := string(body)
	if isDDGCaptcha(text) {
		return nil, true, ""
	}
	return parseDDGHTML(text), false, ""
}

func (t WebSearchTool) searchInstantAnswer(ctx context.Context, client *http.Client, query string) ([]searchHit, string) {
	base := t.InstantBaseURL
	if base == "" {
		base = duckDuckGoInstant
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return nil, err.Error()
	}
	q := endpoint.Query()
	q.Set("q", query)
	q.Set("format", "json")
	q.Set("no_redirect", "1")
	q.Set("no_html", "1")
	endpoint.RawQuery = q.Encode()

	body, errMsg := webSearchGET(ctx, client, endpoint.String(), nil)
	if errMsg != "" {
		return nil, errMsg
	}

	var parsed ddgInstantResponse
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Sprintf("parse response: %v", err)
	}
	return instantAnswerHits(parsed), ""
}

func webSearchGET(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) ([]byte, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, err.Error()
	}
	req.Header.Set("User-Agent", webSearchUserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
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

type searxResponse struct {
	Results []searxResult `json:"results"`
}

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type braveResponse struct {
	Web braveWeb `json:"web"`
}

type braveWeb struct {
	Results []braveResult `json:"results"`
}

type braveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type ddgInstantResponse struct {
	Abstract       string `json:"Abstract"`
	AbstractText   string `json:"AbstractText"`
	AbstractSource string `json:"AbstractSource"`
	AbstractURL    string `json:"AbstractURL"`
	Heading        string `json:"Heading"`
	RelatedTopics  []any  `json:"RelatedTopics"`
}

func instantAnswerHits(r ddgInstantResponse) []searchHit {
	hits := make([]searchHit, 0, MaxWebSearchResults)
	summary := r.AbstractText
	if summary == "" {
		summary = r.Abstract
	}
	if r.Heading != "" || summary != "" {
		title := r.Heading
		if title == "" {
			title = "Instant Answer"
		}
		if r.AbstractSource != "" {
			title = fmt.Sprintf("%s (%s)", title, r.AbstractSource)
		}
		hits = append(hits, searchHit{
			Title:   title,
			URL:     strings.TrimSpace(r.AbstractURL),
			Snippet: strings.TrimSpace(summary),
		})
	}
	for _, topic := range r.RelatedTopics {
		item, ok := topic.(map[string]any)
		if !ok {
			continue
		}
		textVal, ok := item["Text"].(string)
		if !ok {
			continue
		}
		text := strings.TrimSpace(textVal)
		if text == "" {
			continue
		}
		firstURL := ""
		if fu, ok := item["FirstURL"].(string); ok {
			firstURL = strings.TrimSpace(fu)
		}
		hits = append(hits, searchHit{Title: text, URL: firstURL})
		if len(hits) >= MaxWebSearchResults {
			break
		}
	}
	return hits
}

func isDDGCaptcha(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "anomaly.js") ||
		strings.Contains(lower, "challenge-form") ||
		strings.Contains(lower, "id=\"challenge-form\"") ||
		strings.Contains(lower, "please complete the captcha")
}

func parseDDGHTML(body string) []searchHit {
	links := ddgResultLinkRe.FindAllStringSubmatch(body, MaxWebSearchResults*2)
	snippets := ddgSnippetRe.FindAllStringSubmatch(body, MaxWebSearchResults*2)
	hits := make([]searchHit, 0, MaxWebSearchResults)
	for i, link := range links {
		if len(link) < 3 {
			continue
		}
		title := collapseSpace(stripTags(html.UnescapeString(link[2])))
		href := decodeDDGHref(html.UnescapeString(link[1]))
		if title == "" && href == "" {
			continue
		}
		snippet := ""
		if i < len(snippets) && len(snippets[i]) >= 2 {
			snippet = collapseSpace(stripTags(html.UnescapeString(snippets[i][1])))
		}
		hits = append(hits, searchHit{Title: title, URL: href, Snippet: snippet})
		if len(hits) >= MaxWebSearchResults {
			break
		}
	}
	return hits
}

func decodeDDGHref(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}
	if uddg := parsed.Query().Get("uddg"); uddg != "" {
		if decoded, err := url.QueryUnescape(uddg); err == nil && decoded != "" {
			return decoded
		}
	}
	return href
}

func stripTags(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
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
