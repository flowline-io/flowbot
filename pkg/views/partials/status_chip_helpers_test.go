package partials

import "testing"

func TestRunsStatusClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{name: "success", status: 2, want: "flowbot-chip flowbot-chip-success"},
		{name: "failed", status: 4, want: "flowbot-chip flowbot-chip-error"},
		{name: "cancelled", status: 3, want: "flowbot-chip flowbot-chip-warning"},
		{name: "started default", status: 0, want: "flowbot-chip flowbot-chip-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := runsStatusClass(tt.status); got != tt.want {
				t.Fatalf("runsStatusClass(%d) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatusBadgeClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "success", status: "success", want: "flowbot-chip flowbot-chip-success"},
		{name: "failed", status: "failed", want: "flowbot-chip flowbot-chip-error"},
		{name: "other muted", status: "pending", want: "flowbot-chip flowbot-chip-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := statusBadgeClass(tt.status); got != tt.want {
				t.Fatalf("statusBadgeClass(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestActionBadgeClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		action string
		want   string
	}{
		{name: "throttle warning", action: "throttle", want: "flowbot-chip flowbot-chip-warning"},
		{name: "aggregate primary", action: "aggregate", want: "flowbot-chip flowbot-chip-primary"},
		{name: "drop error", action: "drop", want: "flowbot-chip flowbot-chip-error"},
		{name: "mute muted", action: "mute", want: "flowbot-chip flowbot-chip-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := actionBadgeClass(tt.action); got != tt.want {
				t.Fatalf("actionBadgeClass(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}
