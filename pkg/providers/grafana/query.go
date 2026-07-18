package grafana

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
)

// BackendKind identifies an observability backend queried through Grafana datasources.
type BackendKind string

const (
	BackendPrometheus BackendKind = "prometheus"
	BackendLoki       BackendKind = "loki"
	BackendTempo      BackendKind = "tempo"
	BackendPyroscope  BackendKind = "pyroscope"
	BackendAlloy      BackendKind = "alloy"
)

// QueryRequest holds parameters for a Grafana datasource query.
type QueryRequest struct {
	Backend       BackendKind
	Expr          string
	DatasourceUID string
	From          string // Grafana time, e.g. "now-1h"
	To            string // Grafana time, e.g. "now"
	MaxLines      int    // Loki max lines; ignored for other backends
}

// QueryResult is a simplified datasource query response.
type QueryResult struct {
	Backend        BackendKind    `json:"backend"`
	DatasourceUID  string         `json:"datasource_uid"`
	DatasourceType string         `json:"datasource_type"`
	Frames         []QueryFrame   `json:"frames,omitzero"`
	Raw            map[string]any `json:"raw,omitzero"`
}

// QueryFrame is one data frame summary from a query result.
type QueryFrame struct {
	Name   string            `json:"name,omitzero"`
	RefID  string            `json:"ref_id,omitzero"`
	Fields []QueryFrameField `json:"fields,omitzero"`
}

// QueryFrameField summarizes a frame field.
type QueryFrameField struct {
	Name   string `json:"name"`
	Type   string `json:"type,omitzero"`
	Values any    `json:"values,omitzero"`
}

var backendDatasourceTypes = map[BackendKind][]string{
	BackendPrometheus: {"prometheus"},
	BackendAlloy:      {"prometheus"}, // Alloy metrics are typically exposed via a Prometheus datasource
	BackendLoki:       {"loki"},
	BackendTempo:      {"tempo"},
	BackendPyroscope:  {"grafana-pyroscope", "phlare", "pyroscope"},
}

// Query runs a query against a Grafana-managed datasource for the given backend.
// When DatasourceUID is empty, the first matching datasource type is used.
func (g *Grafana) Query(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if req.Expr == "" {
		return nil, fmt.Errorf("grafana query: expr is required")
	}
	dsType, err := resolveDatasourceType(req.Backend)
	if err != nil {
		return nil, err
	}
	uid := req.DatasourceUID
	if uid == "" {
		uid, err = g.findDatasourceUID(ctx, req.Backend)
		if err != nil {
			return nil, err
		}
	}
	body, err := buildDSQueryBody(req, dsType, uid)
	if err != nil {
		return nil, err
	}
	resp, err := g.c.R().SetContext(ctx).SetBody(body).Post("/api/ds/query")
	if err != nil {
		return nil, fmt.Errorf("grafana query: %w", err)
	}
	if err := checkStatus(resp, "query"); err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := sonic.Unmarshal(resp.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("grafana query decode: %w", err)
	}
	return &QueryResult{
		Backend:        req.Backend,
		DatasourceUID:  uid,
		DatasourceType: dsType,
		Frames:         extractFrames(raw),
		Raw:            raw,
	}, nil
}

func buildDSQueryBody(req QueryRequest, dsType, uid string) (map[string]any, error) {
	from := req.From
	if from == "" {
		from = "now-1h"
	}
	to := req.To
	if to == "" {
		to = "now"
	}
	query, err := buildBackendQuery(req, dsType, uid)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"from":    from,
		"to":      to,
		"queries": []map[string]any{query},
	}, nil
}

