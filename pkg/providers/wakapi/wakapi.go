package wakapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "wakapi"
	EndpointKey = "endpoint"
	APIKeyKey   = "api_key"
)

// Wakapi is an HTTP client for the Wakapi API.
type Wakapi struct {
	c *resty.Client
}

// GetClient builds a Wakapi client from vendors.wakapi config.
// Returns nil when endpoint is not configured.
func GetClient() *Wakapi {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, APIKeyKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewWakapi(endpoint.String(), apiKey.String())
}

// NewWakapi creates a Wakapi client. Auth uses Basic base64(api_key).
// Returns nil when endpoint is empty.
func NewWakapi(endpoint, apiKey string) *Wakapi {
	if endpoint == "" {
		return nil
	}
	v := &Wakapi{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(strings.TrimRight(endpoint, "/"))
	if apiKey != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(apiKey))
		v.c.SetHeader("Authorization", "Basic "+encoded)
	}
	return v
}

// Health checks GET /api/health (no auth required).
func (w *Wakapi) Health(ctx context.Context) (*HealthStatus, error) {
	if w == nil || w.c == nil {
		return nil, fmt.Errorf("wakapi: not configured")
	}
	resp, err := w.c.R().SetContext(ctx).Get("/api/health")
	if err != nil {
		return nil, fmt.Errorf("wakapi health: %w", err)
	}
	if err := checkStatus(resp, "health"); err != nil {
		return nil, err
	}
	return parseHealthStatus(string(resp.Bytes())), nil
}

// GetSummary returns a native activity summary for the given interval or date range.
func (w *Wakapi) GetSummary(ctx context.Context, params SummaryParams) (*Summary, error) {
	if params.Interval == "" && params.From == "" && params.To == "" {
		params.Interval = "today"
	}
	req := w.c.R().SetContext(ctx)
	applySummaryQuery(req, params)
	resp, err := req.Get("/api/summary")
	if err != nil {
		return nil, fmt.Errorf("wakapi summary: %w", err)
	}
	if err := checkStatus(resp, "summary"); err != nil {
		return nil, err
	}
	var result Summary
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi summary decode: %w", err)
	}
	return &result, nil
}

// GetStats returns WakaTime-compatible statistics including total_seconds.
func (w *Wakapi) GetStats(ctx context.Context, params StatsParams) (*Stats, error) {
	user := params.User
	if user == "" {
		user = CurrentUser
	}
	rangeVal := params.Range
	if rangeVal == "" {
		rangeVal = "today"
	}
	req := w.c.R().SetContext(ctx)
	applyStatsQuery(req, params)
	path := fmt.Sprintf("/api/compat/wakatime/v1/users/%s/stats/%s", user, rangeVal)
	resp, err := req.Get(path)
	if err != nil {
		return nil, fmt.Errorf("wakapi stats: %w", err)
	}
	if err := checkStatus(resp, "stats"); err != nil {
		return nil, err
	}
	var result StatsResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi stats decode: %w", err)
	}
	return &result.Data, nil
}

// GetAllTime returns cumulative coding time since account creation.
func (w *Wakapi) GetAllTime(ctx context.Context, user string) (*AllTime, error) {
	if user == "" {
		user = CurrentUser
	}
	path := fmt.Sprintf("/api/compat/wakatime/v1/users/%s/all_time_since_today", user)
	resp, err := w.c.R().SetContext(ctx).Get(path)
	if err != nil {
		return nil, fmt.Errorf("wakapi all time: %w", err)
	}
	if err := checkStatus(resp, "all time"); err != nil {
		return nil, err
	}
	var result AllTimeResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi all time decode: %w", err)
	}
	return &result.Data, nil
}

// ListHeartbeats returns heartbeats for a user on a given date (YYYY-MM-DD).
func (w *Wakapi) ListHeartbeats(ctx context.Context, params HeartbeatsParams) (*HeartbeatsResult, error) {
	user := params.User
	if user == "" {
		user = CurrentUser
	}
	if params.Date == "" {
		return nil, fmt.Errorf("wakapi heartbeats: date required")
	}
	path := fmt.Sprintf("/api/compat/wakatime/v1/users/%s/heartbeats", user)
	resp, err := w.c.R().
		SetContext(ctx).
		SetQueryParam("date", params.Date).
		Get(path)
	if err != nil {
		return nil, fmt.Errorf("wakapi heartbeats: %w", err)
	}
	if err := checkStatus(resp, "heartbeats"); err != nil {
		return nil, err
	}
	var result HeartbeatsResult
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi heartbeats decode: %w", err)
	}
	return &result, nil
}

// ListProjects returns tracked projects (WakaTime-compatible).
func (w *Wakapi) ListProjects(ctx context.Context, query string) ([]Project, error) {
	req := w.c.R().SetContext(ctx)
	if query != "" {
		req.SetQueryParam("q", query)
	}
	resp, err := req.Get("/api/compat/wakatime/v1/users/current/projects")
	if err != nil {
		return nil, fmt.Errorf("wakapi list projects: %w", err)
	}
	if err := checkStatus(resp, "list projects"); err != nil {
		return nil, err
	}
	var result ProjectsResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi list projects decode: %w", err)
	}
	return result.Data, nil
}

func applySummaryQuery(req *resty.Request, p SummaryParams) {
	if p.Interval != "" {
		req.SetQueryParam("interval", p.Interval)
	}
	if p.From != "" {
		req.SetQueryParam("from", p.From)
	}
	if p.To != "" {
		req.SetQueryParam("to", p.To)
	}
	if p.Recompute != nil {
		req.SetQueryParam("recompute", strconv.FormatBool(*p.Recompute))
	}
	if p.Project != "" {
		req.SetQueryParam("project", p.Project)
	}
	if p.Language != "" {
		req.SetQueryParam("language", p.Language)
	}
	if p.Editor != "" {
		req.SetQueryParam("editor", p.Editor)
	}
	if p.OperatingSystem != "" {
		req.SetQueryParam("operating_system", p.OperatingSystem)
	}
	if p.Machine != "" {
		req.SetQueryParam("machine", p.Machine)
	}
	if p.Label != "" {
		req.SetQueryParam("label", p.Label)
	}
}

func applyStatsQuery(req *resty.Request, p StatsParams) {
	if p.Project != "" {
		req.SetQueryParam("project", p.Project)
	}
	if p.Language != "" {
		req.SetQueryParam("language", p.Language)
	}
	if p.Editor != "" {
		req.SetQueryParam("editor", p.Editor)
	}
	if p.OperatingSystem != "" {
		req.SetQueryParam("operating_system", p.OperatingSystem)
	}
	if p.Machine != "" {
		req.SetQueryParam("machine", p.Machine)
	}
	if p.Label != "" {
		req.SetQueryParam("label", p.Label)
	}
}

func parseHealthStatus(raw string) *HealthStatus {
	status := &HealthStatus{Raw: strings.TrimSpace(raw)}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "app":
			status.AppOK = val == "1" || strings.EqualFold(val, "true")
		case "db":
			status.DBOK = val == "1" || strings.EqualFold(val, "true")
		}
	}
	return status
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("wakapi %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("wakapi %s: status %d", op, code)
}
