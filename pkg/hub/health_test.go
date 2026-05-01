package hub

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecker(t *testing.T) {
	registry := NewRegistry()
	checker := NewChecker(registry)
	require.NotNil(t, checker)
}

func TestChecker_CheckEmptyRegistry(t *testing.T) {
	old := homelab.DefaultRegistry.Permissions()
	defer homelab.DefaultRegistry.SetPermissions(old)

	checker := NewChecker(NewRegistry())
	result := checker.Check(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthDegraded, result.Status)
	assert.Empty(t, result.Details)
	assert.False(t, result.Timestamp.IsZero())
}

func TestChecker_CheckHealthyDescriptors(t *testing.T) {
	old := homelab.DefaultRegistry.Permissions()
	defer homelab.DefaultRegistry.SetPermissions(old)

	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Description: "bookmark service", Healthy: true, Instance: "ok"}))
	require.NoError(t, r.Register(Descriptor{Type: CapArchive, Backend: "archivebox", App: "archivebox", Description: "archive service", Healthy: true, Instance: "ok"}))

	checker := NewChecker(r)
	result := checker.Check(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthHealthy, result.Status)
	require.Len(t, result.Details, 2)

	// Sorted by capability type: archive < bookmark
	healthByType := make(map[CapabilityType]HealthStatus)
	for _, d := range result.Details {
		healthByType[d.Type] = d.Status
	}
	assert.Equal(t, HealthHealthy, healthByType[CapArchive])
	assert.Equal(t, HealthHealthy, healthByType[CapBookmark])
}

func TestChecker_CheckUnhealthyDescriptor(t *testing.T) {
	old := homelab.DefaultRegistry.Permissions()
	defer homelab.DefaultRegistry.SetPermissions(old)

	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: false, Instance: "ok"}))

	checker := NewChecker(r)
	result := checker.Check(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthDegraded, result.Status)
	require.Len(t, result.Details, 1)
	assert.Equal(t, HealthUnhealthy, result.Details[0].Status)
}

func TestChecker_CheckDegradedDescriptorNilInstance(t *testing.T) {
	old := homelab.DefaultRegistry.Permissions()
	defer homelab.DefaultRegistry.SetPermissions(old)

	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true, Instance: nil}))

	checker := NewChecker(r)
	result := checker.Check(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthDegraded, result.Status)
	require.Len(t, result.Details, 1)
	assert.Equal(t, HealthDegraded, result.Details[0].Status)
}

func TestChecker_CheckMixedHealth(t *testing.T) {
	old := homelab.DefaultRegistry.Permissions()
	defer homelab.DefaultRegistry.SetPermissions(old)

	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true, Instance: "ok"}))
	require.NoError(t, r.Register(Descriptor{Type: CapArchive, Backend: "archivebox", App: "archivebox", Healthy: false, Instance: "ok"}))
	require.NoError(t, r.Register(Descriptor{Type: CapReader, Backend: "miniflux", App: "miniflux", Healthy: true, Instance: nil}))

	checker := NewChecker(r)
	result := checker.Check(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthDegraded, result.Status)
	require.Len(t, result.Details, 3)

	// Sorted by capability type: archive < bookmark < reader
	healthByType := make(map[CapabilityType]HealthStatus)
	for _, d := range result.Details {
		healthByType[d.Type] = d.Status
	}
	assert.Equal(t, HealthUnhealthy, healthByType[CapArchive])
	assert.Equal(t, HealthHealthy, healthByType[CapBookmark])
	assert.Equal(t, HealthDegraded, healthByType[CapReader])
}

func TestChecker_CheckIncludesAppStatuses(t *testing.T) {
	oldList := homelab.DefaultRegistry.List()
	defer homelab.DefaultRegistry.Replace(oldList)

	homelab.DefaultRegistry.Replace([]homelab.App{
		{Name: "archivebox", Status: homelab.AppStatusRunning, Health: homelab.HealthHealthy},
		{Name: "karakeep", Status: homelab.AppStatusStopped, Health: homelab.HealthUnhealthy},
	})

	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapArchive, Backend: "archivebox", App: "archivebox", Healthy: true, Instance: "ok"}))

	checker := NewChecker(r)
	result := checker.Check(context.Background())

	require.NotNil(t, result)
	require.Len(t, result.AppStatuses, 2)

	assert.Equal(t, "archivebox", result.AppStatuses[0].Name)
	assert.Equal(t, homelab.AppStatusRunning, result.AppStatuses[0].Status)
	assert.Equal(t, homelab.HealthHealthy, result.AppStatuses[0].Health)

	assert.Equal(t, "karakeep", result.AppStatuses[1].Name)
	assert.Equal(t, homelab.AppStatusStopped, result.AppStatuses[1].Status)
	assert.Equal(t, homelab.HealthUnhealthy, result.AppStatuses[1].Health)
}

func TestChecker_CheckCapabilityFound(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Description: "bookmark service", Healthy: true, Instance: "ok"}))

	checker := NewChecker(r)
	ch, err := checker.CheckCapability(context.Background(), CapBookmark)

	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, CapBookmark, ch.Type)
	assert.Equal(t, "karakeep", ch.Backend)
	assert.Equal(t, "karakeep", ch.App)
	assert.Equal(t, "bookmark service", ch.Description)
	assert.Equal(t, HealthHealthy, ch.Status)
}

func TestChecker_CheckCapabilityNotFound(t *testing.T) {
	r := NewRegistry()
	checker := NewChecker(r)

	ch, err := checker.CheckCapability(context.Background(), CapBookmark)

	require.Error(t, err)
	assert.Nil(t, ch)
	assert.Contains(t, err.Error(), "capability bookmark not found")
}

func TestChecker_CheckCapabilityUnhealthy(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: false, Instance: "ok"}))

	checker := NewChecker(r)
	ch, err := checker.CheckCapability(context.Background(), CapBookmark)

	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, HealthUnhealthy, ch.Status)
}

func TestChecker_CheckCapabilityDegradedNilInstance(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true, Instance: nil}))

	checker := NewChecker(r)
	ch, err := checker.CheckCapability(context.Background(), CapBookmark)

	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.Equal(t, HealthDegraded, ch.Status)
}
