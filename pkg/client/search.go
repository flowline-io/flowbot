package client

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/validate"
)

// SearchClient provides access to the search API.
type SearchClient struct {
	c *Client
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Source  string `json:"source"`
	URL     string `json:"url,omitempty"`
}

// Search performs a full-text search.
// The source parameter filters by data source (optional).
func (s *SearchClient) Search(ctx context.Context, query string, source string) ([]SearchResult, error) {
	if err := validateSearchQuery(query); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/service/search/query?q=%s", query)
	if source != "" {
		path += fmt.Sprintf("&source=%s", source)
	}

	var result types.KV
	err := s.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}

	return extractSearchResults(result), nil
}

func validateSearchQuery(query string) error {
	if query == "" {
		return fmt.Errorf("query is required")
	}
	if len(query) > validate.QueryMaxLen {
		return fmt.Errorf("query exceeds maximum length of %d", validate.QueryMaxLen)
	}
	return nil
}

// Autocomplete performs a search autocomplete query.
// The source parameter filters by data source (optional).
func (s *SearchClient) Autocomplete(ctx context.Context, query string, source string) ([]SearchResult, error) {
	if err := validateSearchQuery(query); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/service/search/autocomplete?q=%s", query)
	if source != "" {
		path += fmt.Sprintf("&source=%s", source)
	}

	var result types.KV
	err := s.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}

	return extractSearchResults(result), nil
}

func extractSearchResults(data types.KV) []SearchResult {
	results := make([]SearchResult, 0)
	if items, ok := data["hits"].([]any); ok {
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				result := SearchResult{
					ID:     getString(m, "id"),
					Title:  getString(m, "title"),
					Source: getString(m, "source"),
					URL:    getString(m, "url"),
				}
				if content, ok := m["content"].(string); ok {
					result.Content = content
				}
				results = append(results, result)
			}
		}
	}
	return results
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
