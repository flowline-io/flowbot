package example

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	example "github.com/flowline-io/flowbot/pkg/ability/example"
	provider "github.com/flowline-io/flowbot/pkg/providers/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

type conformanceWrapper struct {
	*Adapter
	cfg example.Config
}

func (w *conformanceWrapper) ListItems(_ context.Context, _ *example.ListQuery) (*ability.ListResult[ability.Host], error) {
	if w.cfg.ListErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", w.cfg.ListErr)
	}
	if w.cfg.ListItems != nil {
		return &ability.ListResult[ability.Host]{Items: w.cfg.ListItems}, nil
	}
	return w.Adapter.ListItems(context.Background(), &example.ListQuery{})
}

func (w *conformanceWrapper) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if w.cfg.HealthErr != nil {
		return false, types.WrapError(types.ErrProvider, "health check failed", w.cfg.HealthErr)
	}
	if !w.cfg.HealthOk && w.cfg.HealthErr == nil {
		return false, nil
	}
	return w.Adapter.HealthCheck(ctx)
}

func TestExampleConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "runs example conformance test suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			example.RunExampleConformance(t, func(_ *testing.T, cfg example.Config) example.Service {
				c := &fakeClient{
					getErr:    cfg.GetErr,
					postErr:   cfg.CreateErr,
					deleteErr: cfg.DeleteErr,
					statusErr: cfg.HealthErr,
				}
				if cfg.GetItem != nil {
					c.getResp = &provider.Response{Origin: cfg.GetItem.Name, URL: cfg.GetItem.Status}
				}
				if cfg.CreateItem != nil {
					c.postResp = &provider.Response{URL: "https://example.com"}
				}
				if cfg.HealthOk {
					c.statusResp = &provider.Response{}
				}
				if cfg.ListErr != nil {
					c.getErr = cfg.ListErr
				}
				if cfg.CreateErr != nil {
					c.postErr = cfg.CreateErr
				}
				if cfg.DeleteErr != nil {
					c.deleteErr = cfg.DeleteErr
				}
				if cfg.HealthErr != nil {
					c.statusErr = cfg.HealthErr
				}
				a, ok := NewWithClient(c).(*Adapter)
				if !ok {
					t.Fatal("unexpected type")
				}
				return &conformanceWrapper{Adapter: a, cfg: cfg}
			})
		})
	}
}

func TestConformance_FakeClient_ImplementsClient(t *testing.T) {
	t.Run("fakeClient satisfies client interface", func(_ *testing.T) {
		var _ client = (*fakeClient)(nil)
	})
}
