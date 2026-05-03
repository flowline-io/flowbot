package hub

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/types"
)

type HealthResult struct {
	Status      HealthStatus       `json:"status"`
	Timestamp   time.Time          `json:"timestamp"`
	Details     []CapabilityHealth `json:"details,omitempty"`
	AppStatuses []AppHealth        `json:"app_statuses,omitempty"`
}

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type CapabilityHealth struct {
	Type        CapabilityType `json:"capability"`
	Backend     string         `json:"backend"`
	App         string         `json:"app"`
	Status      HealthStatus   `json:"status"`
	Description string         `json:"description,omitempty"`
}

type AppHealth struct {
	Name   string               `json:"name"`
	Status homelab.AppStatus    `json:"status"`
	Health homelab.HealthStatus `json:"health"`
}

type Checker struct {
	registry *Registry
}

func NewChecker(registry *Registry) *Checker {
	return &Checker{registry: registry}
}

func (c *Checker) Check(ctx context.Context) *HealthResult {
	result := &HealthResult{
		Status:    HealthHealthy,
		Timestamp: time.Now(),
	}

	descriptors := c.registry.List()
	for _, desc := range descriptors {
		ch := CapabilityHealth{
			Type:        desc.Type,
			Backend:     desc.Backend,
			App:         desc.App,
			Description: desc.Description,
			Status:      HealthHealthy,
		}

		if !desc.Healthy {
			ch.Status = HealthUnhealthy
			result.Status = HealthDegraded
		} else if desc.Instance == nil {
			ch.Status = HealthDegraded
			if result.Status == HealthHealthy {
				result.Status = HealthDegraded
			}
		}

		result.Details = append(result.Details, ch)
	}

	// Add endpoint-level health for discovered capabilities.
	endpointChecker := NewEndpointHealthChecker(5 * time.Second)
	for _, ch := range endpointChecker.CheckCapabilities(ctx, c.registry) {
		result.Details = append(result.Details, ch)
		if ch.Status != HealthHealthy {
			result.Status = HealthDegraded
		}
	}

	apps := homelab.DefaultRegistry.List()
	for _, app := range apps {
		ah := AppHealth{
			Name:   app.Name,
			Status: app.Status,
			Health: app.Health,
		}
		result.AppStatuses = append(result.AppStatuses, ah)
	}

	if len(result.Details) == 0 {
		result.Status = HealthDegraded
	}

	return result
}

func (c *Checker) CheckCapability(ctx context.Context, capType CapabilityType) (*CapabilityHealth, error) {
	desc, ok := c.registry.Get(capType)
	if !ok {
		return nil, types.Errorf(types.ErrNotFound, "capability %s not found", capType)
	}

	ch := &CapabilityHealth{
		Type:        desc.Type,
		Backend:     desc.Backend,
		App:         desc.App,
		Description: desc.Description,
		Status:      HealthHealthy,
	}

	if !desc.Healthy {
		ch.Status = HealthUnhealthy
	} else if desc.Instance == nil {
		ch.Status = HealthDegraded
	}

	return ch, nil
}
