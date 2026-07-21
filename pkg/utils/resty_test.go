package utils

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestDefaultRestyClient(t *testing.T) {
	t.Parallel()

	client := DefaultRestyClient()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "returns non-nil client",
			fn: func(t *testing.T) {
				t.Parallel()
				require.NotNil(t, client, "DefaultRestyClient() returned nil")
			},
		},
		{
			name: "can create requests",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				assert.NotNil(t, req, "DefaultRestyClient() client.R() returned nil")
			},
		},
		{
			name: "request has headers initialized",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				assert.NotNil(t, req.Header, "DefaultRestyClient() request headers are nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestDefaultRestyClientTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		expectError bool
	}{
		{
			name:        "nanosecond timeout triggers error immediately",
			timeout:     1 * time.Nanosecond,
			expectError: true,
		},
		{
			name:        "default timeout does not prevent request creation",
			timeout:     0,
			expectError: false,
		},
		{
			name:        "moderate timeout still allows request creation",
			timeout:     30 * time.Second,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := DefaultRestyClient()
			if tt.timeout > 0 {
				client.SetTimeout(tt.timeout)
			}

			if tt.expectError {
				start := time.Now()
				_, err := client.R().Get("http://example.com/")
				elapsed := time.Since(start)

				require.Error(t, err, "Expected timeout error but got nil")
				assert.LessOrEqual(t, elapsed, 5*time.Second, "Request took too long: %v, expected immediate timeout", elapsed)
			} else {
				assert.NotNil(t, client, "DefaultRestyClient should return valid client")
				req := client.R()
				assert.NotNil(t, req, "should be able to create requests with default timeout")
			}
		})
	}
}

