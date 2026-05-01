package infra

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type HostQuery struct {
	Page   ability.PageRequest
	Status string
}

type MetricQuery struct {
	HostID string
	From   time.Time
	To     time.Time
	Step   time.Duration
}

type MetricPoint struct {
	Time  time.Time `json:"time"`
	Name  string    `json:"name"`
	Value float64   `json:"value"`
}

type Service interface {
	ListHosts(ctx context.Context, q *HostQuery) (*ability.ListResult[ability.Host], error)
	GetHost(ctx context.Context, id string) (*ability.Host, error)
	GetMetrics(ctx context.Context, q *MetricQuery) ([]MetricPoint, error)
}
