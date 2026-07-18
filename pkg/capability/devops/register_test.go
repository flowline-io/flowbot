package devops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "devops", svc: nil, wantErr: false},
		{name: "valid service", app: "devops", svc: NewWithClients(Clients{Dozzle: &stubDozzle{}}), wantErr: false},
		{name: "empty app with valid service", app: "", svc: NewWithClients(Clients{Grafana: &stubGrafana{}}), wantErr: false},
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
	require.NoError(t, Register("devops", NewWithClients(Clients{Dozzle: &stubDozzle{}})))
	desc, ok := hub.Default.Get(hub.CapDevops)
	require.True(t, ok)
	assert.Equal(t, hub.CapDevops, desc.Type)
	assert.NotEmpty(t, desc.Operations)

	tests := []struct {
		name string
		op   string
	}{
		{name: "status", op: OpStatus},
		{name: "beszel list", op: OpBeszelListSystems},
		{name: "grafana health", op: OpGrafanaHealth},
		{name: "dozzle health", op: OpDozzleHealth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, op := range desc.Operations {
				assert.NotContains(t, op.Name, ".")
				if op.Name == tt.op {
					found = true
				}
			}
			assert.True(t, found, "missing op %s", tt.op)
		})
	}
}
