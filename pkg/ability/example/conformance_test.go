package example

import (
	"context"
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type conformanceService struct {
	cfg Config
}

func checkCtx(ctx context.Context) error {
	return ctx.Err()
}

func (c *conformanceService) GetItem(ctx context.Context, id string) (*ability.Host, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.New("id must not be empty")
	}
	if c.cfg.GetErr != nil {
		return nil, c.cfg.GetErr
	}
	return c.cfg.GetItem, nil
}
func (c *conformanceService) ListItems(ctx context.Context, _ *ListQuery) (*ability.ListResult[ability.Host], error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if c.cfg.ListErr != nil {
		return nil, c.cfg.ListErr
	}
	return &ability.ListResult[ability.Host]{Items: c.cfg.ListItems}, nil
}
func (c *conformanceService) CreateItem(ctx context.Context, title string) (*ability.Host, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, errors.New("title must not be empty")
	}
	if c.cfg.CreateErr != nil {
		return nil, c.cfg.CreateErr
	}
	return c.cfg.CreateItem, nil
}
func (c *conformanceService) UpdateItem(ctx context.Context, _ string, _ map[string]any) (*ability.Host, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if c.cfg.UpdateErr != nil {
		return nil, c.cfg.UpdateErr
	}
	return c.cfg.UpdateItem, nil
}
func (c *conformanceService) DeleteItem(ctx context.Context, _ string) error {
	if err := checkCtx(ctx); err != nil {
		return err
	}
	return c.cfg.DeleteErr
}
func (c *conformanceService) HealthCheck(ctx context.Context) (bool, error) {
	if err := checkCtx(ctx); err != nil {
		return false, err
	}
	if c.cfg.HealthErr != nil {
		return false, c.cfg.HealthErr
	}
	return c.cfg.HealthOk, nil
}
func (c *conformanceService) ListRawEvents(ctx context.Context, _ string) ([]any, string, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, "", err
	}
	if c.cfg.RawErr != nil {
		return nil, "", c.cfg.RawErr
	}
	return c.cfg.RawItems, c.cfg.RawCursor, nil
}

func TestRunExampleConformance(t *testing.T) {
	t.Run("runs example conformance test suite", func(t *testing.T) {
		RunExampleConformance(t, func(_ *testing.T, cfg Config) Service {
			return &conformanceService{cfg: cfg}
		})
	})
}
