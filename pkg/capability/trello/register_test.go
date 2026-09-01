package trello

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
	assert.Equal(t, hub.CapTrello, spec.Type)
	assert.NotEmpty(t, spec.Ops)
}

func TestRegisterNilServiceSkips(t *testing.T) {
	t.Parallel()
	err := Register("trello-unconfigured", nil)
	require.NoError(t, err)
}

func TestRegisterWithMockService(t *testing.T) {
	svc := &mockService{}
	err := Register("trello", svc)
	require.NoError(t, err)
	desc, ok := hub.Default.Get(hub.CapTrello)
	require.True(t, ok)
	assert.Equal(t, hub.CapTrello, desc.Type)
}

type mockService struct{}

func (*mockService) ListBoards(_ context.Context, _ *ListQuery) (*capability.ListResult[capability.TrelloBoard], error) {
	return nil, nil
}
func (*mockService) GetBoard(_ context.Context, _ string) (*capability.TrelloBoard, error) {
	return nil, nil
}
func (*mockService) ListLists(_ context.Context, _ string) ([]*capability.TrelloList, error) {
	return nil, nil
}
func (*mockService) ListCards(_ context.Context, _ string, _ *ListQuery) (*capability.ListResult[capability.TrelloCard], error) {
	return nil, nil
}
func (*mockService) GetCard(_ context.Context, _ string) (*capability.TrelloCard, error) {
	return nil, nil
}
func (*mockService) SearchCards(_ context.Context, _ string, _ int) ([]*capability.TrelloCard, error) {
	return nil, nil
}
func (*mockService) CreateCard(_ context.Context, _, _, _ string) (*capability.TrelloCard, error) {
	return nil, nil
}
func (*mockService) UpdateCard(_ context.Context, _, _, _ string) (*capability.TrelloCard, error) {
	return nil, nil
}
func (*mockService) MoveCard(_ context.Context, _, _ string) (*capability.TrelloCard, error) {
	return nil, nil
}
func (*mockService) DeleteCard(_ context.Context, _ string) error { return nil }
func (*mockService) RegisterWebhook(_ context.Context, _, _, _ string) (*capability.TrelloWebhook, error) {
	return nil, nil
}
func (*mockService) DeleteWebhook(_ context.Context, _ string) error { return nil }
func (*mockService) HealthCheck(_ context.Context) (bool, error)     { return true, nil }
