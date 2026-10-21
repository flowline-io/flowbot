package drone

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (mockService) Deploy(context.Context, DeployInput) (*Build, error) {
	return &Build{Number: 1, Namespace: "alice", Repo: "deploy"}, nil
}

func (mockService) HealthCheck(context.Context) (bool, error) { return true, nil }

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: mockService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: mockService{}, wantErr: false},
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

func TestRegisterOperations(t *testing.T) {
	require.NoError(t, Register("drone", mockService{}))
	desc, ok := hub.Default.Get(hub.CapDrone)
	require.True(t, ok)
	assert.Equal(t, hub.CapDrone, desc.Type)
	assert.Equal(t, "drone", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 2)

	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	assert.Contains(t, opNames, OpDeploy)
	assert.Contains(t, opNames, OpHealth)

	res, err := invokeHealth(mockService{})(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, true, res.Data)

	_, err = invokeDeploy(mockService{}, "drone")(context.Background(), map[string]any{"uid": "u"})
	require.NoError(t, err)
}
