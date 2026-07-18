package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeQueryBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	assert.NoError(t, sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body))
	return body
}

func firstQuery(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	queries, ok := body["queries"].([]any)
	if !assert.True(t, ok) || !assert.NotEmpty(t, queries) {
		return map[string]any{}
	}
	q0, ok := queries[0].(map[string]any)
	if !assert.True(t, ok) {
		return map[string]any{}
	}
	return q0
}

func TestGrafana_Query(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     QueryRequest
		handler http.HandlerFunc
		wantErr bool
		check   func(t *testing.T, got *QueryResult)
	}{
		{
			name: "prometheus instant query",
			req:  QueryRequest{Backend: BackendPrometheus, Expr: "up"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/datasources":
					_ = sonic.ConfigDefault.NewEncoder(w).Encode([]Datasource{
						{UID: "prom1", Name: "Prometheus", Type: "prometheus"},
					})
				case "/api/ds/query":
					assert.Equal(t, http.MethodPost, r.Method)
					q0 := firstQuery(t, decodeQueryBody(t, r))
					assert.Equal(t, "up", q0["expr"])
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]any{
						"results": map[string]any{
							"A": map[string]any{
								"frames": []any{
									map[string]any{
										"schema": map[string]any{
											"name": "up",
											"fields": []any{
												map[string]any{"name": "Time", "type": "time"},
												map[string]any{"name": "Value", "type": "number"},
											},
										},
										"data": map[string]any{
											"values": []any{[]any{1}, []any{1}},
										},
									},
								},
							},
						},
					})
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
			check: func(t *testing.T, got *QueryResult) {
				assert.Equal(t, BackendPrometheus, got.Backend)
				assert.Equal(t, "prom1", got.DatasourceUID)
				require.Len(t, got.Frames, 1)
				assert.Equal(t, "up", got.Frames[0].Name)
			},
		},
		{
			name: "alloy prefers named datasource",
			req:  QueryRequest{Backend: BackendAlloy, Expr: "up{job=\"alloy\"}"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/datasources":
					_ = sonic.ConfigDefault.NewEncoder(w).Encode([]Datasource{
						{UID: "prom1", Name: "Prometheus", Type: "prometheus"},
						{UID: "alloy1", Name: "Alloy", Type: "prometheus"},
					})
				case "/api/ds/query":
					q0 := firstQuery(t, decodeQueryBody(t, r))
					ds, ok := q0["datasource"].(map[string]any)
					assert.True(t, ok)
					assert.Equal(t, "alloy1", ds["uid"])
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]any{"results": map[string]any{}})
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
			check: func(t *testing.T, got *QueryResult) {
				assert.Equal(t, "alloy1", got.DatasourceUID)
			},
		},
		{
			name:    "missing expr",
			req:     QueryRequest{Backend: BackendLoki},
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
			wantErr: true,
		},
		{
			name: "loki query",
			req:  QueryRequest{Backend: BackendLoki, Expr: `{job="varlogs"}`, DatasourceUID: "loki1"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/ds/query", r.URL.Path)
				q0 := firstQuery(t, decodeQueryBody(t, r))
				assert.Equal(t, "range", q0["queryType"])
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]any{"results": map[string]any{}})
			},
			check: func(t *testing.T, got *QueryResult) {
				assert.Equal(t, BackendLoki, got.Backend)
				assert.Equal(t, "loki1", got.DatasourceUID)
			},
		},
		{
			name: "tempo query",
			req:  QueryRequest{Backend: BackendTempo, Expr: `{status=error}`, DatasourceUID: "tempo1"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				q0 := firstQuery(t, decodeQueryBody(t, r))
				assert.Equal(t, "traceql", q0["queryType"])
				assert.Equal(t, `{status=error}`, q0["query"])
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]any{"results": map[string]any{}})
			},
			check: func(t *testing.T, got *QueryResult) {
				assert.Equal(t, BackendTempo, got.Backend)
			},
		},
		{
			name: "pyroscope query",
			req:  QueryRequest{Backend: BackendPyroscope, Expr: `{service_name="flowbot"}`, DatasourceUID: "pyro1"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				q0 := firstQuery(t, decodeQueryBody(t, r))
				assert.Equal(t, "profile", q0["queryType"])
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]any{"results": map[string]any{}})
			},
			check: func(t *testing.T, got *QueryResult) {
				assert.Equal(t, BackendPyroscope, got.Backend)
			},
		},
		{
			name: "unsupported backend",
			req:  QueryRequest{Backend: BackendKind("unknown"), Expr: "x"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			got, err := NewGrafana(server.URL, "tok").Query(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
