package chatagent

import "context"

type runIOKey struct{}

// runIO carries per-run SSE publisher and confirm gate on the request context
// so pooled harness hooks see the current turn without session-map lookups.
type runIO struct {
	Publisher EventPublisher
	Confirm   *ConfirmGate
}

func withRunIO(ctx context.Context, api *APIRunOptions) context.Context {
	if api == nil || (api.Publisher == nil && api.Confirm == nil) {
		return ctx
	}
	return context.WithValue(ctx, runIOKey{}, &runIO{
		Publisher: api.Publisher,
		Confirm:   api.Confirm,
	})
}

func runIOFromContext(ctx context.Context) *runIO {
	if ctx == nil {
		return nil
	}
	io, ok := ctx.Value(runIOKey{}).(*runIO)
	if !ok {
		return nil
	}
	return io
}
