package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestNewChecker(t *testing.T) {
	t.Parallel()

	t.Run("creates non-nil checker", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		checker := NewChecker(registry)
		require.NotNil(t, checker)
	})
}

func TestChecker_Check(t *testing.T) {
	tests := []struct {
		name        string
		descriptors []Descriptor
		setup       func() func()
		check       func(*testing.T, *HealthResult)
	}{
		{
			name:        "empty registry",
			descriptors: nil,
			setup: func() func() {
				old := homelab.DefaultRegistry.Permissions()
				return func() { homelab.DefaultRegistry.SetPermissions(old) }
			},
			check: func(t *testing.T, result *HealthResult) {
				require.NotNil(t, result)
				assert.Equal(t, HealthDegraded, result.Status)
				assert.Empty(t, result.Details)
				assert.False(t, result.Timestamp.IsZero())
			},
		},
		{
			name: "all healthy descriptors",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Description: "bookmark service", Healthy: true, Instance: "ok"},
				{Type: CapArchive, Backend: "archivebox", App: "archivebox", Description: "archive service", Healthy: true, Instance: "ok"},
			},
			setup: func() func() {
				old := homelab.DefaultRegistry.Permissions()
				return func() { homelab.DefaultRegistry.SetPermissions(old) }
			},
			check: func(t *testing.T, result *HealthResult) {
				require.NotNil(t, result)
				assert.Equal(t, HealthHealthy, result.Status)
				require.Len(t, result.Details, 2)

				healthByType := make(map[CapabilityType]HealthStatus)
				for _, d := range result.Details {
					healthByType[d.Type] = d.Status
				}
				assert.Equal(t, HealthHealthy, healthByType[CapArchive])
				assert.Equal(t, HealthHealthy, healthByType[CapBookmark])
			},
		},
		{
			name: "unhealthy descriptor",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: false, Instance: "ok"},
			},
			setup: func() func() {
				old := homelab.DefaultRegistry.Permissions()
				return func() { homelab.DefaultRegistry.SetPermissions(old) }
			},
			check: func(t *testing.T, result *HealthResult) {
				require.NotNil(t, result)
				assert.Equal(t, HealthDegraded, result.Status)
				require.Len(t, result.Details, 1)
				assert.Equal(t, HealthUnhealthy, result.Details[0].Status)
			},
		},
		{
			name: "degraded nil instance",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true, Instance: nil},
			},
			setup: func() func() {
				old := homelab.DefaultRegistry.Permissions()
				return func() { homelab.DefaultRegistry.SetPermissions(old) }
			},
			check: func(t *testing.T, result *HealthResult) {
				require.NotNil(t, result)
				assert.Equal(t, HealthDegraded, result.Status)
				require.Len(t, result.Details, 1)
				assert.Equal(t, HealthDegraded, result.Details[0].Status)
			},
		},
		{
			name: "mixed health",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true, Instance: "ok"},
				{Type: CapArchive, Backend: "archivebox", App: "archivebox", Healthy: false, Instance: "ok"},
				{Type: CapReader, Backend: "miniflux", App: "miniflux", Healthy: true, Instance: nil},
			},
			setup: func() func() {
				old := homelab.DefaultRegistry.Permissions()
				return func() { homelab.DefaultRegistry.SetPermissions(old) }
			},
			check: func(t *testing.T, result *HealthResult) {
				require.NotNil(t, result)
				assert.Equal(t, HealthDegraded, result.Status)
				require.Len(t, result.Details, 3)

				healthByType := make(map[CapabilityType]HealthStatus)
				for _, d := range result.Details {
					healthByType[d.Type] = d.Status
				}
				assert.Equal(t, HealthUnhealthy, healthByType[CapArchive])
				assert.Equal(t, HealthHealthy, healthByType[CapBookmark])
				assert.Equal(t, HealthDegraded, healthByType[CapReader])
			},
		},
		{
			name: "includes app statuses",
			descriptors: []Descriptor{
				{Type: CapArchive, Backend: "archivebox", App: "archivebox", Healthy: true, Instance: "ok"},
			},
			setup: func() func() {
				oldList := homelab.DefaultRegistry.List()
				homelab.DefaultRegistry.Replace([]homelab.App{
					{Name: "archivebox", Status: homelab.AppStatusRunning, Health: homelab.HealthHealthy},
					{Name: "karakeep", Status: homelab.AppStatusStopped, Health: homelab.HealthUnhealthy},
				})
				return func() { homelab.DefaultRegistry.Replace(oldList) }
			},
			check: func(t *testing.T, result *HealthResult) {
				require.NotNil(t, result)
				require.Len(t, result.AppStatuses, 2)

				assert.Equal(t, "archivebox", result.AppStatuses[0].Name)
				assert.Equal(t, homelab.AppStatusRunning, result.AppStatuses[0].Status)
				assert.Equal(t, homelab.HealthHealthy, result.AppStatuses[0].Health)

				assert.Equal(t, "karakeep", result.AppStatuses[1].Name)
				assert.Equal(t, homelab.AppStatusStopped, result.AppStatuses[1].Status)
				assert.Equal(t, homelab.HealthUnhealthy, result.AppStatuses[1].Health)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}
			r := NewRegistry()
			for _, d := range tt.descriptors {
				require.NoError(t, r.Register(d))
			}
			checker := NewChecker(r)
			result := checker.Check(t.Context())
			tt.check(t, result)
		})
	}
}

func TestChecker_CheckCapability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		descriptors []Descriptor
		capType     CapabilityType
		wantErr     bool
		errContains string
		check       func(*testing.T, *CapabilityHealth)
	}{
		{
			name: "capability found and healthy",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Description: "bookmark service", Healthy: true, Instance: "ok"},
			},
			capType: CapBookmark,
			check: func(t *testing.T, ch *CapabilityHealth) {
				require.NotNil(t, ch)
				assert.Equal(t, CapBookmark, ch.Type)
				assert.Equal(t, "karakeep", ch.Backend)
				assert.Equal(t, "karakeep", ch.App)
				assert.Equal(t, "bookmark service", ch.Description)
				assert.Equal(t, HealthHealthy, ch.Status)
			},
		},
		{
			name:        "capability not found",
			descriptors: nil,
			capType:     CapBookmark,
			wantErr:     true,
			errContains: "capability bookmark not found",
		},
		{
			name: "capability found but unhealthy",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: false, Instance: "ok"},
			},
			capType: CapBookmark,
			check: func(t *testing.T, ch *CapabilityHealth) {
				require.NotNil(t, ch)
				assert.Equal(t, HealthUnhealthy, ch.Status)
			},
		},
		{
			name: "capability found with nil instance",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true, Instance: nil},
			},
			capType: CapBookmark,
			check: func(t *testing.T, ch *CapabilityHealth) {
				require.NotNil(t, ch)
				assert.Equal(t, HealthDegraded, ch.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			for _, d := range tt.descriptors {
				require.NoError(t, r.Register(d))
			}
			checker := NewChecker(r)
			ch, err := checker.CheckCapability(t.Context(), tt.capType)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, ch)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			tt.check(t, ch)
		})
	}
}
