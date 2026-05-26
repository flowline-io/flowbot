package example

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	exsvc "github.com/flowline-io/flowbot/pkg/ability/example"
	provider "github.com/flowline-io/flowbot/pkg/providers/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

type conformanceWrapper struct {
	*Adapter
	cfg conformance.ExampleConfig
}

func (w *conformanceWrapper) ListItems(ctx context.Context, _ *exsvc.ListQuery) (*ability.ListResult[ability.Host], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if w.cfg.ListErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", w.cfg.ListErr)
	}
	if w.cfg.ListItems != nil {
		return &ability.ListResult[ability.Host]{Items: w.cfg.ListItems}, nil
	}
	return &ability.ListResult[ability.Host]{Items: []*ability.Host{}, Page: &ability.PageInfo{}}, nil
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

func newFakeClientFromConfig(cfg conformance.ExampleConfig) *fakeClient {
	c := &fakeClient{
		getErr:    cfg.GetErr,
		postErr:   cfg.CreateErr,
		putErr:    cfg.UpdateErr,
		deleteErr: cfg.DeleteErr,
		statusErr: cfg.HealthErr,
	}
	if cfg.GetItem != nil {
		c.getResp = &provider.Response{Title: cfg.GetItem.Name, Body: cfg.GetItem.Status}
	}
	if cfg.CreateItem != nil {
		c.postResp = &provider.Response{ID: 101}
	}
	if cfg.UpdateItem != nil {
		c.putResp = &provider.Response{Title: cfg.UpdateItem.Name, Body: cfg.UpdateItem.Status}
	}
	return c
}

func applyConditionalConfig(c *fakeClient, cfg conformance.ExampleConfig) {
	if cfg.HealthOk {
		c.statusResp = &provider.Response{}
	}
	if cfg.ListErr != nil {
		c.getErr = cfg.ListErr
	}
	if cfg.CreateErr != nil {
		c.postErr = cfg.CreateErr
	}
	if cfg.UpdateErr != nil {
		c.putErr = cfg.UpdateErr
	}
	if cfg.DeleteErr != nil {
		c.deleteErr = cfg.DeleteErr
	}
	if cfg.HealthErr != nil {
		c.statusErr = cfg.HealthErr
	}
	if cfg.RawErr != nil {
		c.listRawErr = cfg.RawErr
	}
	if cfg.RawItems != nil {
		items := make([]map[string]any, 0, len(cfg.RawItems))
		for _, item := range cfg.RawItems {
			if m, ok := item.(map[string]any); ok {
				items = append(items, m)
			}
		}
		c.listRawResp = items
	}
	if cfg.RawCursor != "" {
		c.listRawNext = cfg.RawCursor
	}
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
			conformance.RunExampleConformance(t, func(_ *testing.T, cfg conformance.ExampleConfig) exsvc.Service {
				c := newFakeClientFromConfig(cfg)
				applyConditionalConfig(c, cfg)
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
