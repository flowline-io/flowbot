package utils

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"resty.dev/v3"
)

// httpTransport is a shared HTTP transport with tuned connection pool settings
// for inter-service provider calls. Cloned from http.DefaultTransport to preserve
// default TLS, proxy, and dial settings while overriding pool limits.
var httpTransport = func() *http.Transport {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		}
	}
	tr := t.Clone()
	tr.MaxIdleConns = 100
	tr.MaxIdleConnsPerHost = 10
	return tr
}()

// HTTPTransport returns the shared HTTP transport with connection pool tuning.
// Provider implementations that create raw http.Client instances should use this
// instead of http.DefaultTransport to ensure consistent pool behavior across all
// outgoing provider calls.
func HTTPTransport() *http.Transport {
	return httpTransport
}

// detachBodyTransport copies the request body before the underlying RoundTrip
// so pooled buffers (e.g. resty bodyBuf) can be recycled without racing the
// net/http transport writeLoop.
type detachBodyTransport struct {
	base http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *detachBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	detachHTTPRequestBody(req)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// detachHTTPRequestBody replaces req.Body with an owned copy of its bytes when
// net/http attached GetBody (typically *bytes.Buffer / *bytes.Reader).
// Resty v3 passes a pooled *bytes.Buffer as http.Request.Body and returns that
// buffer to sync.Pool when Execute finishes; the transport writeLoop may still
// read it, which races under parallel provider calls (-race CI).
// Streaming bodies (plain io.Reader, GetBody unset) are left untouched.
func detachHTTPRequestBody(req *http.Request) {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return
	}
	if req.GetBody == nil {
		return
	}
	data, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(nil))
		req.ContentLength = 0
		req.GetBody = nil
		return
	}
	owned := data
	req.Body = io.NopCloser(bytes.NewReader(owned))
	req.ContentLength = int64(len(owned))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(owned)), nil
	}
}

func DefaultRestyClient() *resty.Client {
	c := resty.New()
	c.SetLoggerWarnLevel(true)
	c.SetTimeout(time.Minute)
	// Copy bodies before otelhttp/net/http so resty's pooled bodyBuf cannot race
	// with writeLoop when recycled under concurrent requests.
	c.SetTransport(&detachBodyTransport{base: otelhttp.NewTransport(httpTransport)})
	c.AddContentTypeEncoder("json", EncodeJSON)
	c.AddContentTypeDecoder("json", DecodeJSON)

	return c
}

// RestyClientWithTrace returns a resty client configured with OTel HTTP tracing.
// It is an alias of DefaultRestyClient; providers should call SetContext(ctx) on
// individual requests so client spans nest under the caller's span.
func RestyClientWithTrace() *resty.Client {
	return DefaultRestyClient()
}

func EncodeJSON(w io.Writer, v any) error {
	return EncodeJSONEscapeHTML(w, v, true)
}

func EncodeJSONEscapeHTML(w io.Writer, v any, esc bool) error {
	enc := sonic.Config{EscapeHTML: esc}.Froze().NewEncoder(w)
	return enc.Encode(v)
}

func EncodeJSONEscapeHTMLIndent(w io.Writer, v any, esc bool, indent string) error {
	enc := sonic.Config{EscapeHTML: esc}.Froze().NewEncoder(w)
	enc.SetIndent("", indent)
	return enc.Encode(v)
}

func DecodeJSON(r io.Reader, v any) error {
	dec := sonic.ConfigStd.NewDecoder(r)
	for {
		if err := dec.Decode(v); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}
