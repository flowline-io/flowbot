package drone

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

// Build is a Drone CI build created for the deploy repository.
type Build struct {
	Number    int    `json:"number"`
	Namespace string `json:"namespace"`
	Repo      string `json:"repo"`
	URL       string `json:"url,omitempty"`
}

// DeployInput carries the user who should receive the deploy notification.
type DeployInput struct {
	UID types.Uid
}

// Service triggers the deploy repository build.
type Service interface {
	Deploy(ctx context.Context, in DeployInput) (*Build, error)
	HealthCheck(ctx context.Context) (bool, error)
}
