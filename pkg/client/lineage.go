package client

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
)

// LineageClient calls /service/lineage and related analytics read APIs.
type LineageClient struct {
	c *Client
}

// FlowClient calls /service/flows.
type FlowClient struct {
	c *Client
}

// EventsClient calls /service/events.
type EventsClient struct {
	c *Client
}

// LineageTraceOptions is the query for GET /service/lineage/trace.
type LineageTraceOptions struct {
	App           string
	EntityID      string
	EventID       string
	CorrelationID string
	Direction     string
	Depth         int
}

// LineageSearchOptions is the query for GET /service/lineage/search.
type LineageSearchOptions struct {
	Query  string
	Limit  int
	Cursor string
}

// LineageSearchResult is the REST envelope for lineage.search.
type LineageSearchResult struct {
	Items []types.ResourceRef `json:"items"`
	Page  map[string]any      `json:"page"`
}

// FlowGraphOptions is the query for GET /service/flows/graph.
type FlowGraphOptions struct {
	Since         string
	Until         string
	App           string
	IncludeStatic bool
}

// EventsTimelineOptions is the query for GET /service/events/timeline.
type EventsTimelineOptions struct {
	Since          string
	Until          string
	App            string
	EventType      string
	CorrelationID  string
	Limit          int
	Cursor         string
	IncludePayload bool
}

// EventsTimelineResult is the REST envelope for events.timeline.
type EventsTimelineResult struct {
	Items []types.EventsTimelineItem `json:"items"`
	Page  map[string]any             `json:"page"`
}

// Trace returns multi-hop lineage from one seed.
func (l *LineageClient) Trace(ctx context.Context, opts LineageTraceOptions) (*types.LineageTrace, error) {
	var result types.LineageTrace
	err := l.c.Get(ctx, "/service/lineage/trace"+lineageTraceQuery(opts), &result)
	return &result, err
}

// Search returns distinct resource nodes matching query.
func (l *LineageClient) Search(ctx context.Context, opts LineageSearchOptions) (*LineageSearchResult, error) {
	var result LineageSearchResult
	err := l.c.Get(ctx, "/service/lineage/search"+lineageSearchQuery(opts), &result)
	return &result, err
}

// Graph returns the live-aggregated flow topology.
func (f *FlowClient) Graph(ctx context.Context, opts FlowGraphOptions) (*types.FlowGraph, error) {
	var result types.FlowGraph
	err := f.c.Get(ctx, "/service/flows/graph"+flowGraphQuery(opts), &result)
	return &result, err
}

// Timeline returns a paged data_events window.
func (e *EventsClient) Timeline(ctx context.Context, opts EventsTimelineOptions) (*EventsTimelineResult, error) {
	var result EventsTimelineResult
	err := e.c.Get(ctx, "/service/events/timeline"+eventsTimelineQuery(opts), &result)
	return &result, err
}

func lineageTraceQuery(opts LineageTraceOptions) string {
	q := url.Values{}
	setQuery(q, "app", opts.App)
	setQuery(q, "entity_id", opts.EntityID)
	setQuery(q, "event_id", opts.EventID)
	setQuery(q, "correlation_id", opts.CorrelationID)
	setQuery(q, "direction", opts.Direction)
	if opts.Depth > 0 {
		q.Set("depth", strconv.Itoa(opts.Depth))
	}
	return encodeQuery(q)
}

func lineageSearchQuery(opts LineageSearchOptions) string {
	q := url.Values{}
	setQuery(q, "query", opts.Query)
	setQuery(q, "cursor", opts.Cursor)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	return encodeQuery(q)
}

func flowGraphQuery(opts FlowGraphOptions) string {
	q := url.Values{}
	setQuery(q, "since", opts.Since)
	setQuery(q, "until", opts.Until)
	setQuery(q, "app", opts.App)
	if opts.IncludeStatic {
		q.Set("include_static", "true")
	}
	return encodeQuery(q)
}

func eventsTimelineQuery(opts EventsTimelineOptions) string {
	q := url.Values{}
	setQuery(q, "since", opts.Since)
	setQuery(q, "until", opts.Until)
	setQuery(q, "app", opts.App)
	setQuery(q, "event_type", opts.EventType)
	setQuery(q, "correlation_id", opts.CorrelationID)
	setQuery(q, "cursor", opts.Cursor)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.IncludePayload {
		q.Set("include_payload", "true")
	}
	return encodeQuery(q)
}

func setQuery(q url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		q.Set(key, value)
	}
}

func encodeQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}
