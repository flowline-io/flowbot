package utils

import (
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

func DefaultRestyClient() *resty.Client {
	c := resty.New()
	c.SetLoggerWarnLevel(true)
	c.SetTimeout(time.Minute)
	c.SetTransport(otelhttp.NewTransport(httpTransport))
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
