package ability

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

type stubWebhookConverter struct {
	path string
}

func (s *stubWebhookConverter) WebhookPath() string { return s.path }
func (*stubWebhookConverter) VerifySignature(_ map[string]string, _ []byte) error {
	return nil
}
func (*stubWebhookConverter) Convert(_ []byte, _ map[string]string) ([]types.DataEvent, error) {
	return nil, nil
}

type stubPollingResource struct {
	name     string
	interval time.Duration
	items    []any
	cursor   string
}

func (s *stubPollingResource) ResourceName() string           { return s.name }
func (s *stubPollingResource) DefaultInterval() time.Duration { return s.interval }
func (*stubPollingResource) DiffKey(item any) string {
	if v, ok := item.(string); ok {
		return v
	}
	return ""
}
func (*stubPollingResource) ContentHash(item any) string {
	if v, ok := item.(string); ok {
		return v
	}
	return ""
}
func (*stubPollingResource) CursorField() string { return "id" }
func (s *stubPollingResource) List(_ context.Context, _ string) (PollResult, error) {
	return PollResult{Items: s.items, NextCursor: s.cursor}, nil
}

func TestEventSourceManager_RegisterWebhook(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
	}{
		{
			name:  "register single webhook",
			paths: []string{"github/events"},
		},
		{
			name:  "register multiple webhooks",
			paths: []string{"github/events", "gitea/webhooks", "miniflux/entries"},
		},
		{
			name:  "register webhook with complex path",
			paths: []string{"some-provider/v2/hooks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEventSourceManager(nil, nil, nil)
			for _, path := range tt.paths {
				mgr.RegisterWebhook(&stubWebhookConverter{path: path})
			}
			for _, path := range tt.paths {
				if _, ok := mgr.webhooks[path]; !ok {
					t.Errorf("webhook %s not registered", path)
				}
			}
		})
	}
}

func TestEventSourceManager_RegisterPolling(t *testing.T) {
	tests := []struct {
		name      string
		resources []string
	}{
		{
			name:      "register single polling resource",
			resources: []string{"github/starred"},
		},
		{
			name:      "register multiple polling resources",
			resources: []string{"github/starred", "miniflux/entries", "gitea/issues"},
		},
		{
			name:      "register with custom resource name",
			resources: []string{"custom-provider/resource-type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEventSourceManager(nil, nil, nil)
			for _, name := range tt.resources {
				mgr.RegisterPolling(&stubPollingResource{name: name, interval: 1 * time.Minute})
			}
			for _, name := range tt.resources {
				if _, ok := mgr.pollers[name]; !ok {
					t.Errorf("poller %s not registered", name)
				}
			}
		})
	}
}

func TestEventSourceManager_Start_Empty(t *testing.T) {
	mgr := NewEventSourceManager(nil, nil, nil)
	err := mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("Start on empty manager should succeed: %v", err)
	}
}

func TestEventSourceManager_StartStop(t *testing.T) {
	mgr := NewEventSourceManager(nil, nil, nil)
	mgr.RegisterPolling(&stubPollingResource{
		name: "test/rsrc", interval: time.Hour,
		items: nil, cursor: "",
	})

	err := mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	err = mgr.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
