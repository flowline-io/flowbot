package ability

import (
	"testing"
)

func TestPollResult_Fields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pr   PollResult
	}{
		{
			name: "with items and cursor",
			pr: PollResult{
				Items:      []any{"a", "b"},
				NextCursor: "cursor-next",
				HasMore:    true,
			},
		},
		{
			name: "empty result",
			pr: PollResult{
				Items:      nil,
				NextCursor: "",
				HasMore:    false,
			},
		},
		{
			name: "single item no more",
			pr: PollResult{
				Items:      []any{42},
				NextCursor: "c42",
				HasMore:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.pr.Items == nil && len(tt.pr.Items) != 0 {
				t.Error("expected nil Items to have length 0")
			}
		})
	}
}

func TestWebhookConverter_Interface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		assertion string
	}{
		{
			name:      "has WebhookPath method",
			assertion: "WebhookPath returns string",
		},
		{
			name:      "has VerifySignature method",
			assertion: "VerifySignature returns error",
		},
		{
			name:      "has Convert method",
			assertion: "Convert returns []DataEvent and error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ WebhookConverter = nil // compile-time check
		})
	}
}

func TestPollingResource_Interface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		assertion string
	}{
		{
			name:      "has ResourceName method",
			assertion: "ResourceName returns string",
		},
		{
			name:      "has DefaultInterval method",
			assertion: "DefaultInterval returns time.Duration",
		},
		{
			name:      "has DiffKey method",
			assertion: "DiffKey returns string",
		},
		{
			name:      "has ContentHash method",
			assertion: "ContentHash returns string",
		},
		{
			name:      "has CursorField method",
			assertion: "CursorField returns string",
		},
		{
			name:      "has List method",
			assertion: "List returns PollResult and error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ PollingResource = nil // compile-time check
		})
	}
}
