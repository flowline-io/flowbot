package confluence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestCatalogSpec(t *testing.T) {
	t.Parallel()
	spec := CatalogSpec()
	assert.Equal(t, hub.CapConfluence, spec.Type)
	assert.NotEmpty(t, spec.Ops)
}

func TestRegisterNilServiceSkips(t *testing.T) {
	t.Parallel()
	err := Register("confluence-unconfigured", nil)
	require.NoError(t, err)
}

func TestRegisterWithMockService(t *testing.T) {
	svc := &mockService{}
	err := Register("confluence", svc)
	require.NoError(t, err)
	desc, ok := hub.Default.Get(hub.CapConfluence)
	require.True(t, ok)
	assert.Equal(t, hub.CapConfluence, desc.Type)
}

type mockService struct{}

func (*mockService) ListSpaces(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.ConfluenceSpace], error) {
	return nil, nil
}
func (*mockService) ListPages(_ context.Context, _ string, _ *ListQuery) (*capability.ListResult[capability.ConfluencePage], error) {
	return nil, nil
}
func (*mockService) GetPage(_ context.Context, _ string) (*capability.ConfluencePage, error) {
	return nil, nil
}
func (*mockService) GetPageContent(_ context.Context, _ string) (string, error) { return "", nil }
func (*mockService) SearchPages(_ context.Context, _ string, _ *ListQuery) (*capability.ListResult[capability.ConfluencePage], error) {
	return nil, nil
}
func (*mockService) CreatePage(_ context.Context, _, _, _ string) (*capability.ConfluencePage, error) {
	return nil, nil
}
func (*mockService) UpdatePage(_ context.Context, _, _, _ string) (*capability.ConfluencePage, error) {
	return nil, nil
}
func (*mockService) DeletePage(_ context.Context, _ string) error { return nil }
func (*mockService) HealthCheck(_ context.Context) (bool, error)  { return true, nil }