func buildBackendQuery(req QueryRequest, dsType, uid string) (map[string]any, error) {
	query := map[string]any{
		"refId": "A",
		"datasource": map[string]any{
			"type": dsType,
			"uid":  uid,
		},
	}
	switch req.Backend {
	case BackendPrometheus, BackendAlloy:
		query["expr"] = req.Expr
		query["instant"] = true
	case BackendLoki:
		query["expr"] = req.Expr
		query["queryType"] = "range"
		maxLines := req.MaxLines
		if maxLines <= 0 {
			maxLines = 100
		}
		query["maxLines"] = maxLines
	case BackendTempo:
		query["query"] = req.Expr
		query["queryType"] = "traceql"
	case BackendPyroscope:
		query["labelSelector"] = req.Expr
		query["queryType"] = "profile"
		query["profileTypeId"] = "process_cpu:cpu:nanoseconds:cpu:nanoseconds"
	default:
		return nil, fmt.Errorf("grafana query: unsupported backend %q", req.Backend)
	}
	return query, nil
}

func resolveDatasourceType(backend BackendKind) (string, error) {
	types, ok := backendDatasourceTypes[backend]
	if !ok || len(types) == 0 {
		return "", fmt.Errorf("grafana query: unsupported backend %q", backend)
	}
	return types[0], nil
}

func (g *Grafana) findDatasourceUID(ctx context.Context, backend BackendKind) (string, error) {
	wanted, ok := backendDatasourceTypes[backend]
	if !ok {
		return "", fmt.Errorf("grafana query: unsupported backend %q", backend)
	}
	list, err := g.ListDatasources(ctx)
	if err != nil {
		return "", err
	}
	wantedSet := make(map[string]struct{}, len(wanted))
	for _, t := range wanted {
		wantedSet[strings.ToLower(t)] = struct{}{}
	}
	hint := strings.ToLower(string(backend))
	var fallback string
	for _, ds := range list {
		if _, match := wantedSet[strings.ToLower(ds.Type)]; !match {
			continue
		}
		if fallback == "" {
			fallback = ds.UID
		}
		if strings.Contains(strings.ToLower(ds.Name), hint) || strings.Contains(strings.ToLower(ds.UID), hint) {
			return ds.UID, nil
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("grafana query: no %s datasource configured", backend)
}

func extractFrames(raw map[string]any) []QueryFrame {
	results, ok := raw["results"].(map[string]any)
	if !ok {
		return nil
	}
	var frames []QueryFrame
	for refID, entry := range results {
		frames = append(frames, extractEntryFrames(refID, entry)...)
	}
	return frames
}

func extractEntryFrames(refID string, entry any) []QueryFrame {
	em, ok := entry.(map[string]any)
	if !ok {
		return nil
	}
	frameList, ok := em["frames"].([]any)
	if !ok {
		return nil
	}
	out := make([]QueryFrame, 0, len(frameList))
	for _, f := range frameList {
		if qf, ok := parseQueryFrame(refID, f); ok {
			out = append(out, qf)
		}
	}
	return out
}

func parseQueryFrame(refID string, f any) (QueryFrame, bool) {
	fm, ok := f.(map[string]any)
	if !ok {
		return QueryFrame{}, false
	}
	schema, _ := mapAny(fm["schema"])
	name, _ := asString(schema["name"])
	fieldsRaw, _ := sliceAny(schema["fields"])
	data, _ := mapAny(fm["data"])
	values, _ := sliceAny(data["values"])
	qf := QueryFrame{Name: name, RefID: refID}
	for i, fr := range fieldsRaw {
		fieldMap, ok := fr.(map[string]any)
		if !ok {
			continue
		}
		fname, _ := asString(fieldMap["name"])
		ftype, _ := asString(fieldMap["type"])
		var vals any
		if i < len(values) {
			vals = values[i]
		}
		qf.Fields = append(qf.Fields, QueryFrameField{Name: fname, Type: ftype, Values: vals})
	}
	return qf, true
}

func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func mapAny(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}, false
	}
	return m, true
}

func sliceAny(v any) ([]any, bool) {
	s, ok := v.([]any)
	if !ok {
		return nil, false
	}
	return s, true
}
