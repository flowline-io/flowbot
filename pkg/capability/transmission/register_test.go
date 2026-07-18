package transmission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (*mockService) AddTorrent(_ context.Context, _ AddTorrentInput) (*capability.Torrent, error) {
	return &capability.Torrent{ID: 1, Name: "a"}, nil
}
func (*mockService) ListTorrents(_ context.Context) ([]*capability.Torrent, error) {
	return []*capability.Torrent{{ID: 1}}, nil
}
func (*mockService) StopTorrents(_ context.Context, _ StopTorrentsInput) error { return nil }
func (*mockService) RemoveTorrents(_ context.Context, _ RemoveTorrentsInput) error {
	return nil
}
func (*mockService) HealthCheck(_ context.Context) (bool, error) { return true, nil }

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockService{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.app, tt.svc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister_Operations(t *testing.T) {
	require.NoError(t, Register("transmission", &mockService{}))
	desc, ok := hub.Default.Get(hub.CapTransmission)
	require.True(t, ok)
	assert.Equal(t, hub.CapTransmission, desc.Type)
	assert.Equal(t, "transmission", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 5)

	tests := []struct {
		name string
		op   string
	}{
		{"has add", OpAdd},
		{"has list", OpList},
		{"has stop", OpStop},
		{"has remove", OpRemove},
		{"has health", OpHealth},
	}
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, opNames, tt.op)
		})
	}
}

func TestInvokeAddStop(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		invoke  capability.Invoker
	}{
		{
			name:   "add success",
			params: map[string]any{"url": "magnet:?xt=urn:btih:abc"},
			invoke: invokeAdd(&mockService{}, "transmission"),
		},
		{
			name:    "add missing url",
			params:  map[string]any{},
			wantErr: true,
			invoke:  invokeAdd(&mockService{}, "transmission"),
		},
		{
			name:   "stop success",
			params: map[string]any{"ids": []any{float64(1), float64(2)}},
			invoke: invokeStop(&mockService{}),
		},
		{
			name:    "stop missing ids",
			params:  map[string]any{},
			wantErr: true,
			invoke:  invokeStop(&mockService{}),
		},
		{
			name:   "remove success",
			params: map[string]any{"ids": []int64{9}},
			invoke: invokeRemove(&mockService{}),
		},
		{
			name:   "list success",
			params: map[string]any{},
			invoke: invokeList(&mockService{}),
		},
		{
			name:   "health success",
			params: map[string]any{},
			invoke: invokeHealth(&mockService{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := tt.invoke(context.Background(), tt.params)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.NotNil(t, res.Data)
		})
	}
}

func TestRequiredInt64Slice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  map[string]any
		want    []int64
		wantErr bool
	}{
		{name: "int64 slice", params: map[string]any{"ids": []int64{1, 2}}, want: []int64{1, 2}},
		{name: "any float slice", params: map[string]any{"ids": []any{float64(3)}}, want: []int64{3}},
		{name: "missing", params: map[string]any{}, wantErr: true},
		{name: "empty", params: map[string]any{"ids": []int64{}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := requiredInt64Slice(tt.params, "ids")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
