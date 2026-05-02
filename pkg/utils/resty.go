package utils

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"resty.dev/v3"
)

func DefaultRestyClient() *resty.Client {
	c := resty.New()
	c.SetDisableWarn(true)
	c.SetTimeout(time.Minute)
	c.AddContentTypeEncoder("json", EncodeJSON)
	c.AddContentTypeDecoder("json", DecodeJSON)

	return c
}

// RestyClientWithTrace returns a resty client configured with OTel HTTP tracing.
// The underlying http.Transport is wrapped with otelhttp.NewTransport which automatically
// propagates W3C TraceContext headers and creates client spans for outgoing requests.
//
// Providers should use SetContext(ctx) on individual requests to ensure the span context
// is propagated from the caller's context.
func RestyClientWithTrace() *resty.Client {
	c := DefaultRestyClient()
	c.SetTransport(otelhttp.NewTransport(http.DefaultTransport))
	return c
}