func TestDefaultRestyClientHeaders(t *testing.T) {
	t.Parallel()

	client := DefaultRestyClient()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "can create request object",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				assert.NotNil(t, req, "DefaultRestyClient() should be able to create requests")
			},
		},
		{
			name: "can set and get custom header on request",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				req.SetHeader("Test-Header", "test-value")
				assert.Equal(t, "test-value", req.Header.Get("Test-Header"), "Should be able to set headers on requests")
			},
		},
		{
			name: "can override default headers",
			fn: func(t *testing.T) {
				t.Parallel()
				req := client.R()
				req.SetHeader("Content-Type", "application/json")
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"), "Should be able to override Content-Type header")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestHTTPTransport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "returns non-nil transport",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				require.NotNil(t, tr, "HTTPTransport() returned nil")
			},
		},
		{
			name: "has MaxIdleConnsPerHost set to 10",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				assert.Equal(t, 10, tr.MaxIdleConnsPerHost, "HTTPTransport() MaxIdleConnsPerHost should be 10")
			},
		},
		{
			name: "has MaxIdleConns set to 100",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				assert.Equal(t, 100, tr.MaxIdleConns, "HTTPTransport() MaxIdleConns should be 100")
			},
		},
		{
			name: "returns same instance on repeated calls",
			fn: func(t *testing.T) {
				t.Parallel()
				tr1 := HTTPTransport()
				tr2 := HTTPTransport()
				assert.Same(t, tr1, tr2, "HTTPTransport() should return the same instance")
			},
		},
		{
			name: "transport is a clone of http.DefaultTransport",
			fn: func(t *testing.T) {
				t.Parallel()
				tr := HTTPTransport()
				_, ok := http.DefaultTransport.(*http.Transport)
				require.True(t, ok, "http.DefaultTransport should be *http.Transport")
				require.NotNil(t, tr.TLSClientConfig, "TLS config should not be nil in clone")
				assert.Equal(t, []string{"h2", "http/1.1"}, tr.TLSClientConfig.NextProtos, "clone should preserve NextProtos from DefaultTransport")
				assert.NotNil(t, tr.Proxy, "Proxy func should be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

// TestEncodeJSON tests the EncodeJSON function
func TestEncodeJSON(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "simple_struct",
			input: testStruct{
				Name: "John",
				Age:  30,
			},
			wantErr: false,
		},
		{
			name:    "string_input",
			input:   "test string",
			wantErr: false,
		},
		{
			name:    "number_input",
			input:   123,
			wantErr: false,
		},
		{
			name:    "map_input",
			input:   map[string]string{"key": "value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := EncodeJSON(&buf, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, buf.Len(), "EncodeJSON() produced empty output")
		})
	}
}

// TestEncodeJSONEscapeHTML tests the EncodeJSONEscapeHTML function
func TestEncodeJSONEscapeHTML(t *testing.T) {
	t.Parallel()
	type args struct {
		v   any
		esc bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		check   func(string) bool
	}{
		{
			name: "escape_html_true",
			args: args{
				v:   map[string]string{"html": "<script>alert('xss')</script>"},
				esc: true,
			},
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(output, "\\u003cscript\\u003e")
			},
		},
		{
			name: "escape_html_false",
			args: args{
				v:   map[string]string{"html": "<script>alert('xss')</script>"},
				esc: false,
			},
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(output, "<script>")
			},
		},
		{
			name: "simple_struct_no_html",
			args: args{
				v:   map[string]string{"name": "hello"},
				esc: true,
			},
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(output, `"name":"hello"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := EncodeJSONEscapeHTML(&buf, tt.args.v, tt.args.esc)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			output := buf.String()
			if tt.check != nil {
				assert.True(t, tt.check(output), "EncodeJSONEscapeHTML() output check failed: %s", output)
			}
		})
	}
}

// TestEncodeJSONEscapeHTMLIndent tests the EncodeJSONEscapeHTMLIndent function
func TestEncodeJSONEscapeHTMLIndent(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		v       any
		esc     bool
		indent  string
		wantErr bool
	}{
		{
			name: "indent_with_spaces",
			v: testStruct{
				Name: "John",
				Age:  30,
			},
			esc:     true,
			indent:  "  ",
			wantErr: false,
		},
		{
			name: "indent_with_tabs",
			v: map[string]any{
				"key1": "value1",
				"key2": 123,
			},
			esc:     false,
			indent:  "\t",
			wantErr: false,
		},
		{
			name:    "empty_indent_compact",
			v:       map[string]any{"key": "value"},
			esc:     true,
			indent:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := EncodeJSONEscapeHTMLIndent(&buf, tt.v, tt.esc, tt.indent)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			output := buf.String()
			if tt.indent == "  " {
				assert.Contains(t, output, "  ", "EncodeJSONEscapeHTMLIndent() should contain indentation")
			}
		})
	}
}

// TestDecodeJSON tests the DecodeJSON function
func TestDecodeJSON(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		input   string
		target  any
		wantErr bool
	}{
		{
			name:    "valid_json",
			input:   `{"name":"John","age":30}`,
			target:  &testStruct{},
			wantErr: false,
		},
		{
			name:    "invalid_json",
			input:   `{"name":"John","age":}`,
			target:  &testStruct{},
			wantErr: true,
		},
		{
			name:    "empty_json",
			input:   `{}`,
			target:  &testStruct{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reader := strings.NewReader(tt.input)
			err := DecodeJSON(reader, tt.target)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRestyClientWithTrace(t *testing.T) {
	tests := []struct {
		name            string
		setContext      bool
		wantTraceparent bool
		wantClientChild bool
		useTraceAlias   bool
	}{
		{name: "SetContext nests client under parent", setContext: true, wantTraceparent: true, wantClientChild: true},
		{name: "without SetContext still injects orphan client", setContext: false, wantTraceparent: true, wantClientChild: false},
		{name: "RestyClientWithTrace alias nests like DefaultRestyClient", setContext: true, wantTraceparent: true, wantClientChild: true, useTraceAlias: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			prevTP := otel.GetTracerProvider()
			prevProp := otel.GetTextMapPropagator()
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
			))
			t.Cleanup(func() {
				_ = tp.Shutdown(context.Background())
				otel.SetTracerProvider(prevTP)
				otel.SetTextMapPropagator(prevProp)
			})

			var gotTraceparent string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotTraceparent = r.Header.Get("traceparent")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(srv.Close)

			parentCtx, parent := tp.Tracer("test").Start(context.Background(), "parent")
			defer parent.End()

			client := DefaultRestyClient()
			if tt.useTraceAlias {
				client = RestyClientWithTrace()
			}
			req := client.R()
			if tt.setContext {
				req.SetContext(parentCtx)
			}
			resp, err := req.Get(srv.URL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode())

			if tt.wantTraceparent {
				assert.NotEmpty(t, gotTraceparent)
			} else {
				assert.Empty(t, gotTraceparent)
			}

			var clientSpan sdktrace.ReadOnlySpan
			for _, s := range recorder.Ended() {
				if s.SpanKind() == oteltrace.SpanKindClient {
					clientSpan = s
					break
				}
			}
			require.NotNil(t, clientSpan, "expected client span")
			if tt.wantClientChild {
				assert.Equal(t, parent.SpanContext().TraceID(), clientSpan.SpanContext().TraceID())
				assert.Equal(t, parent.SpanContext().SpanID(), clientSpan.Parent().SpanID())
			} else {
				assert.NotEqual(t, parent.SpanContext().SpanID().String(), clientSpan.Parent().SpanID().String())
			}
		})
	}
}

func TestDetachHTTPRequestBody(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setup      func() *http.Request
		wantCopied bool
		wantLen    int64
	}{
		{
			name: "nil request is no-op",
			setup: func() *http.Request {
				return nil
			},
		},
		{
			name: "nil body is no-op",
			setup: func() *http.Request {
				return &http.Request{Header: make(http.Header)}
			},
		},
		{
			name: "NoBody is no-op",
			setup: func() *http.Request {
				return &http.Request{Body: http.NoBody, Header: make(http.Header)}
			},
		},
		{
			name: "streaming reader without GetBody is preserved",
			setup: func() *http.Request {
				return &http.Request{
					Body:   io.NopCloser(strings.NewReader("stream")),
					Header: make(http.Header),
				}
			},
			wantCopied: false,
		},
		{
			name: "bytes buffer with GetBody is copied",
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "http://example.com", bytes.NewBufferString(`{"id":1}`))
				require.NoError(t, err)
				return req
			},
			wantCopied: true,
			wantLen:    8,
		},
		{
			name: "empty buffer becomes NoBody and is skipped",
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "http://example.com", bytes.NewBuffer(nil))
				require.NoError(t, err)
				return req
			},
			wantCopied: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := tt.setup()
			var beforeBody io.ReadCloser
			if req != nil {
				beforeBody = req.Body
			}
			detachHTTPRequestBody(req)
			if req == nil {
				return
			}
			if !tt.wantCopied {
				assert.Equal(t, beforeBody, req.Body)
				return
			}
			require.NotNil(t, req.Body)
			assert.NotEqual(t, beforeBody, req.Body)
			assert.Equal(t, tt.wantLen, req.ContentLength)
			got, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Len(t, got, int(tt.wantLen))
			require.NotNil(t, req.GetBody)
			rc, err := req.GetBody()
			require.NoError(t, err)
			again, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, got, again)
		})
	}
}

func TestDefaultRestyClientParallelJSONBodies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		body   any
	}{
		{name: "parallel post", method: http.MethodPost, body: map[string]any{"n": 1}},
		{name: "parallel patch", method: http.MethodPatch, body: map[string]any{"n": 2}},
		{name: "parallel delete with payload", method: http.MethodDelete, body: []map[string]any{{"Id": 3}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(server.Close)

			const workers = 32
			errCh := make(chan error, workers)
			for i := 0; i < workers; i++ {
				go func() {
					client := DefaultRestyClient()
					client.SetBaseURL(server.URL)
					client.SetMethodDeleteAllowPayload(true)
					req := client.R().SetBody(tt.body)
					var err error
					switch tt.method {
					case http.MethodPost:
						_, err = req.Post("/")
					case http.MethodPatch:
						_, err = req.Patch("/")
					case http.MethodDelete:
						_, err = req.Delete("/")
					}
					errCh <- err
				}()
			}
			for i := 0; i < workers; i++ {
				require.NoError(t, <-errCh)
			}
		})
	}
}
