package ability

import (
	"testing"
)

func TestPollResult_Fields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		pr          PollResult
		wantLen     int
		wantCursor  string
		wantHasMore bool
	}{
		{
			name: "with items and cursor",
			pr: PollResult{
				Items:      []any{"a", "b"},
				NextCursor: "cursor-next",
				HasMore:    true,
			},
			wantLen:     2,
			wantCursor:  "cursor-next",
			wantHasMore: true,
		},
		{
			name: "empty result",
			pr: PollResult{
				Items:      nil,
				NextCursor: "",
				HasMore:    false,
			},
			wantLen:     0,
			wantCursor:  "",
			wantHasMore: false,
		},
		{
			name: "single item no more",
			pr: PollResult{
				Items:      []any{42},
				NextCursor: "c42",
				HasMore:    false,
			},
			wantLen:     1,
			wantCursor:  "c42",
			wantHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := len(tt.pr.Items); got != tt.wantLen {
				t.Errorf("len(Items) = %d, want %d", got, tt.wantLen)
			}
			if got := tt.pr.NextCursor; got != tt.wantCursor {
				t.Errorf("NextCursor = %q, want %q", got, tt.wantCursor)
			}
			if got := tt.pr.HasMore; got != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", got, tt.wantHasMore)
			}
		})
	}
}

func TestWebhookConverter_Interface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "has WebhookPath method"},
		{name: "has VerifySignature method"},
		{name: "has Convert method"},
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
		name string
	}{
		{name: "has ResourceName method"},
		{name: "has DefaultInterval method"},
		{name: "has DiffKey method"},
		{name: "has ContentHash method"},
		{name: "has CursorField method"},
		{name: "has List method"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ PollingResource = nil // compile-time check
		})
	}
}
